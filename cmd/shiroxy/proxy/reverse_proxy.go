package proxy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/textproto"
	"net/url"
	"shiroxy/pkg/logger"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/http/httpguts"
)

// A BufferPool is an interface for getting and returning temporary
// byte slices for use by io.CopyBuffer.
type BufferPool interface {
	Get() []byte
	Put([]byte)
}

// Shiroxy is an HTTP Handler that takes an incoming request and
// sends it to another server, proxying the response back to the
// client.
//
// 1xx responses are forwarded to the client if the underlying
// transport supports ClientTrace.Got1xxResponse.
type Shiroxy struct {
	// Logger is utility for logging, it can log to terminal,
	// file and also to remote source using syslog protocol.
	Logger *logger.Logger

	// Rewrite must be a function which modifies
	// the request into a new request to be sent
	// using Transport. Its response is then copied
	// back to the original client unmodified.
	// Rewrite must not access the provided ProxyRequest
	// or its contents after returning.
	//
	// The Forwarded, X-Forwarded, X-Forwarded-Host,
	// and X-Forwarded-Proto headers are removed from the
	// outbound request before Rewrite is called. See also
	// the ProxyRequest.SetXForwarded method.
	//
	// Unparsable query parameters are removed from the
	// outbound request before Rewrite is called.
	// The Rewrite function may copy the inbound URL's
	// RawQuery to the outbound URL to preserve the original
	// parameter string. Note that this can lead to security
	// issues if the proxy's interpretation of query parameters
	// does not match that of the downstream server.
	//
	// At most one of Rewrite or Director may be set.
	Rewrite func(*ProxyRequest)

	// Director is a function which modifies
	// the request into a new request to be sent
	// using Transport. Its response is then copied
	// back to the original client unmodified.
	// Director must not access the provided Request
	// after returning.
	//
	// By default, the X-Forwarded-For header is set to the
	// value of the client IP address. If an X-Forwarded-For
	// header already exists, the client IP is appended to the
	// existing values. As a special case, if the header
	// exists in the Request.Header map but has a nil value
	// (such as when set by the Director func), the X-Forwarded-For
	// header is not modified.
	//
	// To prevent IP spoofing, be sure to delete any pre-existing
	// X-Forwarded-For header coming from the client or
	// an untrusted proxy.
	//
	// Hop-by-hop headers are removed from the request after
	// Director returns, which can remove headers added by
	// Director. Use a Rewrite function instead to ensure
	// modifications to the request are preserved.
	//
	// Unparsable query parameters are removed from the outbound
	// request if Request.Form is set after Director returns.
	//
	// At most one of Rewrite or Director may be set.
	Director func(*http.Request)

	// The transport used to perform proxy requests.
	// If nil, http.DefaultTransport is used.
	Transport http.RoundTripper

	// FlushInterval specifies the flush interval
	// to flush to the client while copying the
	// response body.
	// If zero, no periodic flushing is done.
	// A negative value means to flush immediately
	// after each write to the client.
	// The FlushInterval is ignored when Shiroxy
	// recognizes a response as a streaming response, or
	// if its ContentLength is -1; for such responses, writes
	// are flushed to the client immediately.
	FlushInterval time.Duration

	// ErrorLog specifies an optional logger for errors
	// that occur when attempting to proxy the request.
	// If nil, logging is done via the log package's standard logger.
	ErrorLog *log.Logger

	// BufferPool optionally specifies a buffer pool to
	// get byte slices for use by io.CopyBuffer when
	// copying HTTP response bodies.
	BufferPool BufferPool

	// ModifyResponse is an optional function that modifies the
	// Response from the backend. It is called if the backend
	// returns a response at all, with any HTTP status code.
	// If the backend is unreachable, the optional ErrorHandler is
	// called without any call to ModifyResponse.
	//
	// If ModifyResponse returns an error, ErrorHandler is called
	// with its error value. If ErrorHandler is nil, its default
	// implementation is used.
	ModifyResponse func(*http.Response) error

	// ErrorHandler is an optional function that handles errors
	// reaching the backend or errors from ModifyResponse.
	//
	// If nil, the default is to log the provided error and return
	// a 502 Status Bad Gateway response.
	ErrorHandler func(http.ResponseWriter, *http.Request, error)
}

