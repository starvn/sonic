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
	"fmt"
)

func Example() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c, err := New(ctx, "base64key://smGbjm71Nxd1Ig5FS0wj9SlbzAIrnolCz9bQQ6uAhl4=")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer c.Close()

	plainKey := make([]byte, 32)
	_, _ = rand.Read(plainKey)

	cypherKey, err := c.EncryptKey(ctx, plainKey)
	if err != nil {
		fmt.Println(err)
		return
	}

	plainText := "asdfghjklñqwertyuiozxcvbnm,"

	cypherText, err := c.Encrypt(ctx, []byte(plainText), cypherKey)
	if err != nil {
		fmt.Println(err)
		return
	}

	result, err := c.Decrypt(ctx, cypherText, cypherKey)
	if err != nil {
		fmt.Println(err)
		return
	}

	if r := string(result); r != plainText {
		fmt.Printf("unexpected result: %s", r)
	}

	// output:
}

func ExampleEncrypt() {
	msg := "zxcvbnmasdfghjklqwertyuiop1234567890"
	passphrase := "some secret"

	cypherMsg, err := Encrypt([]byte(msg), []byte(passphrase))
	if err != nil {
		fmt.Println(err)
		return
	}

	cypherMsg2, err2 := Encrypt([]byte(msg), []byte(passphrase))
	if err2 != nil {
		fmt.Println(err2)
		return
	}

	if string(cypherMsg) == string(cypherMsg2) {
		fmt.Println("two executions with the same input shall not generate the same output")
	}

	// output:
}

func ExampleDecrypt() {
	msg := "zxcvbnmasdfghjklqwertyuiop1234567890"
	passphrase := "some secret"

	cypherMsg1, err := Encrypt([]byte(msg), []byte(passphrase))
	if err != nil {
		fmt.Println(err)
		return
	}

	cypherMsg2, err2 := Encrypt([]byte(msg), []byte(passphrase))
	if err2 != nil {
		fmt.Println(err2)
		return
	}

	if string(cypherMsg1) == string(cypherMsg2) {
		fmt.Println("two executions with the same input shall not generate the same output")
		return
	}

	res1, err3 := Decrypt(cypherMsg1, []byte(passphrase))
	if err != nil {
		fmt.Println(err3)
		return
	}

	res2, err4 := Decrypt(cypherMsg2, []byte(passphrase))
	if err != nil {
		fmt.Println(err4)
		return
	}

	if string(res1) != string(res2) {
		fmt.Println("results are different:", string(res1), string(res2))
		return
	}

	// output:
}
