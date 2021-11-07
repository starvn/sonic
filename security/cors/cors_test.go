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

package cors

import (
	"encoding/json"
	"testing"
	"time"
)

func TestConfigGetter(t *testing.T) {
	sampleCfg := map[string]interface{}{}
	serialized := []byte(`{ "github.com/starvn/sonic/security/cors": {
			"allow_origins": [ "http://localhost", "http://www.example.com" ],
			"allow_headers": [ "X-Test", "Content-Type"],
			"allow_methods": [ "POST", "GET" ],
			"expose_headers": [ "Content-Type" ],
			"allow_credentials": false,
			"max_age": "24h"
			}
		}`)
	_ = json.Unmarshal(serialized, &sampleCfg)
	testCfg := ConfigGetter(sampleCfg).(Config)

	if len(testCfg.AllowOrigins) != 2 {
		t.Error("Should have exactly 2 allowed origins.\n")
	}
	for i, v := range []string{"http://localhost", "http://www.example.com"} {
		if testCfg.AllowOrigins[i] != v {
			t.Errorf("Invalid value %s should be %s\n", testCfg.AllowOrigins[i], v)
		}
	}
	if len(testCfg.AllowHeaders) != 2 {
		t.Error("Should have exactly 2 allowed headers.\n")
	}
	for i, v := range []string{"X-Test", "Content-Type"} {
		if testCfg.AllowHeaders[i] != v {
			t.Errorf("Invalid value %s should be %s\n", testCfg.AllowHeaders[i], v)
		}
	}
	if len(testCfg.AllowMethods) != 2 {
		t.Error("Should have exactly 2 allowed headers.\n")
	}
	for i, v := range []string{"POST", "GET"} {
		if testCfg.AllowMethods[i] != v {
			t.Errorf("Invalid value %s should be %s\n", testCfg.AllowMethods[i], v)
		}
	}
	if len(testCfg.ExposeHeaders) != 1 {
		t.Error("Should have exactly 2 allowed headers.\n")
	}
	for i, v := range []string{"Content-Type"} {
		if testCfg.ExposeHeaders[i] != v {
			t.Errorf("Invalid value %s should be %s\n", testCfg.ExposeHeaders[i], v)
		}
	}
	if testCfg.AllowCredentials {
		t.Error("Allow Credentials should be disabled.\n")
	}

	if testCfg.MaxAge != 24*time.Hour {
		t.Errorf("Unexpected collection time: %v\n", testCfg.MaxAge)
	}
}

func TestDefaultConfiguration(t *testing.T) {
	sampleCfg := map[string]interface{}{}
	serialized := []byte(`{ "github.com/starvn/sonic/security/cors": {
			"allow_origins": [ "http://www.example.com" ]
	}}`)
	_ = json.Unmarshal(serialized, &sampleCfg)
	defaultCfg := ConfigGetter(sampleCfg).(Config)
	if defaultCfg.AllowOrigins[0] != "http://www.example.com" {
		t.Error("Wrong AllowOrigin.\n")
	}
}

func TestWrongConfiguration(t *testing.T) {
	sampleCfg := map[string]interface{}{}
	if _, ok := ConfigGetter(sampleCfg).(Config); ok {
		t.Error("The config should be nil\n")
	}
	badCfg := map[string]interface{}{Namespace: "test"}
	if _, ok := ConfigGetter(badCfg).(Config); ok {
		t.Error("The config should be nil\n")
	}
}

func TestEmptyConfiguration(t *testing.T) {
	noOriginCfg := map[string]interface{}{}
	serialized := []byte(`{ "github.com/starvn/sonic/security/cors": {
			}
		}`)
	_ = json.Unmarshal(serialized, &noOriginCfg)
	if v, ok := ConfigGetter(noOriginCfg).(Config); !ok {
		t.Errorf("The configuration should not be empty: %v\n", v)
	}
}
