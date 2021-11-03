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

package gin

import (
	"bytes"
	"errors"
	"github.com/alexeyco/binder"
	"github.com/gin-gonic/gin"
	lua "github.com/starvn/sonic/interpreter"
	"github.com/starvn/sonic/interpreter/route"
	"github.com/starvn/turbo/config"
	"github.com/starvn/turbo/log"
	"github.com/starvn/turbo/proxy"
	sgin "github.com/starvn/turbo/route/gin"
	"io/ioutil"
	"net/http"
	"net/url"
)

func Register(l log.Logger, extraConfig config.ExtraConfig, engine *gin.Engine) {
	logPrefix := "[Service: Gin][Lua]"
	cfg, err := lua.Parse(l, extraConfig, route.Namespace)
	if err != nil {
		if err != lua.ErrNoExtraConfig {
			l.Debug(logPrefix, err.Error())
		}
		return
	}

	l.Debug(logPrefix, "Middleware is now ready")

	engine.Use(func(c *gin.Context) {
		if err := process(c, cfg); err != nil {
			_ = c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		c.Next()
	})
}

func HandlerFactory(l log.Logger, next sgin.HandlerFactory) sgin.HandlerFactory {
	return func(remote *config.EndpointConfig, p proxy.Proxy) gin.HandlerFunc {
		logPrefix := "[ENDPOINT: " + remote.Endpoint + "][Lua]"
		handlerFunc := next(remote, p)

		cfg, err := lua.Parse(l, remote.ExtraConfig, route.Namespace)
		if err != nil {
			if err != lua.ErrNoExtraConfig {
				l.Debug(logPrefix, err.Error())
			}
			return handlerFunc
		}

		l.Debug(logPrefix, "Middleware is now ready")

		return func(c *gin.Context) {
			if err := process(c, cfg); err != nil {
				err = lua.ToError(err)
				if errhttp, ok := err.(errHTTP); ok {
					_ = c.AbortWithError(errhttp.StatusCode(), err)
					return
				}
				_ = c.AbortWithError(http.StatusInternalServerError, err)
				return
			}

			handlerFunc(c)
		}
	}
}

type errHTTP interface {
	error
	StatusCode() int
}

func process(c *gin.Context, cfg lua.Config) error {
	b := binder.New(binder.Options{
		SkipOpenLibs:        !cfg.AllowOpenLibs,
		IncludeGoStackTrace: true,
	})

	lua.RegisterErrors(b)
	registerCtxTable(c, b)

	for _, source := range cfg.Sources {
		src, ok := cfg.Get(source)
		if !ok {
			return lua.ErrUnknownSource(source)
		}
		if err := b.DoString(src); err != nil {
			return err
		}
	}

	return b.DoString(cfg.PreCode)
}

func registerCtxTable(c *gin.Context, b *binder.Binder) {
	r := &ginContext{c}

	t := b.Table("ctx")

	t.Static("load", func(c *binder.Context) error {
		c.Push().Data(r, "ctx")
		return nil
	})

	t.Dynamic("method", r.method)
	t.Dynamic("url", r.url)
	t.Dynamic("host", r.host)
	t.Dynamic("query", r.query)
	t.Dynamic("params", r.params)
	t.Dynamic("headers", r.requestHeaders)
	t.Dynamic("body", r.requestBody)
}

type ginContext struct {
	*gin.Context
}

func (r *ginContext) method(c *binder.Context) error {
	req, ok := c.Arg(1).Data().(*ginContext)
	if !ok {
		return errContextExpected
	}

	if c.Top() == 1 {
		c.Push().String(req.Request.Method)
	} else {
		req.Request.Method = c.Arg(2).String()
	}

	return nil
}

func (r *ginContext) url(c *binder.Context) error {
	req, ok := c.Arg(1).Data().(*ginContext)
	if !ok {
		return errContextExpected
	}

	if c.Top() == 1 {
		c.Push().String(req.Request.URL.String())
	} else {
		req.Request.URL, _ = url.Parse(c.Arg(2).String())
	}

	return nil
}

func (r *ginContext) host(c *binder.Context) error {
	req, ok := c.Arg(1).Data().(*ginContext)
	if !ok {
		return errContextExpected
	}

	if c.Top() == 1 {
		c.Push().String(req.Request.Host)
	} else {
		req.Request.Host = c.Arg(2).String()
	}

	return nil
}

func (r *ginContext) query(c *binder.Context) error {
	req, ok := c.Arg(1).Data().(*ginContext)
	if !ok {
		return errContextExpected
	}

	switch c.Top() {
	case 1:
		return errNeedsArguments
	case 2:
		c.Push().String(req.Query(c.Arg(2).String()))
	case 3:
		q := req.Request.URL.Query()
		q.Set(c.Arg(2).String(), c.Arg(3).String())
		req.Request.URL.RawQuery = q.Encode()
	}

	return nil
}

func (r *ginContext) params(c *binder.Context) error {
	req, ok := c.Arg(1).Data().(*ginContext)
	if !ok {
		return errContextExpected
	}
	switch c.Top() {
	case 1:
		return errNeedsArguments
	case 2:
		c.Push().String(req.Params.ByName(c.Arg(2).String()))
	case 3:
		key := c.Arg(2).String()
		for i, p := range req.Params {
			if p.Key == key {
				req.Params[i].Value = c.Arg(3).String()
				return nil
			}
		}
		req.Params = append(req.Params, gin.Param{Key: c.Arg(2).String(), Value: c.Arg(3).String()})
	}

	return nil
}

func (r *ginContext) requestHeaders(c *binder.Context) error {
	req, ok := c.Arg(1).Data().(*ginContext)
	if !ok {
		return errContextExpected
	}
	switch c.Top() {
	case 1:
		return errNeedsArguments
	case 2:
		c.Push().String(req.Request.Header.Get(c.Arg(2).String()))
	case 3:
		req.Request.Header.Set(c.Arg(2).String(), c.Arg(3).String())
	}

	return nil
}

func (r *ginContext) requestBody(c *binder.Context) error {
	req, ok := c.Arg(1).Data().(*ginContext)
	if !ok {
		return errContextExpected
	}

	if c.Top() == 2 {
		req.Request.Body = ioutil.NopCloser(bytes.NewBufferString(c.Arg(2).String()))
		return nil
	}

	var b []byte
	if req.Request.Body != nil {
		b, _ = ioutil.ReadAll(req.Request.Body)
		_ = req.Request.Body.Close()
	}
	req.Request.Body = ioutil.NopCloser(bytes.NewBuffer(b))
	c.Push().String(string(b))

	return nil
}

var (
	errNeedsArguments  = errors.New("need arguments")
	errContextExpected = errors.New("ginContext expected")
)
