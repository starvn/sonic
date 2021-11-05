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

package jose

import (
	"github.com/starvn/turbo/config"
	"github.com/starvn/turbo/log"
	"testing"
)

func TestChainedRejecterFactory(t *testing.T) {
	rf := ChainedRejecterFactory([]RejecterFactory{
		NopRejecterFactory{},
		RejecterFactoryFunc(func(_ log.Logger, _ *config.EndpointConfig) Rejecter {
			return RejecterFunc(func(in map[string]interface{}) bool {
				v, ok := in["key"].(int)
				return ok && v == 42
			})
		}),
	})

	rejecter := rf.New(nil, nil)

	for _, tc := range []struct {
		name     string
		in       map[string]interface{}
		expected bool
	}{
		{
			name: "empty",
			in:   map[string]interface{}{},
		},
		{
			name:     "reject",
			in:       map[string]interface{}{"key": 42},
			expected: true,
		},
		{
			name: "pass-1",
			in:   map[string]interface{}{"key": "42"},
		},
		{
			name: "pass-2",
			in:   map[string]interface{}{"key": 9876},
		},
	} {
		if v := rejecter.Reject(tc.in); tc.expected != v {
			t.Errorf("unexpected result for %s. have %v, want %v", tc.name, v, tc.expected)
		}
	}
}
