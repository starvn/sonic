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

package gin

import (
	"context"
	"fmt"
	"github.com/auth0-community/go-auth0"
	"github.com/gin-gonic/gin"
	sjose "github.com/starvn/sonic/endpoint/jose"
	"github.com/starvn/turbo/config"
	"github.com/starvn/turbo/log"
	"github.com/starvn/turbo/proxy"
	sgin "github.com/starvn/turbo/route/gin"
	"gopkg.in/square/go-jose.v2/jwt"
	"net/http"
	"regexp"
	"strings"
)

func HandlerFactory(hf sgin.HandlerFactory, logger log.Logger, rejecterF sjose.RejecterFactory) sgin.HandlerFactory {
	return TokenSignatureValidator(TokenSigner(hf, logger), logger, rejecterF)
}

func TokenSigner(hf sgin.HandlerFactory, logger log.Logger) sgin.HandlerFactory {
	return func(cfg *config.EndpointConfig, prxy proxy.Proxy) gin.HandlerFunc {
		logPrefix := "[ENDPOINT: " + cfg.Endpoint + "][JWTSigner]"
		signerCfg, signer, err := sjose.NewSigner(cfg, nil)
		if err == sjose.ErrNoSignerCfg {
			logger.Debug(logPrefix, "Signer disabled")
			return hf(cfg, prxy)
		}
		if err != nil {
			logger.Error(logPrefix, "Unable to create the signer:", err.Error())
			return hf(cfg, prxy)
		}

		logger.Debug(logPrefix, "Signer enabled")

		return func(c *gin.Context) {
			proxyReq := sgin.NewRequest(cfg.HeadersToPass)(c, cfg.QueryString)
			ctx, cancel := context.WithTimeout(c, cfg.Timeout)
			defer cancel()

			response, err := prxy(ctx, proxyReq)
			if err != nil {
				logger.Error(logPrefix, "Proxy response:", err.Error())
				c.AbortWithStatus(http.StatusBadRequest)
				return
			}

			if response == nil {
				logger.Error(logPrefix, "Empty proxy response")
				c.AbortWithStatus(http.StatusBadRequest)
				return
			}

			if err := sjose.SignFields(signerCfg.KeysToSign, signer, response); err != nil {
				logger.Error(logPrefix, "Signing fields:", err.Error())
				c.AbortWithStatus(http.StatusBadRequest)
				return
			}

			for k, v := range response.Metadata.Headers {
				c.Header(k, v[0])
			}
			c.JSON(response.Metadata.StatusCode, response.Data)
		}
	}
}

