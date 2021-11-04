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

package xml

import (
	"github.com/clbanning/mxj"
	"strings"
	"testing"
)

func TestNewDecoder_map(t *testing.T) {
	decoder := NewDecoder(false)
	original := strings.NewReader(`
			<Person id="1">
				<FullName>Alice</FullName>
				<Company>Acme</Company>
				<Email where="home">
					<Addr>gre@example.com</Addr>
				</Email>
				<Email where='work'>
					<Addr>gre@work.com</Addr>
				</Email>
				<Group>
					<Value>Friends</Value>
					<Value>Squash</Value>
				</Group>
			</Person>`)
	var result map[string]interface{}
	if err := decoder(original, &result); err != nil {
		t.Error("Unexpected error:", err.Error())
	}
	if len(result) != 1 {
		t.Error("Unexpected result:", result)
	}
	if v, ok := result["Person"]; !ok || len(v.(map[string]interface{})) != 5 {
		t.Error("result with wrong len:", result)
	}
	if result["Person"].(map[string]interface{})["FullName"] != "Alice" {
		t.Error("wrong result:", result)
	}
	if result["Person"].(map[string]interface{})["Company"] != "Acme" {
		t.Error("wrong result:", result)
	}
}

func TestNewDecoder_collection(t *testing.T) {
	decoder := NewDecoder(true)
	original := strings.NewReader(`
		<People>
			<Person id="1">
				<FullName>Alice</FullName>
				<Company>Acme</Company>
			</Person>
			<Person id="2">
				<FullName>Bob</FullName>
				<Company>Acme</Company>
			</Person>
			<Person id="3">
				<FullName>Charles</FullName>
				<Company>Acme</Company>
			</Person>
		</People>`)
	var result map[string]interface{}
	if err := decoder(original, &result); err != nil {
		t.Error("Unexpected error:", err.Error())
	}
	if len(result) != 1 {
		t.Error("Unexpected result:", result)
	}
	v, ok := result["collection"]
	if !ok {
		t.Error("wrong result:", result)
	}
	embedded := v.(mxj.Map).Old()["People"].(map[string]interface{})["Person"].([]interface{})
	if len(embedded) != 3 {
		t.Error("wrong result:", embedded)
	}
	if embedded[0].(map[string]interface{})["FullName"].(string) != "Alice" {
		t.Error("wrong result:", result)
	}
	if embedded[1].(map[string]interface{})["FullName"].(string) != "Bob" {
		t.Error("wrong result:", result)
	}
	if embedded[2].(map[string]interface{})["FullName"].(string) != "Charles" {
		t.Error("wrong result:", result)
	}
}

func TestNewDecoder_utf8(t *testing.T) {
	decoder := NewDecoder(false)
	original := strings.NewReader(`
			<?xml version="1.0" encoding="UTF-8"?>
			<Person id="1">
				<FullName>Alice</FullName>
				<Company>Acme</Company>
				<Email where="home">
					<Addr>gre@example.com</Addr>
				</Email>
				<Email where='work'>
					<Addr>gre@work.com</Addr>
				</Email>
				<Group>
					<Value>Friends</Value>
					<Value>Squash</Value>
				</Group>
			</Person>`)
	var result map[string]interface{}
	if err := decoder(original, &result); err != nil {
		t.Error("Unexpected error:", err.Error())
	}
	if len(result) != 1 {
		t.Error("Unexpected result:", result)
	}
	if v, ok := result["Person"]; !ok || len(v.(map[string]interface{})) != 5 {
		t.Error("result with wrong len:", result)
	}
	if result["Person"].(map[string]interface{})["FullName"] != "Alice" {
		t.Error("wrong result:", result)
	}
	if result["Person"].(map[string]interface{})["Company"] != "Acme" {
		t.Error("wrong result:", result)
	}
}

func TestNewDecoder_iso88591(t *testing.T) {
	decoder := NewDecoder(false)
	original := strings.NewReader(`
			<?xml version="1.0" encoding="iso-8859-1"?>
			<Person id="1">
				<FullName>Alice</FullName>
				<Company>Acme</Company>
				<Email where="home">
					<Addr>gre@example.com</Addr>
				</Email>
				<Email where='work'>
					<Addr>gre@work.com</Addr>
				</Email>
				<Group>
					<Value>Friends</Value>
					<Value>Squash</Value>
				</Group>
			</Person>`)
	var result map[string]interface{}
	if err := decoder(original, &result); err != nil {
		t.Error("Unexpected error:", err.Error())
	}
	if len(result) != 1 {
		t.Error("Unexpected result:", result)
	}
	if v, ok := result["Person"]; !ok || len(v.(map[string]interface{})) != 5 {
		t.Error("result with wrong len:", result)
	}
	if result["Person"].(map[string]interface{})["FullName"] != "Alice" {
		t.Error("wrong result:", result)
	}
	if result["Person"].(map[string]interface{})["Company"] != "Acme" {
		t.Error("wrong result:", result)
	}
}

func TestNewDecoder_ko(t *testing.T) {
	for _, testCase := range []bool{true, false} {
		decoder := NewDecoder(testCase)
		original := strings.NewReader(`3`)
		var result map[string]interface{}
		if err := decoder(original, &result); err == nil {
			t.Error("Expecting error!")
		}
	}
}
