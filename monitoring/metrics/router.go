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
	"crypto/tls"
	"github.com/rcrowley/go-metrics"
)

func NewRouterMetrics(parent *metrics.Registry) *RouterMetrics {
	r := metrics.NewPrefixedChildRegistry(*parent, "router.")

	return &RouterMetrics{
		ProxyMetrics{r},
		metrics.NewRegisteredCounter("connected", r),
		metrics.NewRegisteredCounter("disconnected", r),
		metrics.NewRegisteredCounter("connected-total", r),
		metrics.NewRegisteredCounter("disconnected-total", r),
		metrics.NewRegisteredGauge("connected-gauge", r),
		metrics.NewRegisteredGauge("disconnected-gauge", r),
	}
}

type RouterMetrics struct {
	ProxyMetrics
	connected         metrics.Counter
	disconnected      metrics.Counter
	connectedTotal    metrics.Counter
	disconnectedTotal metrics.Counter
	connectedGauge    metrics.Gauge
	disconnectedGauge metrics.Gauge
}

func (rm *RouterMetrics) Connection(TLS *tls.ConnectionState) {
	rm.connected.Inc(1)
	if TLS == nil {
		return
	}

	rm.Counter("tls_version", tlsVersion[TLS.Version], "count").Inc(1)
	rm.Counter("tls_cipher", tlsCipherSuite[TLS.CipherSuite], "count").Inc(1)
}

func (rm *RouterMetrics) Disconnection() {
	rm.disconnected.Inc(1)
}

func (rm *RouterMetrics) Aggregate() {
	con := rm.connected.Count()
	rm.connectedGauge.Update(con)
	rm.connectedTotal.Inc(con)
	rm.connected.Clear()
	disconnected := rm.disconnected.Count()
	rm.disconnectedGauge.Update(disconnected)
	rm.disconnectedTotal.Inc(disconnected)
	rm.disconnected.Clear()
}

func (rm *RouterMetrics) RegisterResponseWriterMetrics(name string) {
	rm.Counter("response", name, "status")
	rm.Histogram("response", name, "size")
	rm.Histogram("response", name, "time")
}
