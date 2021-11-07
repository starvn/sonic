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

package gauge

import (
	"github.com/influxdata/influxdb/client/v2"
	"github.com/starvn/turbo/log"
	"time"
)

func Points(hostname string, now time.Time, counters map[string]int64, logger log.Logger) []*client.Point {
	res := make([]*client.Point, 4)

	in := map[string]interface{}{
		"gauge": int(counters["sonic.router.connected-gauge"]),
	}
	incoming, err := client.NewPoint("router", map[string]string{"host": hostname, "direction": "in"}, in, now)
	if err != nil {
		logger.Error("creating incoming connection counters point:", err.Error())
		return res
	}
	res[0] = incoming

	out := map[string]interface{}{
		"gauge": int(counters["sonic.router.disconnected-gauge"]),
	}
	outgoing, err := client.NewPoint("router", map[string]string{"host": hostname, "direction": "out"}, out, now)
	if err != nil {
		logger.Error("creating outgoing connection counters point:", err.Error())
		return res
	}
	res[1] = outgoing

	debug := map[string]interface{}{}
	runtime := map[string]interface{}{}

	for k, v := range counters {
		if k == "sonic.router.connected-gauge" || k == "sonic.router.disconnected-gauge" {
			continue
		}
		if k[:22] == "sonic.service.debug." {
			debug[k[22:]] = int(v)
			continue
		}
		if k[:24] == "sonic.service.runtime." {
			runtime[k[24:]] = int(v)
			continue
		}
		logger.Debug("unknown gauge key:", k)
	}

	debugPoint, err := client.NewPoint("debug", map[string]string{"host": hostname}, debug, now)
	if err != nil {
		logger.Error("creating debug counters point:", err.Error())
		return res
	}
	res[2] = debugPoint

	runtimePoint, err := client.NewPoint("runtime", map[string]string{"host": hostname}, runtime, now)
	if err != nil {
		logger.Error("creating runtime counters point:", err.Error())
		return res
	}
	res[3] = runtimePoint

	return res
}
