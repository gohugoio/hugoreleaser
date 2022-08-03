package templ

import (
	"bytes"
	"strings"
	"text/template"
)

// We add a limited set of useful funcs, mostly string handling, to the Go built-ins.
var builtInFuncs = template.FuncMap{
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
// It (currently) panics on errors.
func Sprintt(t string, ctx any) string {
	tmpl := template.New("").Funcs(builtInFuncs)
	var err error
	tmpl, err = tmpl.Parse(t)
	if err != nil {
		panic(err)
	}
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, ctx)
	if err != nil {
		panic(err)
	}
	return buf.String()
}
