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

package gin

import (
	"context"
	"github.com/gin-gonic/gin"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type trackingResponseWriter struct {
	gin.ResponseWriter
	ctx     context.Context
	reqSize int64
	start   time.Time
	endOnce sync.Once
}

var _ gin.ResponseWriter = (*trackingResponseWriter)(nil)

func (t *trackingResponseWriter) end() {
	t.endOnce.Do(func() {
		m := []stats.Measurement{
			ochttp.ServerLatency.M(float64(time.Since(t.start)) / float64(time.Millisecond)),
			ochttp.ServerResponseBytes.M(int64(t.Size())),
		}
		if t.reqSize >= 0 {
			m = append(m, ochttp.ServerRequestBytes.M(t.reqSize))
		}
		status := t.Status()
		if status == 0 {
			status = http.StatusOK
		}
		ctx, _ := tag.New(t.ctx, tag.Upsert(ochttp.StatusCode, strconv.Itoa(status)))
		stats.Record(ctx, m...)
	})
}
