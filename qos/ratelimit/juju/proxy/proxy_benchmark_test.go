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

package proxy

import (
	"context"
	"github.com/starvn/turbo/config"
	"github.com/starvn/turbo/proxy"
	"testing"
)

func BenchmarkNewMiddleware_ok(b *testing.B) {
	p := NewMiddleware(&config.Backend{
		ExtraConfig: map[string]interface{}{Namespace: map[string]interface{}{"maxRate": 10000000000000.0, "capacity": 100000000000.0}},
	})(dummyProxy(&proxy.Response{}, nil))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = p(context.Background(), &proxy.Request{
			Path: "/turbo",
		})
	}
}

func BenchmarkNewCMiddleware_ko(b *testing.B) {
	p := NewMiddleware(&config.Backend{
		ExtraConfig: map[string]interface{}{Namespace: map[string]interface{}{"maxRate": 1.0, "capacity": 1.0}},
	})(dummyProxy(&proxy.Response{}, nil))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = p(context.Background(), &proxy.Request{
			Path: "/turbo",
		})
	}
}
