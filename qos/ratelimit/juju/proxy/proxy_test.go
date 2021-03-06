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
	"context"
	srate "github.com/starvn/sonic/qos/ratelimit"
	"github.com/starvn/turbo/config"
	"github.com/starvn/turbo/proxy"
	"sync/atomic"
	"testing"
)

func TestNewMiddleware_multipleNext(t *testing.T) {
	defer func() {
		if r := recover(); r != proxy.ErrTooManyProxies {
			t.Errorf("The code did not panic\n")
		}
	}()
	NewMiddleware(&config.Backend{})(proxy.NoopProxy, proxy.NoopProxy)
}

func TestNewMiddleware_zeroConfig(t *testing.T) {
	for _, cfg := range []*config.Backend{
		{},
		{ExtraConfig: map[string]interface{}{Namespace: 42}},
	} {
		resp := proxy.Response{}
		mdw := NewMiddleware(cfg)
		p := mdw(dummyProxy(&resp, nil))

		request := proxy.Request{
			Path: "/turbo",
		}

		for i := 0; i < 100; i++ {
			r, err := p(context.Background(), &request)
			if err != nil {
				t.Error(err.Error())
				return
			}
			if &resp != r {
				t.Fail()
			}
		}
	}
}

func TestNewMiddleware_ok(t *testing.T) {
	resp := proxy.Response{}
	mdw := NewMiddleware(&config.Backend{
		ExtraConfig: map[string]interface{}{Namespace: map[string]interface{}{"maxRate": 10000.0, "capacity": 10000.0}},
	})
	p := mdw(dummyProxy(&resp, nil))

	request := proxy.Request{
		Path: "/turbo",
	}

	for i := 0; i < 1000; i++ {
		r, err := p(context.Background(), &request)
		if err != nil {
			t.Error(err.Error())
			return
		}
		if &resp != r {
			t.Fail()
		}
	}
}

func TestNewMiddleware_ko(t *testing.T) {
	expected := proxy.Response{}
	calls := uint64(0)
	mdw := NewMiddleware(&config.Backend{
		ExtraConfig: map[string]interface{}{Namespace: map[string]interface{}{"maxRate": 1.0, "capacity": 1.0}},
	})
	p := mdw(func(_ context.Context, _ *proxy.Request) (*proxy.Response, error) {
		total := atomic.AddUint64(&calls, 1)
		if total > 2 {
			t.Error("This proxy shouldn't been executed!")
		}
		return &expected, nil
	})

	request := proxy.Request{
		Path: "/turbo",
	}

	for i := 0; i < 100; i++ {
		_, _ = p(context.Background(), &request)
	}

	r, err := p(context.Background(), &request)
	if err != srate.ErrLimited {
		t.Errorf("error expected")
	}
	if nil != r {
		t.Error("unexpected response")
	}
	if calls != 1 {
		t.Error("unexpected number of calls to the proxy")
	}
}

func dummyProxy(r *proxy.Response, err error) proxy.Proxy {
	return func(_ context.Context, _ *proxy.Request) (*proxy.Response, error) {
		return r, err
	}
}
