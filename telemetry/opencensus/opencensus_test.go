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
	"github.com/starvn/turbo/config"
	"net/http"
	"testing"
)

func TestGetAggregatedPathForMetrics(t *testing.T) {
	for i, tc := range []struct {
		cfg      *config.EndpointConfig
		expected string
	}{
		{
			cfg:      &config.EndpointConfig{Endpoint: "/api/:foo/:bar"},
			expected: "/api/{foo}/{bar}",
		},
		{
			cfg: &config.EndpointConfig{
				Endpoint: "/api/:foo/:bar",
				ExtraConfig: config.ExtraConfig{
					Namespace: map[string]interface{}{"path_aggregation": "pattern"},
				},
			},
			expected: "/api/{foo}/{bar}",
		},
		{
			cfg: &config.EndpointConfig{
				Endpoint: "/api/:foo/:bar",
				ExtraConfig: config.ExtraConfig{
					Namespace: map[string]interface{}{"path_aggregation": "lastparam"},
				},
			},
			expected: "/api/foo/{bar}",
		},
		{
			cfg: &config.EndpointConfig{
				Endpoint: "/api/:foo/:bar",
				ExtraConfig: config.ExtraConfig{
					Namespace: map[string]interface{}{"path_aggregation": "off"},
				},
			},
			expected: "/api/foo/bar",
		},
		{
			expected: "/api/foo/bar",
		},
	} {
		extractor := GetAggregatedPathForMetrics(tc.cfg)
		r, _ := http.NewRequest("GET", "http://example.tld/api/foo/bar", nil)
		if tag := extractor(r); tag != tc.expected {
			t.Errorf("tc-%d: unexpected result: %s", i, tag)
		}
	}
}

func TestGetAggregatedPathForBackendMetrics(t *testing.T) {
	for i, tc := range []struct {
		cfg      *config.Backend
		expected string
	}{
		{
			cfg:      &config.Backend{URLPattern: "/api/{{.Foo}}/{{.Bar}}"},
			expected: "/api/{foo}/{bar}",
		},
		{
			cfg: &config.Backend{
				URLPattern: "/api/{{.Foo}}/{{.Bar}}",
				ExtraConfig: config.ExtraConfig{
					Namespace: map[string]interface{}{"path_aggregation": "pattern"},
				},
			},
			expected: "/api/{foo}/{bar}",
		},
		{
			cfg: &config.Backend{
				URLPattern: "/api/{{.Foo}}/{{.Bar}}",
				ExtraConfig: config.ExtraConfig{
					Namespace: map[string]interface{}{"path_aggregation": "lastparam"},
				},
			},
			expected: "/api/foo/{bar}",
		},
		{
			cfg: &config.Backend{
				URLPattern: "/api/{{.Foo}}/{{.Bar}}",
				ExtraConfig: config.ExtraConfig{
					Namespace: map[string]interface{}{"path_aggregation": "off"},
				},
			},
			expected: "/api/foo/bar",
		},
		{
			expected: "/api/foo/bar",
		},
	} {
		extractor := GetAggregatedPathForBackendMetrics(tc.cfg)
		r, _ := http.NewRequest("GET", "http://example.tld/api/foo/bar", nil)
		if tag := extractor(r); tag != tc.expected {
			t.Errorf("tc-%d: unexpected result: %s", i, tag)
		}
	}
}
