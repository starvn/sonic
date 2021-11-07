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

package opencensus

import (
	"context"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/plugin/ochttp/propagation/b3"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
	"go.opencensus.io/trace"
	"go.opencensus.io/trace/propagation"
	"io"
	"net/http"
	"net/http/httptrace"
	"strconv"
	"sync"
	"time"
)

var defaultFormat propagation.HTTPFormat = &b3.HTTPFormat{}

type Transport struct {
	Base            http.RoundTripper
	Propagation     propagation.HTTPFormat
	StartOptions    trace.StartOptions
	GetStartOptions func(*http.Request) trace.StartOptions
	FormatSpanName  func(*http.Request) string
	NewClientTrace  func(*http.Request, *trace.Span) *httptrace.ClientTrace
	tags            []tagGenerator
}

type tagGenerator func(*http.Request) tag.Mutator

func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	rt := t.base()
	if isHealthEndpoint(req.URL.Path) {
		return rt.RoundTrip(req)
	}
	format := t.Propagation
	if format == nil {
		format = defaultFormat
	}
	spanNameFormatter := t.FormatSpanName
	if spanNameFormatter == nil {
		spanNameFormatter = SpanNameFromURL
	}

	startOpts := t.StartOptions
	if t.GetStartOptions != nil {
		startOpts = t.GetStartOptions(req)
	}

	rt = &traceTransport{
		base:   rt,
		format: format,
		startOptions: trace.StartOptions{
			Sampler:  startOpts.Sampler,
			SpanKind: trace.SpanKindClient,
		},
		formatSpanName: spanNameFormatter,
		newClientTrace: t.NewClientTrace,
	}
	rt = statsTransport{base: rt, tags: t.tags}
	return rt.RoundTrip(req)
}

func (t *Transport) base() http.RoundTripper {
	if t.Base != nil {
		return t.Base
	}
	return http.DefaultTransport
}

func (t *Transport) CancelRequest(req *http.Request) {
	type canceler interface {
		CancelRequest(*http.Request)
	}
	if cr, ok := t.base().(canceler); ok {
		cr.CancelRequest(req)
	}
}

type traceTransport struct {
	base           http.RoundTripper
	startOptions   trace.StartOptions
	format         propagation.HTTPFormat
	formatSpanName func(*http.Request) string
	newClientTrace func(*http.Request, *trace.Span) *httptrace.ClientTrace
}

func (t *traceTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	name := t.formatSpanName(req)
	ctx, span := trace.StartSpan(req.Context(), name,
		trace.WithSampler(t.startOptions.Sampler),
		trace.WithSpanKind(trace.SpanKindClient))

	if t.newClientTrace != nil {
		req = req.WithContext(httptrace.WithClientTrace(ctx, t.newClientTrace(req, span)))
	} else {
		req = req.WithContext(ctx)
	}

	if t.format != nil {
		header := make(http.Header)
		for k, v := range req.Header {
			header[k] = v
		}
		req.Header = header
		t.format.SpanContextToRequest(span.SpanContext(), req)
	}

	span.AddAttributes(RequestAttrs(req)...)
	resp, err := t.base.RoundTrip(req)
	if err != nil {
		span.SetStatus(trace.Status{Code: trace.StatusCodeUnknown, Message: err.Error()})
		span.End()
		return resp, err
	}

	span.AddAttributes(ResponseAttrs(resp)...)
	span.SetStatus(TraceStatus(resp.StatusCode, resp.Status))

	bt := &bodyTracker{rc: resp.Body, span: span}
	resp.Body = wrappedBody(bt, resp.Body)
	return resp, err
}

func wrappedBody(wrapper io.ReadCloser, body io.ReadCloser) io.ReadCloser {
	var (
		wr, i0 = body.(io.Writer)
	)
	switch {
	case !i0:
		return struct {
			io.ReadCloser
		}{wrapper}
	case i0:
		return struct {
			io.ReadCloser
			io.Writer
		}{wrapper, wr}
	default:
		return struct {
			io.ReadCloser
		}{wrapper}
	}
}

type bodyTracker struct {
	rc   io.ReadCloser
	span *trace.Span
}

var _ io.ReadCloser = (*bodyTracker)(nil)

func (bt *bodyTracker) Read(b []byte) (int, error) {
	n, err := bt.rc.Read(b)

	switch err {
	case nil:
		return n, nil
	case io.EOF:
		bt.span.End()
	default:
		bt.span.SetStatus(trace.Status{
			Code:    2,
			Message: err.Error(),
		})
	}
	return n, err
}

func (bt *bodyTracker) Close() error {
	bt.span.End()
	return bt.rc.Close()
}

