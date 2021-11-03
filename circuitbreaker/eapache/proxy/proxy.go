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
	"github.com/starvn/sonic/circuitbreaker/eapache"
	"github.com/starvn/turbo/config"
	"github.com/starvn/turbo/proxy"
)

func BackendFactory(next proxy.BackendFactory) proxy.BackendFactory {
	return func(cfg *config.Backend) proxy.Proxy {
		return NewMiddleware(cfg)(next(cfg))
	}
}

func NewMiddleware(remote *config.Backend) proxy.Middleware {
	data := eapache.ConfigGetter(remote.ExtraConfig).(eapache.Config)
	if data == eapache.ZeroCfg {
		return proxy.EmptyMiddleware
	}
	cb := eapache.NewCircuitBreaker(data)

	return func(next ...proxy.Proxy) proxy.Proxy {
		if len(next) > 1 {
			panic(proxy.ErrTooManyProxies)
		}
		return func(ctx context.Context, request *proxy.Request) (*proxy.Response, error) {
			var res *proxy.Response
			if err := cb.Run(func() error {
				var err1 error
				res, err1 = next[0](ctx, request)
				return err1
			}); err != nil {
				return nil, err
			}
			return res, nil
		}
	}
}
