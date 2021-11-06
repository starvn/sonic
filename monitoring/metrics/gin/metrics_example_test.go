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
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/starvn/sonic/monitoring/metrics"
	"github.com/starvn/turbo/config"
	"github.com/starvn/turbo/encoding"
	"github.com/starvn/turbo/log"
	"github.com/starvn/turbo/proxy"
	"github.com/starvn/turbo/transport/http/client"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"time"
)

var defaultCfg = map[string]interface{}{metrics.Namespace: map[string]interface{}{"collection_time": "100ms"}}

func Example_router() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	buf := bytes.NewBuffer(make([]byte, 1024))
	l, _ := log.NewLogger("DEBUG", buf, "")

	metricProducer := New(ctx, defaultCfg, l)

	engine := gin.New()
	hf := metricProducer.NewHTTPHandlerFactory(func(_ *config.EndpointConfig, _ proxy.Proxy) gin.HandlerFunc {
		return gin.WrapF(dummyHTTPHandler)
	})
	engine.GET("/test", hf(&config.EndpointConfig{Endpoint: "/some/{url}"}, proxy.NoopProxy))
	engine.GET("/stats", metricProducer.NewExpHandler())

	for i := 0; i < 10; i++ {
		w := httptest.NewRecorder()
		req, err := http.NewRequest("GET", "/test", ioutil.NopCloser(strings.NewReader("")))
		if err != nil {
			fmt.Println(err)
		}
		engine.ServeHTTP(w, req)
		resp := w.Result()
		if resp.Header.Get("x-test") != "ok" {
			fmt.Println("unexpected header:", resp.Header.Get("x-test"))
		}
		if resp.StatusCode != 200 {
			fmt.Println("unexpected status code:", resp.StatusCode)
		}
	}
	metricProducer.Router.Aggregate()

	w := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "/stats", ioutil.NopCloser(strings.NewReader("")))
	if err != nil {
		fmt.Println(err)
	}
	engine.ServeHTTP(w, req)
	resp := w.Result()
	data, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	if len(data) < 5000 {
		fmt.Println("unexpected response size:", len(data), string(data))
	}

	// Uncomment to see what's been logged
	// fmt.Println(buf.String())

	// Uncomment the next line and remove the # to see the stats report as the test fails
	// fmt.Println(string(data))
	// #Output: golly

	// Output:
}

func Example_proxy() {
	ctx := context.Background()
	buf := bytes.NewBuffer(make([]byte, 1024))
	l, _ := log.NewLogger("DEBUG", buf, "")
	cfg := &config.EndpointConfig{
		Endpoint: "/test/endpoint",
	}

	metricProducer := New(ctx, defaultCfg, l)

	response := proxy.Response{Data: map[string]interface{}{}, IsComplete: true}
	fakeFactory := proxy.FactoryFunc(func(_ *config.EndpointConfig) (proxy.Proxy, error) {
		return func(_ context.Context, _ *proxy.Request) (*proxy.Response, error) { return &response, nil }, nil
	})
	pf := metricProducer.ProxyFactory("proxy_layer", fakeFactory)

	p, err := pf.New(cfg)
	if err != nil {
		fmt.Println("Error:", err)
	}
	req := proxy.Request{}
	for i := 0; i < 10; i++ {
		resp, err := p(ctx, &req)
		if err != nil {
			fmt.Println("Error:", err)
		}
		if resp != &response {
			fmt.Println("Unexpected response:", *resp)
		}
	}

	engine := gin.New()
	engine.GET("/stats", metricProducer.NewExpHandler())
	w := httptest.NewRecorder()
	reqHTTP, err := http.NewRequest("GET", "/stats", ioutil.NopCloser(strings.NewReader("")))
	if err != nil {
		fmt.Println(err)
	}
	engine.ServeHTTP(w, reqHTTP)
	resp := w.Result()
	data, _ := ioutil.ReadAll(resp.Body)
	_ = resp.Body.Close()

	if len(data) < 5000 {
		fmt.Println("unexpected response size:", len(data), string(data))
	}

	// Uncomment to see what's been logged
	// fmt.Println(buf.String())

	// Uncomment the next line and remove the # to see the stats report as the test fails
	// fmt.Println(string(data))
	// #Output: golly

	// Output:
}

func Example_backend() {
	ctx := context.Background()
	buf := bytes.NewBuffer(make([]byte, 1024))
	l, _ := log.NewLogger("DEBUG", buf, "")

	metricProducer := New(ctx, defaultCfg, l)

	bf := metricProducer.BackendFactory("backend_layer", proxy.CustomHTTPProxyFactory(client.NewHTTPClient))

	p := bf(&config.Backend{URLPattern: "/some/{url}", Decoder: encoding.JSONDecoder})

	ts1 := httptest.NewServer(http.HandlerFunc(dummyHTTPHandler))
	defer ts1.Close()

	parsedURL, _ := url.Parse(ts1.URL)
	req := proxy.Request{URL: parsedURL, Method: "GET", Body: ioutil.NopCloser(strings.NewReader(""))}
	for i := 0; i < 10; i++ {
		resp, err := p(ctx, &req)
		if err != nil {
			fmt.Println("Error:", err)
		}
		if !resp.IsComplete {
			fmt.Println("Unexpected response:", *resp)
		}
	}

	engine := gin.New()
	engine.GET("/stats", metricProducer.NewExpHandler())
	w := httptest.NewRecorder()
	reqHTTP, err := http.NewRequest("GET", "/stats", ioutil.NopCloser(strings.NewReader("")))
	if err != nil {
		fmt.Println(err)
	}
	engine.ServeHTTP(w, reqHTTP)
	resp := w.Result()
	data, _ := ioutil.ReadAll(resp.Body)
	_ = resp.Body.Close()

	if len(data) < 5000 {
		fmt.Println("unexpected response size:", len(data), string(data))
	}
	// Uncomment to see what's been logged
	// fmt.Println(buf.String())

	// Uncomment the next line and remove the # to see the stats report as the test fails
	// fmt.Println(string(data))
	// #Output: golly

	// Output:
}

func dummyHTTPHandler(w http.ResponseWriter, r *http.Request) {
	time.Sleep(time.Millisecond)
	w.Header().Set("x-test", "ok")
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(200)
	_, _ = w.Write([]byte(`{"status":true}`))
}
