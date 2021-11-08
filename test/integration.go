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

// Package test implements utility functions to help with API Gateway testing
package test

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/textproto"
	"os"
	"os/exec"
	"path"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	defaultBinPath     *string = flag.String("sonic_bin_path", ".././sonic", "The default path to the sonic bin")
	defaultSpecsPath   *string = flag.String("sonic_spec_path", "./fixture/spec", "The default path to the spec folder")
	defaultBackendPort *int    = flag.Int("sonic_backend_port", 8081, "The port for the mocked backend api")
	defaultCfgPath     *string = flag.String(
		"sonic_config_path",
		"fixture/sonic.json",
		"The default path to the sonic config",
	)
	defaultDelay *time.Duration = flag.Duration(
		"sonic_delay",
		200*time.Millisecond,
		"The delay for the delayed backend endpoint",
	)
	defaultEnvironPatterns *string = flag.String(
		"sonic_envar_pattern",
		"",
		"Comma separated list of patterns to use to filter the envars to pass (set to \".*\" to pass everything)",
	)
	notFollowRedirects = flag.Bool("client_not_follow_redirects", false, "The test http client should not follow http redirects")
)

type TestCase struct {
	Name string `json:"name"`
	Err  string `json:"error"`
	In   Input  `json:"in"`
	Out  Output `json:"out"`
}

type Input struct {
	URL    string            `json:"url"`
	Method string            `json:"method"`
	Header map[string]string `json:"header"`
	Body   string            `json:"body"`
}

type Output struct {
	StatusCode int                 `json:"status_code"`
	Body       interface{}         `json:"body"`
	Header     map[string][]string `json:"header"`
}

type CmdBuilder interface {
	New(*Config) *exec.Cmd
}

type BackendBuilder interface {
	New(*Config) http.Server
}

type Config struct {
	BinPath         string
	CfgPath         string
	SpecsPath       string
	EnvironPatterns string
	BackendPort     int
	Delay           time.Duration
	HttpClient      *http.Client
}

func (c *Config) getBinPath() string {
	if c.BinPath != "" {
		return c.BinPath
	}
	return *defaultBinPath
}

func (c *Config) getCfgPath() string {
	if c.CfgPath != "" {
		return c.CfgPath
	}
	return *defaultCfgPath
}

func (c *Config) getSpecsPath() string {
	if c.SpecsPath != "" {
		return c.SpecsPath
	}
	return *defaultSpecsPath
}

func (c *Config) getBackendPort() int {
	if c.BackendPort != 0 {
		return c.BackendPort
	}
	return *defaultBackendPort
}

func (c *Config) getDelay() time.Duration {
	if c.Delay != 0 {
		return c.Delay
	}
	return *defaultDelay
}

func (c *Config) getEnvironPatterns() string {
	if c.EnvironPatterns != "" {
		return c.EnvironPatterns
	}
	return *defaultEnvironPatterns
}

func (c *Config) getHttpClient() *http.Client {
	if c.HttpClient != nil {
		return c.HttpClient
	}
	return defaultHttpClient()

}

func defaultHttpClient() *http.Client {
	if *notFollowRedirects {
		return &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
	}
	return http.DefaultClient
}

var defaultConfig Config

func NewIntegration(cfg *Config, cb CmdBuilder, bb BackendBuilder) (*Runner, []TestCase, error) {
	if cfg == nil {
		cfg = &defaultConfig
	}

	if cb == nil {
		cb = defaultCmdBuilder
	}
	cmd := cb.New(cfg)

	var tcs []TestCase
	if err := cmd.Start(); err != nil {
		return nil, tcs, err
	}
	closeFuncs := []func(){
		func() {
			_ = cmd.Process.Kill()
		},
	}

	go func() { fmt.Println(cmd.Wait()) }()

	var err error
	tcs, err = testCases(*cfg)
	if err != nil {
		_ = cmd.Process.Kill()
		return nil, tcs, err
	}

	if bb == nil {
		bb = defaultBackendBuilder
	}

	backend := bb.New(cfg)
	closeFuncs = append(closeFuncs, func() { _ = backend.Close() })

	go func() {
		if err := backend.ListenAndServe(); err != nil {
			log.Printf("backend closed: %v", err)
		}
	}()

	select {
	case <-time.After(1500 * time.Millisecond):
	}

	return &Runner{
		closeFuncs: closeFuncs,
		once:       new(sync.Once),
		httpClient: cfg.getHttpClient(),
	}, tcs, nil
}

