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

package main

import (
	"context"
	"flag"
	"github.com/gin-gonic/gin"
	"github.com/starvn/sonic/support/viper"
	metricsgin "github.com/starvn/sonic/telemetry/metrics/gin"
	metricsmux "github.com/starvn/sonic/telemetry/metrics/mux"
	"github.com/starvn/turbo/config"
	logging "github.com/starvn/turbo/log"
	"github.com/starvn/turbo/proxy"
	sgin "github.com/starvn/turbo/route/gin"
	"github.com/starvn/turbo/route/gorilla"
	"github.com/starvn/turbo/route/mux"
	"log"
	"os"
)

func main() {
	port := flag.Int("p", 0, "Port of the service")
	logLevel := flag.String("l", "ERROR", "Logging level")
	debug := flag.Bool("d", false, "Enable the debug")
	useGorilla := flag.Bool("gorilla", false, "Use the gorilla router (gin is used by default)")
	configFile := flag.String("c", "telemetry/metrics/example/sonic.json", "Path to the configuration filename")
	flag.Parse()

	if *useGorilla {
		config.RoutingPattern = config.BracketsRouterPatternBuilder
	}
	parser := viper.New()
	serviceConfig, err := parser.Parse(*configFile)
	if err != nil {
		log.Fatal("ERROR:", err.Error())
	}
	serviceConfig.Debug = serviceConfig.Debug || *debug
	if *port != 0 {
		serviceConfig.Port = *port
	}

	ctx := context.Background()

	logger, err := logging.NewLogger(*logLevel, os.Stdout, "[SONIC]")
	if err != nil {
		log.Fatal("ERROR:", err.Error())
	}

	if *useGorilla {
		metric := metricsmux.New(ctx, serviceConfig.ExtraConfig, logger)
		pf := proxy.NewDefaultFactory(metric.DefaultBackendFactory(), logger)
		routerCfg := gorilla.DefaultConfig(metric.ProxyFactory("pipe", pf), logger)
		defaultHandlerFactory := routerCfg.HandlerFactory
		routerCfg.HandlerFactory = metric.NewHTTPHandlerFactory(defaultHandlerFactory)
		routerFactory := mux.NewFactory(routerCfg)

		routerFactory.NewWithContext(ctx).Run(serviceConfig)

	} else {
		metric := metricsgin.New(ctx, serviceConfig.ExtraConfig, logger)
		pf := proxy.NewDefaultFactory(metric.DefaultBackendFactory(), logger)
		engine := gin.Default()
		routerFactory := sgin.NewFactory(sgin.Config{
			HandlerFactory: metric.NewHTTPHandlerFactory(sgin.EndpointHandler),
			ProxyFactory:   metric.ProxyFactory("pipe", pf),
			Engine:         engine,
			Middlewares:    []gin.HandlerFunc{},
			Logger:         logger,
		})

		routerFactory.NewWithContext(ctx).Run(serviceConfig)
	}
}
