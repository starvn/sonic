//go:build integration
// +build integration

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

package test

import (
	"testing"
)

func TestNewIntegration(t *testing.T) {
	runner, tcs, err := NewIntegration(nil, nil, nil)
	if err != nil {
		t.Error(err)
		return
	}
	defer runner.Close()

	for _, tc := range tcs {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			if err := runner.Check(tc); err != nil {
				t.Error(err)
			}
		})
	}
}
