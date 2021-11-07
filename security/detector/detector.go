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

// Package detector provides a bot detector middleware for the Sonic API Gateway
package detector

import (
	lru "github.com/hashicorp/golang-lru"
	"net/http"
	"regexp"
)

type Config struct {
	DenyList  []string `json:"deny"`
	AllowList []string `json:"allow"`
	Patterns  []string `json:"patterns"`
	CacheSize int      `json:"cache_size"`
}

type DetectorFunc func(r *http.Request) bool

func New(cfg Config) (DetectorFunc, error) {
	if cfg.CacheSize == 0 {
		d, err := NewDetector(cfg)
		return d.IsBot, err
	}

	d, err := NewLRU(cfg)
	return d.IsBot, err
}

func NewDetector(cfg Config) (*Detector, error) {
	deny := make(map[string]struct{}, len(cfg.DenyList))
	for _, e := range cfg.DenyList {
		deny[e] = struct{}{}
	}
	allow := make(map[string]struct{}, len(cfg.AllowList))
	for _, e := range cfg.AllowList {
		allow[e] = struct{}{}
	}
	patterns := make([]*regexp.Regexp, len(cfg.Patterns))
	for i, p := range cfg.Patterns {
		rp, err := regexp.Compile(p)
		if err != nil {
			return nil, err
		}
		patterns[i] = rp
	}
	return &Detector{
		deny:     deny,
		allow:    allow,
		patterns: patterns,
	}, nil
}

type Detector struct {
	deny     map[string]struct{}
	allow    map[string]struct{}
	patterns []*regexp.Regexp
}

func (d *Detector) IsBot(r *http.Request) bool {
	userAgent := r.Header.Get("User-Agent")

	if userAgent == "" {
		return false
	}
	if _, ok := d.allow[userAgent]; ok {
		return false
	}
	if _, ok := d.deny[userAgent]; ok {
		return true
	}
	for _, p := range d.patterns {
		if p.MatchString(userAgent) {
			return true
		}
	}
	return false
}

func NewLRU(cfg Config) (*LRUDetector, error) {
	d, err := NewDetector(cfg)
	if err != nil {
		return nil, err
	}

	cache, err := lru.New(cfg.CacheSize)
	if err != nil {
		return nil, err
	}

	return &LRUDetector{
		detectorFunc: d.IsBot,
		cache:        cache,
	}, nil
}

type LRUDetector struct {
	detectorFunc DetectorFunc
	cache        *lru.Cache
}

func (d *LRUDetector) IsBot(r *http.Request) bool {
	userAgent := r.Header.Get("User-Agent")
	cached, ok := d.cache.Get(userAgent)
	if ok {
		return cached.(bool)
	}

	res := d.detectorFunc(r)
	d.cache.Add(userAgent, res)

	return res
}
