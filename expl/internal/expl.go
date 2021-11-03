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

package internal

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	"github.com/starvn/turbo/config"
	"github.com/starvn/turbo/log"
	"strings"
)

type InterpretableDefinition struct {
	CheckExpression string `json:"check_expr"`
	ModExpression   string `json:"mod_expr"`
}

func ConfigGetter(e config.ExtraConfig) ([]InterpretableDefinition, bool) {
	var def []InterpretableDefinition
	v, ok := e[Namespace]
	if !ok {
		return def, ok
	}
	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(&v); err != nil {
		return def, false
	}

	if err := json.NewDecoder(buf).Decode(&def); err != nil {
		return def, false
	}
	return def, true
}

const Namespace = "github.com/starvn/sonic/expl"

var (
	ErrParsing  = errors.New("cel: error parsing the expression")
	ErrChecking = errors.New("cel: error checking the expression and its param definition")
	ErrNoExpr   = errors.New("cel: no expression")
)

type ErrorChecking struct {
	details error
}

func (e ErrorChecking) Error() string {
	return fmt.Sprintf("error parsing the expression %s", e.details.Error())
}

func NewCheckExpressionParser(l log.Logger) Parser {
	return Parser{
		extractor: extractCheckExpr,
		l:         l,
	}
}

func NewModExpressionParser(l log.Logger) Parser {
	return Parser{
		extractor: extractModExpr,
		l:         l,
	}
}

type Parser struct {
	extractor func(InterpretableDefinition) string
	l         log.Logger
}

func (p Parser) Parse(definition InterpretableDefinition) (cel.Program, error) {
	expr := p.extractor(definition)
	if expr == "" {
		return nil, ErrNoExpr
	}
	p.l.Debug("[CEL]", fmt.Sprintf("Parsing expression: %v", expr))
	env, err := cel.NewEnv(defaultDeclarations())
	if err != nil {
		return nil, err
	}

	ast, iss := env.Parse(p.extractor(definition))
	if iss != nil && iss.Err() != nil {
		return nil, fmt.Errorf("error parsing the expression %s", iss.Err())
	}
	c, iss := env.Check(ast)
	if iss != nil && iss.Err() != nil {
		return nil, ErrorChecking{details: iss.Err()}
	}

	return env.Program(c)
}

func (p Parser) ParsePre(definitions []InterpretableDefinition) ([]cel.Program, error) {
	return p.parseByKey(definitions, PreKey)
}

func (p Parser) ParsePost(definitions []InterpretableDefinition) ([]cel.Program, error) {
	return p.parseByKey(definitions, PostKey)
}

func (p Parser) ParseJWT(definitions []InterpretableDefinition) ([]cel.Program, error) {
	return p.parseByKey(definitions, JwtKey)
}

func (p Parser) parseByKey(definitions []InterpretableDefinition, key string) ([]cel.Program, error) {
	var res []cel.Program
	for _, def := range definitions {
		if !strings.Contains(p.extractor(def), key) {
			continue
		}
		v, err := p.Parse(def)
		if _, ok := err.(ErrorChecking); ok {
			p.l.Debug("[CEL]", err.Error())
			continue
		}

		if err != nil {
			return res, err
		}
		res = append(res, v)
	}
	return res, nil
}

func defaultDeclarations() cel.EnvOption {
	return cel.Declarations(
		decls.NewConst(NowKey, decls.String, nil),
		decls.NewConst(PreKey+"_method", decls.String, nil),
		decls.NewConst(PreKey+"_path", decls.String, nil),
		decls.NewConst(PreKey+"_params", decls.NewMapType(decls.String, decls.String), nil),
		decls.NewConst(PreKey+"_headers", decls.NewMapType(decls.String, decls.NewListType(decls.String)), nil),
		decls.NewConst(PreKey+"_querystring", decls.NewMapType(decls.String, decls.NewListType(decls.String)), nil),
		decls.NewConst(PostKey+"_completed", decls.Bool, nil),
		decls.NewConst(PostKey+"_metadata_status", decls.Int, nil),
		decls.NewConst(PostKey+"_metadata_headers", decls.NewMapType(decls.String, decls.NewListType(decls.String)), nil),
		decls.NewConst(PostKey+"_data", decls.NewMapType(decls.String, decls.Dyn), nil),
		decls.NewConst(JwtKey, decls.NewMapType(decls.String, decls.Dyn), nil),
	)
}

func extractCheckExpr(i InterpretableDefinition) string { return i.CheckExpression }
func extractModExpr(i InterpretableDefinition) string   { return i.ModExpression }

const (
	PreKey  = "req"
	PostKey = "resp"
	JwtKey  = "JWT"
	NowKey  = "now"
)
