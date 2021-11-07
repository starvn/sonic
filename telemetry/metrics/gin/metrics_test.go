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

package gin

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/starvn/sonic/telemetry/metrics"
	"github.com/starvn/turbo/config"
	"github.com/starvn/turbo/log"
	"github.com/starvn/turbo/proxy"
	sgin "github.com/starvn/turbo/route/gin"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
)

func TestDisabledRouterMetrics(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	buf := bytes.NewBuffer(make([]byte, 1024))
	l, _ := log.NewLogger("DEBUG", buf, "")
	cfg := map[string]interface{}{metrics.Namespace: map[string]interface{}{"router_disabled": true}}
	metric := New(ctx, cfg, l)
	hf := metric.NewHTTPHandlerFactory(sgin.EndpointHandler)
	if reflect.ValueOf(hf).Pointer() != reflect.ValueOf(sgin.EndpointHandler).Pointer() {
		t.Error("The endpoint handler should be the default since the Router metrics are disabled.")
	}
}

func TestNew(t *testing.T) {
	rand.Seed(time.Now().Unix())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	buf := bytes.NewBuffer(make([]byte, 1024))
	l, _ := log.NewLogger("DEBUG", buf, "")
	defaultCfg := map[string]interface{}{metrics.Namespace: map[string]interface{}{"collection_time": "1s"}}
	metric := New(ctx, defaultCfg, l)

	engine := gin.New()

	response := proxy.Response{Data: map[string]interface{}{}, IsComplete: true}
	max := 1000
	min := 1
	p := func(_ context.Context, _ *proxy.Request) (*proxy.Response, error) {
		time.Sleep(time.Microsecond * time.Duration(rand.Intn(max-min)+min))
		return &response, nil
	}
	hf := metric.NewHTTPHandlerFactory(sgin.EndpointHandler)
	cfg := &config.EndpointConfig{
		Endpoint: "/test/{var}",
		Timeout:  10 * time.Second,
		CacheTTL: time.Second,
	}
	engine.GET("/test/:var", hf(cfg, p))
	statsEngine := metric.NewEngine()

	for i := 0; i < 100; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test/a", nil)
		engine.ServeHTTP(w, req)
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
	req, _ := http.NewRequest("GET", "/__stats", nil)
	statsEngine.ServeHTTP(w, req)

	if w.Result().StatusCode != 200 {
		t.Errorf("unexpected status code: %d\n", w.Result().StatusCode)
	}
}

func TestStatsEndpoint(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	buf := bytes.NewBuffer(make([]byte, 1024))
	l, _ := log.NewLogger("DEBUG", buf, "")
	cfg := map[string]interface{}{metrics.Namespace: map[string]interface{}{"collection_time": "100ms", "listen_address": ":8990"}}
	_ = New(ctx, cfg, l)
	<-time.After(500 * time.Millisecond)
	resp, err := http.Get("http://localhost:8990/__stats")
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
		t.Errorf("Proble unmarshaling stats endpoint response: %s\n", err.Error())
	}
	if _, ok := stats["cmdline"]; !ok {
		t.Error("Key cmdline should exists in the response.\n")
	}
}
