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

package sonic

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/go-contrib/uuid"
	sonicbf "github.com/starvn/go-bloom-filter/sonic"
	"github.com/starvn/sonic/auth/jose"
	"github.com/starvn/sonic/backend/pubsub"
	cors "github.com/starvn/sonic/security/cors/gin"
	cmd "github.com/starvn/sonic/support/cobra"
	"github.com/starvn/sonic/support/usage/client"
	"github.com/starvn/sonic/telemetry/gelf"
	"github.com/starvn/sonic/telemetry/gologging"
	"github.com/starvn/sonic/telemetry/influxdb"
	"github.com/starvn/sonic/telemetry/logstash"
	metrics "github.com/starvn/sonic/telemetry/metrics/gin"
	"github.com/starvn/sonic/telemetry/opencensus"
	_ "github.com/starvn/sonic/telemetry/opencensus/exporter/datadog"
	_ "github.com/starvn/sonic/telemetry/opencensus/exporter/influx"
	_ "github.com/starvn/sonic/telemetry/opencensus/exporter/jaeger"
	_ "github.com/starvn/sonic/telemetry/opencensus/exporter/ocagent"
	_ "github.com/starvn/sonic/telemetry/opencensus/exporter/prometheus"
	_ "github.com/starvn/sonic/telemetry/opencensus/exporter/stackdriver"
	_ "github.com/starvn/sonic/telemetry/opencensus/exporter/xray"
	_ "github.com/starvn/sonic/telemetry/opencensus/exporter/zipkin"
	"github.com/starvn/sonic/validation/explang"
	"github.com/starvn/turbo/config"
	"github.com/starvn/turbo/core"
	"github.com/starvn/turbo/log"
	"github.com/starvn/turbo/proxy"
	router "github.com/starvn/turbo/route/gin"
	serverhttp "github.com/starvn/turbo/transport/http/server"
	server "github.com/starvn/turbo/transport/http/server/plugin"
	"io"
	"net/http"
	"os"
	"time"
)

func NewExecutor(ctx context.Context) cmd.Executor {
	eb := new(ExecutorBuilder)
	return eb.NewCmdExecutor(ctx)
}

type PluginLoader interface {
	Load(folder, pattern string, logger log.Logger)
}

type SubscriberFactoriesRegister interface {
	Register(context.Context, config.ServiceConfig, log.Logger) func(string, int)
}

type TokenRejecterFactory interface {
	NewTokenRejecter(context.Context, config.ServiceConfig, log.Logger, func(string, int)) (jose.ChainedRejecterFactory, error)
}

type MetricsAndTracesRegister interface {
	Register(context.Context, config.ServiceConfig, log.Logger) *metrics.Metrics
}

type EngineFactory interface {
	NewEngine(config.ServiceConfig, log.Logger, io.Writer) *gin.Engine
}

type ProxyFactory interface {
	NewProxyFactory(log.Logger, proxy.BackendFactory, *metrics.Metrics) proxy.Factory
}

type BackendFactory interface {
	NewBackendFactory(context.Context, log.Logger, *metrics.Metrics) proxy.BackendFactory
}

type HandlerFactory interface {
	NewHandlerFactory(log.Logger, *metrics.Metrics, jose.RejecterFactory) router.HandlerFactory
}

type LoggerFactory interface {
	NewLogger(config.ServiceConfig) (log.Logger, io.Writer, error)
}

type RunServer func(context.Context, config.ServiceConfig, http.Handler) error

type RunServerFactory interface {
	NewRunServer(log.Logger, router.RunServerFunc) RunServer
}

type ExecutorBuilder struct {
	LoggerFactory               LoggerFactory
	PluginLoader                PluginLoader
	SubscriberFactoriesRegister SubscriberFactoriesRegister
	TokenRejecterFactory        TokenRejecterFactory
	MetricsAndTracesRegister    MetricsAndTracesRegister
	EngineFactory               EngineFactory
	ProxyFactory                ProxyFactory
	BackendFactory              BackendFactory
	HandlerFactory              HandlerFactory
	RunServerFactory            RunServerFactory

	Middlewares []gin.HandlerFunc
}

