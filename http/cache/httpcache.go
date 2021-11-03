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

// Package cache introduces an in-memory-cached http client into the Sonic stack
package cache

import (
	"context"
	"github.com/gregjones/httpcache"
	"github.com/starvn/turbo/config"
	"github.com/starvn/turbo/proxy"
	"github.com/starvn/turbo/transport/http/client"
	"net/http"
)

const Namespace = "github.com/starvn/sonic/http/cache"

var (
	memTransport = httpcache.NewMemoryCacheTransport()
	memClient    = http.Client{Transport: memTransport}
)

func NewHTTPClient(cfg *config.Backend) client.HTTPClientFactory {
	_, ok := cfg.ExtraConfig[Namespace]
	if !ok {
		return client.NewHTTPClient
	}
	return func(_ context.Context) *http.Client {
		return &memClient
	}
}

func BackendFactory(cfg *config.Backend) proxy.BackendFactory {
	return proxy.CustomHTTPProxyFactory(NewHTTPClient(cfg))
}