func TokenSignatureValidator(hf sgin.HandlerFactory, logger log.Logger, rejecterF sjose.RejecterFactory) sgin.HandlerFactory {
	return func(cfg *config.EndpointConfig, prxy proxy.Proxy) gin.HandlerFunc {
		logPrefix := "[ENDPOINT: " + cfg.Endpoint + "][JWTValidator]"
		if rejecterF == nil {
			rejecterF = new(sjose.NopRejecterFactory)
		}
		rejecter := rejecterF.New(logger, cfg)

		handler := hf(cfg, prxy)
		scfg, err := sjose.GetSignatureConfig(cfg)
		if err == sjose.ErrNoValidatorCfg {
			logger.Info(logPrefix, "Validator disabled for this endpoint")
			return handler
		}
		if err != nil {
			logger.Warning(logPrefix, "Unable to parse the configuration:", err.Error())
			return handler
		}

		validator, err := sjose.NewValidator(scfg, FromCookie)
		if err != nil {
			logger.Fatal(logPrefix, "Unable to create the validator:", err.Error())
		}

		var aclCheck func(string, map[string]interface{}, []string) bool

		if scfg.RolesKeyIsNested && strings.Contains(scfg.RolesKey, ".") && scfg.RolesKey[:4] != "http" {
			logger.Debug(logPrefix, fmt.Sprintf("Roles will be matched against the nested key: '%s'", scfg.RolesKey))
			aclCheck = sjose.CanAccessNested
		} else {
			logger.Debug(logPrefix, fmt.Sprintf("Roles will be matched against the key: '%s'", scfg.RolesKey))
			aclCheck = sjose.CanAccess
		}

		var scopesMatcher func(string, map[string]interface{}, []string) bool

		if len(scfg.Scopes) > 0 && scfg.ScopesKey != "" {
			if scfg.ScopesMatcher == "all" {
				logger.Debug(logPrefix, fmt.Sprintf("Constraint added: tokens must contain a claim '%s' with all these scopes: %v", scfg.ScopesKey, scfg.Scopes))
				scopesMatcher = sjose.ScopesAllMatcher
			} else {
				logger.Debug(logPrefix, fmt.Sprintf("Constraint added: tokens must contain a claim '%s' with any of these scopes: %v", scfg.ScopesKey, scfg.Scopes))
				scopesMatcher = sjose.ScopesAnyMatcher
			}
		} else {
			logger.Debug(logPrefix, "No scope validation required")
			scopesMatcher = sjose.ScopesDefaultMatcher
		}

		if scfg.OperationDebug {
			logger.Debug(logPrefix, "Validator enabled for this endpoint. Operation debug is enabled")
		} else {
			logger.Debug(logPrefix, "Validator enabled for this endpoint")
		}

		paramExtractor := extractRequiredJWTClaims(cfg)

		return func(c *gin.Context) {
			token, err := validator.ValidateRequest(c.Request)
			if err != nil {
				if scfg.OperationDebug {
					logger.Error(logPrefix, "Unable to validate the token:", err.Error())
				}
				c.AbortWithStatus(http.StatusUnauthorized)
				return
			}

			claims := map[string]interface{}{}
			err = validator.Claims(c.Request, token, &claims)
			if err != nil {
				if scfg.OperationDebug {
					logger.Error(logPrefix, "Token sent by client is invalid:", err.Error())
				}
				c.AbortWithStatus(http.StatusUnauthorized)
				return
			}

			if rejecter.Reject(claims) {
				if scfg.OperationDebug {
					logger.Error(logPrefix, "Token sent by client rejected")
				}
				c.AbortWithStatus(http.StatusUnauthorized)
				return
			}

			if !aclCheck(scfg.RolesKey, claims, scfg.Roles) {
				if scfg.OperationDebug {
					logger.Error(logPrefix, "Token sent by client does not have sufficient roles")
				}
				c.AbortWithStatus(http.StatusForbidden)
				return
			}

			if !scopesMatcher(scfg.ScopesKey, claims, scfg.Scopes) {
				if scfg.OperationDebug {
					logger.Error(logPrefix, "Token sent by client does not have the required scopes")
				}
				c.AbortWithStatus(http.StatusForbidden)
				return
			}

			propagateHeaders(cfg, scfg.PropagateClaimsToHeader, claims, c, logger)

			paramExtractor(c, claims)

			handler(c)
		}
	}
}

func propagateHeaders(cfg *config.EndpointConfig, propagationCfg [][]string, claims map[string]interface{}, c *gin.Context, logger log.Logger) {
	logPrefix := "[ENDPOINT: " + cfg.Endpoint + "][PropagateHeaders]"
	if len(propagationCfg) > 0 {
		headersToPropagate, err := sjose.CalculateHeadersToPropagate(propagationCfg, claims)
		if err != nil {
			logger.Warning(logPrefix, err.Error())
		}
		for k, v := range headersToPropagate {
			c.Request.Header.Set(k, v)
		}
	}
}

var jwtParamsPattern = regexp.MustCompile(`{{\.JWT\.([^}]*)}}`)

func extractRequiredJWTClaims(cfg *config.EndpointConfig) func(*gin.Context, map[string]interface{}) {
	required := []string{}
	for _, backend := range cfg.Backend {
		for _, match := range jwtParamsPattern.FindAllStringSubmatch(backend.URLPattern, -1) {
			if len(match) < 2 {
				continue
			}
			required = append(required, match[1])
		}
	}
	if len(required) == 0 {
		return func(_ *gin.Context, _ map[string]interface{}) {}
	}

	return func(c *gin.Context, claims map[string]interface{}) {
		cl := sjose.Claims(claims)
		for _, param := range required {
			// TODO: check for nested claims
			v, ok := cl.Get(param)
			if !ok {
				continue
			}
			params := append(c.Params, gin.Param{Key: "JWT." + param, Value: v})
			c.Params = params
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
