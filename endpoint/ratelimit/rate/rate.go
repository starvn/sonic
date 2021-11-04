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

// Package rate provides a set of rate-limit proxy and router middlewares using the golang.org/x/time/rate lib
package rate

import (
	"context"
	sonicrate "github.com/starvn/sonic/endpoint/ratelimit"
	"golang.org/x/time/rate"
)

func NewLimiter(maxRate float64, capacity int) Limiter {
	return Limiter{rate.NewLimiter(rate.Limit(maxRate), capacity)}
}

type Limiter struct {
	limiter *rate.Limiter
}

func (l Limiter) Allow() bool {
	return l.limiter.Allow()
}

func NewLimiterStore(maxRate float64, capacity int, backend sonicrate.Backend) sonicrate.LimiterStore {
	f := func() interface{} { return NewLimiter(maxRate, capacity) }
	return func(t string) sonicrate.Limiter {
		return backend.Load(t, f).(Limiter)
	}
}

func NewMemoryStore(maxRate float64, capacity int) sonicrate.LimiterStore {
	return NewLimiterStore(maxRate, capacity, sonicrate.DefaultShardedMemoryBackend(context.Background()))
}
