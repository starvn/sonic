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

// Package cors provides CORS support
package cors

import (
	"github.com/starvn/turbo/config"
	"time"
)

const Namespace = "github.com/starvn/sonic/security/cors"

type Config struct {
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	ExposeHeaders    []string
	AllowCredentials bool
	MaxAge           time.Duration
	Debug            bool
}

func ConfigGetter(e config.ExtraConfig) interface{} {
	v, ok := e[Namespace]
	if !ok {
		return nil
	}

	tmp, ok := v.(map[string]interface{})
	if !ok {
		return nil
	}

	cfg := Config{}
	cfg.AllowOrigins = getList(tmp, "allow_origins")
	cfg.AllowMethods = getList(tmp, "allow_methods")
	cfg.AllowHeaders = getList(tmp, "allow_headers")
	cfg.ExposeHeaders = getList(tmp, "expose_headers")

	if allowCredentials, ok := tmp["allow_credentials"]; ok {
		if v, ok := allowCredentials.(bool); ok {
			cfg.AllowCredentials = v
		}
	}

	if debug, ok := tmp["debug"]; ok {
		v, ok := debug.(bool)
		cfg.Debug = ok && v
	}

	if maxAge, ok := tmp["max_age"]; ok {
		if d, err := time.ParseDuration(maxAge.(string)); err == nil {
			cfg.MaxAge = d
		}
	}
	return cfg
}

func getList(data map[string]interface{}, name string) []string {
	var out []string
	if vs, ok := data[name]; ok {
		if v, ok := vs.([]interface{}); ok {
			for _, s := range v {
				if j, ok := s.(string); ok {
					out = append(out, j)
				}
			}
		}
	}
	return out
}
