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
	"errors"
	"github.com/gin-gonic/gin"
	hsec "github.com/starvn/sonic/secure"
	"github.com/starvn/turbo/config"
	"github.com/unrolled/secure"
)

var ErrNoConfig = errors.New("no config present for the secure module")

func Register(cfg config.ExtraConfig, engine *gin.Engine) error {
	opt, ok := hsec.ConfigGetter(cfg).(secure.Options)
	if !ok {
		return ErrNoConfig
	}
	engine.Use(secureMw(opt))
	return nil
}

func NewSecureMw(cfg config.ExtraConfig) gin.HandlerFunc {
	opt, ok := hsec.ConfigGetter(cfg).(secure.Options)
	if !ok {
		return func(c *gin.Context) {}
	}

	return secureMw(opt)
}

func secureMw(opt secure.Options) gin.HandlerFunc {
	secureMiddleware := secure.New(opt)

	return func(c *gin.Context) {
		err := secureMiddleware.Process(c.Writer, c.Request)

		if err != nil {
			c.Abort()
			return
		}

		if status := c.Writer.Status(); status > 300 && status < 399 {
			c.Abort()
		}
	}
}
