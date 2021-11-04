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

package explang

import (
	"fmt"
	"github.com/google/cel-go/cel"
	"github.com/starvn/sonic/explang/internal"
	"github.com/starvn/turbo/config"
	"github.com/starvn/turbo/log"
	"time"
)

func NewRejecter(l log.Logger, cfg *config.EndpointConfig) *Rejecter {
	logPrefix := "[ENDPOINT: " + cfg.Endpoint + "][CEL]"
	def, ok := internal.ConfigGetter(cfg.ExtraConfig)
	if !ok {
		return nil
	}

	p := internal.NewCheckExpressionParser(l)
	evaluators, err := p.ParseJWT(def)
	if err != nil {
		l.Debug(logPrefix, "Error building the JWT rejecter:", err.Error())
		return nil
	}

	return &Rejecter{
		name:       logPrefix,
		evaluators: evaluators,
		logger:     l,
	}
}

type Rejecter struct {
	name       string
	evaluators []cel.Program
	logger     log.Logger
}

func (r *Rejecter) Reject(data map[string]interface{}) bool {
	now := timeNow().Format(time.RFC3339)
	reqActivation := map[string]interface{}{
		internal.JwtKey: data,
		internal.NowKey: now,
	}
	for i, eval := range r.evaluators {
		res, _, err := eval.Eval(reqActivation)
		if err != nil {
			r.logger.Info(fmt.Sprintf("%s Rejecter #%d failed: %v", r.name, i, res))
			return true
		}

		resultMsg := fmt.Sprintf("%s Rejecter #%d result: %v", r.name, i, res)
		if v, ok := res.Value().(bool); !ok || !v {
			r.logger.Info(resultMsg)
			return true
		}
		r.logger.Debug(resultMsg)
	}
	return false
}
