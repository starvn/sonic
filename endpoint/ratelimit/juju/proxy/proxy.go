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

// Package proxy provides a rate-limit proxy middleware using the github.com/juju/ratelimit lib
package proxy

import (
	"context"
	srate "github.com/starvn/sonic/endpoint/ratelimit"
	"github.com/starvn/sonic/endpoint/ratelimit/juju"
	"github.com/starvn/turbo/config"
	"github.com/starvn/turbo/proxy"
)

const Namespace = "github.com/starvn/sonic/endpoint/ratelimit/juju/proxy"

type Config struct {
	MaxRate  float64
	Capacity int64
}

func BackendFactory(next proxy.BackendFactory) proxy.BackendFactory {
	return func(cfg *config.Backend) proxy.Proxy {
		return NewMiddleware(cfg)(next(cfg))
	}
}

func NewMiddleware(remote *config.Backend) proxy.Middleware {
	cfg := ConfigGetter(remote.ExtraConfig).(Config)
	if cfg == ZeroCfg || cfg.MaxRate <= 0 {
		return proxy.EmptyMiddleware
	}
	tb := juju.NewLimiter(cfg.MaxRate, cfg.Capacity)
	return func(next ...proxy.Proxy) proxy.Proxy {
		if len(next) > 1 {
			panic(proxy.ErrTooManyProxies)
		}
		return func(ctx context.Context, request *proxy.Request) (*proxy.Response, error) {
			if !tb.Allow() {
				return nil, srate.ErrLimited
			}
			return next[0](ctx, request)
		}
	}
}

var ZeroCfg = Config{}

func ConfigGetter(e config.ExtraConfig) interface{} {
	v, ok := e[Namespace]
	if !ok {
		return ZeroCfg
	}
	tmp, ok := v.(map[string]interface{})
	if !ok {
		return ZeroCfg
	}
	cfg := Config{}
	if v, ok := tmp["maxRate"]; ok {
		switch val := v.(type) {
		case float64:
			cfg.MaxRate = val
		case int:
			cfg.MaxRate = float64(val)
		case int64:
			cfg.MaxRate = float64(val)
		}
	}
	if v, ok := tmp["capacity"]; ok {
		switch val := v.(type) {
		case int64:
			cfg.Capacity = val
		case int:
			cfg.Capacity = int64(val)
		case float64:
			cfg.Capacity = int64(val)
		}
	}
	return cfg
}
