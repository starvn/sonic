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
	"fmt"
	"github.com/starvn/sonic/qos/circuitbreaker/gobreaker"
	"github.com/starvn/turbo/config"
	"github.com/starvn/turbo/log"
	"github.com/starvn/turbo/proxy"
)

func BackendFactory(next proxy.BackendFactory, logger log.Logger) proxy.BackendFactory {
	return func(cfg *config.Backend) proxy.Proxy {
		return NewMiddleware(cfg, logger)(next(cfg))
	}
}

func NewMiddleware(remote *config.Backend, logger log.Logger) proxy.Middleware {
	data := gobreaker.ConfigGetter(remote.ExtraConfig).(gobreaker.Config)
	if data == gobreaker.ZeroCfg {
		return proxy.EmptyMiddleware
	}
	cb := gobreaker.NewCircuitBreaker(data, logger)

	logger.Debug(fmt.Sprintf("[BACKEND: %s][CB] Creating the circuit breaker named '%s'", remote.URLPattern, data.Name))

	return func(next ...proxy.Proxy) proxy.Proxy {
		if len(next) > 1 {
			panic(proxy.ErrTooManyProxies)
		}
		return func(ctx context.Context, request *proxy.Request) (*proxy.Response, error) {
			result, err := cb.Execute(func() (interface{}, error) { return next[0](ctx, request) })
			if err != nil {
				return nil, err
			}
			return result.(*proxy.Response), err
		}
	}
}
