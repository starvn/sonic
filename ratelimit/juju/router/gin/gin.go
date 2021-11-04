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
	"github.com/gin-gonic/gin"
	srate "github.com/starvn/sonic/ratelimit"
	"github.com/starvn/sonic/ratelimit/juju"
	"github.com/starvn/sonic/ratelimit/juju/router"
	"github.com/starvn/turbo/config"
	"github.com/starvn/turbo/proxy"
	sgin "github.com/starvn/turbo/route/gin"
	"net"
	"net/http"
	"strings"
)

var HandlerFactory = NewRateLimiterMw(sgin.EndpointHandler)

func NewRateLimiterMw(next sgin.HandlerFactory) sgin.HandlerFactory {
	return func(remote *config.EndpointConfig, p proxy.Proxy) gin.HandlerFunc {
		handlerFunc := next(remote, p)

		cfg := router.ConfigGetter(remote.ExtraConfig).(router.Config)
		if cfg == router.ZeroCfg || (cfg.MaxRate <= 0 && cfg.ClientMaxRate <= 0) {
			return handlerFunc
		}

		if cfg.MaxRate > 0 {
			handlerFunc = NewEndpointRateLimiterMw(juju.NewLimiter(float64(cfg.MaxRate), cfg.MaxRate))(handlerFunc)
		}
		if cfg.ClientMaxRate > 0 {
			switch strings.ToLower(cfg.Strategy) {
			case "ip":
				handlerFunc = NewIpLimiterWithKeyMw(cfg.Key, float64(cfg.ClientMaxRate), cfg.ClientMaxRate)(handlerFunc)
			case "header":
				handlerFunc = NewHeaderLimiterMw(cfg.Key, float64(cfg.ClientMaxRate), cfg.ClientMaxRate)(handlerFunc)
			}
		}
		return handlerFunc
	}
}

type EndpointMw func(gin.HandlerFunc) gin.HandlerFunc

func NewEndpointRateLimiterMw(tb juju.Limiter) EndpointMw {
	return func(next gin.HandlerFunc) gin.HandlerFunc {
		return func(c *gin.Context) {
			if !tb.Allow() {
				_ = c.AbortWithError(503, srate.ErrLimited)
				return
			}
			next(c)
		}
	}
}

func NewHeaderLimiterMw(header string, maxRate float64, capacity int64) EndpointMw {
	return NewTokenLimiterMw(HeaderTokenExtractor(header), juju.NewMemoryStore(maxRate, capacity))
}

func NewIpLimiterMw(maxRate float64, capacity int64) EndpointMw {
	return NewTokenLimiterMw(IPTokenExtractor, juju.NewMemoryStore(maxRate, capacity))
}

func NewIpLimiterWithKeyMw(header string, maxRate float64, capacity int64) EndpointMw {
	if header == "" {
		return NewIpLimiterMw(maxRate, capacity)
	}
	return NewTokenLimiterMw(NewIPTokenExtractor(header), juju.NewMemoryStore(maxRate, capacity))
}

type TokenExtractor func(*gin.Context) string

func IPTokenExtractor(c *gin.Context) string { return c.ClientIP() }

func NewIPTokenExtractor(header string) TokenExtractor {
	return func(c *gin.Context) string {
		if clientIP := strings.TrimSpace(strings.Split(c.Request.Header.Get(header), ",")[0]); clientIP != "" {
			ip := strings.Split(clientIP, ":")[0]
			if parsedIP := net.ParseIP(ip); parsedIP != nil {
				return ip
			}
		}
		return IPTokenExtractor(c)
	}
}

func HeaderTokenExtractor(header string) TokenExtractor {
	return func(c *gin.Context) string { return c.Request.Header.Get(header) }
}

func NewTokenLimiterMw(tokenExtractor TokenExtractor, limiterStore srate.LimiterStore) EndpointMw {
	return func(next gin.HandlerFunc) gin.HandlerFunc {
		return func(c *gin.Context) {
			tokenKey := tokenExtractor(c)
			if tokenKey == "" {
				_ = c.AbortWithError(http.StatusTooManyRequests, srate.ErrLimited)
				return
			}
			if !limiterStore(tokenKey).Allow() {
				_ = c.AbortWithError(http.StatusTooManyRequests, srate.ErrLimited)
				return
			}
			next(c)
		}
	}
}
