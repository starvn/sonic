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

package usage

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/catalinc/hashcash"
	"io"
	"net/http"
	"runtime"
	"time"
)

const (
	defaultURL       = "https://usage.sonic.io"
	HashBits         = 20
	SaltChars        = 40
	DefaultExtension = ""

	sessionEndpoint = "/session"
	reportEndpoint  = "/report"

	timeout = 15 * time.Second
)

var (
	hasher      = hashcash.New(HashBits, SaltChars, DefaultExtension)
	reportLapse = 12 * time.Hour
)

type SessionRequest struct {
	ClusterID string `json:"cluster_id"`
	ServerID  string `json:"server_id"`
}

type SessionReply struct {
	Token string `json:"token"`
}

type ReportRequest struct {
	Token string    `json:"token"`
	Pow   string    `json:"pow"`
	Data  UsageData `json:"data"`
}

type UsageData struct {
	Version   string `json:"version"`
	Arch      string `json:"arch"`
	OS        string `json:"os"`
	ClusterID string `json:"cluster_id"`
	ServerID  string `json:"server_id"`
	Uptime    int64  `json:"uptime"`
	Time      int64  `json:"time"`
}

func (u *UsageData) Hash() (string, error) {
	buf := new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(u)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(buf.Bytes())
	return base64.URLEncoding.EncodeToString(sum[:]), nil
}

func (u *UsageData) Expired() bool {
	return time.Since(time.Unix(u.Time, 0)) > time.Minute
}

type ReportReply struct {
	Status  int    `json:"status"`
	Message string `json:"message,omitempty"`
}

type Minter interface {
	Mint(string) (string, error)
}

type UsageClient interface {
	NewSession(context.Context, *SessionRequest) (*SessionReply, error)
	SendReport(context.Context, *ReportRequest) (*ReportReply, error)
}

type HTTPClient interface {
	Send(context.Context, string, interface{}, interface{}) error
}

type Reporter struct {
	client    UsageClient
	clusterID string
	serverID  string
	start     time.Time
	minter    Minter
	token     string
	version   string
}

func New(url, clusterID, serverID, Version string) (*Reporter, error) {
	if url == "" {
		url = defaultURL
	}

	usageClient := &client{
		HTTPClient: &httpClient{
			c:   &http.Client{},
			URL: url,
		},
		sessionEndpoint: sessionEndpoint,
		reportEndpoint:  reportEndpoint,
	}
	localCtx, _ := context.WithTimeout(context.Background(), timeout)
	ses, err := usageClient.NewSession(localCtx, &SessionRequest{
		ClusterID: clusterID,
		ServerID:  serverID,
	})
	if err != nil {
		return nil, err
	}

	r := &Reporter{
		client:    usageClient,
		start:     time.Now(),
		minter:    hasher,
		token:     ses.Token,
		clusterID: clusterID,
		serverID:  serverID,
		version:   Version,
	}

	return r, nil
}

func (r *Reporter) Report(ctx context.Context) {
	for {
		localCtx, _ := context.WithTimeout(ctx, timeout)
		_ = r.SingleReport(localCtx)
		select {
		case <-ctx.Done():
			return
		case <-time.After(reportLapse):
		}
	}
}

func (r *Reporter) SingleReport(ctx context.Context) error {
	ud := UsageData{
		Version:   r.version,
		Arch:      runtime.GOARCH,
		OS:        runtime.GOOS,
		ClusterID: r.clusterID,
		ServerID:  r.serverID,
		Uptime:    int64(time.Since(r.start).Truncate(time.Second).Seconds()),
		Time:      time.Now().Unix(),
	}

	base, err := ud.Hash()
	if err != nil {
		return err
	}

	pow, err := r.minter.Mint(r.token + base)
	if err != nil {
		return err
	}

	_, err = r.client.SendReport(ctx, &ReportRequest{
		Token: r.token,
		Pow:   pow,
		Data:  ud,
	})
	return err
}

type client struct {
	HTTPClient
	sessionEndpoint string
	reportEndpoint  string
}

func (c *client) NewSession(ctx context.Context, in *SessionRequest) (*SessionReply, error) {
	reply := &SessionReply{}
	if err := c.Send(ctx, c.sessionEndpoint, in, reply); err != nil {
		return nil, err
	}

	return reply, nil
}

func (c *client) SendReport(ctx context.Context, in *ReportRequest) (*ReportReply, error) {
	reply := &ReportReply{}
	if err := c.Send(ctx, c.reportEndpoint, in, reply); err != nil {
		return nil, err
	}

	return reply, nil
}

type httpClient struct {
	c   *http.Client
	URL string
}

func (c *httpClient) Send(ctx context.Context, path string, in, out interface{}) error {
	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(in); err != nil {
		return err
	}

	req, err := http.NewRequest("POST", c.URL+path, buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.c.Do(req.WithContext(ctx))
	if err != nil {
		return err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)
	if resp.StatusCode != 200 {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
