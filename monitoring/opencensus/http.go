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

package opencensus

import (
	"context"
	"github.com/starvn/turbo/config"
	transport "github.com/starvn/turbo/transport/http/client"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/tag"
	"go.opencensus.io/trace"
	"net/http"
)

var defaultClient = &http.Client{Transport: &ochttp.Transport{}}

func NewHTTPClient(ctx context.Context) *http.Client {
	if !IsBackendEnabled() {
		return transport.NewHTTPClient(ctx)
	}
	return defaultClient
}

func HTTPRequestExecutor(clientFactory transport.HTTPClientFactory) transport.HTTPRequestExecutor {
	return HTTPRequestExecutorFromConfig(clientFactory, nil)
}

func HTTPRequestExecutorFromConfig(clientFactory transport.HTTPClientFactory, cfg *config.Backend) transport.HTTPRequestExecutor {
	if !IsBackendEnabled() {
		return transport.DefaultHTTPRequestExecutor(clientFactory)
	}
	pathExtractor := GetAggregatedPathForBackendMetrics(cfg)
	return func(ctx context.Context, req *http.Request) (*http.Response, error) {
		client := clientFactory(ctx)
		if _, ok := client.Transport.(*Transport); !ok {
			tags := []tagGenerator{
				func(r *http.Request) tag.Mutator { return tag.Upsert(ochttp.KeyClientHost, req.Host) },
				func(r *http.Request) tag.Mutator { return tag.Upsert(ochttp.KeyClientPath, pathExtractor(r)) },
				func(r *http.Request) tag.Mutator { return tag.Upsert(ochttp.KeyClientMethod, req.Method) },
			}
			client.Transport = &Transport{Base: client.Transport, tags: tags}
		}

		return client.Do(req.WithContext(trace.NewContext(ctx, fromContext(ctx))))
	}
}
