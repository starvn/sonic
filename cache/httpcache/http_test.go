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

package httpcache

import (
	"bytes"
	"context"
	"fmt"
	"github.com/starvn/turbo/config"
	"github.com/starvn/turbo/encoding"
	"github.com/starvn/turbo/proxy"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
)

func TestClient_ok(t *testing.T) {
	testCacheSystem(t, func(t *testing.T, URL string) {
		testClient(t, sampleCfg, URL)
	}, 1)
}

func TestClient_ko(t *testing.T) {
	cfg := &config.Backend{
		Decoder:     encoding.JSONDecoder,
		ExtraConfig: map[string]interface{}{},
	}
	testCacheSystem(t, func(t *testing.T, URL string) {
		testClient(t, cfg, URL)
	}, 100)
}

func testClient(t *testing.T, cfg *config.Backend, URL string) {
	clientFactory := NewHTTPClient(cfg)
	client := clientFactory(context.Background())

	for i := 0; i < 100; i++ {
		resp, err := client.Get(URL)
		if err != nil {
			log.Println(err)
			t.Error(err)
			return
		}
		response, err := ioutil.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			t.Error(err)
			return
		}
		if string(response) != statusOKMsg {
			t.Error("unexpected body:", string(response))
		}
	}
}

func TestBackendFactory(t *testing.T) {
	testCacheSystem(t, func(t *testing.T, testURL string) {
		backendFactory := BackendFactory(sampleCfg)
		backendProxy := backendFactory(sampleCfg)
		ctx := context.Background()
		URL, _ := url.Parse(testURL)

		for i := 0; i < 100; i++ {
			req := &proxy.Request{
				Method: "GET",
				URL:    URL,
				Body:   ioutil.NopCloser(bytes.NewBufferString("")),
			}
			resp, err := backendProxy(ctx, req)
			if err != nil {
				t.Error(err)
				return
			}
			if !resp.IsComplete {
				t.Error("incomplete response:", *resp)
			}
		}
	}, 1)
}

var (
	statusOKMsg = `{"status": "ok"}`
	sampleCfg   = &config.Backend{
		Decoder: encoding.JSONDecoder,
		ExtraConfig: map[string]interface{}{
			Namespace: map[string]interface{}{},
		},
	}
)

func testCacheSystem(t *testing.T, f func(*testing.T, string), expected uint64) {
	var ops uint64 = 0
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddUint64(&ops, 1)
		w.Header().Set("Cache-Control", "public, max-age=300")
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, statusOKMsg)
	}))
	defer testServer.Close()

	f(t, testServer.URL)

	opsFinal := atomic.LoadUint64(&ops)
	if opsFinal != expected {
		t.Errorf("the server should not being hited just %d time(s). Total requests: %d\n", expected, opsFinal)
	}
}
