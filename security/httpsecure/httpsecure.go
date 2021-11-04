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

// Package httpsecure provides a complete http security layer for the Sonic API Gateway
package httpsecure

import (
	"github.com/starvn/turbo/config"
	"github.com/unrolled/secure"
)

const Namespace = "github.com/starvn/sonic/security/httpsecure"

func ConfigGetter(e config.ExtraConfig) interface{} {
	v, ok := e[Namespace]
	if !ok {
		return nil
	}
	tmp, ok := v.(map[string]interface{})
	if !ok {
		return nil
	}

	cfg := secure.Options{}

	getStrings(tmp, "allowed_hosts", &cfg.AllowedHosts)
	getStrings(tmp, "host_proxy_headers", &cfg.HostsProxyHeaders)
	getInt64(tmp, "sts_seconds", &cfg.STSSeconds)
	getString(tmp, "custom_frame_options_value", &cfg.CustomFrameOptionsValue)
	getString(tmp, "content_security_policy", &cfg.ContentSecurityPolicy)
	getString(tmp, "ssl_host", &cfg.SSLHost)
	getString(tmp, "referrer_policy", &cfg.ReferrerPolicy)
	getBool(tmp, "content_type_nosniff", &cfg.ContentTypeNosniff)
	getBool(tmp, "browser_xss_filter", &cfg.BrowserXssFilter)
	getBool(tmp, "is_development", &cfg.IsDevelopment)
	getBool(tmp, "sts_include_subdomains", &cfg.STSIncludeSubdomains)
	getBool(tmp, "frame_deny", &cfg.FrameDeny)
	getBool(tmp, "ssl_redirect", &cfg.SSLRedirect)

	return cfg
}

func getStrings(data map[string]interface{}, key string, v *[]string) {
	if vs, ok := data[key]; ok {
		var result []string
		for _, v := range vs.([]interface{}) {
			if s, ok := v.(string); ok {
				result = append(result, s)
			}
		}
		*v = result
	}
}

func getString(data map[string]interface{}, key string, v *string) {
	if val, ok := data[key]; ok {
		*v = val.(string)
		if s, ok := val.(string); ok {
			*v = s
		}
	}
}

func getBool(data map[string]interface{}, key string, v *bool) {
	if val, ok := data[key]; ok {
		if b, ok := val.(bool); ok {
			*v = b
		}
	}
}

func getInt64(data map[string]interface{}, key string, v *int64) {
	if val, ok := data[key]; ok {
		switch i := val.(type) {
		case int64:
			*v = i
		case int:
			*v = int64(i)
		case float64:
			*v = int64(i)
		}
	}
}
