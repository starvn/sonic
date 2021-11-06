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

package mux

import (
	"github.com/starvn/sonic/monitoring/opencensus"
	"github.com/starvn/turbo/config"
	"github.com/starvn/turbo/proxy"
	"github.com/starvn/turbo/route/mux"
	"go.opencensus.io/plugin/ochttp"
	"net/http"
)

func New(hf mux.HandlerFactory) mux.HandlerFactory {
	if !opencensus.IsRouterEnabled() {
		return hf
	}
	return func(cfg *config.EndpointConfig, p proxy.Proxy) http.HandlerFunc {
		handler := ochttp.Handler{Handler: tagAggregationMiddleware(hf(cfg, p), cfg)}
		return handler.ServeHTTP
	}
}

func tagAggregationMiddleware(next http.Handler, cfg *config.EndpointConfig) http.Handler {
	pathExtractor := opencensus.GetAggregatedPathForMetrics(cfg)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ochttp.SetRoute(r.Context(), pathExtractor(r))
		next.ServeHTTP(w, r)
	})
}
