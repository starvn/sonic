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
	"github.com/starvn/sonic/monitoring/metrics"
	"testing"
)

func Test_isEmpty(t *testing.T) {
	for _, h := range []metrics.HistogramData{
		{
			Max: 42,
		},
		{
			Min: 42,
		},
		{
			Mean: 42.0,
		},
		{
			Stddev: 42.0,
		},
		{
			Variance: 42.0,
		},
		{
			Percentiles: []float64{42.0, 0, 10},
		},
	} {
		if isEmpty(h) {
			t.Errorf("the histogram %v is not empty", h)
		}
	}

	if !isEmpty(metrics.HistogramData{}) {
		t.Errorf("unable to detect an empty histogram")
	}
}
