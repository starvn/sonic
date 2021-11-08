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

package sonic

import (
	"github.com/gin-gonic/gin"
	lua "github.com/starvn/sonic/modifier/interpreter/route/gin"
	gindec "github.com/starvn/sonic/security/detector/gin"
	ginsec "github.com/starvn/sonic/security/httpsecure/gin"
	"github.com/starvn/turbo/config"
	"github.com/starvn/turbo/log"
	turbogin "github.com/starvn/turbo/route/gin"
	"io"
)

func NewEngine(cfg config.ServiceConfig, logger log.Logger, w io.Writer) *gin.Engine {
	engine := turbogin.NewEngine(cfg, logger, w)
	logPrefix := "[SERVICE: Gin]"
	if err := ginsec.Register(cfg.ExtraConfig, engine); err != nil && err != ginsec.ErrNoConfig {
		logger.Warning(logPrefix+"[HTTPSecure]", err)
	} else if err == nil {
		logger.Debug(logPrefix + "[HTTPSecure] Successfully loaded module")
	}

	lua.Register(logger, cfg.ExtraConfig, engine)

	gindec.Register(cfg, logger, engine)

	return engine
}

type engineFactory struct{}

func (e engineFactory) NewEngine(cfg config.ServiceConfig, l log.Logger, w io.Writer) *gin.Engine {
	return NewEngine(cfg, l, w)
}