func (e *ExecutorBuilder) NewCmdExecutor(ctx context.Context) cmd.Executor {
	e.checkCollaborators()

	return func(cfg config.ServiceConfig) {
		logger, gelfWriter, gelfErr := e.LoggerFactory.NewLogger(cfg)
		if gelfErr != nil {
			return
		}

		logger.Info("Listening on port:", cfg.Port)

		startReporter(ctx, logger, cfg)

		if cfg.Plugin != nil {
			e.PluginLoader.Load(cfg.Plugin.Folder, cfg.Plugin.Pattern, logger)
		}

		metricCollector := e.MetricsAndTracesRegister.Register(ctx, cfg, logger)

		tokenRejecterFactory, err := e.TokenRejecterFactory.NewTokenRejecter(
			ctx,
			cfg,
			logger,
			e.SubscriberFactoriesRegister.Register(ctx, cfg, logger),
		)
		if err != nil && err != sonicbf.ErrNoConfig {
			logger.Warning("[SERVICE: Bloomfilter]", err.Error())
		}

		routerFactory := router.NewFactory(router.Config{
			Engine: e.EngineFactory.NewEngine(cfg, logger, gelfWriter),
			ProxyFactory: e.ProxyFactory.NewProxyFactory(
				logger,
				e.BackendFactory.NewBackendFactory(ctx, logger, metricCollector),
				metricCollector,
			),
			Middlewares:    e.Middlewares,
			Logger:         logger,
			HandlerFactory: e.HandlerFactory.NewHandlerFactory(logger, metricCollector, tokenRejecterFactory),
			RunServer:      router.RunServerFunc(e.RunServerFactory.NewRunServer(logger, serverhttp.RunServer)),
		})

		routerFactory.NewWithContext(ctx).Run(cfg)
	}
}

func (e *ExecutorBuilder) checkCollaborators() {
	if e.PluginLoader == nil {
		e.PluginLoader = new(pluginLoader)
	}
	if e.SubscriberFactoriesRegister == nil {
		e.SubscriberFactoriesRegister = new(registerSubscriberFactories)
	}
	if e.TokenRejecterFactory == nil {
		e.TokenRejecterFactory = new(BloomFilterJWT)
	}
	if e.MetricsAndTracesRegister == nil {
		e.MetricsAndTracesRegister = new(MetricsAndTraces)
	}
	if e.EngineFactory == nil {
		e.EngineFactory = new(engineFactory)
	}
	if e.ProxyFactory == nil {
		e.ProxyFactory = new(proxyFactory)
	}
	if e.BackendFactory == nil {
		e.BackendFactory = new(backendFactory)
	}
	if e.HandlerFactory == nil {
		e.HandlerFactory = new(handlerFactory)
	}
	if e.LoggerFactory == nil {
		e.LoggerFactory = new(LoggerBuilder)
	}
	if e.RunServerFactory == nil {
		e.RunServerFactory = new(DefaultRunServerFactory)
	}
}

type DefaultRunServerFactory struct{}

func (d *DefaultRunServerFactory) NewRunServer(l log.Logger, next router.RunServerFunc) RunServer {
	return RunServer(server.New(
		l,
		server.RunServer(cors.NewRunServer(cors.NewRunServerWithLogger(cors.RunServer(next), l))),
	))
}

type LoggerBuilder struct{}

