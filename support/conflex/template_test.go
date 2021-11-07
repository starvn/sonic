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

package conflex

import (
	"fmt"
	"github.com/Masterminds/sprig"
	"github.com/starvn/turbo/config"
	"io/ioutil"
	"os"
	"text/template"
)

func ExampleTemplateParser_marshal() {
	tmpFile, err := ioutil.TempFile("", "Sonic_parsed_config_template_0_")
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	defer func(name string) {
		_ = os.Remove(name)
	}(tmpFile.Name())
	if _, err := tmpFile.Write(originalTemplate); err != nil {
		fmt.Println(err.Error())
		return
	}
	if err := tmpFile.Close(); err != nil {
		fmt.Println(err.Error())
		return
	}

	expectedCfg := config.ServiceConfig{
		Port:    1234,
		Version: 42,
	}
	tmpl := TemplateParser{
		Vars: map[string]interface{}{
			"Namespace1": map[string]interface{}{
				"Namespace1-key1": "value1",
				"Namespace1-key2": 2,
			},
			"Namespace2": map[string]interface{}{
				"Namespace2-key1": "value1000",
				"Namespace2-key2": 2000,
			},
			"Jsonplaceholder": "http://example.com",
			"Port":            1234,
		},
		Parser: config.ParserFunc(func(tmpPath string) (config.ServiceConfig, error) {
			data, err := ioutil.ReadFile(tmpPath)
			fmt.Println(string(data))
			if err != nil {
				fmt.Println(err.Error())
				return expectedCfg, err
			}
			return expectedCfg, nil
		}),
	}

	tmpl.funcMap = sprig.GenericFuncMap()
	tmpl.funcMap["marshal"] = tmpl.marshal

	res, err := tmpl.Parse(tmpFile.Name())
	if err != nil {
		fmt.Println(err.Error())
	}
	if res.Port != expectedCfg.Port {
		fmt.Println("unexpected cfg")
	}
	if res.Version != expectedCfg.Version {
		fmt.Println("unexpected cfg")
	}

	// Output:
	// {
	//     "version": 42,
	//     "port": 1234,
	//     "endpoints": [
	//         {
	//             "endpoint": "/combination/{id}",
	//             "backend": [
	//                 {
	//                     "host": [
	//                         "http://example.com"
	//                     ],
	//                     "url_pattern": "/posts?userId={id}",
	//                     "is_collection": true,
	//                     "mapping": {
	//                         "collection": "posts"
	//                     },
	//                     "disable_host_sanitize": true,
	//                     "extra_config": {
	//                     	"namespace1": {"Namespace1-key1":"value1","Namespace1-key2":2}
	// 				    }
	//                 },
	//                 {
	//                     "host": [
	//                         "http://example.com"
	//                     ],
	//                     "url_pattern": "/users/{id}",
	//                     "mapping": {
	//                         "email": "personal_email"
	//                     },
	//                     "disable_host_sanitize": true,
	//                     "extra_config": {
	//                     	"namespace1": {"Namespace1-key1":"value1","Namespace1-key2":2},
	//                     	"namespace2": {"Namespace2-key1":"value1000","Namespace2-key2":2000}
	// 				    }
	//                 }
	//             ],
	//             "extra_config": {
	//             	"namespace3": { "sonic": "turbo" },
	//             	"namespace2": {"Namespace2-key1":"value1000","Namespace2-key2":2000}
	// 		    }
	//         }
	//     ],
	//     "extra_config": {
	//         "namespace2": {"Namespace2-key1":"value1000","Namespace2-key2":2000}
	//     }
	// }
}

