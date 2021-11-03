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

// Package gelf just return a gelf writer with the configuration provided via Sonic ExtraConfig
package gelf

import (
	"fmt"
	"github.com/starvn/turbo/config"
	"gopkg.in/Graylog2/go-gelf.v2/gelf"
	"io"
)

const Namespace = "github.com/starvn/sonic/logging/gelf"

var (
	newTCPWriter   = gelf.NewTCPWriter
	newUDPWriter   = gelf.NewUDPWriter
	ErrWrongConfig = fmt.Errorf("getting the extra config for the sonic/logging/gelf package")
	ErrMissingAddr = fmt.Errorf("missing addr to send gelf logs")
)

func NewWriter(cfg config.ExtraConfig) (io.Writer, error) {
	logConfig, ok := ConfigGetter(cfg).(Config)
	if !ok {
		return nil, ErrWrongConfig
	}
	if logConfig.Addr == "" {
		return nil, ErrMissingAddr
	}

	if logConfig.EnableTCP {
		return newTCPWriter(logConfig.Addr)
	}
	return newUDPWriter(logConfig.Addr)
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
	if v, ok := tmp["address"]; ok {
		cfg.Addr = v.(string)
	}
	if v, ok := tmp["enable_tcp"]; ok {
		cfg.EnableTCP = v.(bool)
	}
	return cfg
}

type Config struct {
	Addr      string
	EnableTCP bool
}
