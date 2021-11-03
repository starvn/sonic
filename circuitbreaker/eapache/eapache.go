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

// Package eapache provides a circuit breaker adapter using the github.com/eapache/go-resiliency/breaker lib
package eapache

import (
	"github.com/eapache/go-resiliency/breaker"
	"github.com/starvn/turbo/config"
	"time"
)

const Namespace = "github.com/starvn/circuitbreaker/eapache"

type Config struct {
	Error   int
	Success int
	Timeout time.Duration
}

var ZeroCfg = Config{}

func ConfigGetter(e config.ExtraConfig) interface{} {
	v, ok := e[Namespace]
	if !ok {
		return ZeroCfg
	}
	tmp, ok := v.(map[string]interface{})
	if !ok {
		return ZeroCfg
	}
	cfg := Config{}
	if v, ok := tmp["error"]; ok {
		switch i := v.(type) {
		case int:
			cfg.Error = i
		case float64:
			cfg.Error = int(i)
		}
	}
	if v, ok := tmp["success"]; ok {
		switch i := v.(type) {
		case int:
			cfg.Success = i
		case float64:
			cfg.Success = int(i)
		}
	}
	if v, ok := tmp["timeout"]; ok {
		if d, err := time.ParseDuration(v.(string)); err == nil {
			cfg.Timeout = d
		}
	}
	return cfg
}

func NewCircuitBreaker(cfg Config) *breaker.Breaker {
	return breaker.New(cfg.Error, cfg.Success, cfg.Timeout)
}
