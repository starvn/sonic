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

package expl

import (
	"bytes"
	"fmt"
	"github.com/starvn/sonic/expl/internal"
	"github.com/starvn/turbo/config"
	"github.com/starvn/turbo/log"
	"testing"
	"time"
)

func TestRejecter_Reject(t *testing.T) {
	timeNow = func() time.Time {
		loc, _ := time.LoadLocation("UTC")
		return time.Date(2018, 12, 10, 0, 0, 0, 0, loc)
	}
	defer func() { timeNow = time.Now }()

	buff := bytes.NewBuffer(make([]byte, 1024))
	logger, err := log.NewLogger("DEBUG", buff, "pref")
	if err != nil {
		t.Error("building the logger:", err.Error())
		return
	}

	rejecter := NewRejecter(logger, &config.EndpointConfig{
		Endpoint: "/",
		ExtraConfig: config.ExtraConfig{
			internal.Namespace: []internal.InterpretableDefinition{
				{CheckExpression: "has(JWT.user_id) && has(JWT.enabled_days) && (timestamp(now).getDayOfWeek() in JWT.enabled_days)"},
			},
		},
	})

	defer func() {
		fmt.Println(buff.String())
	}()

	if rejecter == nil {
		t.Error("nil rejecter")
		return
	}

	for _, tc := range []struct {
		data     map[string]interface{}
		expected bool
	}{
		{
			data:     map[string]interface{}{},
			expected: true,
		},
		{
			data: map[string]interface{}{
				"user_id": 1,
			},
			expected: true,
		},
		{
			data: map[string]interface{}{
				"user_id":      1,
				"enabled_days": []int{},
			},
			expected: true,
		},
		{
			data: map[string]interface{}{
				"user_id":      1,
				"enabled_days": []int{1, 2, 3, 4, 5},
			},
			expected: false,
		},
	} {
		if res := rejecter.Reject(tc.data); res != tc.expected {
			t.Errorf("%+v => unexpected response %v", tc.data, res)
		}
	}
}