// Hop-by-hop headers. These are removed when sent to the backend.
// As of RFC 7230, hop-by-hop headers are required to appear in the
// Connection header field. These are the headers defined by the
// obsoleted RFC 2616 (section 13.5.1) and are used for backward
// compatibility.
var hopHeaders = []string{
	"Connection",
	"Proxy-Connection", // non-standard but still sent by libcurl and rejected by e.g. google
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Te",      // canonicalized version of "TE"
	"Trailer", // not Trailers per URL above; https://www.rfc-editor.org/errata_search.php?eid=4522
	"Transfer-Encoding",
	"Upgrade",
}

func (p *Shiroxy) DefaultErrorHandler(rw http.ResponseWriter, req *http.Request, err error) {
	// p.logf("http: shiroxy error: %v", err)
	p.Logger.LogError(err.Error(), "Shiroxy", "Error")
	rw.WriteHeader(http.StatusBadGateway)
}

func (p *Shiroxy) getErrorHandler() func(http.ResponseWriter, *http.Request, error) {
	if p.ErrorHandler != nil {
		return p.ErrorHandler
	}
	return p.DefaultErrorHandler
}

// modifyResponse conditionally runs the optional ModifyResponse hook
// and reports whether the request should proceed.
func (p *Shiroxy) modifyResponse(rw http.ResponseWriter, res *http.Response, req *http.Request) bool {
	if p.ModifyResponse == nil {
		return true
	}
	if err := p.ModifyResponse(res); err != nil {
		res.Body.Close()
		p.getErrorHandler()(rw, req, err)
		return false
	}
	return true
}

