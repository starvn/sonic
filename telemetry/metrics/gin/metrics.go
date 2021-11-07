/*
 * Copyright (c) 2021 Huy Duc Dao
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package gin defines a set of basic building blocks for instrumenting Sonic gateways built using the gin router
package gin

import (
	"context"
	"github.com/gin-gonic/gin"
	metrics2 "github.com/starvn/sonic/telemetry/metrics"
	"github.com/starvn/sonic/telemetry/metrics/mux"
	"github.com/starvn/turbo/config"
	"github.com/starvn/turbo/log"
	"github.com/starvn/turbo/proxy"
	sgin "github.com/starvn/turbo/route/gin"
	"net/http"
	"strconv"
	"time"
)

func New(ctx context.Context, e config.ExtraConfig, l log.Logger) *Metrics {
	metricsCollector := Metrics{metrics2.New(ctx, e, l)}
	if metricsCollector.Config != nil && !metricsCollector.Config.EndpointDisabled {
		metricsCollector.RunEndpoint(ctx, metricsCollector.NewEngine(), l)
	}
	return &metricsCollector
}

type Metrics struct {
	*metrics2.Metrics
}

func (m *Metrics) RunEndpoint(ctx context.Context, e *gin.Engine, l log.Logger) {
	logPrefix := "[SERVICE: Stats]"
	server := &http.Server{
		Addr:    m.Config.ListenAddr,
		Handler: e,
	}
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			l.Error(logPrefix, err.Error())
		}
	}()

	go func() {
		<-ctx.Done()
		l.Info(logPrefix, "Shutting down the metrics endpoint handler")
		ctx, cancel := context.WithTimeout(ctx, time.Second)
		_ = server.Shutdown(ctx)
		cancel()
	}()

	l.Debug(logPrefix, "The endpoint /__stats is now available on", m.Config.ListenAddr)
}

func (m *Metrics) NewEngine() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.RedirectTrailingSlash = true
	engine.RedirectFixedPath = true
	engine.HandleMethodNotAllowed = true

	engine.GET("/__stats", m.NewExpHandler())
	return engine
}

func (m *Metrics) NewExpHandler() gin.HandlerFunc {
	return gin.WrapH(mux.NewExpHandler(m.Registry))
}

func (m *Metrics) NewHTTPHandlerFactory(hf sgin.HandlerFactory) sgin.HandlerFactory {
	if m.Config == nil || m.Config.RouterDisabled {
		return hf
	}
	return NewHTTPHandlerFactory(m.Router, hf)
}

func NewHTTPHandlerFactory(rm *metrics2.RouterMetrics, hf sgin.HandlerFactory) sgin.HandlerFactory {
	return func(cfg *config.EndpointConfig, p proxy.Proxy) gin.HandlerFunc {
		next := hf(cfg, p)
		rm.RegisterResponseWriterMetrics(cfg.Endpoint)
		return func(c *gin.Context) {
			rw := &ginResponseWriter{c.Writer, cfg.Endpoint, time.Now(), rm}
			c.Writer = rw
			rm.Connection(c.Request.TLS)

			next(c)

			rw.end()
			rm.Disconnection()
		}
	}
}

type ginResponseWriter struct {
	gin.ResponseWriter
	name  string
	begin time.Time
	rm    *metrics2.RouterMetrics
}

func (w *ginResponseWriter) end() {
	duration := time.Since(w.begin)
	w.rm.Counter("response", w.name, "status", strconv.Itoa(w.Status()), "count").Inc(1)
	w.rm.Histogram("response", w.name, "size").Update(int64(w.Size()))
	w.rm.Histogram("response", w.name, "time").Update(int64(duration))
}
