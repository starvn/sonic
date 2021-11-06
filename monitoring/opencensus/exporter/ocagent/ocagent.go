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

package ocagent

import (
	"context"
	"contrib.go.opencensus.io/exporter/ocagent"
	"errors"
	"github.com/starvn/sonic/monitoring/opencensus"
	_ "google.golang.org/grpc/encoding/gzip"
	"time"
)

func init() {
	opencensus.RegisterExporterFactories(func(ctx context.Context, cfg opencensus.Config) (interface{}, error) {
		return Exporter(ctx, cfg)
	})
}

func Exporter(ctx context.Context, cfg opencensus.Config) (*ocagent.Exporter, error) {

	var options []ocagent.ExporterOption
	if cfg.Exporters.Ocagent == nil {
		return nil, errors.New("ocagent exporter disabled")
	}

	if cfg.Exporters.Ocagent.Address == "" {
		return nil, errors.New("missing ocagent address")
	}
	options = append(options, ocagent.WithAddress(cfg.Exporters.Ocagent.Address))

	if cfg.Exporters.Ocagent.ServiceName == "" {
		cfg.Exporters.Ocagent.ServiceName = "Sonic-Opencensus"
	}
	options = append(options, ocagent.WithServiceName(cfg.Exporters.Ocagent.ServiceName))

	if cfg.Exporters.Ocagent.Headers != nil {
		options = append(options, ocagent.WithHeaders(cfg.Exporters.Ocagent.Headers))
	}

	if cfg.Exporters.Ocagent.Insecure {
		options = append(options, ocagent.WithInsecure())
	}

	if cfg.Exporters.Ocagent.EnaableCompression {
		options = append(options, ocagent.UseCompressor("gzip"))
	}

	if cfg.Exporters.Ocagent.Reconnection != "" {
		period, err := time.ParseDuration(cfg.Exporters.Ocagent.Reconnection)
		if err != nil {
			return nil, errors.New("cannot parse reconnection period")
		}
		options = append(options, ocagent.WithReconnectionPeriod(period))
	}
	return ocagent.NewExporter(options...)
}
