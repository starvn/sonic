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

package metrics

import "time"

func NewStats() Stats {
	return Stats{
		Time:       time.Now().UnixNano(),
		Counters:   map[string]int64{},
		Gauges:     map[string]int64{},
		Histograms: map[string]HistogramData{},
	}
}

type Stats struct {
	Time       int64
	Counters   map[string]int64
	Gauges     map[string]int64
	Histograms map[string]HistogramData
}

type HistogramData struct {
	Max         int64
	Min         int64
	Mean        float64
	Stddev      float64
	Variance    float64
	Percentiles []float64
}
