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
	"github.com/starvn/turbo/log"
	"github.com/starvn/turbo/proxy"
	smux "github.com/starvn/turbo/route/mux"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTokenSignatureValidator(t *testing.T) {
	server := httptest.NewServer(jwkEndpoint("public"))
	defer server.Close()

	validatorEndpointCfg := newVerifierEndpointCfg("RS256", server.URL, []string{"role_a"})

	forbiddenEndpointCfg := newVerifierEndpointCfg("RS256", server.URL, []string{"role_c"})
	forbiddenEndpointCfg.Endpoint = "/forbidden"

	registeredEndpointCfg := newVerifierEndpointCfg("RS256", server.URL, []string{})
	registeredEndpointCfg.Endpoint = "/registered"

	propagateHeadersEndpointCfg := newVerifierEndpointCfg("RS256", server.URL, []string{})
	propagateHeadersEndpointCfg.Endpoint = "/propagateheaders"

	token := "eyJhbGciOiJSUzI1NiIsImtpZCI6IjIwMTEtMDQtMjkifQ.eyJhdWQiOiJodHRwOi8vYXBpLmV4YW1wbGUuY29tIiwiZXhwIjoxNzM1Njg5NjAwLCJpc3MiOiJodHRwOi8vZXhhbXBsZS5jb20iLCJqdGkiOiJtbmIyM3Zjc3J0NzU2eXVpb21uYnZjeDk4ZXJ0eXVpb3AiLCJyb2xlcyI6WyJyb2xlX2EiLCJyb2xlX2IiXSwic3ViIjoiMTIzNDU2Nzg5MHF3ZXJ0eXVpbyJ9.NrLwxZK8UhS6CV2ijdJLUfAinpjBn5_uliZCdzQ7v-Dc8lcv1AQA9cYsG63RseKWH9u6-TqPKMZQ56WfhqL028BLDdQCiaeuBoLzYU1tQLakA1V0YmouuEVixWLzueVaQhyGx-iKuiuFhzHWZSqFqSehiyzI9fb5O6Gcc2L6rMEoxQMaJomVS93h-t013MNq3ADLWTXRaO-negydqax_WmzlVWp_RDroR0s5J2L2klgmBXVwh6SYy5vg7RrnuN3S8g4oSicJIi9NgnG-dDikuaOg2DeFUt-mYq_j_PbNXf9TUl5hl4kEy7E0JauJ17d1BUuTl3ChY4BOmhQYRN0dYg"

	dummyProxy := func(ctx context.Context, req *proxy.Request) (*proxy.Response, error) {
		return &proxy.Response{
			Data: map[string]interface{}{
				"aaaa": map[string]interface{}{
					"foo": "a",
					"bar": "b",
				},
				"bbbb": true,
				"cccc": 1234567890,
			},
			IsComplete: true,
			Metadata: proxy.Metadata{
				StatusCode: 200,
			},
		}, nil
	}

	buf := new(bytes.Buffer)
	logger, _ := log.NewLogger("DEBUG", buf, "")
	hf := HandlerFactory(smux.EndpointHandler, dummyParamsExtractor, logger, nil)

	engine := smux.DefaultEngine()

	engine.Handle(validatorEndpointCfg.Endpoint, "GET", hf(validatorEndpointCfg, dummyProxy))
	engine.Handle(forbiddenEndpointCfg.Endpoint, "GET", hf(forbiddenEndpointCfg, dummyProxy))
	engine.Handle(registeredEndpointCfg.Endpoint, "GET", hf(registeredEndpointCfg, dummyProxy))
	engine.Handle(propagateHeadersEndpointCfg.Endpoint, "GET", hf(propagateHeadersEndpointCfg, dummyProxy))

	req := httptest.NewRequest("GET", forbiddenEndpointCfg.Endpoint, new(bytes.Buffer))

	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("unexpected status code: %d", w.Code)
	}
	if body := w.Body.String(); body != "Token not found\n" {
		t.Errorf("unexpected body: '%s'", body)
	}

	req = httptest.NewRequest("GET", validatorEndpointCfg.Endpoint, new(bytes.Buffer))
	req.Header.Set("Authorization", "BEARER "+token)

	w = httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("unexpected status code: %d", w.Code)
	}
	if body := w.Body.String(); body != `{"aaaa":{"bar":"b","foo":"a"},"bbbb":true,"cccc":1234567890}` {
		t.Errorf("unexpected body: %s", body)
	}

	if logging := buf.String(); !strings.Contains(logging, "INFO: JOSE: signer disabled for the endpoint /private") {
		t.Error(logging)
	}

	req = httptest.NewRequest("GET", forbiddenEndpointCfg.Endpoint, new(bytes.Buffer))
	req.Header.Set("Authorization", "BEARER "+token)

	w = httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("unexpected status code: %d", w.Code)
	}
	if body := w.Body.String(); body != "\n" {
		t.Errorf("unexpected body: %s", body)
	}

	req = httptest.NewRequest("GET", registeredEndpointCfg.Endpoint, new(bytes.Buffer))
	req.Header.Set("Authorization", "BEARER "+token)

	w = httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("unexpected status code: %d", w.Code)
	}
	if body := w.Body.String(); body != `{"aaaa":{"bar":"b","foo":"a"},"bbbb":true,"cccc":1234567890}` {
		t.Errorf("unexpected body: %s", body)
	}

	req = httptest.NewRequest("GET", propagateHeadersEndpointCfg.Endpoint, new(bytes.Buffer))
	req.Header.Set("Authorization", "BEARER "+token)
	// Check header-overwrite: it must be overwritten by a claim in the JWT!
	req.Header.Set("x-sonic-replace", "abc")

	w = httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if req.Header.Get("x-sonic-jti") == "" {
		t.Error("JWT claim not propagated to header: jti")
	} else if req.Header.Get("x-sonic-jti") != "mnb23vcsrt756yuiomnbvcx98ertyuiop" {
		t.Errorf("wrong JWT claim propagated for 'jti': %v", req.Header.Get("x-sonic-jti"))
	}

	// Check that existing header values are overwritten
	if req.Header.Get("x-sonic-replace") == "abc" {
		t.Error("JWT claim not propagated to x-sonic-replace header: sub")
	} else if req.Header.Get("x-sonic-replace") != "1234567890qwertyuio" {
		t.Errorf("wrong JWT claim propagated for 'sub': %v", req.Header.Get("x-sonic-replace"))
	}

	if req.Header.Get("x-sonic-sub") == "" {
		t.Error("JWT claim not propagated to header: sub")
	} else if req.Header.Get("x-sonic-sub") != "1234567890qwertyuio" {
		t.Errorf("wrong JWT claim propagated for 'sub': %v", req.Header.Get("x-sonic-sub"))
	}

	if req.Header.Get("x-sonic-ne") != "" {
		t.Error("JWT claim propagated, although it shouldn't: nonexistent")
	}

	if w.Code != http.StatusOK {
		t.Errorf("unexpected status code: %d", w.Code)
	}
	if body := w.Body.String(); body != `{"aaaa":{"bar":"b","foo":"a"},"bbbb":true,"cccc":1234567890}` {
		t.Errorf("unexpected body: %s", body)
	}
}

func jwkEndpoint(name string) http.HandlerFunc {
	data, err := ioutil.ReadFile("../fixture/" + name + ".json")
	return func(rw http.ResponseWriter, _ *http.Request) {
		if err != nil {
			rw.WriteHeader(500)
			return
		}
		rw.Header().Set("Content-Type", "application/json")
		_, _ = rw.Write(data)
	}
}

func dummyParamsExtractor(_ *http.Request) map[string]string {
	return map[string]string{}
}
