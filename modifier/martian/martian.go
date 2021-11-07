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

package martian

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/google/martian"
	_ "github.com/google/martian/body"
	_ "github.com/google/martian/cookie"
	_ "github.com/google/martian/fifo"
	_ "github.com/google/martian/header"
	_ "github.com/google/martian/martianurl"
	"github.com/google/martian/parse"
	_ "github.com/google/martian/port"
	_ "github.com/google/martian/priority"
	_ "github.com/google/martian/stash"
	_ "github.com/google/martian/status"
	"github.com/starvn/turbo/config"
	"github.com/starvn/turbo/log"
	"github.com/starvn/turbo/proxy"
	"github.com/starvn/turbo/transport/http/client"
	"io/ioutil"
	"net/http"
)

func NewBackendFactory(logger log.Logger, re client.HTTPRequestExecutor) proxy.BackendFactory {
	return NewConfiguredBackendFactory(logger, func(_ *config.Backend) client.HTTPRequestExecutor { return re })
}

func NewConfiguredBackendFactory(logger log.Logger, ref func(*config.Backend) client.HTTPRequestExecutor) proxy.BackendFactory {
	parse.Register("static.Modifier", staticModifierFromJSON)

	return func(remote *config.Backend) proxy.Proxy {
		re := ref(remote)
		result, ok := ConfigGetter(remote.ExtraConfig).(Result)
		if !ok {
			return proxy.NewHTTPProxyWithHTTPExecutor(remote, re, remote.Decoder)
		}
		switch result.Err {
		case nil:
			return proxy.NewHTTPProxyWithHTTPExecutor(remote, HTTPRequestExecutor(result.Result, re), remote.Decoder)
		case ErrEmptyValue:
			return proxy.NewHTTPProxyWithHTTPExecutor(remote, re, remote.Decoder)
		default:
			logger.Error(result, remote.ExtraConfig)
			return proxy.NewHTTPProxyWithHTTPExecutor(remote, re, remote.Decoder)
		}
	}
}

func HTTPRequestExecutor(result *parse.Result, re client.HTTPRequestExecutor) client.HTTPRequestExecutor {
	return func(ctx context.Context, req *http.Request) (resp *http.Response, err error) {
		if err = modifyRequest(result.RequestModifier(), req); err != nil {
			return
		}

		mctx, ok := req.Context().(*Context)
		if !ok || !mctx.SkippingRoundTrip() {
			resp, err = re(ctx, req)
			if err != nil {
				return
			}
			if resp == nil {
				err = ErrEmptyResponse
				return
			}
		} else if resp == nil {
			resp = &http.Response{
				Request:    req,
				Header:     http.Header{},
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(bytes.NewBufferString("")),
			}
		}

		err = modifyResponse(result.ResponseModifier(), resp)
		return
	}
}

func modifyRequest(mod martian.RequestModifier, req *http.Request) error {
	if req.Body == nil {
		req.Body = ioutil.NopCloser(bytes.NewBufferString(""))
	}
	if req.Header == nil {
		req.Header = http.Header{}
	}

	if mod == nil {
		return nil
	}
	return mod.ModifyRequest(req)
}

func modifyResponse(mod martian.ResponseModifier, resp *http.Response) error {
	if resp.Body == nil {
		resp.Body = ioutil.NopCloser(bytes.NewBufferString(""))
	}
	if resp.Header == nil {
		resp.Header = http.Header{}
	}
	if resp.StatusCode == 0 {
		resp.StatusCode = http.StatusOK
	}

	if mod == nil {
		return nil
	}
	return mod.ModifyResponse(resp)
}

const Namespace = "github.com/starvn/sonic/modifier/martian"

type Result struct {
	Result *parse.Result
	Err    error
}

func ConfigGetter(e config.ExtraConfig) interface{} {
	cfg, ok := e[Namespace]
	if !ok {
		return Result{nil, ErrEmptyValue}
	}

	data, ok := cfg.(map[string]interface{})
	if !ok {
		return Result{nil, ErrBadValue}
	}

	raw, err := json.Marshal(data)
	if err != nil {
		return Result{nil, ErrMarshallingValue}
	}

	r, err := parse.FromJSON(raw)

	return Result{r, err}
}

var (
	ErrEmptyValue       = errors.New("getting the extra config for the martian module")
	ErrBadValue         = errors.New("casting the extra config for the martian module")
	ErrMarshallingValue = errors.New("marshalling the extra config for the martian module")
	ErrEmptyResponse    = errors.New("getting the http response from the request executor")
)
