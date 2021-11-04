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

// Package queue provides an AMQP compatible backend for the Sonic API Gateway
package queue

import (
	"context"
	"github.com/starvn/turbo/config"
	"github.com/starvn/turbo/log"
	"github.com/starvn/turbo/proxy"
	"github.com/streadway/amqp"
	"sync"
)

func NewBackendFactory(ctx context.Context, logger log.Logger, bf proxy.BackendFactory) proxy.BackendFactory {
	f := backendFactory{
		logger:    logger,
		bf:        bf,
		ctx:       ctx,
		mu:        new(sync.Mutex),
		consumers: map[string]<-chan amqp.Delivery{},
	}

	return f.New
}

type backendFactory struct {
	ctx       context.Context
	logger    log.Logger
	bf        proxy.BackendFactory
	consumers map[string]<-chan amqp.Delivery
	mu        *sync.Mutex
}

func (f backendFactory) New(remote *config.Backend) proxy.Proxy {
	if prxy, err := f.initConsumer(f.ctx, remote); err == nil {
		return prxy
	}

	if prxy, err := f.initProducer(f.ctx, remote); err == nil {
		return prxy
	}

	return f.bf(remote)
}

func (f backendFactory) newChannel(path string) (*amqp.Channel, closer, error) {
	conn, err := amqp.Dial(path)
	if err != nil {
		return nil, nopCloser, err
	}
	ch, err := conn.Channel()
	return ch, conn.Close, nil
}

type closer func() error

func nopCloser() error { return nil }

type queueCfg struct {
	Name          string   `json:"name"`
	Exchange      string   `json:"exchange"`
	RoutingKey    []string `json:"routing_key"`
	Durable       bool     `json:"durable"`
	Delete        bool     `json:"delete"`
	Exclusive     bool     `json:"exclusive"`
	NoWait        bool     `json:"no_wait"`
	PrefetchCount int      `json:"prefetch_count"`
	PrefetchSize  int      `json:"prefetch_size"`
}
