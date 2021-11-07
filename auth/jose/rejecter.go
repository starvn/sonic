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
	"github.com/starvn/turbo/config"
	"github.com/starvn/turbo/log"
)

type Rejecter interface {
	Reject(map[string]interface{}) bool
}

type RejecterFunc func(map[string]interface{}) bool

func (r RejecterFunc) Reject(v map[string]interface{}) bool { return r(v) }

type FixedRejecter bool

func (f FixedRejecter) Reject(_ map[string]interface{}) bool { return bool(f) }

type RejecterFactory interface {
	New(log.Logger, *config.EndpointConfig) Rejecter
}

type RejecterFactoryFunc func(log.Logger, *config.EndpointConfig) Rejecter

func (f RejecterFactoryFunc) New(l log.Logger, cfg *config.EndpointConfig) Rejecter {
	return f(l, cfg)
}

type NopRejecterFactory struct{}

func (NopRejecterFactory) New(_ log.Logger, _ *config.EndpointConfig) Rejecter {
	return FixedRejecter(false)
}

type ChainedRejecterFactory []RejecterFactory

func (c ChainedRejecterFactory) New(l log.Logger, cfg *config.EndpointConfig) Rejecter {
	rejecters := []Rejecter{}
	for _, rf := range c {
		rejecters = append(rejecters, rf.New(l, cfg))
	}
	return RejecterFunc(func(v map[string]interface{}) bool {
		for _, r := range rejecters {
			if r.Reject(v) {
				return true
			}
		}
		return false
	})
}
