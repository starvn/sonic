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
	"bytes"
	"context"
	"github.com/rcrowley/go-metrics"
	"github.com/starvn/turbo/log"
	"github.com/starvn/turbo/proxy"
	"github.com/starvn/turbo/transport/http/client"
	"net/url"
	"reflect"
	"testing"
	"time"
)

func TestNewProxyMiddleware(t *testing.T) {
	URL, _ := url.Parse("http://example.com/12345")
	request := &proxy.Request{URL: URL}
	response := &proxy.Response{Data: map[string]interface{}{}, IsComplete: true}
	assertion := func(_ context.Context, req *proxy.Request) (*proxy.Response, error) {
		if request != req {
			t.Errorf("unexpected request! want [%v], have [%v]\n", request, req)
		}
		time.Sleep(time.Millisecond)
		return response, nil
	}

	registry := metrics.NewRegistry()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	proxyMetric := NewProxyMetrics(&registry)

	mw := NewProxyMiddleware("some", "none", proxyMetric)

	for i := 0; i < 100; i++ {
		resp, err := mw(assertion)(ctx, request)
		if err != nil {
			t.Error("unexpected error:", err)
			return
		}
		if resp != response {
			t.Errorf("unexpected response! want [%v], have [%v]\n", response, resp)
			return
		}
	}

	expected := map[string]struct{}{
		"proxy.latency.layer.some.name.none.complete.true.error.true":    {},
		"proxy.latency.layer.some.name.none.complete.true.error.false":   {},
		"proxy.latency.layer.some.name.none.complete.false.error.true":   {},
		"proxy.latency.layer.some.name.none.complete.false.error.false":  {},
		"proxy.requests.layer.some.name.none.complete.true.error.true":   {},
		"proxy.requests.layer.some.name.none.complete.true.error.false":  {},
		"proxy.requests.layer.some.name.none.complete.false.error.true":  {},
		"proxy.requests.layer.some.name.none.complete.false.error.false": {},
	}
	var tracked []string
	proxyMetric.register.Each(func(k string, v interface{}) {
		tracked = append(tracked, k)
	})
	if len(tracked) != len(expected) {
		t.Error("unexpected size of the tracked list", tracked)
	}
	for _, k := range tracked {
		if _, ok := expected[k]; !ok {
			t.Error("the key", k, " has not been tracked")
		}
	}
}

func TestDefaultBackendFactory(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	buf := bytes.NewBuffer(make([]byte, 1024))
	l, _ := log.NewLogger("DEBUG", buf, "")
	cfg := map[string]interface{}{Namespace: map[string]interface{}{"backend_disabled": true}}
	metric := New(ctx, cfg, l)
	bf := metric.DefaultBackendFactory()
	if reflect.ValueOf(bf).Pointer() != reflect.ValueOf(proxy.CustomHTTPProxyFactory(client.NewHTTPClient)).Pointer() {
		t.Error("The backend factory should be the default since the backend metrics are disabled.")
	}
}
