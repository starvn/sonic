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

package histogram

import (
	"github.com/influxdata/influxdb/client/v2"
	"github.com/starvn/sonic/telemetry/metrics"
	"github.com/starvn/turbo/log"
	"regexp"
	"time"
)

func Points(hostname string, now time.Time, histograms map[string]metrics.HistogramData, logger log.Logger) []*client.Point {
	points := latencyPoints(hostname, now, histograms, logger)
	points = append(points, routerPoints(hostname, now, histograms, logger)...)
	if p := debugPoint(hostname, now, histograms, logger); p != nil {
		points = append(points, p)
	}
	if p := runtimePoint(hostname, now, histograms, logger); p != nil {
		points = append(points, p)
	}
	return points
}

var (
	latencyPattern = `sonic\.proxy\.latency\.layer\.([a-zA-Z]+)\.name\.(.*)\.complete\.(true|false)\.error\.(true|false)`
	latencyRegexp  = regexp.MustCompile(latencyPattern)

	routerPattern = `sonic\.router\.response\.(.*)\.(size|time)`
	routerRegexp  = regexp.MustCompile(routerPattern)
)

func latencyPoints(hostname string, now time.Time, histograms map[string]metrics.HistogramData, logger log.Logger) []*client.Point {
	var res []*client.Point
	for k, histogram := range histograms {
		if !latencyRegexp.MatchString(k) {
			continue
		}

		if isEmpty(histogram) {
			continue
		}

		params := latencyRegexp.FindAllStringSubmatch(k, -1)[0][1:]
		tags := map[string]string{
			"host":     hostname,
			"layer":    params[0],
			"name":     params[1],
			"complete": params[2],
			"error":    params[3],
		}

		histogramPoint, err := client.NewPoint("requests", tags, newFields(histogram), now)
		if err != nil {
			logger.Error("creating histogram point:", err.Error())
			continue
		}
		res = append(res, histogramPoint)
	}
	return res
}

func routerPoints(hostname string, now time.Time, histograms map[string]metrics.HistogramData, logger log.Logger) []*client.Point {
	var res []*client.Point
	for k, histogram := range histograms {
		if !routerRegexp.MatchString(k) {
			continue
		}

		if isEmpty(histogram) {
			continue
		}

		params := routerRegexp.FindAllStringSubmatch(k, -1)[0][1:]
		tags := map[string]string{
			"host": hostname,
			"name": params[0],
		}

		histogramPoint, err := client.NewPoint("router.response-"+params[1], tags, newFields(histogram), now)
		if err != nil {
			logger.Error("creating histogram point:", err.Error())
			continue
		}
		res = append(res, histogramPoint)
	}
	return res
}

func debugPoint(hostname string, now time.Time, histograms map[string]metrics.HistogramData, logger log.Logger) *client.Point {
	hd, ok := histograms["sonic.service.debug.GCStats.Pause"]
	if !ok {
		return nil
	}
	tags := map[string]string{
		"host": hostname,
	}

	histogramPoint, err := client.NewPoint("service.debug.GCStats.Pause", tags, newFields(hd), now)
	if err != nil {
		logger.Error("creating histogram point:", err.Error())
		return nil
	}
	return histogramPoint
}

func runtimePoint(hostname string, now time.Time, histograms map[string]metrics.HistogramData, logger log.Logger) *client.Point {
	hd, ok := histograms["sonic.service.runtime.MemStats.PauseNs"]
	if !ok {
		return nil
	}
	tags := map[string]string{
		"host": hostname,
	}

	histogramPoint, err := client.NewPoint("service.runtime.MemStats.PauseNs", tags, newFields(hd), now)
	if err != nil {
		logger.Error("creating histogram point:", err.Error())
		return nil
	}
	return histogramPoint
}

func isEmpty(histogram metrics.HistogramData) bool {
	return histogram.Max == 0 && histogram.Min == 0 &&
		histogram.Mean == .0 && histogram.Stddev == .0 && histogram.Variance == 0 &&
		(len(histogram.Percentiles) == 0 ||
			histogram.Percentiles[0] == .0 && histogram.Percentiles[len(histogram.Percentiles)-1] == .0)
}

func newFields(h metrics.HistogramData) map[string]interface{} {
	fields := map[string]interface{}{
		"max":      int(h.Max),
		"min":      int(h.Min),
		"mean":     int(h.Mean),
		"stddev":   int(h.Stddev),
		"variance": int(h.Variance),
	}

	if len(h.Percentiles) != 7 {
		return fields
	}

	fields["p0.1"] = h.Percentiles[0]
	fields["p0.25"] = h.Percentiles[1]
	fields["p0.5"] = h.Percentiles[2]
	fields["p0.75"] = h.Percentiles[3]
	fields["p0.9"] = h.Percentiles[4]
	fields["p0.95"] = h.Percentiles[5]
	fields["p0.99"] = h.Percentiles[6]

	return fields
}
