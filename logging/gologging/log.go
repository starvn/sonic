//go:build !windows && !plan9
// +build !windows,!plan9

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

// Package gologging provides a logger implementation based on the github.com/op/go-log pkg
package gologging

import (
	"fmt"
	gologging "github.com/op/go-logging"
	"github.com/starvn/turbo/config"
	"github.com/starvn/turbo/log"
	"io"
	"log/syslog"
	"os"
)

const Namespace = "github.com/starvn/logging/gologging"

var (
	ErrWrongConfig           = fmt.Errorf("getting the extra config for the sonic/logging/gologging package")
	LogstashPattern          = `{"@timestamp":"%{time:2006-01-02T15:04:05.000+00:00}", "@version": 1, "level": "%{level}", "message": "%{message}", "module": "%{module}"}`
	DefaultPattern           = ` %{time:2006/01/02 - 15:04:05.000} %{color}▶ %{level}%{color:reset} %{message}`
	ActivePattern            = DefaultPattern
	defaultFormatterSelector = func(io.Writer) string { return ActivePattern }
)

func SetFormatterSelector(f func(io.Writer) string) {
	defaultFormatterSelector = f
}

func NewLogger(cfg config.ExtraConfig, ws ...io.Writer) (log.Logger, error) {
	logConfig, ok := ConfigGetter(cfg).(Config)
	if !ok {
		return nil, ErrWrongConfig
	}
	module := "SONIC"
	loggr := gologging.MustGetLogger(module)

	if logConfig.StdOut {
		ws = append(ws, os.Stdout)
	}

	if logConfig.Syslog {
		var err error
		var w *syslog.Writer
		w, err = syslog.New(syslog.LOG_CRIT, logConfig.Prefix)
		if err != nil {
			return nil, err
		}
		ws = append(ws, w)
	}

	if logConfig.Format == "logstash" {
		ActivePattern = LogstashPattern
		logConfig.Prefix = ""
	}

	if logConfig.Format == "custom" {
		ActivePattern = logConfig.CustomFormat
		logConfig.Prefix = ""
	}

	var backends []gologging.Backend
	for _, w := range ws {
		backend := gologging.NewLogBackend(w, logConfig.Prefix, 0)
		pattern := defaultFormatterSelector(w)
		format := gologging.MustStringFormatter(pattern)
		backendLeveled := gologging.AddModuleLevel(gologging.NewBackendFormatter(backend, format))
		logLevel, err := gologging.LogLevel(logConfig.Level)
		if err != nil {
			return nil, err
		}
		backendLeveled.SetLevel(logLevel, module)
		backends = append(backends, backendLeveled)
	}

	gologging.SetBackend(backends...)
	return Logger{loggr}, nil
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
	if v, ok := tmp["stdout"]; ok {
		cfg.StdOut = v.(bool)
	}
	if v, ok := tmp["syslog"]; ok {
		cfg.Syslog = v.(bool)
	}
	if v, ok := tmp["level"]; ok {
		cfg.Level = v.(string)
	}
	if v, ok := tmp["prefix"]; ok {
		cfg.Prefix = v.(string)
	}
	if v, ok := tmp["format"]; ok {
		cfg.Format = v.(string)
	}
	if v, ok := tmp["custom_format"]; ok {
		cfg.CustomFormat = v.(string)
	}
	return cfg
}

type Config struct {
	Level        string
	StdOut       bool
	Syslog       bool
	Prefix       string
	Format       string
	CustomFormat string
}

type Logger struct {
	logger *gologging.Logger
}

func (l Logger) Debug(v ...interface{}) {
	l.logger.Debug(v...)
}

func (l Logger) Info(v ...interface{}) {
	l.logger.Info(v...)
}

func (l Logger) Warning(v ...interface{}) {
	l.logger.Warning(v...)
}

func (l Logger) Error(v ...interface{}) {
	l.logger.Error(v...)
}

func (l Logger) Critical(v ...interface{}) {
	l.logger.Critical(v...)
}

func (l Logger) Fatal(v ...interface{}) {
	l.logger.Fatal(v...)
}
