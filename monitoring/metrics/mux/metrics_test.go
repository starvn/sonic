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

package mux

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/rcrowley/go-metrics"
	smetrics "github.com/starvn/sonic/monitoring/metrics"
	"github.com/starvn/turbo/config"
	"github.com/starvn/turbo/log"
	"github.com/starvn/turbo/proxy"
	"github.com/starvn/turbo/route/mux"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestDisabledRouterMetrics(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	buf := bytes.NewBuffer(make([]byte, 1024))
	l, _ := log.NewLogger("DEBUG", buf, "")
	cfg := map[string]interface{}{smetrics.Namespace: map[string]interface{}{"router_disabled": true}}
	metric := New(ctx, cfg, l)
	hf := metric.NewHTTPHandlerFactory(mux.EndpointHandler)
	if reflect.ValueOf(hf).Pointer() != reflect.ValueOf(mux.EndpointHandler).Pointer() {
		t.Error("The endpoint handler should be the default since the Router metrics are disabled.")
	}
}

func TestNew(t *testing.T) {
	rand.Seed(time.Now().Unix())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	buf := bytes.NewBuffer(make([]byte, 1024))
	l, _ := log.NewLogger("DEBUG", buf, "")
	metricsCfg := map[string]interface{}{smetrics.Namespace: map[string]interface{}{"collection_time": "1s"}}
	metric := New(ctx, metricsCfg, l)

	response := proxy.Response{Data: map[string]interface{}{}, IsComplete: true}
	max := 1000
	min := 1
	p := func(_ context.Context, _ *proxy.Request) (*proxy.Response, error) {
		time.Sleep(time.Microsecond * time.Duration(rand.Intn(max-min)+min))
		return &response, nil
	}
	hf := metric.NewHTTPHandlerFactory(mux.EndpointHandler)
	cfg := &config.EndpointConfig{
		Endpoint: "/test/{var}",
		Timeout:  10 * time.Second,
		CacheTTL: time.Second,
		Method:   "GET",
	}
	// engine.GET("/test", hf(cfg, p))
	// engine.GET("/__stats", metric.NewExpHandler())

	for i := 0; i < 100; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test/something", ioutil.NopCloser(strings.NewReader("")))
		hf(cfg, p)(w, req)
	}

	metric.Router.Aggregate()
	snapshot := metric.TakeSnapshot()

	expected := map[string]int64{
		"sonic.router.response./test/{var}.status.200.count": 100,
		"sonic.router.connected":                             0,
		"sonic.router.disconnected":                          0,
		"sonic.router.connected-total":                       100,
		"sonic.router.disconnected-total":                    100,
		"sonic.router.response./test/{var}.status":           0,
	}
	for k, v := range snapshot.Counters {
		if exp, ok := expected[k]; !ok || int(exp) != int(v) {
			t.Errorf("unexpected metric: got [%s: %d] want [%s: %d]", k, v, k, exp)
		}
	}

	if _, ok := snapshot.Histograms["sonic.router.response./test/{var}.size"]; !ok {
		t.Error("expected histogram not present")
	}

	expected = map[string]int64{
		"sonic.router.connected-gauge":    100,
		"sonic.router.disconnected-gauge": 100,
	}
	for k, exp := range expected {
		if v, ok := snapshot.Gauges[k]; !ok || int(exp) != int(v) {
			t.Errorf("unexpected metric: got [%s: %d] want [%s: %d]", k, v, k, exp)
		}
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/__stats", ioutil.NopCloser(strings.NewReader("")))
	metric.NewExpHandler().ServeHTTP(w, req)

	if w.Result().StatusCode != 200 {
		t.Errorf("unexpected status code: %d\n", w.Result().StatusCode)
	}
}

func TestNewHTTPHandler(t *testing.T) {
	registry := metrics.NewRegistry()

	rm := smetrics.NewRouterMetrics(&registry)
	assertion := func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(time.Millisecond)
		w.Header().Set("x-test", "ok")
		w.WriteHeader(200)
		_, _ = w.Write([]byte("okidoki"))
	}
	h := NewHTTPHandler("test", http.HandlerFunc(assertion), rm)
	ts := httptest.NewServer(h)

	for i := 0; i < 10; i++ {
		resp, err := http.Get(ts.URL)
		if err != nil {
			t.Error(err)
		}
		if resp.Header.Get("x-test") != "ok" {
			t.Errorf("unexpected header: %s\n", resp.Header.Get("x-test"))
		}
		if resp.StatusCode != 200 {
			t.Errorf("unexpected status code: %d\n", resp.StatusCode)
		}
	}
	rm.Aggregate()
	ts.Close()

	expected := map[string]struct{}{
		"router.connected":                      {},
		"router.disconnected":                   {},
		"router.connected-gauge":                {},
		"router.disconnected-gauge":             {},
		"router.connected-total":                {},
		"router.disconnected-total":             {},
		"router.response.test.status.200.count": {},
		"router.response.test.time":             {},
		"router.response.test.size":             {},
		"router.response.test.status":           {},
	}
	tracked := []string{}
	registry.Each(func(k string, _ interface{}) {
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

	ts = httptest.NewServer(NewExpHandler(&registry))

	resp, err := http.Get(ts.URL)
	if err != nil {
		t.Error(err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("unexpected status code: %d\n", resp.StatusCode)
	}

	ts.Close()
}

func TestDisabledMetricMethods(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	buf := bytes.NewBuffer(make([]byte, 1024))
	l, _ := log.NewLogger("DEBUG", buf, "")
	emptyMetrics := New(ctx, nil, l)
	hf := emptyMetrics.NewHTTPHandlerFactory(mux.EndpointHandler)
	if reflect.ValueOf(hf).Pointer() != reflect.ValueOf(mux.EndpointHandler).Pointer() {
		t.Error("An empty metrics package should implement NewHTTPHandlerFactory method.")
	}
	expHandler := emptyMetrics.NewExpHandler()
	if fmt.Sprintf("%T", expHandler) != "http.HandlerFunc" {
		t.Error("An empty metrics package should implement the NewExpHandler() method.")
	}
	handler := emptyMetrics.NewHTTPHandler("test", http.HandlerFunc(dummyHTTPHandler))
	if fmt.Sprintf("%T", handler) != "http.HandlerFunc" {
		t.Error("An empty metrics package should implement the NewExpHandler() method.")
	}

}

func TestStatsEndpoint(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	buf := bytes.NewBuffer(make([]byte, 1024))
	l, _ := log.NewLogger("DEBUG", buf, "")
	cfg := map[string]interface{}{smetrics.Namespace: map[string]interface{}{"collection_time": "100ms", "listen_address": ":8999"}}
	_ = New(ctx, cfg, l)
	<-time.After(500 * time.Millisecond)
	resp, err := http.Get("http://localhost:8999/__stats")
	if err != nil {
		t.Errorf("Problem with the stats endpoint: %s\n", err.Error())
		return
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("Cannot read body: %s\n", err.Error())
	}
	var stats map[string]interface{}
	err = json.Unmarshal(body, &stats)
	if err != nil {
		t.Errorf("Problem unmarshaling stats endpoint response: %s\n", err.Error())
	}
	if _, ok := stats["cmdline"]; !ok {
		t.Error("Key cmdline should exists in the response.\n")
	}
}
