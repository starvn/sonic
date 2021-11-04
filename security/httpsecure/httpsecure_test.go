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

package httpsecure

import (
	"encoding/json"
	"fmt"
	"github.com/starvn/turbo/config"
)

func ExampleConfigGetter() {
	cfg := ConfigGetter(config.ExtraConfig{
		Namespace: map[string]interface{}{
			"allowed_hosts":      []interface{}{"host1"},
			"host_proxy_headers": []interface{}{"x-custom-header"},
			"sts_seconds":        10.0,
			"ssl_redirect":       true,
			"ssl_host":           "secure.example.com",
		},
	})
	fmt.Println(cfg)

	// output:
	// {false false false false false false true false false false false         secure.example.com [host1] false [x-custom-header] <nil> map[] 10  }
}

func ExampleConfigGetter_fromParsedData() {
	sample := `{
            "allowed_hosts": ["host1"],
            "ssl_proxy_headers": {},
            "sts_seconds": 300,
            "frame_deny": true,
            "sts_include_subdomains": true
        }`
	parsedCfg := map[string]interface{}{}

	if err := json.Unmarshal([]byte(sample), &parsedCfg); err != nil {
		fmt.Println(err)
		return
	}

	cfg := ConfigGetter(config.ExtraConfig{Namespace: parsedCfg})
	fmt.Println(cfg)

	// output:
	// {false false false true false false false false false true false          [host1] false [] <nil> map[] 300  }
}