func (p *Shiroxy) ServeHTTP(rw http.ResponseWriter, req *ShiroxyRequest) error {
	transport := p.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}

	// ====== Implement a new way for handling unexpected closing of http requests.
	ctx := req.Request.Context()
	if ctx.Done() != nil {
		// CloseNotifier predates context.Context, and has been
		// entirely superseded by it. If the request contains
		// a Context that carries a cancellation signal, don't
		// bother spinning up a goroutine to watch the CloseNotify
		// channel (if any).
		//
		// If the request Context has a nil Done channel (which
		// means it is either context.Background, or a custom
		// Context implementation with no cancellation signal),
		// then consult the CloseNotifier if available.
	} else if cn, ok := rw.(http.CloseNotifier); ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithCancel(ctx)
		defer cancel()
		notifyChan := cn.CloseNotify()
		go func() {
			select {
			case <-notifyChan:
				cancel()
			case <-ctx.Done():
			}
		}()
	}
	// ================================================================

	outreq := req.Request.Clone(ctx)
	if req.Request.ContentLength == 0 {
		outreq.Body = nil // Issue 16036: nil Body for http.Transport retries
	}
	if outreq.Body != nil {
		// Reading from the request body after returning from a handler is not
		// allowed, and the RoundTrip goroutine that reads the Body can outlive
		// this handler. This can lead to a crash if the handler panics (see
		// Issue 46866). Although calling Close doesn't guarantee there isn't
		// any Read in flight after the handle returns, in practice it's safe to
		// read after closing it.
		defer outreq.Body.Close()
	}
	if outreq.Header == nil {
		outreq.Header = make(http.Header) // Issue 33142: historical behavior was to always allocate
	}

	if (p.Director != nil) == (p.Rewrite != nil) {
		dErr := errors.New("Shiroxy must have exactly one of Director or Rewrite set")
		p.getErrorHandler()(rw, req.Request, dErr)
		return dErr
	}

	if p.Director != nil {
		p.Director(outreq)
		if outreq.Form != nil {
			outreq.URL.RawQuery = cleanQueryParams(outreq.URL.RawQuery)
		}
	}
	outreq.Close = false

	reqUpType := upgradeType(outreq.Header)

	if !IsPrint(reqUpType) {
		pErr := fmt.Errorf("client tried to switch to invalid protocol %q", reqUpType)
		p.getErrorHandler()(rw, req.Request, pErr)
		return pErr
	}
	removeHopByHopHeaders(outreq.Header)

	// Issue 21096: tell backend applications that care about trailer support
	// that we support trailers. (We do, but we don't go out of our way to
	// advertise that unless the incoming client request thought it was worth
	// mentioning.) Note that we look at req.Header, not outreq.Header, since
	// the latter has passed through removeHopByHopHeaders.
	if httpguts.HeaderValuesContainsToken(req.Request.Header["Te"], "trailers") {
		outreq.Header.Set("Te", "trailers")
	}

	// After stripping all the hop-by-hop connection headers above, add back any
	// necessary for protocol upgrades, such as for websockets.
	if reqUpType != "" {
		outreq.Header.Set("Connection", "Upgrade")
		outreq.Header.Set("Upgrade", reqUpType)
	}

	if p.Rewrite != nil {
		// Strip client-provided forwarding headers.
		// The Rewrite func may use SetXForwarded to set new values
		// for these or copy the previous values from the inbound request.
		outreq.Header.Del("Forwarded")
		outreq.Header.Del("X-Forwarded-For")
		outreq.Header.Del("X-Forwarded-Host")
		outreq.Header.Del("X-Forwarded-Proto")

		// Remove unparsable query parameters from the outbound request.
		outreq.URL.RawQuery = cleanQueryParams(outreq.URL.RawQuery)

		pr := &ProxyRequest{
			In:  req.Request,
			Out: outreq,
		}
		p.Rewrite(pr)
		outreq = pr.Out
	} else {
		if clientIP, _, err := net.SplitHostPort(req.Request.RemoteAddr); err == nil {
			// If we aren't the first proxy retain prior
			// X-Forwarded-For information as a comma+space
			// separated list and fold multiple headers into one.
			prior, ok := outreq.Header["X-Forwarded-For"]
			omit := ok && prior == nil // Issue 38079: nil now means don't populate the header
			if len(prior) > 0 {
				clientIP = strings.Join(prior, ", ") + ", " + clientIP
			}
			if !omit {
				outreq.Header.Set("X-Forwarded-For", clientIP)
			}
		}
	}

	if _, ok := outreq.Header["User-Agent"]; !ok {
		// If the outbound request doesn't have a User-Agent header set,
		// don't send the default Go HTTP client User-Agent.
		outreq.Header.Set("User-Agent", "")
	}

	trace := &httptrace.ClientTrace{
		Got1xxResponse: func(code int, header textproto.MIMEHeader) error {
			h := rw.Header()
			copyHeader(h, http.Header(header))
			rw.WriteHeader(code)

			// Clear headers, it's not automatically done by ResponseWriter.WriteHeader() for 1xx responses
			clear(h)
			return nil
		},
	}
	outreq = outreq.WithContext(httptrace.WithClientTrace(outreq.Context(), trace))

	res, err := transport.RoundTrip(outreq)
	if err != nil {
		// p.getErrorHandler()(rw, outreq, err)
		return err
	}

	// Deal with 101 Switching Protocols responses: (WebSocket, h2c, etc)
	if res.StatusCode == http.StatusSwitchingProtocols {
		if !p.modifyResponse(rw, res, outreq) {
			return nil
		}
		p.handleUpgradeResponse(rw, outreq, res)
		return nil
	}

	removeHopByHopHeaders(res.Header)

	if !p.modifyResponse(rw, res, outreq) {
		return nil
	}

	copyHeader(rw.Header(), res.Header)

	// The "Trailer" header isn't included in the Transport's response,
	// at least for *http.Transport. Build it up from Trailer.
	announcedTrailers := len(res.Trailer)
	if announcedTrailers > 0 {
		trailerKeys := make([]string, 0, len(res.Trailer))
		for k := range res.Trailer {
			trailerKeys = append(trailerKeys, k)
		}
		rw.Header().Add("Trailer", strings.Join(trailerKeys, ", "))
	}

	rw.WriteHeader(res.StatusCode)

	err = p.copyResponse(rw, res.Body, p.flushInterval(res))
	if err != nil {
		defer res.Body.Close()
		// Since we're streaming the response, if we run into an error all we can do
		// is abort the request. Issue 23643: Shiroxy should use ErrAbortHandler
		// on read error while copying body.
		if !shouldPanicOnCopyError(req.Request) {
			p.logf("suppressing panic for copyResponse error in test; copy error: %v", err)
			return nil
		}
		panic(http.ErrAbortHandler)
	}
	res.Body.Close() // close now, instead of defer, to populate res.Trailer

	if len(res.Trailer) > 0 {
		// Force chunking if we saw a response trailer.
		// This prevents net/http from calculating the length for short
		// bodies and adding a Content-Length.
		http.NewResponseController(rw).Flush()
	}

	if len(res.Trailer) == announcedTrailers {
		copyHeader(rw.Header(), res.Trailer)
		return nil
	}

	for k, vv := range res.Trailer {
		k = http.TrailerPrefix + k
		for _, v := range vv {
			rw.Header().Add(k, v)
		}
	}
	return nil
}

