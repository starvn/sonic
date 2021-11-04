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
	"flag"
	"github.com/gin-gonic/gin"
	rateproxy "github.com/starvn/sonic/endpoint/ratelimit/rate/proxy"
	raterouter "github.com/starvn/sonic/endpoint/ratelimit/rate/router/gin"
	"github.com/starvn/turbo/config"
	logging "github.com/starvn/turbo/log"
	"github.com/starvn/turbo/proxy"
	sgin "github.com/starvn/turbo/route/gin"
	"github.com/starvn/turbo/transport/http/client"
	http "github.com/starvn/turbo/transport/http/server"
	"log"
	"os"
)

func main() {
	port := flag.Int("p", 0, "Port of the service")
	logLevel := flag.String("l", "ERROR", "Logging level")
	debug := flag.Bool("d", false, "Enable the debug")
	configFile := flag.String("c", "endpoint/ratelimit/rate/example/sonic.json", "Path to the configuration filename")
	flag.Parse()

	parser := config.NewParser()
	serviceConfig, err := parser.Parse(*configFile)
	if err != nil {
		log.Fatal("ERROR:", err.Error())
	}
	serviceConfig.Debug = serviceConfig.Debug || *debug
	if *port != 0 {
		serviceConfig.Port = *port
	}

	logger, err := logging.NewLogger(*logLevel, os.Stdout, "[SONIC]")
	if err != nil {
		log.Fatal("ERROR:", err.Error())
	}

	routerFactory := sgin.NewFactory(sgin.Config{
		Engine:         gin.Default(),
		ProxyFactory:   proxy.NewDefaultFactory(rateproxy.BackendFactory(proxy.CustomHTTPProxyFactory(client.NewHTTPClient)), logger),
		Middlewares:    []gin.HandlerFunc{},
		Logger:         logger,
		HandlerFactory: raterouter.HandlerFactory,
		RunServer:      http.RunServer,
	})

	routerFactory.New().Run(serviceConfig)
}
