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

package gelf

import (
	"bytes"
	"encoding/json"
	"fmt"
	"gopkg.in/Graylog2/go-gelf.v2/gelf"
	"sync/atomic"
	"testing"
)

const (
	debugMsg     = "Debug msg"
	expectedAddr = "127.0.0.1:12201"
)

func mockerTCPWriter(count *uint64) func(string) (*gelf.TCPWriter, error) {
	return func(addr string) (*gelf.TCPWriter, error) {
		atomic.AddUint64(count, 1)
		if expectedAddr != addr {
			return nil, fmt.Errorf("Unexpected address: %s, expecting: %s", addr, expectedAddr)
		}
		return nil, nil
	}
}

func TestTCPWriter(t *testing.T) {
	var count uint64
	newTCPWriter = mockerTCPWriter(&count)
	cfg := map[string]interface{}{
		Namespace: map[string]interface{}{
			"address":    "127.0.0.1:12201",
			"enable_tcp": true,
		},
	}
	_, err := NewWriter(cfg)
	if err != nil {
		t.Errorf("unexpected error: %s", err.Error())
	}
	if count != 1 {
		t.Errorf("unexpected number of calls: %d", count)
	}
}

func TestUDPWriter(t *testing.T) {
	r, err := gelf.NewReader("127.0.0.1:0")
	if err != nil {
		t.Error(err)
	}
	cfg := map[string]interface{}{
		Namespace: map[string]interface{}{
			"address":    r.Addr(),
			"enable_tcp": false,
		},
	}
	wr, err := NewWriter(cfg)
	if err != nil {
		t.Error(err)
	}

	_, _ = wr.Write([]byte(debugMsg))
	msg, err := r.ReadMessage()
	if err != nil {
		t.Error(err)
	}
	buff := &bytes.Buffer{}
	if err = msg.MarshalJSONBuf(buff); err != nil {
		t.Error(err)
	}
	result := buff.Bytes()

	var message map[string]interface{}
	if err = json.Unmarshal(result, &message); err != nil {
		t.Errorf("Unable to Unmarshal message: %s", err.Error())
	}

	v, ok := message["short_message"]
	if !ok {
		t.Error("unable to decode message")
	}
	resultMsg, ok := v.(string)
	if !ok {
		t.Error("invalid short_message format")
	}
	if resultMsg != debugMsg {
		t.Errorf("the short_message field %s should be equal to %s", resultMsg, debugMsg)
	}
}

func TestInvalidConfig(t *testing.T) {
	cfg := map[string]interface{}{
		"Invalid Namespace": map[string]interface{}{},
	}
	_, err := NewWriter(cfg)
	if err != ErrWrongConfig {
		t.Errorf("The error should be %s, not %s", ErrWrongConfig.Error(), err.Error())
	}

	cfg = map[string]interface{}{
		Namespace: false,
	}
	_, err = NewWriter(cfg)
	if err != ErrWrongConfig {
		t.Errorf("The error should be %s, not %s", ErrWrongConfig.Error(), err.Error())
	}
}

func TestMissingAddr(t *testing.T) {
	cfg := map[string]interface{}{
		Namespace: map[string]interface{}{
			"enable_tcp": false,
		},
	}
	_, err := NewWriter(cfg)
	if err != ErrMissingAddr {
		t.Errorf("The error should be %s, not %s", ErrMissingAddr.Error(), err.Error())
	}
}
