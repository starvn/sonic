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
	b64 "encoding/base64"
	"errors"
	"gopkg.in/square/go-jose.v2"
	"time"
)

var (
	ErrNoKeyFound    = errors.New("no Keys have been found")
	ErrKeyExpired    = errors.New("key exists but is expired")
	MaxKeyAgeNoCheck = time.Duration(-1)
)

type KeyIDGetter interface {
	Get(*jose.JSONWebKey) string
}

type KeyIDGetterFunc func(*jose.JSONWebKey) string

func (f KeyIDGetterFunc) Get(key *jose.JSONWebKey) string {
	return f(key)
}

func DefaultKeyIDGetter(key *jose.JSONWebKey) string {
	return key.KeyID
}

func X5TKeyIDGetter(key *jose.JSONWebKey) string {
	return b64.RawURLEncoding.EncodeToString(key.CertificateThumbprintSHA1)
}

func CompoundX5TKeyIDGetter(key *jose.JSONWebKey) string {
	return key.KeyID + X5TKeyIDGetter(key)
}

func KeyIDGetterFactory(keyIdentifyStrategy string) KeyIDGetter {

	var supportedKeyIdentifyStrategy = map[string]KeyIDGetterFunc{
		"kid":     DefaultKeyIDGetter,
		"x5t":     X5TKeyIDGetter,
		"kid_x5t": CompoundX5TKeyIDGetter,
	}

	if keyGetter, ok := supportedKeyIdentifyStrategy[keyIdentifyStrategy]; ok {
		return keyGetter
	}
	return KeyIDGetterFunc(DefaultKeyIDGetter)
}

type KeyCacher interface {
	Get(keyID string) (*jose.JSONWebKey, error)
	Add(keyID string, webKeys []jose.JSONWebKey) (*jose.JSONWebKey, error)
}

type MemoryKeyCacher struct {
	entries      map[string]keyCacherEntry
	maxKeyAge    time.Duration
	maxCacheSize int
	keyIDGetter  KeyIDGetter
}

type keyCacherEntry struct {
	addedAt time.Time
	jose.JSONWebKey
}

func NewMemoryKeyCacher(maxKeyAge time.Duration, maxCacheSize int, keyIdentifyStrategy string) KeyCacher {
	return &MemoryKeyCacher{
		entries:      map[string]keyCacherEntry{},
		maxKeyAge:    maxKeyAge,
		maxCacheSize: maxCacheSize,
		keyIDGetter:  KeyIDGetterFactory(keyIdentifyStrategy),
	}
}

func (mkc *MemoryKeyCacher) Get(keyID string) (*jose.JSONWebKey, error) {
	searchKey, ok := mkc.entries[keyID]
	if ok {
		if mkc.maxKeyAge == MaxKeyAgeNoCheck || !mkc.keyIsExpired(keyID) {
			return &searchKey.JSONWebKey, nil
		}
		return nil, ErrKeyExpired
	}
	return nil, ErrNoKeyFound
}

func (mkc *MemoryKeyCacher) Add(keyID string, downloadedKeys []jose.JSONWebKey) (*jose.JSONWebKey, error) {

	var addingKey jose.JSONWebKey
	var addingKeyID string
	for _, key := range downloadedKeys {
		cacheKey := mkc.keyIDGetter.Get(&key)
		if cacheKey == keyID {
			addingKey = key
			addingKeyID = cacheKey
		}
		if mkc.maxCacheSize == -1 {
			mkc.entries[cacheKey] = keyCacherEntry{
				addedAt:    time.Now(),
				JSONWebKey: key,
			}
		}
	}
	if addingKey.Key != nil {
		if mkc.maxCacheSize != -1 {
			mkc.entries[addingKeyID] = keyCacherEntry{
				addedAt:    time.Now(),
				JSONWebKey: addingKey,
			}
			mkc.handleOverflow()
		}
		return &addingKey, nil
	}
	return nil, ErrNoKeyFound
}

func (mkc *MemoryKeyCacher) keyIsExpired(keyID string) bool {
	if time.Now().After(mkc.entries[keyID].addedAt.Add(mkc.maxKeyAge)) {
		delete(mkc.entries, keyID)
		return true
	}
	return false
}

func (mkc *MemoryKeyCacher) handleOverflow() {
	if mkc.maxCacheSize < len(mkc.entries) {
		var oldestEntryKeyID string
		var latestAddedTime = time.Now()
		for entryKeyID, entry := range mkc.entries {
			if entry.addedAt.Before(latestAddedTime) {
				latestAddedTime = entry.addedAt
				oldestEntryKeyID = entryKeyID
			}
		}
		delete(mkc.entries, oldestEntryKeyID)
	}
}
