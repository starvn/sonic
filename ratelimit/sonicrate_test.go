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

package ratelimit

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"
)

func TestMemoryBackend(t *testing.T) {
	for _, tc := range []struct {
		name string
		s    int
		f    func(context.Context, time.Duration) Backend
	}{
		{name: "memory", s: 1, f: func(ctx context.Context, ttl time.Duration) Backend { return NewMemoryBackend(ctx, ttl) }},
		{name: "sharded", s: 256, f: func(ctx context.Context, ttl time.Duration) Backend {
			return NewShardedMemoryBackend(ctx, 256, ttl, PseudoFNV64a)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			testBackend(t, tc.s, tc.f)
		})
	}
}

func testBackend(t *testing.T, storesInit int, f func(context.Context, time.Duration) Backend) {
	ttl := time.Second
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mb := f(ctx, ttl)
	total := 1000 * runtime.NumCPU()

	<-time.After(ttl)

	wg := new(sync.WaitGroup)

	for w := 0; w < 10; w++ {
		wg.Add(1)
		go func() {
			for i := 0; i < total; i++ {
				_ = mb.Store(fmt.Sprintf("key-%d", i), i)
			}
			wg.Done()
		}()
	}
	wg.Wait()

	noResult := func() interface{} { return nil }

	for i := 0; i < total; i++ {
		v := mb.Load(fmt.Sprintf("key-%d", i), noResult)
		if v == nil {
			t.Errorf("key %d not present", i)
			return
		}
		if res, ok := v.(int); !ok || res != i {
			t.Errorf("unexpected value. want: %d, have: %v", i, v)
			return
		}
	}

	<-time.After(2 * ttl)

	for i := 0; i < total; i++ {
		if v := mb.Load(fmt.Sprintf("key-%d", i), noResult); v != nil {
			t.Errorf("key %d present after 2 TTL: %v", i, v)
			return
		}
	}
}
