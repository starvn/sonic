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

// Package juju provides a set of rate-limit proxy and router middlewares using the github.com/juju/ratelimit lib
package juju

import (
	"context"
	"github.com/juju/ratelimit"
	srate "github.com/starvn/sonic/ratelimit"
)

func NewLimiter(maxRate float64, capacity int64) Limiter {
	return Limiter{ratelimit.NewBucketWithRate(maxRate, capacity)}
}

type Limiter struct {
	limiter *ratelimit.Bucket
}

func (l Limiter) Allow() bool {
	return l.limiter.TakeAvailable(1) > 0
}

func NewLimiterStore(maxRate float64, capacity int64, backend srate.Backend) srate.LimiterStore {
	f := func() interface{} { return NewLimiter(maxRate, capacity) }
	return func(t string) srate.Limiter {
		return backend.Load(t, f).(Limiter)
	}
}

func NewMemoryStore(maxRate float64, capacity int64) srate.LimiterStore {
	return NewLimiterStore(maxRate, capacity, srate.DefaultShardedMemoryBackend(context.Background()))
}
