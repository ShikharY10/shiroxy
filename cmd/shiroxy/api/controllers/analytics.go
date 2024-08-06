package controllers

import (
	"reflect"
	"shiroxy/cmd/shiroxy/analytics"
	"shiroxy/cmd/shiroxy/api/middlewares"
	"shiroxy/cmd/shiroxy/domains"
	"shiroxy/cmd/shiroxy/proxy"
	"shiroxy/cmd/shiroxy/webhook"

	"github.com/gin-gonic/gin"
)

type AnalyticsController struct {
	Storage          *domains.Storage
	AnalyticsHandler *analytics.AnalyticsConfiguration
	Middlewares      *middlewares.Middlewares
	WebhookHandler   *webhook.WebhookHandler
	HealthChecker    *proxy.HealthChecker
}

func (a *AnalyticsController) FetchDomainAnalytics(c *gin.Context) {
	response := map[string]interface{}{
		"total":          len(a.Storage.DomainMetadata),
		"total_active":   0,
		"total_inactive": 0,
	}

	for _, domain := range a.Storage.DomainMetadata {
		if domain.Status == "active" {
			if response["total_active"] != nil {
				response["total_active"] = (response["total_active"].(int) + 1)
			} else {
				response["total_active"] = 1
			}
		} else {
			if response["total_inactive"] != nil {
				response["total_inactive"] = (response["total_active"].(int) + 1)
			} else {
				response["total_inactive"] = 1
			}
		}
	}

	a.Middlewares.WriteResponse(c, middlewares.ApiResponse{
		Success: true,
		Data:    response,
	}, 200)
}

func (a *AnalyticsController) FetchSystemAnalytics(c *gin.Context) {
	response := map[string]interface{}{}

	analytics, err := a.AnalyticsHandler.ReadAnalytics(true)
	if err != nil {
		a.Middlewares.WriteResponse(c, middlewares.ApiResponse{
			Success: false,
			Data:    response,
		}, 500)
	}

	if analytics == nil {
		a.Middlewares.WriteResponse(c, middlewares.ApiResponse{
			Success: true,
			Data:    response,
		}, 200)
		return
	}

	analyticsReflectValue := reflect.ValueOf(analytics)
	if analyticsReflectValue.Kind() == reflect.Ptr {
		analyticsReflectValue = analyticsReflectValue.Elem()
	}
	for i := 0; i < analyticsReflectValue.NumField(); i++ {
		response[analyticsReflectValue.Type().Field(i).Name] = analyticsReflectValue.Field(i).Interface()
	}

	a.Middlewares.WriteResponse(c, middlewares.ApiResponse{
		Success: true,
		Data:    response,
	}, 200)
}

func (a *AnalyticsController) FetchServerDetails(c *gin.Context) {
	response := map[string]interface{}{}

	var servers []map[string]any = []map[string]any{}
	for _, server := range a.HealthChecker.Servers {
		serverReflect := reflect.ValueOf(server)
		if serverReflect.Kind() == reflect.Ptr {
			serverReflect = serverReflect.Elem()
		}
		serverJson := map[string]any{}
		for i := 0; i < serverReflect.NumField(); i++ {
			serverJson[serverReflect.Type().Field(i).Name] = serverReflect.Field(i).Interface()
		}

		servers = append(servers, serverJson)
	}

	response["servers"] = servers

	a.Middlewares.WriteResponse(c, middlewares.ApiResponse{
		Success: true,
		Data:    response,
	}, 200)
}
