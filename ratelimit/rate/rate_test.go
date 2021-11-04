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

package rate

import "testing"

func TestNewMemoryStore(t *testing.T) {
	store := NewMemoryStore(1, 1)
	limiter1 := store("1")
	if !limiter1.Allow() {
		t.Error("The limiter should allow the first call")
	}
	if limiter1.Allow() {
		t.Error("The limiter should block the second call")
	}
	if store("1").Allow() {
		t.Error("The limiter should block the third call")
	}
	if !store("2").Allow() {
		t.Error("The limiter should allow the fourth call because it requests a new limiter")
	}
}