var inOurTests bool // whether we're in our own tests

// shouldPanicOnCopyError reports whether the reverse proxy should
// panic with http.ErrAbortHandler. This is the right thing to do by
// default, but Go 1.10 and earlier did not, so existing unit tests
// weren't expecting panics. Only panic in our own tests, or when
// running under the HTTP server.
func shouldPanicOnCopyError(req *http.Request) bool {
	if inOurTests {
		// Our tests know to handle this panic.
		return true
	}
	if req.Context().Value(http.ServerContextKey) != nil {
		// We seem to be running under an HTTP server, so
		// it'll recover the panic.
		return true
	}
	// Otherwise act like Go 1.10 and earlier to not break
	// existing tests.
	return false
}

// removeHopByHopHeaders removes hop-by-hop headers.
func removeHopByHopHeaders(h http.Header) {
	// RFC 7230, section 6.1: Remove headers listed in the "Connection" header.
	for _, f := range h["Connection"] {
		for _, sf := range strings.Split(f, ",") {
			if sf = textproto.TrimString(sf); sf != "" {
				h.Del(sf)
			}
		}
	}
	// RFC 2616, section 13.5.1: Remove a set of known hop-by-hop headers.
	// This behavior is superseded by the RFC 7230 Connection header, but
	// preserve it for backwards compatibility.
	for _, f := range hopHeaders {
		h.Del(f)
	}
}

// flushInterval returns the p.FlushInterval value, conditionally
// overriding its value for a specific request/response.
func (p *Shiroxy) flushInterval(res *http.Response) time.Duration {
	resCT := res.Header.Get("Content-Type")

	// For Server-Sent Events responses, flush immediately.
	// The MIME type is defined in https://www.w3.org/TR/eventsource/#text-event-stream
	if baseCT, _, _ := mime.ParseMediaType(resCT); baseCT == "text/event-stream" {
		return -1 // negative means immediately
	}

	// We might have the case of streaming for which Content-Length might be unset.
	if res.ContentLength == -1 {
		return -1
	}

	return p.FlushInterval
}

func (p *Shiroxy) copyResponse(dst http.ResponseWriter, src io.Reader, flushInterval time.Duration) error {
	var w io.Writer = dst

	if flushInterval != 0 {
		mlw := &maxLatencyWriter{
			dst:     dst,
			flush:   http.NewResponseController(dst).Flush,
			latency: flushInterval,
		}
		defer mlw.stop()

		// set up initial timer so headers get flushed even if body writes are delayed
		mlw.flushPending = true
		mlw.t = time.AfterFunc(flushInterval, mlw.delayedFlush)

		w = mlw
	}

	var buf []byte
	if p.BufferPool != nil {
		buf = p.BufferPool.Get()
		defer p.BufferPool.Put(buf)
	}
	_, err := p.copyBuffer(w, src, buf)
	return err
}

// copyBuffer returns any write errors or non-EOF read errors, and the amount
// of bytes written.
func (p *Shiroxy) copyBuffer(dst io.Writer, src io.Reader, buf []byte) (int64, error) {
	if len(buf) == 0 {
		buf = make([]byte, 32*1024)
	}
	var written int64
	for {
		nr, rerr := src.Read(buf)
		if rerr != nil && rerr != io.EOF && rerr != context.Canceled {
			p.logf("httputil: Shiroxy read error during body copy: %v", rerr)
		}
		if nr > 0 {
			nw, werr := dst.Write(buf[:nr])
			if nw > 0 {
				written += int64(nw)
			}
			if werr != nil {
				return written, werr
			}
			if nr != nw {
				return written, io.ErrShortWrite
			}
		}
		if rerr != nil {
			if rerr == io.EOF {
				rerr = nil
			}
			return written, rerr
		}
	}
}

func (p *Shiroxy) logf(format string, args ...any) {
	if p.ErrorLog != nil {
		p.ErrorLog.Printf(format, args...)
	} else {
		log.Printf(format, args...)
	}
}

type maxLatencyWriter struct {
	dst     io.Writer
	flush   func() error
	latency time.Duration // non-zero; negative means to flush immediately

	mu           sync.Mutex // protects t, flushPending, and dst.Flush
	t            *time.Timer
	flushPending bool
}

