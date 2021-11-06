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

package metrics

import (
	"context"
	"github.com/rcrowley/go-metrics"
	"github.com/starvn/turbo/config"
	"github.com/starvn/turbo/proxy"
	"github.com/starvn/turbo/transport/http/client"
	"strconv"
	"strings"
	"time"
)

func (m *Metrics) NewProxyMiddleware(layer, name string) proxy.Middleware {
	return NewProxyMiddleware(layer, name, m.Proxy)
}

func (m *Metrics) ProxyFactory(segmentName string, next proxy.Factory) proxy.FactoryFunc {
	if m.Config == nil || m.Config.ProxyDisabled {
		return next.New
	}
	return proxy.FactoryFunc(func(cfg *config.EndpointConfig) (proxy.Proxy, error) {
		next, err := next.New(cfg)
		if err != nil {
			return proxy.NoopProxy, err
		}
		return m.NewProxyMiddleware(segmentName, cfg.Endpoint)(next), nil
	})
}

func (m *Metrics) BackendFactory(segmentName string, next proxy.BackendFactory) proxy.BackendFactory {
	if m.Config == nil || m.Config.BackendDisabled {
		return next
	}
	return func(cfg *config.Backend) proxy.Proxy {
		return m.NewProxyMiddleware(segmentName, cfg.URLPattern)(next(cfg))
	}
}

func (m *Metrics) DefaultBackendFactory() proxy.BackendFactory {
	return m.BackendFactory("backend", proxy.CustomHTTPProxyFactory(client.NewHTTPClient))
}

func NewProxyMetrics(parent *metrics.Registry) *ProxyMetrics {
	m := metrics.NewPrefixedChildRegistry(*parent, "proxy.")
	return &ProxyMetrics{m}
}

func NewProxyMiddleware(layer, name string, pm *ProxyMetrics) proxy.Middleware {
	registerProxyMiddlewareMetrics(layer, name, pm)
	return func(next ...proxy.Proxy) proxy.Proxy {
		if len(next) > 1 {
			panic(proxy.ErrTooManyProxies)
		}
		return func(ctx context.Context, request *proxy.Request) (*proxy.Response, error) {
			begin := time.Now()
			resp, err := next[0](ctx, request)

			go func(duration int64, resp *proxy.Response, err error) {
				errored := strconv.FormatBool(err != nil)
				complete := strconv.FormatBool(resp != nil && resp.IsComplete)
				labels := "layer." + layer + ".name." + name + ".complete." + complete + ".error." + errored
				pm.Counter("requests." + labels).Inc(1)
				pm.Histogram("latency." + labels).Update(duration)
			}(time.Since(begin).Nanoseconds(), resp, err)

			return resp, err
		}
	}
}

func registerProxyMiddlewareMetrics(layer, name string, pm *ProxyMetrics) {
	labels := "layer." + layer + ".name." + name
	for _, complete := range []string{"true", "false"} {
		for _, errored := range []string{"true", "false"} {
			metrics.GetOrRegisterCounter("requests."+labels+".complete."+complete+".error."+errored, pm.register)

			metrics.GetOrRegisterHistogram("latency."+labels+".complete."+complete+".error."+errored, pm.register, defaultSample())
		}
	}
}

type ProxyMetrics struct {
	register metrics.Registry
}

func (rm *ProxyMetrics) Histogram(labels ...string) metrics.Histogram {
	return metrics.GetOrRegisterHistogram(strings.Join(labels, "."), rm.register, defaultSample())
}

func (rm *ProxyMetrics) Counter(labels ...string) metrics.Counter {
	return metrics.GetOrRegisterCounter(strings.Join(labels, "."), rm.register)
}
