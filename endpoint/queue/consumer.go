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

package queue

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/starvn/turbo/config"
	"github.com/starvn/turbo/proxy"
	"github.com/streadway/amqp"
	"io"
)

const consumerNamespace = "github.com/starvn/sonic/endpoint/queue/consume"

var errNoConsumerCfgDefined = errors.New("no queue consumer defined")
var errNoBackendHostDefined = errors.New("no host backend defined")

type consumerCfg struct {
	queueCfg
	AutoACK bool `json:"auto_ack"`
	NoLocal bool `json:"no_local"`
}

func (f backendFactory) initConsumer(ctx context.Context, remote *config.Backend) (proxy.Proxy, error) {
	if len(remote.Host) < 1 {
		return proxy.NoopProxy, errNoBackendHostDefined
	}

	dns := remote.Host[0]
	logPrefix := "[BACKEND: " + remote.URLPattern + "][AMQP]"
	cfg, err := getConsumerConfig(remote)
	if err != nil {
		if err != errNoConsumerCfgDefined {
			f.logger.Debug(logPrefix, fmt.Sprintf("%s: %s", dns, err.Error()))
		}
		return proxy.NoopProxy, err
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	if msgs, ok := f.consumers[dns+cfg.Name]; ok {
		return consumerBackend(remote, msgs), nil
	}

	ch, close, err := f.newChannel(dns)
	if err != nil {
		f.logger.Error(logPrefix, fmt.Sprintf("getting the channel for %s/%s: %s", dns, cfg.Name, err.Error()))
		return proxy.NoopProxy, err
	}

	err = ch.ExchangeDeclare(
		cfg.Exchange,
		"topic",
		cfg.Durable,
		cfg.Delete,
		cfg.Exclusive,
		cfg.NoWait,
		nil,
	)
	if err != nil {
		f.logger.Error(logPrefix, fmt.Sprintf("declaring the exchange for %s/%s: %s", dns, cfg.Name, err.Error()))
		_ = close()
		return proxy.NoopProxy, err
	}

	q, err := ch.QueueDeclare(
		cfg.Name,
		cfg.Durable,
		cfg.Delete,
		cfg.Exclusive,
		cfg.NoWait,
		nil,
	)
	if err != nil {
		f.logger.Error(logPrefix, fmt.Sprintf("declaring the queue for %s/%s: %s", dns, cfg.Name, err.Error()))
		_ = close()
		return proxy.NoopProxy, err
	}

	for _, k := range cfg.RoutingKey {
		err := ch.QueueBind(
			q.Name,
			k,
			cfg.Exchange,
			false,
			nil,
		)
		if err != nil {
			f.logger.Error(logPrefix, fmt.Sprintf("Error bindind the queue for %s/%s: %s", dns, cfg.Name, err.Error()))
		}
	}

	if cfg.PrefetchCount != 0 || cfg.PrefetchSize != 0 {
		if err := ch.Qos(cfg.PrefetchCount, cfg.PrefetchSize, false); err != nil {
			f.logger.Error(logPrefix, fmt.Sprintf("Error setting the QoS for the consumer %s/%s: %s", dns, cfg.Name, err.Error()))
			_ = close()
			return proxy.NoopProxy, err
		}
	}

	msgs, err := ch.Consume(
		cfg.Name,
		"",
		cfg.AutoACK,
		cfg.Exclusive,
		cfg.NoLocal,
		cfg.NoWait,
		nil,
	)
	if err != nil {
		f.logger.Error(logPrefix, fmt.Sprintf("Error setting up the consumer for %s/%s: %s", dns, cfg.Name, err.Error()))
		_ = close()
		return proxy.NoopProxy, err
	}

	f.consumers[dns+cfg.Name] = msgs

	f.logger.Debug(logPrefix, "Consumer attached")
	go func() {
		<-ctx.Done()
		_ = close()
	}()

	return consumerBackend(remote, msgs), nil
}

func getConsumerConfig(remote *config.Backend) (*consumerCfg, error) {
	v, ok := remote.ExtraConfig[consumerNamespace]
	if !ok {
		return nil, errNoConsumerCfgDefined
	}

	b, _ := json.Marshal(v)
	cfg := &consumerCfg{}
	err := json.Unmarshal(b, cfg)
	return cfg, err
}

func consumerBackend(remote *config.Backend, msgs <-chan amqp.Delivery) proxy.Proxy {
	ef := proxy.NewEntityFormatter(remote)
	return func(ctx context.Context, _ *proxy.Request) (*proxy.Response, error) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case msg := <-msgs:
			var data map[string]interface{}
			err := remote.Decoder(bytes.NewBuffer(msg.Body), &data)
			if err != nil && err != io.EOF {
				_ = msg.Ack(false)
				return nil, err
			}

			_ = msg.Ack(true)

			newResponse := proxy.Response{Data: data, IsComplete: true}
			newResponse = ef.Format(newResponse)
			return &newResponse, nil
		}
	}
}
