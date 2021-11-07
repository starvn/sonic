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

package jose

import (
	"github.com/auth0-community/go-auth0"
	"gopkg.in/square/go-jose.v2/jwt"
	"net/http"
)

type TokenIDGetter interface {
	Get(*jwt.JSONWebToken) string
}

type TokenKeyIDGetterFunc func(*jwt.JSONWebToken) string

func (f TokenKeyIDGetterFunc) Get(token *jwt.JSONWebToken) string {
	return f(token)
}

func DefaultTokenKeyIDGetter(token *jwt.JSONWebToken) string {
	return token.Headers[0].KeyID
}

func X5TTokenKeyIDGetter(token *jwt.JSONWebToken) string {
	x5t, ok := token.Headers[0].ExtraHeaders["x5t"].(string)
	if !ok {
		return token.Headers[0].KeyID
	}
	return x5t
}

func CompoundX5TTokenKeyIDGetter(token *jwt.JSONWebToken) string {
	return token.Headers[0].KeyID + X5TTokenKeyIDGetter(token)
}

func TokenIDGetterFactory(keyIdentifyStrategy string) TokenIDGetter {

	var supportedKeyIdentifyStrategy = map[string]TokenKeyIDGetterFunc{
		"kid":     DefaultTokenKeyIDGetter,
		"x5t":     X5TTokenKeyIDGetter,
		"kid_x5t": CompoundX5TTokenKeyIDGetter,
	}

	if tokenGetter, ok := supportedKeyIdentifyStrategy[keyIdentifyStrategy]; ok {
		return tokenGetter
	}
	return TokenKeyIDGetterFunc(DefaultTokenKeyIDGetter)
}

type JWKClientOptions struct {
	auth0.JWKClientOptions
	KeyIdentifyStrategy string
}

type JWKClient struct {
	*auth0.JWKClient
	extractor     auth0.RequestTokenExtractor
	tokenIDGetter TokenIDGetter
}

func NewJWKClientWithCache(options JWKClientOptions, extractor auth0.RequestTokenExtractor, keyCacher auth0.KeyCacher) *JWKClient {
	return &JWKClient{
		JWKClient:     auth0.NewJWKClientWithCache(options.JWKClientOptions, extractor, keyCacher),
		extractor:     extractor,
		tokenIDGetter: TokenIDGetterFactory(options.KeyIdentifyStrategy),
	}
}

func (j *JWKClient) GetSecret(r *http.Request) (interface{}, error) {
	token, err := j.extractor.Extract(r)
	if err != nil {
		return nil, err
	}

	if len(token.Headers) < 1 {
		return nil, auth0.ErrNoJWTHeaders
	}
	keyID := j.tokenIDGetter.Get(token)
	return j.GetKey(keyID)
}
