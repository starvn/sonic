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

// Sonic sets up a complete Sonic API Gateway ready to serve
package main

import (
	"context"
	"github.com/starvn/sonic"
	cmd "github.com/starvn/sonic/support/cobra"
	"github.com/starvn/sonic/support/conflex"
	"github.com/starvn/sonic/support/viper"
	"github.com/starvn/turbo/config"
	"log"
	"os"
	"os/signal"
	"syscall"
)

const (
	fcPartials  = "FC_PARTIALS"
	fcTemplates = "FC_TEMPLATES"
	fcSettings  = "FC_SETTINGS"
	fcPath      = "FC_OUT"
	fcEnable    = "FC_ENABLE"
)

func main() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		select {
		case sig := <-sigs:
			log.Println("Signal intercepted:", sig)
			cancel()
		case <-ctx.Done():
		}
	}()

	sonic.RegisterEncoders()

	for key, alias := range aliases {
		config.ExtraConfigAlias[alias] = key
	}

	var cfg config.Parser
	cfg = viper.New()
	if os.Getenv(fcEnable) != "" {
		cfg = conflex.NewTemplateParser(conflex.Config{
			Parser:    cfg,
			Partials:  os.Getenv(fcPartials),
			Settings:  os.Getenv(fcSettings),
			Path:      os.Getenv(fcPath),
			Templates: os.Getenv(fcTemplates),
		})
	}

	cmd.Execute(cfg, sonic.NewExecutor(ctx))
}

var aliases = map[string]string{
	"github.com/starvn/turbo/transport/http/server/plugin":       "plugin/http-server",
	"github.com/starvn/turbo/transport/http/client/executor":     "plugin/http-client",
	"github.com/starvn/turbo/proxy/plugin/request":               "plugin/request-modifier",
	"github.com/starvn/turbo/proxy/plugin/response":              "plugin/response-modifier",
	"github.com/starvn/turbo/proxy":                              "proxy",
	"github.com/starvn/turbo/route/gin":                          "router",
	"github.com/starvn/sonic/qos/ratelimit/juju/router":          "qos/ratelimit/router",
	"github.com/starvn/sonic/qos/ratelimit/juju/proxy":           "qos/ratelimit/proxy",
	"github.com/starvn/sonic/qos/httpcache":                      "qos/http-cache",
	"github.com/starvn/sonic/qos/circuitbreaker/gobreaker":       "qos/circuit-breaker",
	"github.com/starvn/auth/oauth":                               "auth/client-credentials",
	"github.com/starvn/sonic/auth/jose/validator":                "auth/validator",
	"github.com/starvn/sonic/auth/jose/signer":                   "auth/signer",
	"github.com/starvn/go-bloom-filter":                          "auth/revoker",
	"github.com/starvn/sonic/security/detector":                  "security/bot-detector",
	"github.com/starvn/sonic/security/httpsecure":                "security/http",
	"github.com/starvn/sonic/security/cors":                      "security/cors",
	"github.com/starvn/sonic/validation/explang":                 "validation/explang",
	"github.com/starvn/sonic/validation/jsonschema":              "validation/json-schema",
	"github.com/starvn/sonic/backend/queue/consume":              "backend/queue/consumer",
	"github.com/starvn/sonic/backend/queue/produce":              "backend/queue/producer",
	"github.com/starvn/sonic/backend/lambda":                     "backend/lambda",
	"github.com/starvn/sonic/backend/pubsub/publisher":           "backend/pubsub/publisher",
	"github.com/starvn/sonic/backend/pubsub/subscriber":          "backend/pubsub/subscriber",
	"github.com/starvn/turbo/transport/http/client/graphql":      "backend/graphql",
	"github.com/starvn/turbo/transport/http/client":              "backend/http-client",
	"github.com/starvn/sonic/telemetry/gelf":                     "telemetry/gelf",
	"github.com/starvn/sonic/telemetry/gologging":                "telemetry/logging",
	"github.com/starvn/sonic/telemetry/logstash":                 "telemetry/logstash",
	"github.com/starvn/sonic/telemetry/metrics":                  "telemetry/metrics",
	"github.com/starvn/sonic/telemetry/influxdb":                 "telemetry/influx",
	"github.com/starvn/sonic/telemetry/opencensus":               "telemetry/opencensus",
	"github.com/starvn/sonic/modifier/interpreter/route":         "modifier/lua-endpoint",
	"github.com/starvn/sonic/modifier/interpreter/proxy":         "modifier/lua-proxy",
	"github.com/starvn/sonic/modifier/interpreter/proxy/backend": "modifier/lua-backend",
	"github.com/starvn/sonic/modifier/martian":                   "modifier/martian",
}