func (m *maxLatencyWriter) Write(p []byte) (n int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	n, err = m.dst.Write(p)
	if m.latency < 0 {
		m.flush()
		return
	}
	if m.flushPending {
		return
	}
	if m.t == nil {
		m.t = time.AfterFunc(m.latency, m.delayedFlush)
	} else {
		m.t.Reset(m.latency)
	}
	m.flushPending = true
	return
}

func (m *maxLatencyWriter) delayedFlush() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.flushPending { // if stop was called but AfterFunc already started this goroutine
		return
	}
	m.flush()
	m.flushPending = false
}

func (m *maxLatencyWriter) stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.flushPending = false
	if m.t != nil {
		m.t.Stop()
	}
}

func upgradeType(h http.Header) string {
	if !httpguts.HeaderValuesContainsToken(h["Connection"], "Upgrade") {
		return ""
	}
	return h.Get("Upgrade")
}

func (p *Shiroxy) handleUpgradeResponse(rw http.ResponseWriter, req *http.Request, res *http.Response) {
	reqUpType := upgradeType(req.Header)
	resUpType := upgradeType(res.Header)
	if !IsPrint(resUpType) { // We know reqUpType is ASCII, it's checked by the caller.
		p.getErrorHandler()(rw, req, fmt.Errorf("backend tried to switch to invalid protocol %q", resUpType))
	}
	if !EqualFold(reqUpType, resUpType) {
		p.getErrorHandler()(rw, req, fmt.Errorf("backend tried to switch protocol %q when %q was requested", resUpType, reqUpType))
		return
	}

	backConn, ok := res.Body.(io.ReadWriteCloser)
	if !ok {
		p.getErrorHandler()(rw, req, fmt.Errorf("internal error: 101 switching protocols response with non-writable body"))
		return
	}

	rc := http.NewResponseController(rw)
	conn, brw, hijackErr := rc.Hijack()
	if errors.Is(hijackErr, http.ErrNotSupported) {
		p.getErrorHandler()(rw, req, fmt.Errorf("can't switch protocols using non-Hijacker ResponseWriter type %T", rw))
		return
	}

	backConnCloseCh := make(chan bool)
	go func() {
		// Ensure that the cancellation of a request closes the backend.
		// See issue https://golang.org/issue/35559.
		select {
		case <-req.Context().Done():
		case <-backConnCloseCh:
		}
		backConn.Close()
	}()
	defer close(backConnCloseCh)

	if hijackErr != nil {
		p.getErrorHandler()(rw, req, fmt.Errorf("Hijack failed on protocol switch: %v", hijackErr))
		return
	}
	defer conn.Close()

	copyHeader(rw.Header(), res.Header)

	res.Header = rw.Header()
	res.Body = nil // so res.Write only writes the headers; we have res.Body in backConn above
	if err := res.Write(brw); err != nil {
		p.getErrorHandler()(rw, req, fmt.Errorf("response write: %v", err))
		return
	}
	if err := brw.Flush(); err != nil {
		p.getErrorHandler()(rw, req, fmt.Errorf("response flush: %v", err))
		return
	}
	errc := make(chan error, 1)
	spc := switchProtocolCopier{user: conn, backend: backConn}
	go spc.copyToBackend(errc)
	go spc.copyFromBackend(errc)
	<-errc
}

// switchProtocolCopier exists so goroutines proxying data back and
// forth have nice names in stacks.
type switchProtocolCopier struct {
	user, backend io.ReadWriter
}

func (c switchProtocolCopier) copyFromBackend(errc chan<- error) {
	_, err := io.Copy(c.user, c.backend)
	errc <- err
}

func (c switchProtocolCopier) copyToBackend(errc chan<- error) {
	_, err := io.Copy(c.backend, c.user)
	errc <- err
}

func cleanQueryParams(s string) string {
	reencode := func(s string) string {
		v, _ := url.ParseQuery(s)
		return v.Encode()
	}
	for i := 0; i < len(s); {
		switch s[i] {
		case ';':
			return reencode(s)
		case '%':
			if i+2 >= len(s) || !ishex(s[i+1]) || !ishex(s[i+2]) {
				return reencode(s)
			}
			i += 3
		default:
			i++
		}
	}
	return s
}

func ishex(c byte) bool {
	switch {
	case '0' <= c && c <= '9':
		return true
	case 'a' <= c && c <= 'f':
		return true
	case 'A' <= c && c <= 'F':
		return true
	}
	return false
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}