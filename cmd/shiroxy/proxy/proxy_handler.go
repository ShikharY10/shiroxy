package proxy

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"shiroxy/cmd/shiroxy/domains"
	"shiroxy/pkg/logger"
	"shiroxy/pkg/models"
	"shiroxy/public"
	"strings"
	"sync"
)

func StartShiroxyHandler(configuration *models.Config, storage *domains.Storage, logHandler *logger.Logger, wg *sync.WaitGroup) (*LoadBalancer, error) {

	var servers []*Server
	for _, server := range configuration.Backend.Servers {
		host := url.URL{
			Scheme: configuration.Default.Mode,
			Host:   fmt.Sprintf("%s:%s", server.Host, server.Port), // The actual address where domain1's server is running
		}

		servers = append(servers, &Server{
			URL:   &host,
			Alive: true,
			Shiroxy: &Shiroxy{
				Logger: logHandler,
				Director: func(req *http.Request) {
					rewriteRequestURL(req, &host)
				},
			},
		})
	}

	loadbalancer := NewLoadBalancer(configuration, servers, wg)
	domainNotFoundErrorResponse := loadErrorPageHtmlContent(public.DOMAIN_NOT_FOUND_ERROR, &configuration.Default.ErrorResponses)

	for _, bind := range configuration.Frontend.Bind {
		if bind.Target == "single" && bind.SecureSetting.SingleTargetMode == "" {
			logHandler.Log("securesetting field is required when bind target is set to 'single'", "Proxy", "Error")
			panic("")
		}

		frontend := Frontends{
			handlerFunc: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				defer func() {
					if rec := recover(); rec != nil {
						logHandler.LogError(fmt.Sprintf("Recovered from panic: %v", rec), "Proxy", "Error")
						http.Error(w, domainNotFoundErrorResponse, http.StatusInternalServerError)
					}
				}()
				if (configuration.Frontend.HttpToHttps) && r.URL.Port() == "80" && r.TLS == nil {
					secureFrontend := loadbalancer.Frontends["443"]
					if secureFrontend != nil {
						url := fmt.Sprintf("https://%s%s", r.Host, r.RequestURI)
						http.Redirect(w, r, url, http.StatusMovedPermanently)
					} else {
						loadbalancer.ServeHTTP(w, r)
					}
				} else {
					loadbalancer.ServeHTTP(w, r)
				}
			}),
		}
		loadbalancer.Frontends[bind.Port] = &frontend

		if bind.Target != "multiple" && bind.Target != "single" {
			logHandler.Log("Invalid target value in frontend configuraton", "Proxy", "Error")
			panic("")
		}

		var server *http.Server
		var secure bool
		var err error

		if bind.Target == "multiple" {
			server, secure, err = createMultipleTargetServer(&bind, storage, frontend.handlerFunc, logHandler, wg)
		} else if bind.Target == "single" {
			fmt.Println("Single mode Reached =========")
			server, secure, err = createSingleTargetServer(&bind, storage, frontend.handlerFunc, logHandler, wg)
		}

		if err != nil {
			return nil, err
		}

		listenAndServe(server, secure, logHandler, wg)
	}
	return loadbalancer, nil
}

func createMultipleTargetServer(bindData *models.FrontendBind, storage *domains.Storage, handlerFunc http.HandlerFunc, logHandler *logger.Logger, wg *sync.WaitGroup) (server *http.Server, secure bool, err error) {
	if bindData.Secure {
		server := &http.Server{
			Addr:    fmt.Sprintf("%s:%s", bindData.Host, bindData.Port),
			Handler: http.HandlerFunc(handlerFunc),
			TLSConfig: &tls.Config{
				ClientAuth: resolveSecurityPolicy(bindData.SecureSetting.SecureVerify),
				GetCertificate: func(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
					var cert tls.Certificate
					var err error

					var domainName string = strings.TrimSpace(info.ServerName)
					domainMetadata := storage.DomainMetadata[domainName]

					if domainMetadata == nil {
						return nil, fmt.Errorf("domain not found")
					}

					if domainMetadata.Status == "active" {
						cert, err = tls.X509KeyPair(domainMetadata.CertPemBlock, domainMetadata.KeyPemBlock)
						if err != nil {
							fmt.Println("tls.X509KeyPair ERROR: ", err.Error())
							return nil, fmt.Errorf("something went wrong")
						}
					} else {
						return nil, fmt.Errorf("routing deactivated")
					}

					return &cert, nil
				},
			},
		}

		return server, true, nil
	} else {
		server := &http.Server{
			Addr:    fmt.Sprintf("%s:%s", bindData.Host, bindData.Port),
			Handler: handlerFunc,
		}
		return server, false, nil
	}
}

