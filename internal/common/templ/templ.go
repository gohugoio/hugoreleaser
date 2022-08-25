// Copyright 2022 The Hugoreleaser Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package templ

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"text/template"
)

// We add a limited set of useful funcs, mostly string handling, to the Go built-ins.
var BuiltInFuncs = template.FuncMap{
	"upper": func(s string) string {
		return strings.ToUpper(s)
	},
	"lower": func(s string) string {
		return strings.ToLower(s)
	},
	"replace": strings.ReplaceAll,
	"trimPrefix": func(prefix, s string) string {
		return strings.TrimPrefix(s, prefix)
	},
	"trimSuffix": func(suffix, s string) string {
		return strings.TrimSuffix(s, suffix)
	},
}

// Sprintt renders the Go template t with the given data in ctx.
func Sprintt(t string, ctx any) (string, error) {
	tmpl := template.New("").Funcs(BuiltInFuncs)
	var err error
	tmpl, err = tmpl.Parse(t)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, ctx)
	if err != nil {
		return "", fmt.Errorf("error executing template: %v; available fields: %v", err, fieldsFromObject(ctx))
	}
	return buf.String(), nil
}

// MustSprintt is like Sprintt but panics on error.
func MustSprintt(t string, ctx any) string {
	s, err := Sprintt(t, ctx)
	if err != nil {
		panic(err)
	}
	return s
}

func fieldsFromObject(in any) []string {
	var fields []string

	if in == nil {
		return fields
	}

	v := reflect.ValueOf(in)
	if v.Kind() != reflect.Struct {
		return fields
	}
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		fields = append(fields, "."+t.Field(i).Name)
	}
	return fields

}
