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
	"context"
	"encoding/json"
	"fmt"
	"github.com/auth0-community/go-auth0"
	sjose "github.com/starvn/sonic/endpoint/jose"
	"github.com/starvn/turbo/config"
	logging "github.com/starvn/turbo/log"
	"github.com/starvn/turbo/proxy"
	smux "github.com/starvn/turbo/route/mux"
	"gopkg.in/square/go-jose.v2/jwt"
	"log"
	"net/http"
	"strings"
)

func HandlerFactory(hf smux.HandlerFactory, paramExtractor smux.ParamExtractor, logger logging.Logger, rejecterF sjose.RejecterFactory) smux.HandlerFactory {
	return TokenSignatureValidator(TokenSigner(hf, paramExtractor, logger), logger, rejecterF)
}

func TokenSigner(hf smux.HandlerFactory, paramExtractor smux.ParamExtractor, logger logging.Logger) smux.HandlerFactory {
	return func(cfg *config.EndpointConfig, prxy proxy.Proxy) http.HandlerFunc {
		signerCfg, signer, err := sjose.NewSigner(cfg, nil)
		if err == sjose.ErrNoSignerCfg {
			logger.Info("JOSE: signer disabled for the endpoint", cfg.Endpoint)
			return hf(cfg, prxy)
		}
		if err != nil {
			logger.Error("JOSE: unable to create the signer for the endpoint", cfg.Endpoint)
			logger.Error(err.Error())
			return hf(cfg, prxy)
		}

		logger.Info("JOSE: signer enabled for the endpoint", cfg.Endpoint)

		return func(w http.ResponseWriter, r *http.Request) {
			proxyReq := smux.NewRequestBuilder(paramExtractor)(r, cfg.QueryString, cfg.HeadersToPass)
			ctx, cancel := context.WithTimeout(r.Context(), cfg.Timeout)
			defer cancel()

			response, err := prxy(ctx, proxyReq)
			if err != nil {
				logger.Error("proxy response error:", err.Error())
				http.Error(w, "", http.StatusBadRequest)
				return
			}

			if response == nil {
				http.Error(w, "", http.StatusBadRequest)
				return
			}

			if err := sjose.SignFields(signerCfg.KeysToSign, signer, response); err != nil {
				logger.Error(err.Error())
				http.Error(w, "", http.StatusBadRequest)
				return
			}

			for k, v := range response.Metadata.Headers {
				w.Header().Set(k, v[0])
			}

			err = jsonRender(w, response)
			if err != nil {
				logger.Error("render answer error:", err.Error())
			}
		}
	}
}

var emptyResponse = []byte("{}")

func jsonRender(w http.ResponseWriter, response *proxy.Response) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(response.Metadata.StatusCode)

	if response == nil {
		_, err := w.Write(emptyResponse)
		return err
	}

	js, err := json.Marshal(response.Data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return err
	}
	_, err = w.Write(js)
	return err
}

func TokenSignatureValidator(hf smux.HandlerFactory, logger logging.Logger, rejecterF sjose.RejecterFactory) smux.HandlerFactory {
	return func(cfg *config.EndpointConfig, prxy proxy.Proxy) http.HandlerFunc {
		if rejecterF == nil {
			rejecterF = new(sjose.NopRejecterFactory)
		}
		rejecter := rejecterF.New(logger, cfg)

		handler := hf(cfg, prxy)
		signatureConfig, err := sjose.GetSignatureConfig(cfg)
		if err == sjose.ErrNoValidatorCfg {
			logger.Info("JOSE: validator disabled for the endpoint", cfg.Endpoint)
			return handler
		}
		if err != nil {
			logger.Warning(fmt.Sprintf("JOSE: validator for %s: %s", cfg.Endpoint, err.Error()))
			return handler
		}

		validator, err := sjose.NewValidator(signatureConfig, FromCookie)
		if err != nil {
			log.Fatalf("%s: %s", cfg.Endpoint, err.Error())
		}

		var aclCheck func(string, map[string]interface{}, []string) bool

		if signatureConfig.RolesKeyIsNested && strings.Contains(signatureConfig.RolesKey, ".") && signatureConfig.RolesKey[:4] != "http" {
			aclCheck = sjose.CanAccessNested
		} else {
			aclCheck = sjose.CanAccess
		}

		var scopesMatcher func(string, map[string]interface{}, []string) bool

		if len(signatureConfig.Scopes) > 0 && signatureConfig.ScopesKey != "" {
			if signatureConfig.ScopesMatcher == "all" {
				scopesMatcher = sjose.ScopesAllMatcher
			} else {
				scopesMatcher = sjose.ScopesAnyMatcher
			}
		} else {
			scopesMatcher = sjose.ScopesDefaultMatcher
		}

		logger.Info("JOSE: validator enabled for the endpoint", cfg.Endpoint)

		return func(w http.ResponseWriter, r *http.Request) {
			token, err := validator.ValidateRequest(r)
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}

			claims := map[string]interface{}{}
			err = validator.Claims(r, token, &claims)
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}

			if rejecter.Reject(claims) {
				http.Error(w, "", http.StatusUnauthorized)
				return
			}

			if !aclCheck(signatureConfig.RolesKey, claims, signatureConfig.Roles) {
				http.Error(w, "", http.StatusForbidden)
				return
			}

			if !scopesMatcher(signatureConfig.ScopesKey, claims, signatureConfig.Scopes) {
				http.Error(w, "", http.StatusForbidden)
				return
			}

			propagateHeaders(cfg, signatureConfig.PropagateClaimsToHeader, claims, r, logger)

			handler(w, r)
		}
	}
}

func FromCookie(key string) func(r *http.Request) (*jwt.JSONWebToken, error) {
	if key == "" {
		key = "access_token"
	}
	return func(r *http.Request) (*jwt.JSONWebToken, error) {
		cookie, err := r.Cookie(key)
		if err != nil {
			return nil, auth0.ErrTokenNotFound
		}
		return jwt.ParseSigned(cookie.Value)
	}
}

func propagateHeaders(cfg *config.EndpointConfig, propagationCfg [][]string, claims map[string]interface{}, r *http.Request, logger logging.Logger) {
	if len(propagationCfg) > 0 {
		headersToPropagate, err := sjose.CalculateHeadersToPropagate(propagationCfg, claims)
		if err != nil {
			logger.Warning(fmt.Sprintf("JOSE: header propagations error for %s: %s", cfg.Endpoint, err.Error()))
		}
		for k, v := range headersToPropagate {
			r.Header.Set(k, v)
		}
	}
}
