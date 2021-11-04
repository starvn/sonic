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

// Package jsonschema provides a JSONschema jsonschema for the Sonic API Gateway
package jsonschema

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/starvn/turbo/config"
	"github.com/starvn/turbo/log"
	"github.com/starvn/turbo/proxy"
	"github.com/xeipuuv/gojsonschema"
	"io/ioutil"
	"net/http"
	"strings"
)

const Namespace = "github.com/starvn/jsonschema"

var ErrEmptyBody = errors.New("could not validate an empty body")

func ProxyFactory(logger log.Logger, pf proxy.Factory) proxy.FactoryFunc {
	return proxy.FactoryFunc(func(cfg *config.EndpointConfig) (proxy.Proxy, error) {
		next, err := pf.New(cfg)
		if err != nil {
			return proxy.NoopProxy, err
		}
		schemaLoader, ok := configGetter(cfg.ExtraConfig).(gojsonschema.JSONLoader)
		if !ok || schemaLoader == nil {
			return next, nil
		}
		logger.Debug("[ENDPOINT: " + cfg.Endpoint + "][JSONSchema] Validator enabled")
		return newProxy(schemaLoader, next), nil
	})
}

func newProxy(schemaLoader gojsonschema.JSONLoader, next proxy.Proxy) proxy.Proxy {
	return func(ctx context.Context, r *proxy.Request) (*proxy.Response, error) {
		if r.Body == nil {
			return nil, ErrEmptyBody
		}
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}
		_ = r.Body.Close()
		r.Body = ioutil.NopCloser(bytes.NewBuffer(body))

		result, err := gojsonschema.Validate(schemaLoader, gojsonschema.NewBytesLoader(body))
		if err != nil {
			return nil, err
		}
		if !result.Valid() {
			return nil, &validationError{errs: result.Errors()}
		}

		return next(ctx, r)
	}
}

func configGetter(cfg config.ExtraConfig) interface{} {
	v, ok := cfg[Namespace]
	if !ok {
		return nil
	}
	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(v); err != nil {
		return nil
	}
	return gojsonschema.NewBytesLoader(buf.Bytes())
}

type validationError struct {
	errs []gojsonschema.ResultError
}

func (v *validationError) Error() string {
	errs := make([]string, len(v.errs))
	for i, desc := range v.errs {
		errs[i] = fmt.Sprintf("- %s", desc)
	}
	return strings.Join(errs, "\n")
}

func (v *validationError) StatusCode() int {
	return http.StatusBadRequest
}
