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
	"fmt"
	"net/http"
	"testing"
)

func TestNew_noLRU(t *testing.T) {
	d, err := New(Config{
		DenyList:  []string{"a", "b"},
		AllowList: []string{"c", "Pingdom.com_bot_version_1.1"},
		Patterns: []string{
			`(Pingdom.com_bot_version_)(\d+)\.(\d+)`,
			`(facebookexternalhit)/(\d+)\.(\d+)`,
		},
	})
	if err != nil {
		t.Error(err)
		return
	}

	if err := testDetection(d); err != nil {
		t.Error(err)
	}
}

func TestNew_LRU(t *testing.T) {
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
		t.Error(err)
		return
	}

	if err := testDetection(d); err != nil {
		t.Error(err)
	}
}

func testDetection(f DetectorFunc) error {
	for i, ua := range []string{
		"abcd",
		"",
		"c",
		"Pingdom.com_bot_version_1.1",
	} {
		req, _ := http.NewRequest("GET", "http://example.com", nil)
		req.Header.Add("User-Agent", ua)
		if f(req) {
			return fmt.Errorf("the req #%d has been detected as a bot: %s", i, ua)
		}
	}

	for i, ua := range []string{
		"a",
		"b",
		"facebookexternalhit/1.1",
		"Pingdom.com_bot_version_1.2",
	} {
		req, _ := http.NewRequest("GET", "http://example.com", nil)
		req.Header.Add("User-Agent", ua)
		if !f(req) {
			return fmt.Errorf("the req #%d has not been detected as a bot: %s", i, ua)
		}
	}
	return nil
}
