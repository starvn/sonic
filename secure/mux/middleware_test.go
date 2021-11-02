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
	"github.com/starvn/sonic/secure"
	"github.com/starvn/turbo/config"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewSecureMw(t *testing.T) {
	cfg := config.ExtraConfig{
		secure.Namespace: map[string]interface{}{
			"allowed_hosts": []interface{}{"host1", "subdomain1.host2", "subdomain2.host2"},
		},
	}
	mw := NewSecureMw(cfg)
	handler := mw.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	for status, URLs := range map[int][]string{
		http.StatusOK: {
			"http://host1/",
			"https://host1/",
			"http://subdomain1.host2/",
			"https://subdomain2.host2/",
		},
		http.StatusInternalServerError: {
			"http://unknown/",
			"https://subdomain.host1/",
			"http://host2/",
			"https://subdomain3.host2/",
		},
	} {
		for _, URL := range URLs {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", URL, nil)
			handler.ServeHTTP(w, req)
			if w.Result().StatusCode != status {
				t.Errorf("request %s unexpected status code! want %d, have %d\n", URL, status, w.Result().StatusCode)
			}
		}
	}
}