type Runner struct {
	closeFuncs []func()
	once       *sync.Once
	httpClient *http.Client
}

func (i *Runner) Close() {
	i.once.Do(func() {
		for _, closeF := range i.closeFuncs {
			closeF()
		}
	})

}

func (i *Runner) Check(tc TestCase) error {
	req, err := newRequest(tc.In)
	if err != nil {
		return err
	}
	resp, err := i.httpClient.Do(req)
	if err != nil && err.Error() != tc.Err {
		return err
	}

	if err != nil {
		return nil
	}

	return assertResponse(resp, tc.Out)
}

type responseError struct {
	errMessage []string
}

func (m responseError) Error() string {
	return "wrong response:\n\t" + strings.Join(m.errMessage, "\n\t")
}

func assertResponse(actual *http.Response, expected Output) error {
	var errMsgs []string
	if actual.StatusCode != expected.StatusCode {
		errMsgs = append(errMsgs, fmt.Sprintf("unexpected status code. have: %d, want: %d", actual.StatusCode, expected.StatusCode))
	}

	for k, vs := range expected.Header {
		k = textproto.CanonicalMIMEHeaderKey(k)
		hs, ok := actual.Header[k]
		isEqual := reflect.DeepEqual(vs, hs)
		if ok && isEqual {
			continue
		}

		if ok {
			errMsgs = append(errMsgs, fmt.Sprintf("unexpected value for header %s. have: %s, want: %s", k, hs, vs))
			continue
		}

		if vs[0] != "" {
			errMsgs = append(errMsgs, fmt.Sprintf("header %s not present: %+v", k, actual.Header))
		}
	}

	var body interface{}

	if actual.Body != nil {
		b, err := ioutil.ReadAll(actual.Body)
		if err != nil {
			return err
		}
		_ = actual.Body.Close()

		switch expected.Body.(type) {
		case string:
			body = string(b)
		default:
			_ = json.Unmarshal(b, &body)
		}
	}

	if !reflect.DeepEqual(body, expected.Body) {
		errMsgs = append(errMsgs, fmt.Sprintf("unexpected body.\n\t\thave: %v\n\t\twant: %v", body, expected.Body))
	}
	if len(errMsgs) == 0 {
		return nil
	}

	return responseError{
		errMessage: errMsgs,
	}
}

func testCases(cfg Config) ([]TestCase, error) {
	var tcs []TestCase
	content, err := readSpecs(cfg.getSpecsPath())
	if err != nil {
		return tcs, err
	}

	for name, c := range content {
		tc, err := parseTestCase(name, c)
		if err != nil {
			return tcs, err
		}
		tcs = append(tcs, tc)
	}

	return tcs, nil
}

func parseTestCase(name string, in []byte) (TestCase, error) {
	tc := TestCase{}
	if err := json.Unmarshal(in, &tc); err != nil {
		return tc, err
	}
	tc.Name = name

	return tc, nil
}

func newRequest(in Input) (*http.Request, error) {
	var body io.Reader
	if in.Body != "" {
		body = bytes.NewBufferString(in.Body)
	}
	req, err := http.NewRequest(in.Method, in.URL, body)
	if err != nil {
		return nil, err
	}
	for k, v := range in.Header {
		req.Header.Add(k, v)
	}
	return req, nil
}

func readSpecs(dirPath string) (map[string][]byte, error) {
	data := map[string][]byte{}
	files, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return data, err
	}

	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".json") {
			continue
		}
		content, err := ioutil.ReadFile(path.Join(dirPath, file.Name()))
		if err != nil {
			return data, err
		}
		data[file.Name()[:len(file.Name())-5]] = content
	}
	return data, nil
}

var defaultCmdBuilder sonicCmdBuilder

type sonicCmdBuilder struct{}

func (k sonicCmdBuilder) New(cfg *Config) *exec.Cmd {
	cmd := exec.Command(cfg.getBinPath(), "run", "-d", "-c", cfg.getCfgPath())
	cmd.Env = k.getEnviron(cfg)
	return cmd
}

