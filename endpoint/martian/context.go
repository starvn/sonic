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

import "context"

func NewContext(parent context.Context) *Context {
	return &Context{
		Context: parent,
	}
}

type Context struct {
	context.Context
	skipRoundTrip bool
}

func (c *Context) SkipRoundTrip() {
	c.skipRoundTrip = true
}

func (c *Context) SkippingRoundTrip() bool {
	return c.skipRoundTrip
}

var _ context.Context = &Context{Context: context.Background()}
