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

package detector

import (
	"net/http"
	"testing"
)

func BenchmarkDetector(b *testing.B) {
	d, err := New(Config{
		DenyList:  []string{"a", "b"},
		AllowList: []string{"c", "Pingdom.com_bot_version_1.1"},
		Patterns: []string{
			`(Pingdom.com_bot_version_)(\d+)\.(\d+)`,
			`(facebookexternalhit)/(\d+)\.(\d+)`,
		},
	})
	if err != nil {
		b.Error(err)
		return
	}

	benchDetection(b, d)
}

func BenchmarkLRUDetector(b *testing.B) {
	d, err := New(Config{
		DenyList:  []string{"a", "b"},
		AllowList: []string{"c", "Pingdom.com_bot_version_1.1"},
		Patterns: []string{
			`(Pingdom.com_bot_version_)(\d+)\.(\d+)`,
			`(facebookexternalhit)/(\d+)\.(\d+)`,
		},
		CacheSize: 10000,
	})
	if err != nil {
		b.Error(err)
		return
	}

	benchDetection(b, d)
}

func benchDetection(b *testing.B, f DetectorFunc) {
	for _, tc := range []struct {
		name string
		ua   string
	}{
		{"ok_1", "abcd"},
		{"ok_2", ""},
		{"ok_3", "c"},
		{"ok_4", "Pingdom.com_bot_version_1.1"},
		{"ko_1", "a"},
		{"ko_2", "b"},
		{"ko_3", "facebookexternalhit/1.1"},
		{"ko_4", "Pingdom.com_bot_version_1.2"},
	} {

		req, _ := http.NewRequest("GET", "http://example.com", nil)
		req.Header.Add("User-Agent", tc.ua)
		b.Run(tc.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				f(req)
			}
		})
	}
}
