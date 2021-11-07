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

package client

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/catalinc/hashcash"
	"github.com/starvn/sonic/support/usage"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type mockUsageServer struct {
	newSession func(context.Context, *usage.SessionRequest) (*usage.SessionReply, error)
	sendReport func(context.Context, *usage.ReportRequest) (*usage.ReportReply, error)
}

func (t *mockUsageServer) NewSession(c context.Context, s *usage.SessionRequest) (*usage.SessionReply, error) {
	return t.newSession(c, s)
}

func (t *mockUsageServer) SendReport(c context.Context, r *usage.ReportRequest) (*usage.ReportReply, error) {
	return t.sendReport(c, r)
}

func TestNew(t *testing.T) {
	done := make(chan bool)
	ctx, cancel := context.WithCancel(context.Background())
	hasher := hashcash.New(usage.HashBits, usage.SaltChars, usage.DefaultExtension)
	s := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		fmt.Println(req)
		rw.Header().Set("Content-Type", "application/json")

		if req.URL.String() == "/session" {
			_, _ = fmt.Fprintf(rw, `{"token":"some_token_value"}`)
			return
		}
		defer func() { done <- true }()

		var msg usage.ReportRequest
		if err := json.NewDecoder(req.Body).Decode(&msg); err != nil {
			t.Error(err)
		}
		_ = req.Body.Close()
		fmt.Printf("%+v\n", msg)
		if !hasher.Check(msg.Pow) {
			t.Errorf("wrong pow: %s", msg.Pow)
		}
		d, err := msg.Data.Hash()
		if err != nil {
			t.Error(err)
		}
		if !strings.Contains(msg.Pow, d) {
			t.Errorf("pow with unexpected hash. have: %s want: %s", msg.Pow, d)
		}
		if msg.Data.Expired() {
			t.Errorf("expired pow. have: %s", time.Unix(msg.Data.Time, 0))
		}

		_, _ = fmt.Fprintf(rw, `{"status":200}`)

	}))
	defer s.Close()

	<-time.After(100 * time.Millisecond)

	if err := StartReporter(ctx, Options{
		ClusterID: "clusterId",
		ServerID:  "serverId",
		URL:       s.URL,
	}); err != nil {
		t.Error(err)
	}
	<-done
	cancel()
}
