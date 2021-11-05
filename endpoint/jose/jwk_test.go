//go:generate go run $GOROOT/src/crypto/tls/generate_cert.go --rsa-bits 1024 --host 127.0.0.1,::1,localhost --ca --start-date "Jan 1 00:00:00 1970" --duration=1000000h

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

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"github.com/starvn/sonic/endpoint/jose/secret"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
)

func TestJWK(t *testing.T) {
	cert, err := tls.LoadX509KeyPair("cert.pem", "key.pem")
	if err != nil {
		t.Error(err)
		return
	}

	for _, tc := range []struct {
		Name string
		Alg  string
		ID   []string
	}{
		{
			Name: "public",
			ID:   []string{"2011-04-29"},
			Alg:  "RS256",
		},
		{
			Name: "public",
			ID:   []string{"1"},
		},
		{
			Name: "private",
			ID:   []string{"2011-04-29"},
			Alg:  "RS256",
		},
		{
			Name: "private",
			ID:   []string{"1"},
		},
		{
			Name: "symmetric",
			ID:   []string{"sim2"},
			Alg:  "HS256",
		},
	} {
		server := httptest.NewUnstartedServer(jwkEndpoint(tc.Name))
		server.TLS = &tls.Config{Certificates: []tls.Certificate{cert}}
		server.StartTLS()

		secretProvider, err := SecretProvider(SecretProviderConfig{URI: server.URL, LocalCA: "cert.pem"}, nil)
		if err != nil {
			t.Error(err)
		}
		for _, k := range tc.ID {
			key, err := secretProvider.GetKey(k)
			if err != nil {
				t.Errorf("[%s] extracting the key %s: %s", tc.Name, k, err.Error())
			}
			if key.Algorithm != tc.Alg {
				t.Errorf("wrong alg. have: %s, want: %s", key.Algorithm, tc.Alg)
			}
		}
		server.Close()
	}
}

func TestJWK_file(t *testing.T) {
	for _, tc := range []struct {
		Name string
		Alg  string
		ID   string
	}{
		{
			Name: "public",
			ID:   "2011-04-29",
			Alg:  "RS256",
		},
		{
			Name: "public",
			ID:   "1",
		},
		{
			Name: "private",
			ID:   "2011-04-29",
			Alg:  "RS256",
		},
		{
			Name: "private",
			ID:   "1",
		},
		{
			Name: "symmetric",
			ID:   "sim2",
			Alg:  "HS256",
		},
	} {
		secretProvider, err := SecretProvider(
			SecretProviderConfig{
				URI:           "",
				AllowInsecure: true,
				LocalPath:     "./fixture/" + tc.Name + ".json",
			},
			nil,
		)
		if err != nil {
			t.Error(err)
		}
		key, err := secretProvider.GetKey(tc.ID)
		if err != nil {
			t.Errorf("[%s] extracting the key %s: %s", tc.Name, tc.ID, err.Error())
		}
		if key.Algorithm != tc.Alg {
			t.Errorf("wrong alg. have: %s, want: %s", key.Algorithm, tc.Alg)
		}
	}
}

func TestJWK_cyperfile(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	url := "base64key://smGbjm71Nxd1Ig5FS0wj9SlbzAIrnolCz9bQQ6uAhl4="

	cypher, err := secret.New(ctx, url)
	if err != nil {
		t.Error(err)
		return
	}
	defer cypher.Close()

	plainKey := make([]byte, 32)
	_, _ = rand.Read(plainKey)

	cypherKey, err := cypher.EncryptKey(ctx, plainKey)
	if err != nil {
		t.Error(err)
		return
	}

	b, _ := ioutil.ReadFile("./fixture/private.json")
	cypherText, err := cypher.Encrypt(ctx, b, cypherKey)
	if err != nil {
		t.Error(err)
		return
	}
	_ = ioutil.WriteFile("./fixture/private.txt", cypherText, 0666)
	defer func() {
		_ = os.Remove("./fixture/private.txt")
	}()

	for k, tc := range []struct {
		Alg string
		ID  string
	}{
		{
			ID:  "2011-04-29",
			Alg: "RS256",
		},
		{
			ID: "1",
		},
	} {
		secretProvider, err := SecretProvider(
			SecretProviderConfig{
				URI:           "",
				AllowInsecure: true,
				LocalPath:     "./fixture/private.txt",
				CipherKey:     cypherKey,
				SecretURL:     url,
			},
			nil,
		)
		if err != nil {
			t.Error(err)
		}
		key, err := secretProvider.GetKey(tc.ID)
		if err != nil {
			t.Errorf("[%d] extracting the key %s: %s", k, tc.ID, err.Error())
		}
		if key.Algorithm != tc.Alg {
			t.Errorf("wrong alg. have: %s, want: %s", key.Algorithm, tc.Alg)
		}
	}
}

