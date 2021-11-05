//go:build integration
// +build integration

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

package pubsub

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/starvn/turbo/config"
	"github.com/starvn/turbo/encoding"
	"github.com/starvn/turbo/log"
	"github.com/starvn/turbo/proxy"
	"gocloud.dev/pubsub"
	_ "gocloud.dev/pubsub/mempubsub"
	"io/ioutil"
	"strings"
	"testing"
)

func TestNew_noConfig(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	called := false

	fallback := func(remote *config.Backend) proxy.Proxy {
		called = true
		return proxy.NoopProxy
	}

	buff := bytes.NewBuffer(make([]byte, 1024))
	logger, _ := log.NewLogger("DEBUG", buff, "")

	bf := NewBackendFactory(ctx, logger, fallback)

	prxy := bf.New(&config.Backend{
		Host:        []string{"schema://host"},
		ExtraConfig: map[string]interface{}{subscriberNamespace: "invalid"},
	})

	_, _ = prxy(context.Background(), &proxy.Request{})

	if !called {
		t.Error("fallback should be called")
	}

	lines := strings.Split(buff.String(), "\n")
	if !strings.HasSuffix(lines[0], "ERROR: [[BACKEND][PubSub] Error initializing subscriber: json: cannot unmarshal string into Go value of type pubsub.subscriberCfg]") {
		t.Error("unexpected first log line:", lines[0])
	}

	if lines[1] != "" {
		t.Error("unexpected final log line:", lines[1])
	}
}

func TestNew_subscriber(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fallback := func(remote *config.Backend) proxy.Proxy {
		t.Error("fallback shouldn't be called")
		return proxy.NoopProxy
	}

	buff := new(bytes.Buffer)
	logger, _ := log.NewLogger("DEBUG", buff, "")

	bf := NewBackendFactory(ctx, logger, fallback)

	topic, err := pubsub.OpenTopic(ctx, "mem://host/subscriber-topic-url")

	prxy := bf.New(&config.Backend{
		Host:    []string{"mem://host"},
		Decoder: encoding.JSONDecoder,
		ExtraConfig: config.ExtraConfig{
			subscriberNamespace: &subscriberCfg{
				SubscriptionURL: "/subscriber-topic-url",
			},
		},
	})

	_ = topic.Send(ctx, &pubsub.Message{
		Body: []byte(`{"foobar":42}`),
	})

	resp, err := prxy(context.Background(), &proxy.Request{})

	if err != nil && err != context.Canceled {
		t.Error(err)
	}

	if logging := buff.String(); strings.HasSuffix(logging, "DEBUG: [[BACKEND: mem://host/subscriber-topic-url][PubSub] Subscriber initialized successfully]") {
		t.Errorf("unexpected log: '%s'", logging)
	}

	if !resp.IsComplete {
		t.Errorf("got an incomplete response")
		return
	}

	v := resp.Data["foobar"]
	foobar, ok := v.(json.Number)
	if !ok {
		t.Errorf("unexpected response: %+v", resp.Data)
		return
	}
	if foobar.String() != "42" {
		t.Errorf("unexpected response: %+v", resp.Data)
		return
	}
}

func TestNew_publisher(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fallback := func(remote *config.Backend) proxy.Proxy {
		t.Error("fallback shouldn't be called")
		return proxy.NoopProxy
	}

	buff := bytes.NewBuffer(make([]byte, 1024))
	logger, _ := log.NewLogger("DEBUG", buff, "")

	bf := NewBackendFactory(ctx, logger, fallback)

	prxy := bf.New(&config.Backend{
		Host: []string{"mem://host"},
		ExtraConfig: config.ExtraConfig{
			publisherNamespace: &publisherCfg{
				TopicURL: "/publisher-topic-url",
			},
		},
	})

	_, _ = prxy(context.Background(), &proxy.Request{Body: ioutil.NopCloser(bytes.NewBufferString(`{"foo":"bar"}`))})

	lines := strings.Split(buff.String(), "\n")
	if !strings.HasSuffix(lines[0], "DEBUG: [[BACKEND: mem://host/publisher-topic-url][PubSub] Publisher initialized successfully]") {
		t.Error("unexpected first log line:", lines[0])
	}
	if lines[1] != "" {
		t.Error("unexpected final log line:", lines[1])
	}
}

func TestNew_publisher_unknownProvider(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	called := false

	fallback := func(remote *config.Backend) proxy.Proxy {
		called = true
		return proxy.NoopProxy
	}

	buff := bytes.NewBuffer(make([]byte, 1024))
	logger, _ := log.NewLogger("DEBUG", buff, "")

	bf := NewBackendFactory(ctx, logger, fallback)

	prxy := bf.New(&config.Backend{
		Host: []string{"schema://host"},
		ExtraConfig: config.ExtraConfig{
			publisherNamespace: &publisherCfg{
				TopicURL: "/publisher-topic-url",
			},
		},
	})

	_, _ = prxy(context.Background(), &proxy.Request{Body: ioutil.NopCloser(bytes.NewBufferString(`{"foo":"bar"}`))})

	if !called {
		t.Error("fallback should be called")
	}

	lines := strings.Split(buff.String(), "\n")
	if !strings.HasSuffix(lines[0], "ERROR: [[BACKEND: schema://host/publisher-topic-url][PubSub]%!(EXTRA string=open pubsub.Topic: no driver registered for \"schema\" for URL \"schema://host/publisher-topic-url\"; available schemes: awssns, awssqs, azuresb, gcppubsub, kafka, mem, nats, rabbit)]") {
		t.Error("unexpected first log line:", lines[0])
	}
	if lines[1] != "" {
		t.Error("unexpected final log line:", lines[1])
	}
}
