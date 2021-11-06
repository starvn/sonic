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
	"strings"
	"testing"
)

func TestRouterMetrics(t *testing.T) {
	p := metrics.NewRegistry()
	rm := NewRouterMetrics(&p)

	rm.Connection(nil)
	rm.Connection(&tls.ConnectionState{Version: tls.VersionTLS11, CipherSuite: tls.TLS_RSA_WITH_AES_128_GCM_SHA256})
	rm.Disconnection()
	rm.Connection(nil)
	rm.Connection(nil)
	rm.Connection(nil)

	rm.Aggregate()

	rm.Connection(&tls.ConnectionState{Version: tls.VersionTLS12, CipherSuite: tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA})
	rm.Disconnection()
	rm.Disconnection()
	rm.Disconnection()

	rm.Aggregate()

	rm.Disconnection()

	p.Each(func(name string, v interface{}) {
		if !strings.HasPrefix(name, "router.") {
			t.Errorf("Unexpected metric: %s", name)
		}
	})

	for k, want := range map[string]int64{
		"router.connected":                                           0,
		"router.disconnected":                                        1,
		"router.connected-total":                                     6,
		"router.disconnected-total":                                  4,
		"router.connected-gauge":                                     1,
		"router.disconnected-gauge":                                  3,
		"router.tls_cipher.TLS_RSA_WITH_AES_128_GCM_SHA256.count":    1,
		"router.tls_cipher.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA.count": 1,
		"router.tls_version.VersionTLS11.count":                      1,
		"router.tls_version.VersionTLS12.count":                      1,
	} {
		var have int64
		switch metric := p.Get(k).(type) {
		case metrics.Counter:
			have = metric.Count()
		case metrics.Gauge:
			have = metric.Value()
		}
		if have != want {
			t.Errorf("Unexpected value for %s. Have: %d, want: %d", k, have, want)
		}
	}
}
