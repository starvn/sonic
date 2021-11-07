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
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/starvn/sonic/security/detector"
	"github.com/starvn/sonic/security/detector/sonic"
	"github.com/starvn/turbo/config"
	"github.com/starvn/turbo/log"
	"github.com/starvn/turbo/proxy"
	sgin "github.com/starvn/turbo/route/gin"
	"net/http"
)

const logPrefix = "[SERVICE: Gin][BotDetector]"

func Register(cfg config.ServiceConfig, l log.Logger, engine *gin.Engine) {
	detectorCfg, err := sonic.ParseConfig(cfg.ExtraConfig)
	if err == sonic.ErrNoConfig {
		return
	}
	if err != nil {
		l.Warning(logPrefix, err.Error())
		return
	}
	d, err := detector.New(detectorCfg)
	if err != nil {
		l.Warning(logPrefix, "Unable to create the bot detector:", err.Error())
		return
	}

	l.Debug(logPrefix, "The bot detector has been registered successfully")
	engine.Use(middleware(d))
}

func New(hf sgin.HandlerFactory, l log.Logger) sgin.HandlerFactory {
	return func(cfg *config.EndpointConfig, p proxy.Proxy) gin.HandlerFunc {
		next := hf(cfg, p)
		logPrefix := "[ENDPOINT: " + cfg.Endpoint + "][Botdetector]"

		detectorCfg, err := sonic.ParseConfig(cfg.ExtraConfig)
		if err == sonic.ErrNoConfig {
			return next
		}
		if err != nil {
			l.Warning(logPrefix, err.Error())
			return next
		}

		d, err := detector.New(detectorCfg)
		if err != nil {
			l.Warning(logPrefix, "Unable to create the bot detector:", err.Error())
			return next
		}

		l.Debug(logPrefix, "The bot detector has been registered successfully")
		return handler(d, next)
	}
}

func middleware(f detector.DetectorFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		if f(c.Request) {
			_ = c.AbortWithError(http.StatusForbidden, errBotRejected)
			return
		}

		c.Next()
	}
}

func handler(f detector.DetectorFunc, next gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		if f(c.Request) {
			_ = c.AbortWithError(http.StatusForbidden, errBotRejected)
			return
		}

		next(c)
	}
}

var errBotRejected = errors.New("bot rejected")
