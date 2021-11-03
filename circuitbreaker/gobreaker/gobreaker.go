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

// Package gobreaker provides a circuit circuitbreaker adapter using the sony/gobreaker lib.
package gobreaker

import (
	"fmt"
	"github.com/sony/gobreaker"
	"github.com/starvn/turbo/config"
	"github.com/starvn/turbo/log"
	"time"
)

const Namespace = "github.com/starvn/sonic/circuitbreaker/gobreaker"

type Config struct {
	Name            string
	Interval        int
	Timeout         int
	MaxErrors       int
	LogStatusChange bool
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
	if v, ok := tmp["name"]; ok {
		if name, ok := v.(string); ok {
			cfg.Name = name
		}
	}
	if v, ok := tmp["interval"]; ok {
		switch i := v.(type) {
		case int:
			cfg.Interval = i
		case float64:
			cfg.Interval = int(i)
		}
	}
	if v, ok := tmp["timeout"]; ok {
		switch i := v.(type) {
		case int:
			cfg.Timeout = i
		case float64:
			cfg.Timeout = int(i)
		}
	}
	if v, ok := tmp["max_errors"]; ok {
		switch i := v.(type) {
		case int:
			cfg.MaxErrors = i
		case float64:
			cfg.MaxErrors = int(i)
		}
	}
	value, ok := tmp["log_status_change"].(bool)
	cfg.LogStatusChange = ok && value

	return cfg
}

func NewCircuitBreaker(cfg Config, logger log.Logger) *gobreaker.CircuitBreaker {
	settings := gobreaker.Settings{
		Name:     cfg.Name,
		Interval: time.Duration(cfg.Interval) * time.Second,
		Timeout:  time.Duration(cfg.Timeout) * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures > uint32(cfg.MaxErrors)
		},
	}

	if cfg.LogStatusChange {
		settings.OnStateChange = func(name string, from gobreaker.State, to gobreaker.State) {
			logger.Warning(fmt.Sprintf("[CB] Circuit circuitbreaker named '%s' went from '%s' to '%s'", name, from.String(), to.String()))
		}
	}

	return gobreaker.NewCircuitBreaker(settings)
}
