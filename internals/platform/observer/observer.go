package observer

import (
	"log"

	"github.com/labstack/echo/v4"
	"github.com/yca-software/2chi-go-api/internals/config"
	chi_observer "github.com/yca-software/2chi-go-observer"
)

type Observer struct {
	Base *chi_observer.Observer
}

func New(cfg *config.Config) *Observer {
	return NewForApp(cfg, "api")
}

func NewForApp(cfg *config.Config, appName string) *Observer {
	base, err := chi_observer.New(chi_observer.ObserverConfig{
		Namespace: cfg.App.Namespace,
		AppName:   appName,
	})
	if err != nil {
		log.Fatalf("failed to create observer: %v", err)
	}

	return &Observer{
		Base: base,
	}
}

func (m *Observer) EchoMiddleware(skipper func(echo.Context) bool) echo.MiddlewareFunc {
	return m.Base.EchoMiddleware(skipper)
}

func (m *Observer) GetQueryMetricsHook() chi_observer.QueryMetricsHook {
	return m.Base.GetQueryMetricsHook()
}

func (m *Observer) RecordRateLimitHit(method, route, principalType, principal, ip string) {
	m.Base.RecordRateLimitHit(method, route, principalType, principal, ip)
}

func (m *Observer) GetJobMetricsHook() chi_observer.JobMetricsHook {
	return m.Base.GetJobMetricsHook()
}
