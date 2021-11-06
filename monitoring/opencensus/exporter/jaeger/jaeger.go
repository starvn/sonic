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

package jaeger

import (
	"context"
	"contrib.go.opencensus.io/exporter/jaeger"
	"errors"
	"github.com/starvn/sonic/monitoring/opencensus"
)

func init() {
	opencensus.RegisterExporterFactories(func(ctx context.Context, cfg opencensus.Config) (interface{}, error) {
		return Exporter(ctx, cfg)
	})
}

func Exporter(ctx context.Context, cfg opencensus.Config) (*jaeger.Exporter, error) {
	if cfg.Exporters.Jaeger == nil {
		return nil, errDisabled
	}
	e, err := jaeger.NewExporter(jaeger.Options{
		CollectorEndpoint: cfg.Exporters.Jaeger.Endpoint,
		BufferMaxCount:    cfg.Exporters.Jaeger.BufferMaxCount,
		Process: jaeger.Process{
			ServiceName: cfg.Exporters.Jaeger.ServiceName,
		},
	})
	if err != nil {
		return e, err
	}

	go func() {
		<-ctx.Done()
		e.Flush()
	}()

	return e, nil

}

var errDisabled = errors.New("opencensus jaeger exporter disabled")
