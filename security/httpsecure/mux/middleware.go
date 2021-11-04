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

package mux

import (
	hsec "github.com/starvn/sonic/security/httpsecure"
	"github.com/starvn/turbo/config"
	"github.com/starvn/turbo/route/mux"
	"github.com/unrolled/secure"
	"net/http"
)

func NewSecureMw(cfg config.ExtraConfig) mux.HandlerMiddleware {
	opt, ok := hsec.ConfigGetter(cfg).(secure.Options)
	if !ok {
		return identityMiddleware{}
	}

	return secure.New(opt)
}

type identityMiddleware struct{}

func (i identityMiddleware) Handler(h http.Handler) http.Handler {
	return h
}
