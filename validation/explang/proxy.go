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

// Package explang provides a Common Expression Language (EXPLANG) module for the Sonic API Gateway
package explang

import (
	"context"
	"fmt"
	"github.com/google/cel-go/cel"
	"github.com/starvn/sonic/validation/explang/internal"
	"github.com/starvn/turbo/config"
	"github.com/starvn/turbo/log"
	"github.com/starvn/turbo/proxy"
	"time"
)

func ProxyFactory(l log.Logger, pf proxy.Factory) proxy.Factory {
	return proxy.FactoryFunc(func(cfg *config.EndpointConfig) (proxy.Proxy, error) {
		logPrefix := "[ENDPOINT: " + cfg.Endpoint + "][EXPLANG]"
		next, err := pf.New(cfg)
		if err != nil {
			return next, err
		}

		def, ok := internal.ConfigGetter(cfg.ExtraConfig)
		if !ok {
			return next, nil
		}
		l.Debug(logPrefix, "Loading configuration")

		p, err := newProxy(l, logPrefix, def, next)
		if err != nil {
			l.Warning(logPrefix, "Error parsing the definitions:", err.Error())
			l.Warning(logPrefix, "Falling back to the next pipe proxy")
			return next, nil
		}
		return p, err
	})
}

func BackendFactory(l log.Logger, bf proxy.BackendFactory) proxy.BackendFactory {
	return func(cfg *config.Backend) proxy.Proxy {
		logPrefix := "[BACKEND: " + cfg.URLPattern + "][EXPLANG]"
		next := bf(cfg)

		def, ok := internal.ConfigGetter(cfg.ExtraConfig)
		if !ok {
			return next
		}
		l.Debug(logPrefix, "Loading configuration")

		p, err := newProxy(l, logPrefix, def, next)
		if err != nil {
			l.Warning(logPrefix, "Error parsing the definitions:", err.Error())
			l.Warning(logPrefix, "Falling back to the next backend proxy")
			return next
		}
		return p
	}
}

func newProxy(l log.Logger, name string, defs []internal.InterpretableDefinition, next proxy.Proxy) (proxy.Proxy, error) {
	p := internal.NewCheckExpressionParser(l)
	preEvaluators, err := p.ParsePre(defs)
	if err != nil {
		return proxy.NoopProxy, err
	}
	postEvaluators, err := p.ParsePost(defs)
	if err != nil {
		return proxy.NoopProxy, err
	}

	l.Debug(name, fmt.Sprintf("%d preEvaluator(s) loaded", len(preEvaluators)))
	l.Debug(name, fmt.Sprintf("%d postEvaluator(s) loaded", len(postEvaluators)))

	return func(ctx context.Context, r *proxy.Request) (*proxy.Response, error) {
		now := timeNow().Format(time.RFC3339)

		if err := evalChecks(l, name+"[pre]", newReqActivation(r, now), preEvaluators); err != nil {
			return nil, err
		}

		resp, err := next(ctx, r)
		if err != nil {
			l.Debug(name, "Delegated execution failed:", err.Error())
			return resp, err
		}

		if err := evalChecks(l, name+"[post]", newRespActivation(resp, now), postEvaluators); err != nil {
			return nil, err
		}

		return resp, nil
	}, nil
}

func evalChecks(l log.Logger, name string, args map[string]interface{}, ps []cel.Program) error {

	for i, eval := range ps {
		res, _, err := eval.Eval(args)
		if err != nil {
			l.Info(fmt.Sprintf("%s Evaluator #%d failed: %v", name, i, res))
			return fmt.Errorf("request aborted by evaluator #%d", i)
		}

		resultMsg := fmt.Sprintf("%s Evaluator #%d result: %v", name, i, res)

		if v, ok := res.Value().(bool); !ok || !v {
			l.Info(resultMsg)
			return fmt.Errorf("request aborted by evaluator #%d", i)
		}
		l.Debug(resultMsg)
	}
	return nil
}

func newReqActivation(r *proxy.Request, now string) map[string]interface{} {
	return map[string]interface{}{
		internal.PreKey + "_method":      r.Method,
		internal.PreKey + "_path":        r.Path,
		internal.PreKey + "_params":      r.Params,
		internal.PreKey + "_headers":     r.Headers,
		internal.PreKey + "_querystring": r.Query,
		internal.NowKey:                  now,
	}
}

func newRespActivation(r *proxy.Response, now string) map[string]interface{} {
	return map[string]interface{}{
		internal.PostKey + "_completed":        r.IsComplete,
		internal.PostKey + "_metadata_status":  r.Metadata.StatusCode,
		internal.PostKey + "_metadata_headers": r.Metadata.Headers,
		internal.PostKey + "_data":             r.Data,
		internal.NowKey:                        now,
	}
}

var timeNow = time.Now