func TestJWK_cache(t *testing.T) {
	cert, err := tls.LoadX509KeyPair("cert.pem", "key.pem")
	if err != nil {
		t.Error(err)
		return
	}

	for _, tc := range []struct {
		Name string
		Alg  string
		ID   []string
	}{
		{
			Name: "public",
			ID:   []string{"2011-04-29"},
			Alg:  "RS256",
		},
		{
			Name: "public",
			ID:   []string{"1"},
		},
		{
			Name: "private",
			ID:   []string{"2011-04-29"},
			Alg:  "RS256",
		},
		{
			Name: "private",
			ID:   []string{"1"},
		},
		{
			Name: "symmetric",
			ID:   []string{"sim2"},
			Alg:  "HS256",
		},
	} {
		var hits uint32
		server := httptest.NewUnstartedServer(jwkEndpointWithCounter(tc.Name, &hits))
		server.TLS = &tls.Config{Certificates: []tls.Certificate{cert}}
		server.StartTLS()

		cfg := SecretProviderConfig{
			URI:          server.URL,
			LocalCA:      "cert.pem",
			CacheEnabled: true,
		}

		secretProvider, err := SecretProvider(cfg, nil)
		if err != nil {
			t.Error(err)
		}

		if hits != 1 {
			t.Errorf("wrong initial number of hits to the jwk endpoint: %d", hits)
		}

		for i := 0; i < 10; i++ {
			for _, k := range tc.ID {
				key, err := secretProvider.GetKey(k)
				if err != nil {
					t.Errorf("[%s] extracting the key %s: %s", tc.Name, k, err.Error())
				}
				if key.Algorithm != tc.Alg {
					t.Errorf("wrong alg. have: %s, want: %s", key.Algorithm, tc.Alg)
				}
			}
		}
		server.Close()

		if hits != 1 {
			t.Errorf("wrong number of hits to the jwk endpoint: %d", hits)
		}
	}
}

func TestDialer_DialTLS_ko(t *testing.T) {
	d := NewDialer(SecretProviderConfig{})
	c, err := d.DialTLS("\t", "addr")
	if err == nil {
		t.Error(err)
	}
	if c != nil {
		t.Errorf("unexpected connection: %v", c)
	}
}

func Test_decodeFingerprints(t *testing.T) {
	_, err := DecodeFingerprints([]string{"not_encoded_message"})
	if err == nil {
		t.Error(err)
	}
}

func TestNewFileKeyCacher(t *testing.T) {
	for _, tc := range []struct {
		Name string
		Alg  string
		ID   string
	}{
		{
			Name: "public",
			ID:   "2011-04-29",
			Alg:  "RS256",
		},
		{
			Name: "public",
			ID:   "1",
		},
		{
			Name: "private",
			ID:   "2011-04-29",
			Alg:  "RS256",
		},
		{
			Name: "private",
			ID:   "1",
		},
		{
			Name: "symmetric",
			ID:   "sim2",
			Alg:  "HS256",
		},
	} {
		b, err := ioutil.ReadFile("./fixture/" + tc.Name + ".json")
		if err != nil {
			t.Error(err)
		}
		kc, err := NewFileKeyCacher(b, "")
		if err != nil {
			t.Error(err)
		}
		if _, err := kc.Get(tc.ID); err != nil {
			t.Error(err)
		}
	}
}

func TestNewFileKeyCacher_unknownKey(t *testing.T) {
	b, err := ioutil.ReadFile("./fixture/symmetric.json")
	if err != nil {
		t.Error(err)
	}
	kc, err := NewFileKeyCacher(b, "")
	if err != nil {
		t.Error(err)
	}
	v, err := kc.Get("unknown")
	if err == nil {
		t.Error("error expected")
	} else if e := err.Error(); e != "key 'unknown' not found in the key set" {
		t.Error("unexpected error:", e)
	}
	if v != nil {
		t.Error("nil value expected")
	}
}

func jwkEndpoint(name string) http.HandlerFunc {
	data, err := ioutil.ReadFile("./fixture/" + name + ".json")
	return func(rw http.ResponseWriter, _ *http.Request) {
		if err != nil {
			rw.WriteHeader(500)
			return
		}
		rw.Header().Set("Content-Type", "application/json")
		_, _ = rw.Write(data)
	}
}

func jwkEndpointWithCounter(name string, hits *uint32) http.HandlerFunc {
	data, err := ioutil.ReadFile("./fixture/" + name + ".json")
	return func(rw http.ResponseWriter, _ *http.Request) {
		if err != nil {
			rw.WriteHeader(500)
			return
		}
		rw.Header().Set("Content-Type", "application/json")
		_, _ = rw.Write(data)
		atomic.AddUint32(hits, 1)
	}
}