func (sonicCmdBuilder) getEnviron(cfg *Config) []string {
	environ := []string{"USAGE_DISABLE=1"}

	var patterns []*regexp.Regexp
	for _, pattern := range strings.Split(cfg.getEnvironPatterns(), ",") {
		re, err := regexp.Compile(pattern)
		if err != nil {
			continue
		}
		patterns = append(patterns, re)
	}

	for _, candidate := range os.Environ() {
		for _, pattern := range patterns {
			if pattern.MatchString(candidate) {
				environ = append(environ, candidate)
				break
			}
		}
	}

	return environ
}

var defaultBackendBuilder mockBackendBuilder

type mockBackendBuilder struct{}

func (mockBackendBuilder) New(cfg *Config) http.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/param_forwarding/", checkXForwardedFor(http.HandlerFunc(echoEndpoint)))
	mux.HandleFunc("/xml", checkXForwardedFor(http.HandlerFunc(xmlEndpoint)))
	mux.HandleFunc("/collection/", checkXForwardedFor(http.HandlerFunc(collectionEndpoint)))
	mux.HandleFunc("/delayed/", checkXForwardedFor(delayedEndpoint(cfg.getDelay(), http.HandlerFunc(echoEndpoint))))
	mux.HandleFunc("/redirect/", checkXForwardedFor(http.HandlerFunc(redirectEndpoint)))
	mux.HandleFunc("/jwk/symmetric", http.HandlerFunc(symmetricJWKEndpoint))

	return http.Server{
		Addr:    fmt.Sprintf(":%v", cfg.getBackendPort()),
		Handler: mux,
	}
}

func collectionEndpoint(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Add("Content-Type", "application/json")
	var res []interface{}

	for i := 0; i < 10; i++ {
		res = append(res, map[string]interface{}{
			"path": r.URL.Path,
			"i":    i,
		})
	}

	_ = json.NewEncoder(rw).Encode(res)
}

func checkXForwardedFor(h http.Handler) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		if ip := net.ParseIP(r.Header.Get("X-Forwarded-For")); ip == nil || !ip.IsLoopback() {
			http.Error(rw, "invalid X-Forwarded-For", 400)
			return
		}
		h.ServeHTTP(rw, r)
	}
}

func delayedEndpoint(d time.Duration, h http.Handler) http.HandlerFunc {
	return func(rw http.ResponseWriter, req *http.Request) {
		<-time.After(d)
		h.ServeHTTP(rw, req)
	}
}

func xmlEndpoint(rw http.ResponseWriter, _ *http.Request) {
	rw.Header().Add("Content-Type", "application/xml; charset=utf-8")
	_, _ = rw.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<user type="admin">
  <name>Elliot</name>
  <social>
    <facebook>https://facebook.com</facebook>
    <twitter>https://twitter.com</twitter>
    <youtube>https://youtube.com</youtube>
  </social>
</user>`))
}

func echoEndpoint(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Add("Content-Type", "application/json")
	rw.Header().Add("Set-Cookie", "test1=test1")
	rw.Header().Add("Set-Cookie", "test2=test2")
	r.Header.Del("X-Forwarded-For")
	_ = json.NewEncoder(rw).Encode(map[string]interface{}{
		"path":    r.URL.Path,
		"query":   r.URL.Query(),
		"headers": r.Header,
		"foo":     42,
	})
}

func redirectEndpoint(rw http.ResponseWriter, r *http.Request) {
	u := r.URL
	u.Path = "/param_forwarding/"

	status, ok2 := r.URL.Query()["status"]
	code := 301
	if !ok2 || status[0] != "301" {
		var err error
		code, err = strconv.Atoi(status[0])
		if err != nil {
			http.Error(rw, err.Error(), http.StatusBadRequest)
			return
		}
	}
	http.Redirect(rw, r, u.String(), code)
}

func symmetricJWKEndpoint(rw http.ResponseWriter, req *http.Request) {
	rw.Header().Add("Content-Type", "application/json")
	_, _ = rw.Write([]byte(`{
  "keys": [
    {
      "kty": "oct",
      "alg": "A128KW",
      "k": "GawgguFyGrWKav7AX4VKUg",
      "kid": "sim1"
    },
    {
      "kty": "oct",
      "k": "AyM1SysPpbyDfgZld3umj1qzKObwVMkoqQ-EstJQLr_T-1qS0gZH75aKtMN3Yj0iPS4hcgUuTwjAzZr1Z9CAow",
      "kid": "sim2",
      "alg": "HS256"
    }
  ]
}`))
}
