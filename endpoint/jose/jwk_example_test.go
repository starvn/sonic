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

package jose

import "fmt"

func Example_Auth0Integration() {
	fs, _ := DecodeFingerprints([]string{"--MBgDH5WGvL9Bcn5Be30cRcL0f5O-NyoXuWtQdX1aI="})
	cfg := SecretProviderConfig{
		URI:          "https://albert-test.auth0.com/.well-known/jwks.json",
		Fingerprints: fs,
	}
	client := SecretProvider(cfg, nil)

	k, err := client.GetKey("MDNGMjU2M0U3RERFQUEwOUUzQUMwQ0NBN0Y1RUY0OEIxNTRDM0IxMw")
	fmt.Println("err:", err)
	fmt.Println("is public:", k.IsPublic())
	fmt.Println("alg:", k.Algorithm)
	fmt.Println("id:", k.KeyID)
	// Output:
	// err: <nil>
	// is public: true
	// alg: RS256
	// id: MDNGMjU2M0U3RERFQUEwOUUzQUMwQ0NBN0Y1RUY0OEIxNTRDM0IxMw
}

func Example_Auth0Integration_badFingerprint() {
	cfg := SecretProviderConfig{
		URI:          "https://albert-test.auth0.com/.well-known/jwks.json",
		Fingerprints: [][]byte{make([]byte, 32)},
	}
	client := SecretProvider(cfg, nil)

	_, err := client.GetKey("MDNGMjU2M0U3RERFQUEwOUUzQUMwQ0NBN0Y1RUY0OEIxNTRDM0IxMw")
	fmt.Println("err:", err)
	// Output:
	// err: Get https://albert-test.auth0.com/.well-known/jwks.json: JWK client did not find a pinned key
}
