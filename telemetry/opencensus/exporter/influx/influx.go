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

package influx

import (
	"context"
	"errors"
	influxdb "github.com/starvn/opencensus-exporter-influxdb"
	"github.com/starvn/sonic/telemetry/opencensus"
	"time"
)

func init() {
	opencensus.RegisterExporterFactories(func(ctx context.Context, cfg opencensus.Config) (interface{}, error) {
		return Exporter(ctx, cfg)
	})
}

func Exporter(ctx context.Context, cfg opencensus.Config) (*influxdb.IFExporter, error) {
	if cfg.Exporters.InfluxDB == nil {
		return nil, errDisabled
	}
	timeout, err := time.ParseDuration(cfg.Exporters.InfluxDB.Timeout)
	if err != nil {
		timeout = 0
	}
	return influxdb.NewExporter(ctx, influxdb.Options{
		Address:         cfg.Exporters.InfluxDB.Address,
		Username:        cfg.Exporters.InfluxDB.Username,
		Password:        cfg.Exporters.InfluxDB.Password,
		Database:        cfg.Exporters.InfluxDB.Database,
		Timeout:         timeout,
		InstanceName:    cfg.Exporters.InfluxDB.InstanceName,
		ReportingPeriod: time.Duration(cfg.ReportingPeriod) * time.Second,
	})
}

var errDisabled = errors.New("opencensus exporter disabled")
