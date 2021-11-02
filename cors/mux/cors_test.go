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
	"encoding/json"
	"github.com/starvn/turbo/log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInvalidCfg(t *testing.T) {
	sampleCfg := map[string]interface{}{}
	corsMw := New(sampleCfg)
	if corsMw != nil {
		t.Error("The corsMw should be nil.\n")
	}
}

func TestNew(t *testing.T) {
	sampleCfg := map[string]interface{}{}
	serialized := []byte(`{ "github.com/starvn/sonic/cors": {
			"allow_origins": [ "http://foobar.com" ],
			"allow_methods": [ "GET" ],
			"max_age": "2h"
			}
		}`)
	_ = json.Unmarshal(serialized, &sampleCfg)
	h := New(sampleCfg)
	res := httptest.NewRecorder()
	req, _ := http.NewRequest("OPTIONS", "http://example.com/foo", nil)
	req.Header.Add("Origin", "http://foobar.com")
	req.Header.Add("Access-Control-Request-Method", "GET")
	req.Header.Add("Access-Control-Request-Headers", "origin")
	handler := h.Handler(testHandler)
	handler.ServeHTTP(res, req)

	assertHeaders(t, res.Header(), map[string]string{
		"Vary":                         "Origin, Access-Control-Request-Method, Access-Control-Request-Headers",
		"Access-Control-Allow-Origin":  "http://foobar.com",
		"Access-Control-Allow-Methods": "GET",
		"Access-Control-Allow-Headers": "Origin",
		"Access-Control-Max-Age":       "7200",
	})
}

func TestNewWithLogger(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	logger, err := log.NewLogger("DEBUG", buf, "")
	if err != nil {
		t.Error(err)
		return
	}
	sampleCfg := map[string]interface{}{}
	serialized := []byte(`{ "github.com/starvn/sonic/cors": {
			"allow_origins": [ "http://foobar.com" ],
			"allow_methods": [ "GET" ],
			"max_age": "2h"
			}
		}`)
	_ = json.Unmarshal(serialized, &sampleCfg)
	h := NewWithLogger(sampleCfg, logger)
	res := httptest.NewRecorder()
	req, _ := http.NewRequest("OPTIONS", "http://example.com/foo", nil)
	req.Header.Add("Origin", "http://foobar.com")
	handler := h.Handler(testHandler)
	handler.ServeHTTP(res, req)
	if res.Code != 200 {
		t.Errorf("Invalid status code: %d should be 200", res.Code)
	}

	assertHeaders(t, res.Header(), map[string]string{
		"Vary":                         "",
		"Access-Control-Allow-Origin":  "",
		"Access-Control-Allow-Methods": "",
		"Access-Control-Allow-Headers": "",
		"Access-Control-Max-Age":       "",
	})

	loggedMsg := buf.String()
	if len(loggedMsg) != 0 {
		t.Error("unexpected logged msg:", loggedMsg)
	}
}

func TestAllowOriginEmpty(t *testing.T) {
	sampleCfg := map[string]interface{}{}
	serialized := []byte(`{ "github.com/starvn/sonic/cors": {
			}
		}`)
	_ = json.Unmarshal(serialized, &sampleCfg)
	h := New(sampleCfg)
	res := httptest.NewRecorder()
	req, _ := http.NewRequest("OPTIONS", "http://example.com/foo", nil)
	req.Header.Add("Access-Control-Request-Method", "GET")
	req.Header.Add("Access-Control-Request-Headers", "origin")
	req.Header.Add("Origin", "http://foobar.com")
	handler := h.Handler(testHandler)
	handler.ServeHTTP(res, req)
	if res.Code != 200 {
		t.Errorf("Invalid status code: %d should be 200", res.Code)
	}

	assertHeaders(t, res.Header(), map[string]string{
		"Vary":                         "Origin, Access-Control-Request-Method, Access-Control-Request-Headers",
		"Access-Control-Allow-Origin":  "*",
		"Access-Control-Allow-Methods": "GET",
		"Access-Control-Allow-Headers": "Origin",
	})
}

var allHeaders = []string{
	"Vary",
	"Access-Control-Allow-Origin",
	"Access-Control-Allow-Methods",
	"Access-Control-Allow-Headers",
	"Access-Control-Allow-Credentials",
	"Access-Control-Max-Age",
	"Access-Control-Expose-Headers",
}

func assertHeaders(t *testing.T, resHeaders http.Header, expHeaders map[string]string) {
	for _, name := range allHeaders {
		got := strings.Join(resHeaders[name], ", ")
		want := expHeaders[name]
		if got != want {
			t.Errorf("Response header %q = %q, want %q", name, got, want)
		}
	}
}

var testHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte("bar"))
})
