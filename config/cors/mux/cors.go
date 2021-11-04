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
	"github.com/rs/cors"
	scors "github.com/starvn/sonic/config/cors"
	"github.com/starvn/turbo/config"
	logging "github.com/starvn/turbo/log"
	"github.com/starvn/turbo/route/mux"
	"io"
	"log"
)

func New(e config.ExtraConfig) mux.HandlerMiddleware {
	return NewWithLogger(e, nil)
}

func NewWithLogger(e config.ExtraConfig, l logging.Logger) mux.HandlerMiddleware {
	tmp := scors.ConfigGetter(e)
	if tmp == nil {
		return nil
	}
	cfg, ok := tmp.(scors.Config)
	if !ok {
		return nil
	}
	c := cors.New(cors.Options{
		AllowedOrigins:   cfg.AllowOrigins,
		AllowedMethods:   cfg.AllowMethods,
		AllowedHeaders:   cfg.AllowHeaders,
		ExposedHeaders:   cfg.ExposeHeaders,
		AllowCredentials: cfg.AllowCredentials,
		MaxAge:           int(cfg.MaxAge.Seconds()),
	})
	if l == nil || !cfg.Debug {
		return c
	}
	r, w := io.Pipe()
	c.Log = log.New(w, "", log.LstdFlags)
	go func() {
		msg := make([]byte, 1024)
		for {
			_, _ = r.Read(msg)
			l.Debug("[CORS]", string(msg))
		}
	}()
	return c
}
