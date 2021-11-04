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

// Package router provides several rate-limit routers using the golang.org/x/time/rate lib
package router

import (
	"fmt"
	"github.com/starvn/turbo/config"
)

const Namespace = "github.com/starvn/sonic/ratelimit/rate/router"

type Config struct {
	MaxRate       int
	Strategy      string
	ClientMaxRate int
	Key           string
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
	if v, ok := tmp["maxRate"]; ok {
		switch val := v.(type) {
		case int:
			cfg.MaxRate = val
		case float64:
			cfg.MaxRate = int(val)
		}
	}
	if v, ok := tmp["strategy"]; ok {
		cfg.Strategy = fmt.Sprintf("%v", v)
	}
	if v, ok := tmp["clientMaxRate"]; ok {
		switch val := v.(type) {
		case int:
			cfg.ClientMaxRate = val
		case float64:
			cfg.ClientMaxRate = int(val)
		}
	}
	if v, ok := tmp["key"]; ok {
		cfg.Key = fmt.Sprintf("%v", v)
	}
	return cfg
}
