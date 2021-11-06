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
	"github.com/google/martian/parse"
	"github.com/starvn/sonic/endpoint/martian/register"
)

func Register() {
	for k, component := range register.Get() {
		parse.Register(k, func(b []byte) (*parse.Result, error) {
			v, err := component.NewFromJSON(b)
			if err != nil {
				return nil, err
			}

			return parse.NewResult(v, toModifierType(component.Scope))
		})
	}
}

func toModifierType(scopes []register.Scope) []parse.ModifierType {
	modifierType := make([]parse.ModifierType, len(scopes))
	for k, s := range scopes {
		modifierType[k] = parse.ModifierType(s)
	}
	return modifierType
}
