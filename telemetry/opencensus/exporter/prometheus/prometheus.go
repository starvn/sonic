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

package prometheus

import (
	"context"
	"contrib.go.opencensus.io/exporter/prometheus"
	"errors"
	"fmt"
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/starvn/sonic/telemetry/opencensus"
	"log"
	"net/http"
	"time"
)

func init() {
	opencensus.RegisterExporterFactories(func(ctx context.Context, cfg opencensus.Config) (interface{}, error) {
		return Exporter(ctx, cfg)
	})
}

func Exporter(ctx context.Context, cfg opencensus.Config) (*prometheus.Exporter, error) {
	if cfg.Exporters.Prometheus == nil {
		return nil, errDisabled
	}

	prometheusRegistry := prom.NewRegistry()
	err := prometheusRegistry.Register(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	if err != nil {
		return nil, err
	}
	err = prometheusRegistry.Register(collectors.NewGoCollector())
	if err != nil {
		return nil, err
	}

	exporter, err := prometheus.NewExporter(prometheus.Options{Namespace: cfg.Exporters.Prometheus.Namespace, Registry: prometheusRegistry})
	if err != nil {
		return exporter, err
	}

	router := http.NewServeMux()
	router.Handle("/metrics", exporter)
	server := http.Server{
		Handler: router,
		Addr:    fmt.Sprintf(":%d", cfg.Exporters.Prometheus.Port),
	}

	go func() {
		if serverErr := server.ListenAndServe(); serverErr != http.ErrServerClosed {
			log.Fatalf("[SERVICE: Opencensus] The Prometheus exporter failed to listen and serve: %v", serverErr)
		}
	}()

	go func() {
		<-ctx.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_ = server.Shutdown(ctx)
		cancel()
	}()

	return exporter, nil
}

var errDisabled = errors.New("opencensus prometheus exporter disabled")