func createSingleTargetServer(bindData *models.FrontendBind, storage *domains.Storage, handlerFunc http.HandlerFunc, logHandler *logger.Logger, wg *sync.WaitGroup) (server *http.Server, secure bool, err error) {
	if bindData.Secure {
		fmt.Println("single and secured")
		var tlsConfig *tls.Config
		if bindData.SecureSetting.SingleTargetMode == "certandkey" {
			tlsConfig = &tls.Config{
				ClientAuth: resolveSecurityPolicy(bindData.SecureSetting.SecureVerify),
				ServerName: bindData.SecureSetting.CertAndKey.Domain,
				GetCertificate: func(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
					cert, err := tls.LoadX509KeyPair(bindData.SecureSetting.CertAndKey.Cert, bindData.SecureSetting.CertAndKey.Key)
					if err != nil {
						return nil, err
					}
					return &cert, nil
				},
			}
		} else if bindData.SecureSetting.SingleTargetMode == "shiroxyshinglesecure" {
			tlsConfig = &tls.Config{
				ClientAuth: resolveSecurityPolicy(bindData.SecureSetting.SecureVerify),
				ServerName: bindData.SecureSetting.CertAndKey.Domain,
				GetCertificate: func(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
					domainMetadata, ok := storage.DomainMetadata[info.ServerName]
					if !ok {
						return nil, errors.New("certificate not found")
					}
					if domainMetadata.Status == "active" {
						cert, err := tls.X509KeyPair(domainMetadata.CertPemBlock, domainMetadata.KeyPemBlock)
						if err != nil {
							fmt.Println("tls.X509KeyPair ERROR: ", err.Error())
							return nil, fmt.Errorf("something went wrong")
						}
						return &cert, nil
					} else {
						return nil, fmt.Errorf("routing deactivated")
					}
				},
			}
		}
		server := &http.Server{
			Addr:      fmt.Sprintf("%s:%s", bindData.Host, bindData.Port),
			Handler:   handlerFunc,
			TLSConfig: tlsConfig,
		}

		return server, true, nil
	} else {
		fmt.Println("single and unsecured")
		server := &http.Server{
			Addr:    fmt.Sprintf("%s:%s", bindData.Host, bindData.Port),
			Handler: handlerFunc,
		}

		return server, false, nil
	}
}

func resolveSecurityPolicy(policy string) tls.ClientAuthType {
	if policy == "none" {
		return tls.NoClientCert
	} else if policy == "optional" {
		return tls.RequestClientCert
	} else if policy == "required" {
		return tls.RequireAndVerifyClientCert
	} else {
		return tls.NoClientCert
	}
}

func loadErrorPageHtmlContent(htmlContent string, config *models.ErrorRespons) string {

	if config.ErrorPageButtonName == "" {
		config.ErrorPageButtonName = "Shiroxy"
	}
	if config.ErrorPageButtonUrl == "" {
		config.ErrorPageButtonUrl = "https://github.com/ShikharY10/shiroxy/issues"
	}

	replacer := strings.NewReplacer(
		"{{button_name}}", config.ErrorPageButtonName,
		"{{button_url}}", config.ErrorPageButtonUrl,
	)

	result := replacer.Replace(htmlContent)

	return result
}

func listenAndServe(server *http.Server, secure bool, logHandler *logger.Logger, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		if secure {
			err := server.ListenAndServeTLS("", "")
			if err != nil {
				logHandler.LogError(fmt.Sprintf("while starting secured server: %s", err.Error()), "Proxy", "Error")
			}
		} else {
			err := server.ListenAndServe()
			if err != nil {
				logHandler.LogError(fmt.Sprintf("while starting unsecured server: %s", err.Error()), "Proxy", "Error")
			}
		}
	}()
}
