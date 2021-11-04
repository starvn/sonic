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

package jsonschema

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/starvn/turbo/config"
	"github.com/starvn/turbo/log"
	"github.com/starvn/turbo/proxy"
	"io/ioutil"
	"strings"
	"testing"
)

func TestProxyFactory_erroredNext(t *testing.T) {
	errExpected := errors.New("proxy factory called")
	pf := ProxyFactory(log.NoOp, proxy.FactoryFunc(func(cfg *config.EndpointConfig) (proxy.Proxy, error) {
		return func(_ context.Context, _ *proxy.Request) (*proxy.Response, error) {
			t.Error("proxy called")
			return nil, errors.New("proxy called")
		}, errExpected
	}))

	_, err := pf.New(&config.EndpointConfig{})
	if err == nil {
		t.Error("error expected")
		return
	}
	if err != errExpected {
		t.Errorf("unexpected error %s", err.Error())
	}
}

func TestProxyFactory_bypass(t *testing.T) {
	errExpected := errors.New("proxy called")
	pf := ProxyFactory(log.NoOp, proxy.FactoryFunc(func(cfg *config.EndpointConfig) (proxy.Proxy, error) {
		return func(_ context.Context, _ *proxy.Request) (*proxy.Response, error) {
			return nil, errExpected
		}, nil
	}))
	p, err := pf.New(&config.EndpointConfig{})
	if err != nil {
		t.Errorf("unexpected error %s", err.Error())
		return
	}
	if _, err := p(context.Background(), &proxy.Request{Body: ioutil.NopCloser(bytes.NewBufferString(""))}); err != errExpected {
		t.Errorf("unexpected error %v", err)
	}
}

func TestProxyFactory_validationFail(t *testing.T) {
	errExpected := "- (root): Invalid type. Expected:"
	pf := ProxyFactory(log.NoOp, proxy.FactoryFunc(func(cfg *config.EndpointConfig) (proxy.Proxy, error) {
		return func(_ context.Context, _ *proxy.Request) (*proxy.Response, error) {
			t.Error("proxy called!")
			return nil, nil
		}, nil
	}))

	for _, tc := range []string{
		`{"type": "string"}`,
		`{"type": "array"}`,
		`{"type": "boolean"}`,
		`{"type": "number"}`,
		`{"type": "integer"}`,
	} {
		cfg := map[string]interface{}{}
		if err := json.Unmarshal([]byte(tc), &cfg); err != nil {
			t.Error(err)
			return
		}
		p, err := pf.New(&config.EndpointConfig{
			ExtraConfig: map[string]interface{}{
				Namespace: cfg,
			},
		})
		if err != nil {
			t.Errorf("unexpected error %s", err.Error())
			return
		}
		_, err = p(context.Background(), &proxy.Request{Body: ioutil.NopCloser(bytes.NewBufferString("{}"))})
		if err == nil {
			t.Error("expecting error")
			return
		}
		if !strings.Contains(err.Error(), errExpected) {
			t.Errorf("unexpected error %s", err.Error())
		}
	}
}

func TestProxyFactory_validationOK(t *testing.T) {
	errExpected := errors.New("proxy called")
	pf := ProxyFactory(log.NoOp, proxy.FactoryFunc(func(cfg *config.EndpointConfig) (proxy.Proxy, error) {
		return func(_ context.Context, _ *proxy.Request) (*proxy.Response, error) {
			return nil, errExpected
		}, nil
	}))

	for _, tc := range []string{
		`{"type": "object"}`,
	} {
		cfg := map[string]interface{}{}
		if err := json.Unmarshal([]byte(tc), &cfg); err != nil {
			t.Error(err)
			return
		}
		p, err := pf.New(&config.EndpointConfig{
			ExtraConfig: map[string]interface{}{
				Namespace: cfg,
			},
		})
		if err != nil {
			t.Errorf("unexpected error %s", err.Error())
			return
		}
		_, err = p(context.Background(), &proxy.Request{Body: ioutil.NopCloser(bytes.NewBufferString("{}"))})
		if err == nil {
			t.Error("expecting error")
			return
		}
		if err != errExpected {
			t.Errorf("unexpected error %s", err.Error())
		}
	}
}