func ExampleTemplateParser_include() {
	tmpFile, err := ioutil.TempFile("", "Sonic_parsed_config_template_1_")
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	defer func(name string) {
		_ = os.Remove(name)
	}(tmpFile.Name())
	if _, err := tmpFile.Write(originalTemplate); err != nil {
		fmt.Println(err.Error())
		return
	}
	if err := tmpFile.Close(); err != nil {
		fmt.Println(err.Error())
		return
	}

	includeTmpFile, err := ioutil.TempFile("", "Sonic_parsed_config_template_2_")
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	defer func(name string) {
		_ = os.Remove(name)
	}(includeTmpFile.Name())
	if _, err := includeTmpFile.Write([]byte(fmt.Sprintf("{{ include \"%s\" }}", tmpFile.Name()))); err != nil {
		fmt.Println(err.Error())
		return
	}
	if err := includeTmpFile.Close(); err != nil {
		fmt.Println(err.Error())
		return
	}

	expectedCfg := config.ServiceConfig{
		Port:    1234,
		Version: 42,
	}
	tmpl := TemplateParser{
		Vars: map[string]interface{}{},
		Parser: config.ParserFunc(func(tmpPath string) (config.ServiceConfig, error) {
			data, err := ioutil.ReadFile(tmpPath)
			fmt.Println(string(data))
			if err != nil {
				fmt.Println(err)
				return expectedCfg, err
			}
			return expectedCfg, nil
		}),
	}

	tmpl.funcMap = template.FuncMap{
		"include": tmpl.include,
	}

	res, err := tmpl.Parse(includeTmpFile.Name())
	if err != nil {
		fmt.Println(err.Error())
	}
	if res.Port != expectedCfg.Port {
		fmt.Println("unexpected cfg")
	}
	if res.Version != expectedCfg.Version {
		fmt.Println("unexpected cfg")
	}

	// Output:
	// {
	//     "version": {{ add 40 2 }},
	//     "port": {{ .Port }},
	//     "endpoints": [
	//         {
	//             "endpoint": "/combination/{id}",
	//             "backend": [
	//                 {
	//                     "host": [
	//                         "{{ .Jsonplaceholder }}"
	//                     ],
	//                     "url_pattern": "/posts?userId={id}",
	//                     "is_collection": true,
	//                     "mapping": {
	//                         "collection": "posts"
	//                     },
	//                     "disable_host_sanitize": true,
	//                     "extra_config": {
	//                     	"namespace1": {{ marshal .Namespace1 }}
	// 				    }
	//                 },
	//                 {
	//                     "host": [
	//                         "{{ .Jsonplaceholder }}"
	//                     ],
	//                     "url_pattern": "/users/{id}",
	//                     "mapping": {
	//                         "email": "personal_email"
	//                     },
	//                     "disable_host_sanitize": true,
	//                     "extra_config": {
	//                     	"namespace1": {{ marshal .Namespace1 }},
	//                     	"namespace2": {{ marshal .Namespace2 }}
	// 				    }
	//                 }
	//             ],
	//             "extra_config": {
	//             	"namespace3": { "sonic": "turbo" },
	//             	"namespace2": {{ marshal .Namespace2 }}
	// 		    }
	//         }
	//     ],
	//     "extra_config": {
	//         "namespace2": {{ marshal .Namespace2 }}
	//     }
	// }
}

var originalTemplate = []byte(`{
    "version": {{ add 40 2 }},
    "port": {{ .Port }},
    "endpoints": [
        {
            "endpoint": "/combination/{id}",
            "backend": [
                {
                    "host": [
                        "{{ .Jsonplaceholder }}"
                    ],
                    "url_pattern": "/posts?userId={id}",
                    "is_collection": true,
                    "mapping": {
                        "collection": "posts"
                    },
                    "disable_host_sanitize": true,
                    "extra_config": {
                    	"namespace1": {{ marshal .Namespace1 }}
				    }
                },
                {
                    "host": [
                        "{{ .Jsonplaceholder }}"
                    ],
                    "url_pattern": "/users/{id}",
                    "mapping": {
                        "email": "personal_email"
                    },
                    "disable_host_sanitize": true,
                    "extra_config": {
                    	"namespace1": {{ marshal .Namespace1 }},
                    	"namespace2": {{ marshal .Namespace2 }}
				    }
                }
            ],
            "extra_config": {
            	"namespace3": { "sonic": "turbo" },
            	"namespace2": {{ marshal .Namespace2 }}
		    }
        }
    ],
    "extra_config": {
        "namespace2": {{ marshal .Namespace2 }}
    }
}`)
