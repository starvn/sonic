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

package influxdb

import (
	"github.com/influxdata/influxdb/client/v2"
)

func NewBuffer(size int) *Buffer {
	return &Buffer{
		data: []client.BatchPoints{},
		size: size,
	}
}

type Buffer struct {
	data []client.BatchPoints
	size int
}

func (b *Buffer) Add(ps ...client.BatchPoints) {
	b.data = append(b.data, ps...)
	if len(b.data) > b.size {
		b.data = b.data[len(b.data)-b.size:]
	}
}

func (b *Buffer) Elements() []client.BatchPoints {
	var res []client.BatchPoints
	res, b.data = b.data, []client.BatchPoints{}
	return res
}