func (LoggerBuilder) NewLogger(cfg config.ServiceConfig) (log.Logger, io.Writer, error) {
	var writers []io.Writer
	gelfWriter, gelfErr := gelf.NewWriter(cfg.ExtraConfig)
	if gelfErr == nil {
		writers = append(writers, gelfWriterWrapper{gelfWriter})
		gologging.SetFormatterSelector(func(w io.Writer) string {
			switch w.(type) {
			case gelfWriterWrapper:
				return "%{message}"
			default:
				return gologging.DefaultPattern
			}
		})
	}
	logger, gologErr := logstash.NewLogger(cfg.ExtraConfig)

	if gologErr != nil {
		logger, gologErr = gologging.NewLogger(cfg.ExtraConfig, writers...)

		if gologErr != nil {
			var err error
			logger, err = log.NewLogger("DEBUG", os.Stdout, "SONIC")
			if err != nil {
				return logger, gelfWriter, err
			}
			logger.Error("[SERVICE: Logging] Unable to create the logger:", gologErr.Error())
		}
	}
	if gelfErr != nil && gelfErr != gelf.ErrWrongConfig {
		logger.Error("[SERVICE: Logging][GELF] Unable to create the writer:", gelfErr.Error())
	}
	return logger, gelfWriter, nil
}

type BloomFilterJWT struct{}

func (t BloomFilterJWT) NewTokenRejecter(ctx context.Context, cfg config.ServiceConfig, l log.Logger, reg func(n string, p int)) (jose.ChainedRejecterFactory, error) {
	rejecter, err := sonicbf.Register(ctx, "sonic-bf", cfg, l, reg)

	return jose.ChainedRejecterFactory([]jose.RejecterFactory{
		jose.RejecterFactoryFunc(func(_ log.Logger, _ *config.EndpointConfig) jose.Rejecter {
			return jose.RejecterFunc(rejecter.RejectToken)
		}),
		jose.RejecterFactoryFunc(func(l log.Logger, cfg *config.EndpointConfig) jose.Rejecter {
			if r := explang.NewRejecter(l, cfg); r != nil {
				return r
			}
			return jose.FixedRejecter(false)
		}),
	}), err
}

type MetricsAndTraces struct{}

func (MetricsAndTraces) Register(ctx context.Context, cfg config.ServiceConfig, l log.Logger) *metrics.Metrics {
	metricCollector := metrics.New(ctx, cfg.ExtraConfig, l)

	if err := influxdb.New(ctx, cfg.ExtraConfig, metricCollector, l); err != nil {
		if err != influxdb.ErrNoConfig {
			l.Warning("[SERVICE: InfluxDB]", err.Error())
		}
	} else {
		l.Debug("[SERVICE: InfluxDB] Service correctly registered")
	}

	if err := opencensus.Register(ctx, cfg, append(opencensus.DefaultViews, pubsub.OpenCensusViews...)...); err != nil {
		if err != opencensus.ErrNoConfig {
			l.Warning("[SERVICE: OpenCensus]", err.Error())
		}
	} else {
		l.Debug("[SERVICE: OpenCensus] Service correctly registered")
	}

	return metricCollector
}

const (
	usageDisable = "USAGE_DISABLE"
	usageDelay   = 5 * time.Second
)

func startReporter(ctx context.Context, logger log.Logger, cfg config.ServiceConfig) {
	logPrefix := "[SERVICE: Telemetry]"
	if os.Getenv(usageDisable) == "1" {
		return
	}

	clusterID, err := cfg.Hash()
	if err != nil {
		logger.Debug(logPrefix, "Unable to create the Cluster ID hash:", err.Error())
		return
	}

	go func() {
		time.Sleep(usageDelay)

		serverID := uuid.NewV4().String()
		logger.Debug(logPrefix, "Registering usage stats for Cluster ID", clusterID)

		if err := client.StartReporter(ctx, client.Options{
			ClusterID: clusterID,
			ServerID:  serverID,
			Version:   core.SonicVersion,
		}); err != nil {
			logger.Debug(logPrefix, "Unable to create the usage report client:", err.Error())
		}
	}()
}

type gelfWriterWrapper struct {
	io.Writer
}
