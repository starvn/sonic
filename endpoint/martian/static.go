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

package martian

import (
	"encoding/json"
	"github.com/google/martian/parse"
	"github.com/google/martian/static"
	"net/http"
)

type StaticModifier struct {
	*static.Modifier
}

type staticJSON struct {
	ExplicitPaths map[string]string    `json:"explicitPaths"`
	RootPath      string               `json:"rootPath"`
	Scope         []parse.ModifierType `json:"scope"`
}

func NewStaticModifier(rootPath string) *StaticModifier {
	return &StaticModifier{
		Modifier: static.NewModifier(rootPath),
	}
}

func (s *StaticModifier) ModifyRequest(req *http.Request) error {
	ctx := NewContext(req.Context())
	ctx.SkipRoundTrip()

	if req.URL.Scheme == "https" {
		req.URL.Scheme = "http"
	}

	*req = *req.WithContext(ctx)

	return nil
}

func staticModifierFromJSON(b []byte) (*parse.Result, error) {
	msg := &staticJSON{}
	if err := json.Unmarshal(b, msg); err != nil {
		return nil, err
	}

	mod := NewStaticModifier(msg.RootPath)
	mod.SetExplicitPathMappings(msg.ExplicitPaths)
	return parse.NewResult(mod, msg.Scope)
}
