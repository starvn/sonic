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
	"github.com/starvn/sonic/auth/jose"
	ginjose "github.com/starvn/sonic/auth/jose/gin"
	lua "github.com/starvn/sonic/modifier/interpreter/route/gin"
	juju "github.com/starvn/sonic/qos/ratelimit/juju/router/gin"
	detector "github.com/starvn/sonic/security/detector/gin"
	metrics "github.com/starvn/sonic/telemetry/metrics/gin"
	opencensus "github.com/starvn/sonic/telemetry/opencensus/router/gin"
	"github.com/starvn/turbo/log"
	router "github.com/starvn/turbo/route/gin"
	"github.com/starvn/turbo/transport/http/server"
)

func NewHandlerFactory(logger log.Logger, metricCollector *metrics.Metrics, rejecter jose.RejecterFactory) router.HandlerFactory {
	handlerFactory := router.CustomErrorEndpointHandler(logger, server.DefaultToHTTPError)
	handlerFactory = juju.NewRateLimiterMw(handlerFactory)
	handlerFactory = lua.HandlerFactory(logger, handlerFactory)
	handlerFactory = ginjose.HandlerFactory(handlerFactory, logger, rejecter)
	handlerFactory = metricCollector.NewHTTPHandlerFactory(handlerFactory)
	handlerFactory = opencensus.New(handlerFactory)
	handlerFactory = detector.New(handlerFactory, logger)
	return handlerFactory
}

type handlerFactory struct{}

func (h handlerFactory) NewHandlerFactory(l log.Logger, m *metrics.Metrics, r jose.RejecterFactory) router.HandlerFactory {
	return NewHandlerFactory(l, m, r)
}
