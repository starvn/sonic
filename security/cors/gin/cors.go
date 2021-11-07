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

package gin

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/rs/cors"
	wrapper "github.com/rs/cors/wrapper/gin"
	scors "github.com/starvn/sonic/security/cors"
	"github.com/starvn/sonic/security/cors/mux"
	"github.com/starvn/turbo/config"
	"github.com/starvn/turbo/log"
	"net/http"
)

func New(e config.ExtraConfig) gin.HandlerFunc {
	tmp := scors.ConfigGetter(e)
	if tmp == nil {
		return nil
	}
	cfg, ok := tmp.(scors.Config)
	if !ok {
		return nil
	}

	return wrapper.New(cors.Options{
		AllowedOrigins:   cfg.AllowOrigins,
		AllowedMethods:   cfg.AllowMethods,
		AllowedHeaders:   cfg.AllowHeaders,
		ExposedHeaders:   cfg.ExposeHeaders,
		AllowCredentials: cfg.AllowCredentials,
		MaxAge:           int(cfg.MaxAge.Seconds()),
	})
}

type RunServer func(context.Context, config.ServiceConfig, http.Handler) error

func NewRunServer(next RunServer) RunServer {
	return NewRunServerWithLogger(next, nil)
}

func NewRunServerWithLogger(next RunServer, l log.Logger) RunServer {
	if l == nil {
		l = log.NoOp
	}
	return func(ctx context.Context, cfg config.ServiceConfig, handler http.Handler) error {
		corsMw := mux.NewWithLogger(cfg.ExtraConfig, l)
		if corsMw == nil {
			return next(ctx, cfg, handler)
		}
		l.Debug("[SERVICE: Gin][CORS] Enabled CORS for all requests")
		return next(ctx, cfg, corsMw.Handler(handler))
	}
}
