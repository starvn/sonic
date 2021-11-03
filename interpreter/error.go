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

package interpreter

import (
	"errors"
	"fmt"
	"github.com/alexeyco/binder"
	"strconv"
	"strings"
)

type ErrWrongChecksumType string

func (e ErrWrongChecksumType) Error() string {
	return "lua: wrong checksum type for source " + string(e)
}

type ErrWrongChecksum struct {
	Source, Actual, Expected string
}

func (e ErrWrongChecksum) Error() string {
	return fmt.Sprintf("lua: wrong checksum for source %s. have: %v, want: %v", e.Source, e.Actual, e.Expected)
}

type ErrUnknownSource string

func (e ErrUnknownSource) Error() string {
	return "lua: unable to load required source " + string(e)
}

var errNeedsArguments = errors.New("need arguments")

type ErrInternal string

func (e ErrInternal) Error() string {
	return string(e)
}

type ErrInternalHTTP struct {
	msg  string
	code int
}

func (e ErrInternalHTTP) StatusCode() int {
	return e.code
}

func (e ErrInternalHTTP) Error() string {
	return e.msg
}

func ToError(e error) error {
	if e == nil {
		return e
	}

	if _, ok := e.(*binder.Error); !ok {
		return e
	}

	originalMsg := e.Error()
	start := strings.Index(originalMsg, ":")

	if l := len(originalMsg); originalMsg[l-1] == ')' && originalMsg[l-5] == '(' {
		code, err := strconv.Atoi(originalMsg[l-4 : l-1])
		if err != nil {
			code = 500
		}
		return ErrInternalHTTP{msg: originalMsg[start+2 : l-6], code: code}
	}

	return ErrInternal(originalMsg[start+2:])
}

func RegisterErrors(b *binder.Binder) {
	b.Func("custom_error", func(c *binder.Context) error {
		switch c.Top() {
		case 0:
			return errNeedsArguments
		case 1:
			return errors.New(c.Arg(1).String())
		default:
			return fmt.Errorf("%s (%d)", c.Arg(1).String(), int(c.Arg(2).Number()))
		}
	})
}
