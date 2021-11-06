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

// Package metrics defines a set of basic building blocks for instrumenting Sonic gateways
package metrics

import (
	"context"
	"fmt"
	"github.com/rcrowley/go-metrics"
	"github.com/starvn/turbo/config"
	"github.com/starvn/turbo/log"
	"strings"
	"time"
)

var defaultListenAddr = ":8090"

func New(ctx context.Context, e config.ExtraConfig, l log.Logger) *Metrics {
	registry := metrics.NewPrefixedRegistry("sonic.")

	var cfg *Config
	if tmp, ok := ConfigGetter(e).(*Config); ok {
		cfg = tmp
	}

	if cfg == nil {
		registry = NewDummyRegistry()
		return &Metrics{
			Registry: &registry,
			Router:   &RouterMetrics{},
			Proxy:    &ProxyMetrics{},
		}
	}

	m := Metrics{
		Config:         cfg,
		Router:         NewRouterMetrics(&registry),
		Proxy:          NewProxyMetrics(&registry),
		Registry:       &registry,
		latestSnapshot: NewStats(),
	}

	m.processMetrics(ctx, m.Config.CollectionTime, logger{l})

	return &m
}

const Namespace = "github.com/starvn/sonic/monitoring/metrics"

type Config struct {
	ProxyDisabled    bool
	RouterDisabled   bool
	BackendDisabled  bool
	CollectionTime   time.Duration
	ListenAddr       string
	EndpointDisabled bool
}

func ConfigGetter(e config.ExtraConfig) interface{} {
	v, ok := e[Namespace]
	if !ok {
		return nil
	}

	tmp, ok := v.(map[string]interface{})
	if !ok {
		return nil
	}

	userCfg := new(Config)
	userCfg.CollectionTime = time.Minute
	if collectionTime, ok := tmp["collection_time"]; ok {
		if d, err := time.ParseDuration(collectionTime.(string)); err == nil {
			userCfg.CollectionTime = d
		}
	}
	userCfg.ListenAddr = defaultListenAddr
	if listenAddr, ok := tmp["listen_address"]; ok {
		if a, ok := listenAddr.(string); ok {
			userCfg.ListenAddr = a
		}
	}
	userCfg.ProxyDisabled = getBool(tmp, "proxy_disabled")
	userCfg.RouterDisabled = getBool(tmp, "router_disabled")
	userCfg.BackendDisabled = getBool(tmp, "backend_disabled")
	userCfg.EndpointDisabled = getBool(tmp, "endpoint_disabled")

	return userCfg
}

func getBool(data map[string]interface{}, name string) bool {
	if flag, ok := data[name]; ok {
		if v, ok := flag.(bool); ok {
			return v
		}
	}
	return false
}

type Metrics struct {
	Config         *Config
	Proxy          *ProxyMetrics
	Router         *RouterMetrics
	Registry       *metrics.Registry
	latestSnapshot Stats
}

func (m *Metrics) Snapshot() Stats {
	return m.latestSnapshot
}

func (m *Metrics) TakeSnapshot() Stats {
	tmp := NewStats()

	(*m.Registry).Each(func(k string, v interface{}) {
		switch metric := v.(type) {
		case metrics.Counter:
			tmp.Counters[k] = metric.Count()
		case metrics.Gauge:
			tmp.Gauges[k] = metric.Value()
		case metrics.Histogram:
			tmp.Histograms[k] = HistogramData{
				Max:         metric.Max(),
				Min:         metric.Min(),
				Mean:        metric.Mean(),
				Stddev:      metric.StdDev(),
				Variance:    metric.Variance(),
				Percentiles: metric.Percentiles(percentiles),
			}
			metric.Clear()
		}
	})
	return tmp
}

func (m *Metrics) processMetrics(ctx context.Context, d time.Duration, l metrics.Logger) {
	r := metrics.NewPrefixedChildRegistry(*(m.Registry), "service.")

	metrics.RegisterDebugGCStats(r)
	metrics.RegisterRuntimeMemStats(r)

	go func() {
		ticker := time.NewTicker(d)
		for {
			select {
			case <-ticker.C:
				metrics.CaptureDebugGCStatsOnce(r)
				metrics.CaptureRuntimeMemStatsOnce(r)
				m.Router.Aggregate()
				m.latestSnapshot = m.TakeSnapshot()
			case <-ctx.Done():
				return
			}
		}
	}()
}

var (
	percentiles   = []float64{0.1, 0.25, 0.5, 0.75, 0.9, 0.95, 0.99}
	defaultSample = func() metrics.Sample { return metrics.NewUniformSample(1028) }
)

type logger struct {
	logger log.Logger
}

func (l logger) Printf(format string, v ...interface{}) {
	l.logger.Debug(strings.TrimRight(fmt.Sprintf(format, v...), "\n"))
}

type DummyRegistry struct{}

func (r DummyRegistry) Each(_ func(string, interface{})) {}
func (r DummyRegistry) Get(_ string) interface{}         { return nil }
func (r DummyRegistry) GetAll() map[string]map[string]interface{} {
	return map[string]map[string]interface{}{}
}
func (r DummyRegistry) GetOrRegister(_ string, i interface{}) interface{} { return i }
func (r DummyRegistry) Register(_ string, _ interface{}) error            { return nil }
func (r DummyRegistry) RunHealthchecks()                                  {}
func (r DummyRegistry) Unregister(_ string)                               {}
func (r DummyRegistry) UnregisterAll()                                    {}

func NewDummyRegistry() metrics.Registry {
	return DummyRegistry{}
}
