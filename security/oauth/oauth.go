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

// Package oauth add support for auth2 client credentials grant to the Sonic API Gateway
package oauth

import (
	"context"
	"github.com/starvn/turbo/config"
	"github.com/starvn/turbo/transport/http/client"
	"golang.org/x/oauth2/clientcredentials"
	"net/http"
	"strings"
)

const Namespace = "github.com/starvn/security/oauth"

func NewHTTPClient(cfg *config.Backend) client.HTTPClientFactory {
	oauth, ok := configGetter(cfg.ExtraConfig).(Config)
	if !ok || oauth.IsDisabled {
		return client.NewHTTPClient
	}
	c := clientcredentials.Config{
		ClientID:       oauth.ClientID,
		ClientSecret:   oauth.ClientSecret,
		TokenURL:       oauth.TokenURL,
		Scopes:         strings.Split(oauth.Scopes, ","),
		EndpointParams: oauth.EndpointParams,
	}
	cli := c.Client(context.Background())
	return func(_ context.Context) *http.Client {
		return cli
	}
}

type Config struct {
	IsDisabled     bool
	ClientID       string
	ClientSecret   string
	TokenURL       string
	Scopes         string
	EndpointParams map[string][]string
}

var ZeroCfg = Config{}

func configGetter(e config.ExtraConfig) interface{} {
	v, ok := e[Namespace]
	if !ok {
		return nil
	}
	tmp, ok := v.(map[string]interface{})
	if !ok {
		return nil
	}
	cfg := Config{}
	if v, ok := tmp["is_disabled"]; ok {
		cfg.IsDisabled = v.(bool)
	}
	if v, ok := tmp["client_id"]; ok {
		cfg.ClientID = v.(string)
	}
	if v, ok := tmp["client_secret"]; ok {
		cfg.ClientSecret = v.(string)
	}
	if v, ok := tmp["token_url"]; ok {
		cfg.TokenURL = v.(string)
	}
	if v, ok := tmp["scopes"]; ok {
		cfg.Scopes = v.(string)
	}
	if v, ok := tmp["endpoint_params"]; ok {
		tmp = v.(map[string]interface{})
		res := map[string][]string{}
		for k, vs := range tmp {
			var values []string
			for _, v := range vs.([]interface{}) {
				values = append(values, v.(string))
			}
			res[k] = values
		}
		cfg.EndpointParams = res
	}
	return cfg
}
