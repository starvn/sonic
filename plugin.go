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
	"github.com/starvn/turbo/log"
	client "github.com/starvn/turbo/transport/http/client/plugin"
	server "github.com/starvn/turbo/transport/http/server/plugin"
)

func LoadPlugins(folder, pattern string, logger log.Logger) {
	n, err := client.LoadWithLogger(
		folder,
		pattern,
		client.RegisterClient,
		logger,
	)
	if err != nil {
		logger.Warning("loading plugins:", err)
	}
	logger.Info("total http executor plugins loaded:", n)

	n, err = server.LoadWithLogger(
		folder,
		pattern,
		server.RegisterHandler,
		logger,
	)
	if err != nil {
		logger.Warning("loading plugins:", err)
	}
	logger.Info("total http handler plugins loaded:", n)
}

type pluginLoader struct{}

func (d pluginLoader) Load(folder, pattern string, logger log.Logger) {
	LoadPlugins(folder, pattern, logger)
}
