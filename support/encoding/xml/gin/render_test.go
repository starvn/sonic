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
	"github.com/starvn/turbo/proxy"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRender(t *testing.T) {
	gin.SetMode(gin.TestMode)
	server := gin.New()
	server.GET("/", func(c *gin.Context) {
		res := &proxy.Response{
			IsComplete: true,
			Data: map[string]interface{}{
				"a": map[string]interface{}{
					"content": "sonic",
				},
				"content": "turbo",
				"foo":     42,
			},
		}
		Render(c, res)
	})

	expected := `<doc><a><content>sonic</content></a><content>turbo</content><foo>42</foo></doc>`

	req, _ := http.NewRequest("GET", "http://127.0.0.1:8080/", nil)

	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(w.Result().Body)

	body, ioerr := ioutil.ReadAll(w.Result().Body)
	if ioerr != nil {
		t.Error("reading response body:", ioerr)
		return
	}

	content := string(body)
	if w.Result().Header.Get("Content-Type") != gin.MIMEXML {
		t.Error("Content-Type error:", w.Result().Header.Get("Content-Type"))
	}
	if w.Result().StatusCode != http.StatusOK {
		t.Error("Unexpected status code:", w.Result().StatusCode)
	}
	if content != expected {
		t.Error("Unexpected body:", content, "expected:", expected)
	}
}
