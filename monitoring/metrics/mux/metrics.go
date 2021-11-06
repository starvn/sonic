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

// Package mux defines a set of basic building blocks for instrumenting Sonic gateways built using the mux router
package mux

import (
	"context"
	"github.com/rcrowley/go-metrics"
	"github.com/rcrowley/go-metrics/exp"
	smetrics "github.com/starvn/sonic/monitoring/metrics"
	"github.com/starvn/turbo/config"
	"github.com/starvn/turbo/log"
	"github.com/starvn/turbo/proxy"
	"github.com/starvn/turbo/route/mux"
	"net/http"
	"strconv"
	"time"
)

func New(ctx context.Context, e config.ExtraConfig, l log.Logger) *Metrics {
	metricsCollector := Metrics{smetrics.New(ctx, e, l)}
	if metricsCollector.Config != nil && !metricsCollector.Config.EndpointDisabled {
		metricsCollector.RunEndpoint(ctx, metricsCollector.NewEngine(), l)
	}
	return &metricsCollector
}

type Metrics struct {
	*smetrics.Metrics
}

func (m *Metrics) RunEndpoint(ctx context.Context, s *http.ServeMux, l log.Logger) {
	server := &http.Server{
		Addr:    m.Config.ListenAddr,
		Handler: s,
	}
	go func() {
		l.Error(server.ListenAndServe())
	}()

	go func() {
		<-ctx.Done()
		l.Info("shutting down the stats handler")
		ctx, cancel := context.WithTimeout(ctx, time.Second)
		_ = server.Shutdown(ctx)
		cancel()
	}()
}

func (m *Metrics) NewEngine() *http.ServeMux {
	muxV := http.NewServeMux()
	muxV.Handle("/__stats", m.NewExpHandler())
	return muxV
}

func (m *Metrics) NewExpHandler() http.Handler {
	return NewExpHandler(m.Registry)
}

func (m *Metrics) NewHTTPHandler(name string, h http.Handler) http.HandlerFunc {
	return NewHTTPHandler(name, h, m.Router)
}

func (m *Metrics) NewHTTPHandlerFactory(defaultHandlerFactory mux.HandlerFactory) mux.HandlerFactory {
	if m.Config == nil || m.Config.RouterDisabled {
		return defaultHandlerFactory
	}
	return func(cfg *config.EndpointConfig, p proxy.Proxy) http.HandlerFunc {
		return m.NewHTTPHandler(cfg.Endpoint, defaultHandlerFactory(cfg, p))
	}
}

func NewExpHandler(parent *metrics.Registry) http.Handler {
	return exp.ExpHandler(*parent)
}

func NewHTTPHandler(name string, h http.Handler, rm *smetrics.RouterMetrics) http.HandlerFunc {
	rm.RegisterResponseWriterMetrics(name)
	return func(w http.ResponseWriter, r *http.Request) {
		rm.Connection(r.TLS)
		rw := newHTTPResponseWriter(name, w, rm)
		h.ServeHTTP(rw, r)
		rw.end()
		rm.Disconnection()
	}
}

func newHTTPResponseWriter(name string, rw http.ResponseWriter, rm *smetrics.RouterMetrics) *responseWriter {
	return &responseWriter{
		ResponseWriter: rw,
		begin:          time.Now(),
		name:           name,
		rm:             rm,
		status:         200,
	}
}

type responseWriter struct {
	http.ResponseWriter
	begin        time.Time
	name         string
	rm           *smetrics.RouterMetrics
	responseSize int
	status       int
}

func (w *responseWriter) WriteHeader(code int) {
	w.ResponseWriter.WriteHeader(code)
	w.status = code
}

func (w *responseWriter) Write(data []byte) (i int, err error) {
	i, err = w.ResponseWriter.Write(data)
	w.responseSize += i
	return
}

func (w responseWriter) end() {
	duration := time.Since(w.begin)
	w.rm.Counter("response", w.name, "status", strconv.Itoa(w.status), "count").Inc(1)
	w.rm.Histogram("response", w.name, "size").Update(int64(w.responseSize))
	w.rm.Histogram("response", w.name, "time").Update(int64(duration))
}
