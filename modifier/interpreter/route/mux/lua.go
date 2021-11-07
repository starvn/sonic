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

package mux

import (
	"bytes"
	"errors"
	"github.com/alexeyco/binder"
	"github.com/starvn/sonic/modifier/interpreter"
	"github.com/starvn/sonic/modifier/interpreter/route"
	"github.com/starvn/turbo/config"
	"github.com/starvn/turbo/log"
	"github.com/starvn/turbo/proxy"
	"github.com/starvn/turbo/route/mux"
	"io/ioutil"
	"net/http"
	"net/url"
)

func RegisterMiddleware(l log.Logger, e config.ExtraConfig, pe mux.ParamExtractor, mws []mux.HandlerMiddleware) []mux.HandlerMiddleware {
	logPrefix := "[Service: Mux][Lua]"
	cfg, err := interpreter.Parse(l, e, route.Namespace)
	if err != nil {
		if err != interpreter.ErrNoExtraConfig {
			l.Debug(logPrefix, err.Error())
		}
		return mws
	}

	l.Debug(logPrefix, "Middleware is now ready")

	return append(mws, &middleware{pe: pe, cfg: cfg})
}

type middleware struct {
	pe  mux.ParamExtractor
	cfg interpreter.Config
}

func (hm *middleware) Handler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := process(r, hm.pe, hm.cfg); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		h.ServeHTTP(w, r)
	})
}

func HandlerFactory(l log.Logger, next mux.HandlerFactory, pe mux.ParamExtractor) mux.HandlerFactory {
	return func(remote *config.EndpointConfig, p proxy.Proxy) http.HandlerFunc {
		logPrefix := "[ENDPOINT: " + remote.Endpoint + "][Lua]"
		handlerFunc := next(remote, p)

		cfg, err := interpreter.Parse(l, remote.ExtraConfig, route.Namespace)
		if err != nil {
			if err != interpreter.ErrNoExtraConfig {
				l.Debug(logPrefix, err.Error())
			}
			return handlerFunc
		}

		l.Debug(logPrefix, "Middleware is now ready")

		return func(w http.ResponseWriter, r *http.Request) {
			if err := process(r, pe, cfg); err != nil {
				err = interpreter.ToError(err)
				if errhttp, ok := err.(errHTTP); ok {
					http.Error(w, err.Error(), errhttp.StatusCode())
					return
				}

				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			handlerFunc(w, r)
		}
	}
}

type errHTTP interface {
	error
	StatusCode() int
}

func process(r *http.Request, pe mux.ParamExtractor, cfg interpreter.Config) error {
	b := binder.New(binder.Options{
		SkipOpenLibs:        !cfg.AllowOpenLibs,
		IncludeGoStackTrace: true,
	})

	interpreter.RegisterErrors(b)
	registerRequestTable(r, pe, b)

	for _, source := range cfg.Sources {
		src, ok := cfg.Get(source)
		if !ok {
			return interpreter.ErrUnknownSource(source)
		}
		if err := b.DoString(src); err != nil {
			return err
		}
	}

	return b.DoString(cfg.PreCode)
}

func registerRequestTable(r *http.Request, pe mux.ParamExtractor, b *binder.Binder) {
	mctx := &muxContext{
		Request: r,
		pe:      pe,
	}

	t := b.Table("ctx")

	t.Static("load", func(c *binder.Context) error {
		c.Push().Data(mctx, "ctx")
		return nil
	})

	t.Dynamic("method", mctx.method)
	t.Dynamic("url", mctx.url)
	t.Dynamic("query", mctx.query)
	t.Dynamic("params", mctx.params)
	t.Dynamic("headers", mctx.headers)
	t.Dynamic("body", mctx.body)
}

type muxContext struct {
	*http.Request
	pe mux.ParamExtractor
}

func (r *muxContext) method(c *binder.Context) error {
	req, ok := c.Arg(1).Data().(*muxContext)
	if !ok {
		return errContextExpected
	}

	if c.Top() == 1 {
		c.Push().String(req.Method)
	} else {
		req.Method = c.Arg(2).String()
	}

	return nil
}

func (r *muxContext) url(c *binder.Context) error {
	req, ok := c.Arg(1).Data().(*muxContext)
	if !ok {
		return errContextExpected
	}

	if c.Top() == 1 {
		c.Push().String(req.URL.String())
	} else {
		req.URL, _ = url.Parse(c.Arg(2).String())
	}

	return nil
}

func (r *muxContext) query(c *binder.Context) error {
	req, ok := c.Arg(1).Data().(*muxContext)
	if !ok {
		return errContextExpected
	}

	switch c.Top() {
	case 1:
		return errNeedsArguments
	case 2:
		c.Push().String(req.URL.Query().Get(c.Arg(2).String()))
	case 3:
		q := req.URL.Query()
		q.Set(c.Arg(2).String(), c.Arg(3).String())
		req.URL.RawQuery = q.Encode()
	}

	return nil
}

func (r *muxContext) params(c *binder.Context) error {
	// req, ok := c.Arg(1).Data().(*muxContext)
	// if !ok {
	// 	return errContextExpected
	// }
	// switch c.Top() {
	// case 1:
	// 	return errNeedsArguments
	// case 2:
	// 	c.Push().String(req.Params.ByName(c.Arg(2).String()))
	// case 3:
	// 	key := c.Arg(2).String()
	// 	for i, p := range req.Params {
	// 		if p.Key == key {
	// 			req.Params[i].Value = c.Arg(3).String()
	// 			return nil
	// 		}
	// 	}
	// 	req.Params = append(req.Params, gin.Param{Key: c.Arg(2).String(), Value: c.Arg(3).String()})
	// }

	return nil
}

func (r *muxContext) headers(c *binder.Context) error {
	req, ok := c.Arg(1).Data().(*muxContext)
	if !ok {
		return errContextExpected
	}
	switch c.Top() {
	case 1:
		return errNeedsArguments
	case 2:
		c.Push().String(req.Header.Get(c.Arg(2).String()))
	case 3:
		req.Header.Set(c.Arg(2).String(), c.Arg(3).String())
	}

	return nil
}

func (r *muxContext) body(c *binder.Context) error {
	req, ok := c.Arg(1).Data().(*muxContext)
	if !ok {
		return errContextExpected
	}

	if c.Top() == 2 {
		req.Body = ioutil.NopCloser(bytes.NewBufferString(c.Arg(2).String()))
		return nil
	}

	var b []byte
	if req.Body != nil {
		b, _ = ioutil.ReadAll(req.Body)
		_ = req.Body.Close()
	}
	req.Body = ioutil.NopCloser(bytes.NewBuffer(b))
	c.Push().String(string(b))

	return nil
}

var (
	errNeedsArguments  = errors.New("need arguments")
	errContextExpected = errors.New("muxContext expected")
)
