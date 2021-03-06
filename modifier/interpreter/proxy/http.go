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

package proxy

import (
	"bytes"
	"context"
	"errors"
	"github.com/alexeyco/binder"
	"github.com/starvn/turbo/transport/http/server"
	lua "github.com/yuin/gopher-lua"
	"io/ioutil"
	"net/http"
	"sync"
)

func registerHTTPRequest(ctx context.Context, b *binder.Binder) {
	t := b.Table("http_response")
	t.Static("new", newHttpResponse(ctx))
	t.Dynamic("statusCode", httpStatus)
	t.Dynamic("headers", httpHeaders)
	t.Dynamic("body", httpBody)
	t.Dynamic("close", closeF)
}

func newHttpResponse(ctx context.Context) func(*binder.Context) error {
	return func(c *binder.Context) error {
		if c.Top() == 0 || c.Top() == 2 {
			return errors.New("need 1, 3 or 4 arguments")
		}

		URL := c.Arg(1).String()
		var req *http.Request

		if c.Top() == 1 {
			req, _ = http.NewRequest("GET", URL, nil)
		} else {
			method := c.Arg(2).String()
			body := c.Arg(3).String()

			var err error
			req, err = http.NewRequest(method, URL, bytes.NewBufferString(body))
			if err != nil {
				return err
			}

			if c.Top() == 4 {
				headers, ok := c.Arg(4).Any().(*lua.LTable)

				if ok {
					headers.ForEach(func(key, value lua.LValue) {
						req.Header.Add(key.String(), value.String())
					})
				}
			}
		}

		resp, err := executeHttpRequest(req.WithContext(ctx))
		if err != nil {
			return err
		}
		if resp == nil {
			return errResponseExpected
		}
		pushHTTPResponse(c, resp)
		return nil
	}
}

func executeHttpRequest(r *http.Request) (*http.Response, error) {
	r.Header.Add("User-Agent", server.UserAgentHeaderValue[0])
	return http.DefaultClient.Do(r)
}

type httpResponse struct {
	once *sync.Once
	r    *http.Response
	body string
}

func (h *httpResponse) Close() {
	if h == nil || h.r == nil || h.r.Body == nil {
		return
	}

	_ = h.r.Body.Close()
	h.r.Body = nil
}

func (h *httpResponse) Body() string {
	h.once.Do(func() {
		b, _ := ioutil.ReadAll(h.r.Body)
		h.Close()
		h.body = string(b)
	})
	return h.body
}

func (h *httpResponse) Header(k string) string {
	return h.r.Header.Get(k)
}

func pushHTTPResponse(c *binder.Context, r *http.Response) {
	c.Push().Data(
		&httpResponse{
			once: new(sync.Once),
			r:    r,
		},
		"http_response",
	)
}

func httpStatus(c *binder.Context) error {
	resp, ok := c.Arg(1).Data().(*httpResponse)
	if !ok {
		return errResponseExpected
	}
	c.Push().Number(float64(resp.r.StatusCode))

	return nil
}

func httpHeaders(c *binder.Context) error {
	resp, ok := c.Arg(1).Data().(*httpResponse)
	if !ok {
		return errResponseExpected
	}
	if c.Top() != 2 {
		return errNeedsArguments
	}
	c.Push().String(resp.Header(c.Arg(2).String()))

	return nil
}

func httpBody(c *binder.Context) error {
	resp, ok := c.Arg(1).Data().(*httpResponse)
	if !ok {
		return errResponseExpected
	}
	c.Push().String(resp.Body())

	return nil
}

func closeF(c *binder.Context) error {
	resp, ok := c.Arg(1).Data().(*httpResponse)
	if !ok {
		return errResponseExpected
	}
	if resp == nil {
		return nil
	}
	resp.Close()
	resp.r = nil
	resp = nil
	return nil
}
