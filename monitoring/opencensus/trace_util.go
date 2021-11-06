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
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/trace"
	"net/http"
)

func SpanNameFromURL(req *http.Request) string {
	return req.URL.Path
}

func RequestAttrs(r *http.Request) []trace.Attribute {
	userAgent := r.UserAgent()

	attrs := make([]trace.Attribute, 0, 5)
	attrs = append(attrs,
		trace.StringAttribute(ochttp.PathAttribute, r.URL.Path),
		trace.StringAttribute(ochttp.URLAttribute, r.URL.String()),
		trace.StringAttribute(ochttp.HostAttribute, r.Host),
		trace.StringAttribute(ochttp.MethodAttribute, r.Method),
	)

	if userAgent != "" {
		attrs = append(attrs, trace.StringAttribute(ochttp.UserAgentAttribute, userAgent))
	}

	return attrs
}

func ResponseAttrs(resp *http.Response) []trace.Attribute {
	return []trace.Attribute{
		trace.Int64Attribute(ochttp.StatusCodeAttribute, int64(resp.StatusCode)),
	}
}
