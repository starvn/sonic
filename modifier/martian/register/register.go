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

package register

import "sync"

const (
	ScopeRequest  Scope = "request"
	ScopeResponse Scope = "response"
)

type Register map[string]Component

type Scope string

type Component struct {
	Scope       []Scope
	NewFromJSON func(b []byte) (interface{}, error)
}

var (
	register = Register{}
	mutex    = &sync.RWMutex{}
)

func Set(name string, scope []Scope, f func(b []byte) (interface{}, error)) {
	mutex.Lock()
	register[name] = Component{
		Scope:       scope,
		NewFromJSON: f,
	}
	mutex.Unlock()
}

func Get() Register {
	mutex.RLock()
	r := make(Register, len(register))
	for k, v := range register {
		r[k] = v
	}
	mutex.RUnlock()
	return r
}
