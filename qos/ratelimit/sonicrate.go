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

// Package ratelimit contains a collection of curated rate limit adaptors for the Sonic API Gateway
package ratelimit

import (
	"context"
	"errors"
	"sync"
	"time"
)

var (
	ErrLimited = errors.New("ERROR: rate limit exceeded")

	DataTTL = 10 * time.Minute

	now           = time.Now
	shards uint64 = 2048
)

type Limiter interface {
	Allow() bool
}

type LimiterStore func(string) Limiter

type Hasher func(string) uint64

type Backend interface {
	Load(string, func() interface{}) interface{}
	Store(string, interface{}) error
}

type ShardedMemoryBackend struct {
	shards []*MemoryBackend
	total  uint64
	hasher Hasher
}

func DefaultShardedMemoryBackend(ctx context.Context) *ShardedMemoryBackend {
	return NewShardedMemoryBackend(ctx, shards, DataTTL, PseudoFNV64a)
}

func NewShardedMemoryBackend(ctx context.Context, shards uint64, ttl time.Duration, h Hasher) *ShardedMemoryBackend {
	b := &ShardedMemoryBackend{
		shards: make([]*MemoryBackend, shards),
		total:  shards,
		hasher: h,
	}
	var i uint64
	for i = 0; i < shards; i++ {
		b.shards[i] = NewMemoryBackend(ctx, ttl)
	}
	return b
}

func (b *ShardedMemoryBackend) shard(key string) uint64 {
	return b.hasher(key) % b.total
}

func (b *ShardedMemoryBackend) Load(key string, f func() interface{}) interface{} {
	return b.shards[b.shard(key)].Load(key, f)
}

func (b *ShardedMemoryBackend) Store(key string, v interface{}) error {
	return b.shards[b.shard(key)].Store(key, v)
}

func (b *ShardedMemoryBackend) del(key ...string) {
	buckets := map[uint64][]string{}
	for _, k := range key {
		h := b.shard(k)
		ks, ok := buckets[h]
		if !ok {
			ks = []string{k}
		} else {
			ks = append(ks, k)
		}
		buckets[h] = ks
	}

	for s, ks := range buckets {
		b.shards[s].del(ks...)
	}
}

func NewMemoryBackend(ctx context.Context, ttl time.Duration) *MemoryBackend {
	m := &MemoryBackend{
		data:       map[string]interface{}{},
		lastAccess: map[string]time.Time{},
		mu:         new(sync.RWMutex),
	}

	go m.manageEvictions(ctx, ttl)

	return m
}

type MemoryBackend struct {
	data       map[string]interface{}
	lastAccess map[string]time.Time
	mu         *sync.RWMutex
}

func (m *MemoryBackend) manageEvictions(ctx context.Context, ttl time.Duration) {
	t := time.NewTicker(ttl)
	for {
		var keysToDel []string

		select {
		case <-ctx.Done():
			t.Stop()
			return
		case now := <-t.C:
			m.mu.RLock()
			for k, v := range m.lastAccess {
				if v.Add(ttl).Before(now) {
					keysToDel = append(keysToDel, k)
				}
			}
			m.mu.RUnlock()
		}

		m.del(keysToDel...)
	}
}

func (m *MemoryBackend) Load(key string, f func() interface{}) interface{} {
	m.mu.RLock()
	v, ok := m.data[key]
	m.mu.RUnlock()

	n := now()

	if ok {
		go func(t time.Time) {
			m.mu.Lock()
			if t0, ok := m.lastAccess[key]; !ok || t.After(t0) {
				m.lastAccess[key] = t
			}
			m.mu.Unlock()
		}(n)

		return v
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	v, ok = m.data[key]
	if ok {
		return v
	}

	v = f()
	m.lastAccess[key] = n
	m.data[key] = v

	return v
}

func (m *MemoryBackend) Store(key string, v interface{}) error {
	m.mu.Lock()
	m.lastAccess[key] = now()
	m.data[key] = v
	m.mu.Unlock()
	return nil
}

func (m *MemoryBackend) del(key ...string) {
	m.mu.Lock()
	for _, k := range key {
		delete(m.data, k)
		delete(m.lastAccess, k)
	}
	m.mu.Unlock()
}
