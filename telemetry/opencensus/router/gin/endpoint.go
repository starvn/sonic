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
	"github.com/gin-gonic/gin"
	opencensus2 "github.com/starvn/sonic/telemetry/opencensus"
	"github.com/starvn/turbo/config"
	"github.com/starvn/turbo/proxy"
	sgin "github.com/starvn/turbo/route/gin"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/plugin/ochttp/propagation/b3"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
	"go.opencensus.io/trace"
	"go.opencensus.io/trace/propagation"
	"net/http"
	"time"
)

func New(hf sgin.HandlerFactory) sgin.HandlerFactory {
	return func(cfg *config.EndpointConfig, p proxy.Proxy) gin.HandlerFunc {
		return HandlerFunc(cfg, hf(cfg, p), nil)
	}
}

func HandlerFunc(cfg *config.EndpointConfig, next gin.HandlerFunc, prop propagation.HTTPFormat) gin.HandlerFunc {
	if !opencensus2.IsRouterEnabled() {
		return next
	}
	if prop == nil {
		prop = &b3.HTTPFormat{}
	}
	pathExtractor := opencensus2.GetAggregatedPathForMetrics(cfg)
	h := &handler{
		name:        cfg.Endpoint,
		propagation: prop,
		Handler:     next,
		StartOptions: trace.StartOptions{
			SpanKind: trace.SpanKindServer,
		},
		tags: []tagGenerator{
			func(r *http.Request) tag.Mutator { return tag.Upsert(ochttp.KeyServerRoute, cfg.Endpoint) },
			func(r *http.Request) tag.Mutator { return tag.Upsert(ochttp.Host, r.Host) },
			func(r *http.Request) tag.Mutator { return tag.Upsert(ochttp.Method, r.Method) },
			func(r *http.Request) tag.Mutator { return tag.Upsert(ochttp.Path, pathExtractor(r)) },
		},
	}
	return h.HandlerFunc
}

type handler struct {
	name             string
	propagation      propagation.HTTPFormat
	Handler          gin.HandlerFunc
	StartOptions     trace.StartOptions
	IsPublicEndpoint bool
	tags             []tagGenerator
}

type tagGenerator func(*http.Request) tag.Mutator

func (h *handler) HandlerFunc(c *gin.Context) {
	var traceEnd, statsEnd func()
	c.Request, traceEnd = h.startTrace(c.Writer, c.Request)
	c.Writer, statsEnd = h.startStats(c.Writer, c.Request)

	c.Set(opencensus2.ContextKey, trace.FromContext(c.Request.Context()))
	h.Handler(c)

	statsEnd()
	traceEnd()
}

func (h *handler) startTrace(_ gin.ResponseWriter, r *http.Request) (*http.Request, func()) {
	ctx := r.Context()
	var span *trace.Span
	sc, ok := h.extractSpanContext(r)

	if ok && !h.IsPublicEndpoint {
		ctx, span = trace.StartSpanWithRemoteParent(
			ctx,
			h.name,
			sc,
			trace.WithSampler(h.StartOptions.Sampler),
			trace.WithSpanKind(h.StartOptions.SpanKind),
		)
	} else {
		ctx, span = trace.StartSpan(
			ctx,
			h.name,
			trace.WithSampler(h.StartOptions.Sampler),
			trace.WithSpanKind(h.StartOptions.SpanKind),
		)

		if ok {
			span.AddLink(trace.Link{
				TraceID:    sc.TraceID,
				SpanID:     sc.SpanID,
				Type:       trace.LinkTypeChild,
				Attributes: nil,
			})
		}
	}

	span.AddAttributes(opencensus2.RequestAttrs(r)...)
	return r.WithContext(ctx), span.End
}

func (h *handler) extractSpanContext(r *http.Request) (trace.SpanContext, bool) {
	return h.propagation.SpanContextFromRequest(r)
}

func (h *handler) startStats(w gin.ResponseWriter, r *http.Request) (gin.ResponseWriter, func()) {
	tags := make([]tag.Mutator, len(h.tags))
	for i, t := range h.tags {
		tags[i] = t(r)
	}
	ctx, _ := tag.New(r.Context(), tags...)
	track := &trackingResponseWriter{
		start:          time.Now(),
		ctx:            ctx,
		ResponseWriter: w,
	}
	if r.Body == nil {
		// TODO: Handle cases where ContentLength is not set.
		track.reqSize = -1
	} else if r.ContentLength > 0 {
		track.reqSize = r.ContentLength
	}
	stats.Record(ctx, ochttp.ServerRequestCount.M(1))
	return track, track.end
}
