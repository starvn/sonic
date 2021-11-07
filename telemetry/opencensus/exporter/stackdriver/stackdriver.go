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

package stackdriver

import (
	"context"
	"contrib.go.opencensus.io/exporter/stackdriver"
	"contrib.go.opencensus.io/exporter/stackdriver/monitoredresource"
	"errors"
	"github.com/starvn/sonic/telemetry/opencensus"
	"time"
)

func init() {
	opencensus.RegisterExporterFactories(func(ctx context.Context, cfg opencensus.Config) (interface{}, error) {
		return Exporter(ctx, cfg)
	})
}

var defaultMetricPrefix = "sonic"

func Exporter(ctx context.Context, cfg opencensus.Config) (*stackdriver.Exporter, error) {
	if cfg.Exporters.Stackdriver == nil {
		return nil, errors.New("stackdriver exporter disabled")
	}
	if cfg.Exporters.Stackdriver.MetricPrefix == "" {
		cfg.Exporters.Stackdriver.MetricPrefix = defaultMetricPrefix
	}

	labels := &stackdriver.Labels{}
	for k, v := range cfg.Exporters.Stackdriver.DefaultLabels {
		labels.Set(k, v, "")
	}

	return stackdriver.NewExporter(stackdriver.Options{
		ProjectID:               cfg.Exporters.Stackdriver.ProjectID,
		MetricPrefix:            cfg.Exporters.Stackdriver.MetricPrefix,
		BundleDelayThreshold:    time.Duration(cfg.ReportingPeriod) * time.Second,
		BundleCountThreshold:    cfg.SampleRate,
		DefaultMonitoringLabels: labels,
		MonitoredResource:       monitoredresource.Autodetect(),
	})
}
