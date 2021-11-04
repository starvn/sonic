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
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/starvn/sonic/config/detector/sonic"
	"github.com/starvn/turbo/config"
	"github.com/starvn/turbo/log"
	"github.com/starvn/turbo/proxy"
	sgin "github.com/starvn/turbo/route/gin"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRegister(t *testing.T) {
	gin.SetMode(gin.TestMode)

	engine := gin.New()

	cfg := config.ServiceConfig{
		ExtraConfig: config.ExtraConfig{
			sonic.Namespace: map[string]interface{}{
				"deny":  []interface{}{"a", "b"},
				"allow": []interface{}{"c", "Pingdom.com_bot_version_1.1"},
				"patterns": []interface{}{
					`(Pingdom.com_bot_version_)(\d+)\.(\d+)`,
					`(facebookexternalhit)/(\d+)\.(\d+)`,
				},
			},
		},
	}

	Register(cfg, log.NoOp, engine)

	engine.GET("/", func(c *gin.Context) {
		c.String(200, "hi!")
	})

	if err := testDetection(engine); err != nil {
		t.Error(err)
	}
}

func TestNew(t *testing.T) {
	gin.SetMode(gin.TestMode)

	engine := gin.New()

	cfg := &config.EndpointConfig{
		ExtraConfig: config.ExtraConfig{
			sonic.Namespace: map[string]interface{}{
				"deny":  []interface{}{"a", "b"},
				"allow": []interface{}{"c", "Pingdom.com_bot_version_1.1"},
				"patterns": []interface{}{
					`(Pingdom.com_bot_version_)(\d+)\.(\d+)`,
					`(facebookexternalhit)/(\d+)\.(\d+)`,
				},
			},
		},
	}

	proxyfunc := func(_ context.Context, _ *proxy.Request) (*proxy.Response, error) {
		return &proxy.Response{IsComplete: true}, nil
	}

	engine.GET("/", New(sgin.EndpointHandler, log.NoOp)(cfg, proxyfunc))

	if err := testDetection(engine); err != nil {
		t.Error(err)
	}
}

func testDetection(engine *gin.Engine) error {
	for i, ua := range []string{
		"abcd",
		"",
		"c",
		"Pingdom.com_bot_version_1.1",
	} {
		req, _ := http.NewRequest("GET", "http://example.com/", nil)
		req.Header.Add("User-Agent", ua)

		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		if w.Result().StatusCode != 200 {
			return fmt.Errorf("the req #%d has been detected as a bot: %s", i, ua)
		}
	}

	for i, ua := range []string{
		"a",
		"b",
		"facebookexternalhit/1.1",
		"Pingdom.com_bot_version_1.2",
	} {
		req, _ := http.NewRequest("GET", "http://example.com/", nil)
		req.Header.Add("User-Agent", ua)

		w := httptest.NewRecorder()
		engine.ServeHTTP(w, req)

		if w.Result().StatusCode != http.StatusForbidden {
			return fmt.Errorf("the req #%d has not been detected as a bot: %s", i, ua)
		}
	}
	return nil
}
