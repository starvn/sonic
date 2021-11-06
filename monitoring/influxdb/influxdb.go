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

// Package influxdb provides metric collector for Sonic
package influxdb

import (
	"context"
	"github.com/influxdata/influxdb/client/v2"
	"github.com/starvn/sonic/monitoring/influxdb/counter"
	"github.com/starvn/sonic/monitoring/influxdb/gauge"
	"github.com/starvn/sonic/monitoring/influxdb/histogram"
	ginmetrics "github.com/starvn/sonic/monitoring/metrics/gin"
	"github.com/starvn/turbo/config"
	"github.com/starvn/turbo/log"
	"os"
	"time"
)

const Namespace = "github.com/starvn/sonic/monitoring/influxdb"
const logPrefix = "[SERVICE: Influx]"

type clientWrapper struct {
	influxClient client.Client
	collector    *ginmetrics.Metrics
	logger       log.Logger
	db           string
	buf          *Buffer
}

func New(ctx context.Context, extraConfig config.ExtraConfig, metricsCollector *ginmetrics.Metrics, logger log.Logger) error {
	cfg, ok := configGetter(extraConfig).(influxConfig)
	if !ok {
		return ErrNoConfig
	}

	logger.Debug(logPrefix, "Creating client")

	influxdbClient, err := client.NewHTTPClient(client.HTTPConfig{
		Addr:     cfg.address,
		Username: cfg.username,
		Password: cfg.password,
		Timeout:  10 * time.Second,
	})
	if err != nil {
		logger.Error(logPrefix, "Client crashed:", err)
		return err
	}

	go func() {
		pingDuration, pingMsg, err := influxdbClient.Ping(time.Second)
		if err != nil {
			logger.Warning(logPrefix, "Unable to ping the InfluxDB server:", err.Error())
			return
		}
		logger.Debug(logPrefix, "Ping results: duration =", pingDuration, "msg =", pingMsg)
	}()

	t := time.NewTicker(cfg.ttl)

	cw := clientWrapper{
		influxClient: influxdbClient,
		collector:    metricsCollector,
		logger:       logger,
		db:           cfg.database,
		buf:          NewBuffer(cfg.bufferSize),
	}

	go cw.keepUpdated(ctx, t.C)

	logger.Debug(logPrefix, "Client up and running")

	return nil
}

func (cw clientWrapper) keepUpdated(ctx context.Context, ticker <-chan time.Time) {
	hostname, err := os.Hostname()
	if err != nil {
		cw.logger.Error("influxdb resolving the local hostname:", err.Error())
	}
	for {
		select {
		case <-ticker:
		case <-ctx.Done():
			return
		}

		cw.logger.Debug(logPrefix, "Preparing data points to send")

		snapshot := cw.collector.Snapshot()

		if shouldSendPoints := len(snapshot.Counters) > 0 || len(snapshot.Gauges) > 0; !shouldSendPoints {
			cw.logger.Debug(logPrefix, "No metrics to send")
			continue
		}

		bp, _ := client.NewBatchPoints(client.BatchPointsConfig{
			Database:  cw.db,
			Precision: "s",
		})
		now := time.Unix(0, snapshot.Time)

		for _, p := range counter.Points(hostname, now, snapshot.Counters, cw.logger) {
			bp.AddPoint(p)
		}

		for _, p := range gauge.Points(hostname, now, snapshot.Gauges, cw.logger) {
			bp.AddPoint(p)
		}

		for _, p := range histogram.Points(hostname, now, snapshot.Histograms, cw.logger) {
			bp.AddPoint(p)
		}

		if err := cw.influxClient.Write(bp); err != nil {
			cw.logger.Error(logPrefix, "Couldn't write to server:", err.Error())
			cw.buf.Add(bp)
			continue
		}

		cw.logger.Debug(logPrefix, len(bp.Points()), "datapoints sent")

		pts := []*client.Point{}
		bpPending := cw.buf.Elements()
		for _, failedBP := range bpPending {
			pts = append(pts, failedBP.Points()...)
		}
		if len(pts) < 1 {
			continue
		}

		retryBatch, _ := client.NewBatchPoints(client.BatchPointsConfig{
			Database:  cw.db,
			Precision: "s",
		})

		retryBatch.AddPoints(pts)

		if err := cw.influxClient.Write(retryBatch); err != nil {
			cw.logger.Error(logPrefix, "Couldn't write to server:", err.Error())
			cw.buf.Add(bpPending...)
			continue
		}
	}
}