func TraceStatus(httpStatusCode int, statusLine string) trace.Status {
	var code int32
	if httpStatusCode < 200 || httpStatusCode >= 400 {
		code = trace.StatusCodeUnknown
	}
	switch httpStatusCode {
	case 499:
		code = trace.StatusCodeCancelled
	case http.StatusBadRequest:
		code = trace.StatusCodeInvalidArgument
	case http.StatusUnprocessableEntity:
		code = trace.StatusCodeInvalidArgument
	case http.StatusGatewayTimeout:
		code = trace.StatusCodeDeadlineExceeded
	case http.StatusNotFound:
		code = trace.StatusCodeNotFound
	case http.StatusForbidden:
		code = trace.StatusCodePermissionDenied
	case http.StatusUnauthorized:
		code = trace.StatusCodeUnauthenticated
	case http.StatusTooManyRequests:
		code = trace.StatusCodeResourceExhausted
	case http.StatusNotImplemented:
		code = trace.StatusCodeUnimplemented
	case http.StatusServiceUnavailable:
		code = trace.StatusCodeUnavailable
	case http.StatusOK:
		code = trace.StatusCodeOK
	}
	return trace.Status{Code: code, Message: codeToStr[code]}
}

var codeToStr = map[int32]string{
	trace.StatusCodeOK:                 `OK`,
	trace.StatusCodeCancelled:          `CANCELLED`,
	trace.StatusCodeUnknown:            `UNKNOWN`,
	trace.StatusCodeInvalidArgument:    `INVALID_ARGUMENT`,
	trace.StatusCodeDeadlineExceeded:   `DEADLINE_EXCEEDED`,
	trace.StatusCodeNotFound:           `NOT_FOUND`,
	trace.StatusCodeAlreadyExists:      `ALREADY_EXISTS`,
	trace.StatusCodePermissionDenied:   `PERMISSION_DENIED`,
	trace.StatusCodeResourceExhausted:  `RESOURCE_EXHAUSTED`,
	trace.StatusCodeFailedPrecondition: `FAILED_PRECONDITION`,
	trace.StatusCodeAborted:            `ABORTED`,
	trace.StatusCodeOutOfRange:         `OUT_OF_RANGE`,
	trace.StatusCodeUnimplemented:      `UNIMPLEMENTED`,
	trace.StatusCodeInternal:           `INTERNAL`,
	trace.StatusCodeUnavailable:        `UNAVAILABLE`,
	trace.StatusCodeDataLoss:           `DATA_LOSS`,
	trace.StatusCodeUnauthenticated:    `UNAUTHENTICATED`,
}

func isHealthEndpoint(path string) bool {
	if path == "/healthz" || path == "/_ah/health" {
		return true
	}
	return false
}

type statsTransport struct {
	base http.RoundTripper

	tags []tagGenerator
}

func (t statsTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	tags := make([]tag.Mutator, len(t.tags))
	for i, tg := range t.tags {
		tags[i] = tg(req)
	}
	ctx, _ := tag.New(req.Context(), tags...)

	req = req.WithContext(ctx)
	track := &tracker{
		start: time.Now(),
		ctx:   ctx,
	}
	if req.Body == nil {
		track.reqSize = -1
	} else if req.ContentLength > 0 {
		track.reqSize = req.ContentLength
	}
	stats.Record(ctx, ochttp.ClientRequestCount.M(1))

	resp, err := t.base.RoundTrip(req)

	if err != nil {
		track.statusCode = http.StatusInternalServerError
		track.end()
	} else {
		track.statusCode = resp.StatusCode
		if req.Method != "HEAD" {
			track.respContentLength = resp.ContentLength
		}
		if resp.Body == nil {
			track.end()
		} else {
			track.body = resp.Body
			resp.Body = wrappedBody(track, resp.Body)
		}
	}
	return resp, err
}

func (t statsTransport) CancelRequest(req *http.Request) {
	type canceler interface {
		CancelRequest(*http.Request)
	}
	if cr, ok := t.base.(canceler); ok {
		cr.CancelRequest(req)
	}
}

type tracker struct {
	ctx               context.Context
	respSize          int64
	respContentLength int64
	reqSize           int64
	start             time.Time
	body              io.ReadCloser
	statusCode        int
	endOnce           sync.Once
}

var _ io.ReadCloser = (*tracker)(nil)

func (t *tracker) end() {
	t.endOnce.Do(func() {
		latencyMs := float64(time.Since(t.start)) / float64(time.Millisecond)
		respSize := t.respSize
		if t.respSize == 0 && t.respContentLength > 0 {
			respSize = t.respContentLength
		}
		m := []stats.Measurement{
			ochttp.ClientSentBytes.M(t.reqSize),
			ochttp.ClientReceivedBytes.M(respSize),
			ochttp.ClientRoundtripLatency.M(latencyMs),
		}

		_ = stats.RecordWithTags(t.ctx, []tag.Mutator{
			tag.Upsert(ochttp.StatusCode, strconv.Itoa(t.statusCode)),
			tag.Upsert(ochttp.KeyClientStatus, strconv.Itoa(t.statusCode)),
		}, m...)
	})
}

func (t *tracker) Read(b []byte) (int, error) {
	n, err := t.body.Read(b)
	t.respSize += int64(n)
	switch err {
	case nil:
		return n, nil
	case io.EOF:
		t.end()
	}
	return n, err
}

func (t *tracker) Close() error {
	t.end()
	return t.body.Close()
}
