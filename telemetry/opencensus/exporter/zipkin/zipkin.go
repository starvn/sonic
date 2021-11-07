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

package zipkin

import (
	"context"
	"contrib.go.opencensus.io/exporter/zipkin"
	"errors"
	"github.com/openzipkin/zipkin-go/model"
	httpreporter "github.com/openzipkin/zipkin-go/reporter/http"
	"github.com/starvn/sonic/telemetry/opencensus"
	"net"
)

func init() {
	opencensus.RegisterExporterFactories(func(ctx context.Context, cfg opencensus.Config) (interface{}, error) {
		return Exporter(ctx, cfg)
	})
}

func Exporter(_ context.Context, cfg opencensus.Config) (*zipkin.Exporter, error) {
	if cfg.Exporters.Zipkin == nil {
		return nil, errDisabled
	}
	return zipkin.NewExporter(
		httpreporter.NewReporter(cfg.Exporters.Zipkin.CollectorURL),
		&model.Endpoint{
			ServiceName: cfg.Exporters.Zipkin.ServiceName,
			IPv4:        net.ParseIP(cfg.Exporters.Zipkin.IP),
			Port:        uint16(cfg.Exporters.Zipkin.Port),
		},
	), nil
}

var errDisabled = errors.New("opencensus zipkin exporter disabled")
