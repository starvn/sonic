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

package influxdb

import (
	"errors"
	"github.com/starvn/turbo/config"
	"time"
)

const defaultBufferSize = 0

type influxConfig struct {
	address    string
	username   string
	password   string
	ttl        time.Duration
	database   string
	bufferSize int
}

func configGetter(extraConfig config.ExtraConfig) interface{} {
	value, ok := extraConfig[Namespace]
	if !ok {
		return nil
	}

	castedConfig, ok := value.(map[string]interface{})
	if !ok {
		return nil
	}

	cfg := influxConfig{}

	if value, ok := castedConfig["address"]; ok {
		cfg.address = value.(string)
	}

	if value, ok := castedConfig["username"]; ok {
		cfg.username = value.(string)
	}

	if value, ok := castedConfig["password"]; ok {
		cfg.password = value.(string)
	}

	if value, ok := castedConfig["buffer_size"]; ok {
		if s, ok := value.(int); ok {
			cfg.bufferSize = s
		}
	}

	if value, ok := castedConfig["ttl"]; ok {
		s, ok := value.(string)

		if !ok {
			return nil
		}
		var err error
		cfg.ttl, err = time.ParseDuration(s)

		if err != nil {
			return nil
		}
	}

	if value, ok := castedConfig["db"]; ok {
		cfg.database = value.(string)
	} else {
		cfg.database = "sonic"
	}

	return cfg
}

var ErrNoConfig = errors.New("influxdb: unable to load custom config")
