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

package sonic

import (
	"context"
	oauth2client "github.com/starvn/sonic/auth/oauth"
	"github.com/starvn/sonic/backend/lambda"
	"github.com/starvn/sonic/backend/pubsub"
	"github.com/starvn/sonic/backend/queue"
	lua "github.com/starvn/sonic/modifier/interpreter/proxy"
	"github.com/starvn/sonic/modifier/martian"
	cb "github.com/starvn/sonic/qos/circuitbreaker/gobreaker/proxy"
	"github.com/starvn/sonic/qos/httpcache"
	juju "github.com/starvn/sonic/qos/ratelimit/juju/proxy"
	metrics "github.com/starvn/sonic/telemetry/metrics/gin"
	"github.com/starvn/sonic/telemetry/opencensus"
	"github.com/starvn/sonic/validation/explang"
	"github.com/starvn/turbo/config"
	"github.com/starvn/turbo/log"
	"github.com/starvn/turbo/proxy"
	"github.com/starvn/turbo/transport/http/client"
	executor "github.com/starvn/turbo/transport/http/client/plugin"
)

func NewBackendFactory(logger log.Logger, metricCollector *metrics.Metrics) proxy.BackendFactory {
	return NewBackendFactoryWithContext(context.Background(), logger, metricCollector)
}

func NewBackendFactoryWithContext(ctx context.Context, logger log.Logger, metricCollector *metrics.Metrics) proxy.BackendFactory {
	requestExecutorFactory := func(cfg *config.Backend) client.HTTPRequestExecutor {
		var clientFactory client.HTTPClientFactory
		if _, ok := cfg.ExtraConfig[oauth2client.Namespace]; ok {
			clientFactory = oauth2client.NewHTTPClient(cfg)
		} else {
			clientFactory = httpcache.NewHTTPClient(cfg)
		}
		return opencensus.HTTPRequestExecutorFromConfig(clientFactory, cfg)
	}
	requestExecutorFactory = executor.HTTPRequestExecutor(logger, requestExecutorFactory)
	backendFactory := martian.NewConfiguredBackendFactory(logger, requestExecutorFactory)
	bf := pubsub.NewBackendFactory(ctx, logger, backendFactory)
	backendFactory = bf.New
	backendFactory = queue.NewBackendFactory(ctx, logger, backendFactory)
	backendFactory = lambda.BackendFactory(logger, backendFactory)
	backendFactory = explang.BackendFactory(logger, backendFactory)
	backendFactory = lua.BackendFactory(logger, backendFactory)
	backendFactory = juju.BackendFactory(backendFactory)
	backendFactory = cb.BackendFactory(backendFactory, logger)
	backendFactory = metricCollector.BackendFactory("backend", backendFactory)
	backendFactory = opencensus.BackendFactory(backendFactory)
	return backendFactory
}

type backendFactory struct{}

func (b backendFactory) NewBackendFactory(ctx context.Context, l log.Logger, m *metrics.Metrics) proxy.BackendFactory {
	return NewBackendFactoryWithContext(ctx, l, m)
}
