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

package queue

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"github.com/starvn/turbo/config"
	"github.com/starvn/turbo/encoding"
	"github.com/starvn/turbo/log"
	"github.com/starvn/turbo/proxy"
	"io/ioutil"
	"testing"
	"time"
)

var (
	rabbitmqHost    = flag.String("rabbitmq", "localhost", "The host of the rabbitmq server")
	totalIterations = flag.Int("iterations", 10000, "The number of produce and consume iterations")
)

func Test(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	batchSize := 10

	buf := new(bytes.Buffer)
	l, _ := log.NewLogger("DEBUG", buf, "")
	defer func() {
		fmt.Println(buf.String())
	}()

	bf := NewBackendFactory(ctx, l, func(_ *config.Backend) proxy.Proxy {
		t.Error("this backend factory shouldn't be called")
		return proxy.NoopProxy
	})
	amqpHost := fmt.Sprintf("amqp://guest:guest@%s:5672", *rabbitmqHost)
	consumerProxy := bf(&config.Backend{
		Host: []string{amqpHost},
		ExtraConfig: config.ExtraConfig{
			consumerNamespace: map[string]interface{}{
				"name":           "queue-1",
				"exchange":       "some-exchange",
				"durable":        true,
				"delete":         false,
				"exclusive":      false,
				"no_wait":        true,
				"auto_ack":       false,
				"no_local":       false,
				"routing_key":    []string{"#"},
				"prefetch_count": batchSize,
			},
		},
		Decoder: encoding.JSONDecoder,
	})

	producerProxy := bf(&config.Backend{
		Host: []string{amqpHost},
		ExtraConfig: config.ExtraConfig{
			producerNamespace: map[string]interface{}{
				"name":      "queue-1",
				"exchange":  "some-exchange",
				"durable":   true,
				"delete":    false,
				"exclusive": false,
				"no_wait":   true,
				"mandatory": true,
				"immediate": false,
			},
		},
	})

	fmt.Println("proxies created. starting the test")

	for i := 0; i < *totalIterations; i++ {
		resp, err := producerProxy(ctx, &proxy.Request{
			Headers: map[string][]string{
				"Content-Type": {"application/json"},
			},
			Params: map[string]string{"routing_key": "some_value"},
			Body:   ioutil.NopCloser(bytes.NewBufferString(fmt.Sprintf("{\"foo\":\"bar\",\"some\":%d}", i))),
		})
		if err != nil {
			t.Error(err)
			return
		}

		if resp == nil || !resp.IsComplete {
			t.Errorf("unexpected response %v", resp)
			return
		}
	}

	for i := 0; i < *totalIterations; i++ {
		localCtx, cancel := context.WithTimeout(ctx, time.Second)
		resp, err := consumerProxy(localCtx, nil)
		cancel()

		if err != nil {
			t.Errorf("#%d: unexpected error %s", i, err.Error())
			return
		}

		if resp == nil || !resp.IsComplete {
			t.Errorf("#%d: unexpected response %v", i, resp)
			return
		}

		res, ok := resp.Data["foo"]
		if !ok {
			t.Errorf("#%d: unexpected response %v", i, resp)
			return
		}
		if v, ok := res.(string); !ok || v != "bar" {
			t.Errorf("#%d: unexpected response %v", i, resp)
			return
		}
	}
}
