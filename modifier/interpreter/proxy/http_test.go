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

package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/alexeyco/binder"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
)

func Example_RegisterBackendModule() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headers, _ := json.Marshal(r.Header)
		fmt.Println(string(headers))
		fmt.Println(r.Method)

		if r.Body != nil {
			body, _ := ioutil.ReadAll(r.Body)
			fmt.Println(string(body))
			_ = r.Body.Close()
		}
		_, _ = fmt.Fprintln(w, "Hello, client")
	}))
	defer ts.Close()

	bindr := binder.New(binder.Options{
		SkipOpenLibs:        true,
		IncludeGoStackTrace: true,
	})

	registerHTTPRequest(context.Background(), bindr)

	code := fmt.Sprintf("local url = '%s'\n%s", ts.URL, sampleLuaCode)

	if err := bindr.DoString(code); err != nil {
		fmt.Println(err)
	}

	// output:
	// lua http test
	//
	// {"123":["456"],"Accept-Encoding":["gzip"],"Content-Length":["13"],"Foo":["bar"],"User-Agent":["Sonic Version undefined"]}
	// POST
	// {"foo":"bar"}
	// 200
	// text/plain; charset=utf-8
	// Hello, client
	//
	// {"Accept-Encoding":["gzip"],"Content-Length":["13"],"User-Agent":["Sonic Version undefined"]}
	// POST
	// {"foo":"bar"}
	// 200
	// text/plain; charset=utf-8
	// Hello, client
	//
	// {"Accept-Encoding":["gzip"],"User-Agent":["Sonic Version undefined"]}
	// GET
	//
	// 200
	// text/plain; charset=utf-8
	// Hello, client
}

const sampleLuaCode = `
print("lua http test\n")
local r = http_response.new(url, "POST", '{"foo":"bar"}', {["foo"] = "bar", ["123"] = "456"})
print(r:statusCode())
print(r:headers('Content-Type'))
print(r:body())
r:close()
local r = http_response.new(url, "POST", '{"foo":"bar"}')
print(r:statusCode())
print(r:headers('Content-Type'))
print(r:body())
r:close()
local r = http_response.new(url)
print(r:statusCode())
print(r:headers('Content-Type'))
print(r:body())
r:close()
`
