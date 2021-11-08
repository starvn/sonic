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
	"context"
	"github.com/starvn/turbo/config"
	"github.com/starvn/turbo/discovery/dns"
	"github.com/starvn/turbo/log"
)

func RegisterSubscriberFactories(ctx context.Context, cfg config.ServiceConfig, logger log.Logger) func(n string, p int) {
	_ = dns.Register()

	return func(name string, port int) {}
}

type registerSubscriberFactories struct{}

func (d registerSubscriberFactories) Register(ctx context.Context, cfg config.ServiceConfig, logger log.Logger) func(n string, p int) {
	return RegisterSubscriberFactories(ctx, cfg, logger)
}
