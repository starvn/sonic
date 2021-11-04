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

// Package xml provides XML encoding support
package xml

import (
	"github.com/clbanning/mxj"
	"github.com/starvn/turbo/encoding"
	"golang.org/x/net/html/charset"
	"io"
)

func Register() error {
	return encoding.GetRegister().Register(Name, NewDecoder)
}

const Name = "xml"

func NewDecoder(isCollection bool) func(io.Reader, *map[string]interface{}) error {
	if isCollection {
		return CollectionDecoder
	}
	return Decoder
}

func Decoder(r io.Reader, v *map[string]interface{}) error {
	mxj.XmlCharsetReader = charset.NewReaderLabel
	mv, err := mxj.NewMapXmlReader(xmlReader{r: r})
	if err != nil {
		return err
	}
	*v = mv
	return nil

}

func CollectionDecoder(r io.Reader, v *map[string]interface{}) error {
	mxj.XmlCharsetReader = charset.NewReaderLabel
	mv, err := mxj.NewMapXmlReader(xmlReader{r: r})
	if err != nil {
		return err
	}
	*(v) = map[string]interface{}{"collection": mv}
	return nil
}

type xmlReader struct {
	r io.Reader
}

func (x xmlReader) Read(p []byte) (n int, err error) {
	n, err = x.r.Read(p)

	if err != io.EOF {
		return n, err
	}

	if len(p) == n {
		return n, nil
	}

	p[n] = ([]byte("\n"))[0]
	return n + 1, err
}
