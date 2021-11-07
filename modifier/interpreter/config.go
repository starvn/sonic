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

// Package interpreter provides a lua interpreter for the Sonic API Gateway
package interpreter

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"github.com/starvn/turbo/config"
	"github.com/starvn/turbo/log"
	"io"
	"io/ioutil"
)

type Config struct {
	Sources       []string
	PreCode       string
	PostCode      string
	SkipNext      bool
	AllowOpenLibs bool
	SourceLoader  SourceLoader
}

func (c *Config) Get(k string) (string, bool) {
	return c.SourceLoader.Get(k)
}

type SourceLoader interface {
	Get(string) (string, bool)
}

func Parse(l log.Logger, e config.ExtraConfig, namespace string) (Config, error) {
	res := Config{}
	v, ok := e[namespace]
	if !ok {
		return res, ErrNoExtraConfig
	}
	c, ok := v.(map[string]interface{})
	if !ok {
		return res, ErrWrongExtraConfig
	}
	if pre, ok := c["pre"].(string); ok {
		res.PreCode = pre
	}
	if post, ok := c["post"].(string); ok {
		res.PostCode = post
	}
	if b, ok := c["skip_next"].(bool); ok && b {
		res.SkipNext = b
	}
	if b, ok := c["allow_open_libs"].(bool); ok && b {
		res.AllowOpenLibs = b
	}

	sources, ok := c["sources"].([]interface{})
	if ok {
		var s []string
		for _, source := range sources {
			if t, ok := source.(string); ok {
				s = append(s, t)
			}
		}
		res.Sources = s
	}

	if b, ok := c["live"].(bool); ok && b {
		res.SourceLoader = new(liveLoader)
		return res, nil
	}

	loader := map[string]string{}

	for _, source := range res.Sources {
		b, err := ioutil.ReadFile(source)
		if err != nil {
			l.Error("[Lua] Opening the source file:", err.Error())
			continue
		}
		loader[source] = string(b)
	}
	res.SourceLoader = onceLoader(loader)

	checksums, ok := c["md5"].(map[string]interface{})
	if !ok {
		return res, nil
	}

	for source, c := range checksums {
		checksum, ok := c.(string)
		if !ok {
			return res, ErrWrongChecksumType(source)
		}
		content, _ := res.SourceLoader.Get(source)
		hash := md5.New()
		if _, err := io.Copy(hash, bytes.NewBuffer([]byte(content))); err != nil {
			return res, err
		}
		hashInBytes := hash.Sum(nil)[:16]
		if actual := hex.EncodeToString(hashInBytes); checksum != actual {
			return res, ErrWrongChecksum{
				Source:   source,
				Actual:   actual,
				Expected: checksum,
			}
		}
	}

	return res, nil
}

type onceLoader map[string]string

func (o onceLoader) Get(k string) (string, bool) {
	v, ok := o[k]
	return v, ok
}

type liveLoader struct{}

func (l *liveLoader) Get(k string) (string, bool) {
	b, err := ioutil.ReadFile(k)
	if err != nil {
		return "", false
	}
	return string(b), true
}

var (
	ErrNoExtraConfig    = errors.New("no extra config")
	ErrWrongExtraConfig = errors.New("wrong extra config")
)
