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

package secret

import (
	"context"
	"crypto/rand"
	"testing"
)

func TestNew(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c, err := New(ctx, "base64key://")
	if err != nil {
		t.Error(err)
		return
	}

	plainKey := make([]byte, 32)
	_, _ = rand.Read(plainKey)

	cypherKey, err := c.EncryptKey(ctx, plainKey)
	if err != nil {
		t.Error(err)
		return
	}

	plainText := "asdfghjklñqwertyuiozxcvbnm,"

	cypherText, err := c.Encrypt(ctx, []byte(plainText), cypherKey)
	if err != nil {
		t.Error(err)
		return
	}

	result, err := c.Decrypt(ctx, cypherText, cypherKey)
	if err != nil {
		t.Error(err)
		return
	}

	if r := string(result); r != plainText {
		t.Errorf("unexpected result: %s", r)
	}
}
