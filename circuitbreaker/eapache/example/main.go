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
	cel "github.com/starvn/sonic/expl"
	"github.com/starvn/turbo/config"
	logging "github.com/starvn/turbo/log"
	"github.com/starvn/turbo/proxy"
	sgin "github.com/starvn/turbo/route/gin"
	"github.com/starvn/turbo/transport/http/client"
	"github.com/starvn/turbo/transport/http/server"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	port := flag.Int("p", 0, "Port of the service")
	logLevel := flag.String("l", "ERROR", "Logging level")
	debug := flag.Bool("d", false, "Enable the debug")
	configFile := flag.String("c", "circuitbreaker/eapache/example/sonic.json", "Path to the configuration filename")
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

	bf := cel.BackendFactory(logger, proxy.CustomHTTPProxyFactory(client.NewHTTPClient))
	pf := cel.ProxyFactory(logger, proxy.NewDefaultFactory(bf, logger))

	routerFactory := sgin.NewFactory(sgin.Config{
		Engine:         gin.Default(),
		ProxyFactory:   pf,
		Logger:         logger,
		HandlerFactory: sgin.EndpointHandler,
		RunServer:      server.RunServer,
	})

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		select {
		case sig := <-sigs:
			logger.Info("Signal intercepted:", sig)
			cancel()
		case <-ctx.Done():
		}
	}()

	routerFactory.NewWithContext(ctx).Run(serviceConfig)
}
