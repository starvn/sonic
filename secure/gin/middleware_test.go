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
	"github.com/gin-gonic/gin"
	"github.com/starvn/sonic/secure"
	"github.com/starvn/turbo/config"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRegister(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	cfg := config.ExtraConfig{
		secure.Namespace: map[string]interface{}{
			"allowed_hosts": []interface{}{"host1", "subdomain1.host2", "subdomain2.host2"},
		},
	}
	if err := Register(cfg, engine); err != nil {
		t.Error(err)
		return
	}
	engine.GET("/should_access", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
	engine.GET("/never_access", func(c *gin.Context) {
		t.Error("unexpected request!", c.Request.URL.String())
		c.JSON(418, gin.H{"status": "ko"})
	})
	engine.GET("/no_headers", func(c *gin.Context) {
		c.Redirect(http.StatusPermanentRedirect, "/target")
	})

	for status, URLs := range map[int][]string{
		http.StatusOK: {
			"http://host1/should_access",
			"https://host1/should_access",
			"http://subdomain1.host2/should_access",
			"https://subdomain2.host2/should_access",
		},
		http.StatusInternalServerError: {
			"http://unknown/never_access",
			"https://subdomain.host1/never_access",
			"http://host2/never_access",
			"https://subdomain3.host2/never_access",
		},
	} {
		for _, URL := range URLs {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", URL, nil)
			engine.ServeHTTP(w, req)
			if w.Result().StatusCode != status {
				t.Errorf("request %s unexpected status code! want %d, have %d\n", URL, status, w.Result().StatusCode)
			}
		}
	}

	URL := "https://host1/no_headers"
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", URL, nil)
	engine.ServeHTTP(w, req)
	if w.Result().StatusCode != http.StatusPermanentRedirect {
		t.Errorf("request %s unexpected status code! want %d, have %d\n", URL, http.StatusPermanentRedirect, w.Result().StatusCode)
	}
	if w.Result().Header.Get("Location") != "/target" {
		t.Error("unexpected value for the location header:", w.Result().Header.Get("Location"))
	}
	if len(w.Result().Header) != 2 {
		t.Error("unexpected number of headers:", len(w.Result().Header), w.Result().Header)
	}
}

func TestRegister_ko(t *testing.T) {
	err := Register(config.ExtraConfig{}, nil)
	if err != ErrNoConfig {
		t.Error("expecting ErrNoConfig. got:", err)
	}
}
