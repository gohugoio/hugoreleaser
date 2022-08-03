package config

import (
	"io"
	"os"
	"reflect"
	"strings"

	"github.com/bep/varexpand"
	"github.com/pelletier/go-toml/v2"
)

var zeroType = reflect.TypeOf((*zeroer)(nil)).Elem()

// DecodeAndApplyDefaults first expand any environment variables in r (${var}),
// decodes it and applies default values.
func DecodeAndApplyDefaults(r io.Reader) (Config, error) {
	cfg := &Config{}

	// Expand environment variables in the source.
	// This is not the most effective, but it's certianly very simple.
	// And the config files should be fairly small.
	var buf strings.Builder
	_, err := io.Copy(&buf, r)
	if err != nil {
		return *cfg, err
	}
	s := buf.String()

	s = varexpand.Expand(s, func(k string) string {
		// TODO(bep) additional env.
		return os.Getenv(k)
	})

	d := toml.NewDecoder(strings.NewReader(s))

	d.DisallowUnknownFields()
	err = d.Decode(cfg)
	if err != nil {
		return *cfg, err
	}

	// Apply defaults.
	if cfg.BuildSettings.GoExe == "" {
		cfg.BuildSettings.GoExe = "go"
	}

	if cfg.BuildSettings.GoProxy == "" {
		cfg.BuildSettings.GoProxy = "https://proxy.golang.org"
	}

	// Merge build settings.
	// We may have build settings on all of project > build > os > arch.
	// Note that this uses the replaces any zero value as defined by IsTruthfulValue (a Hugo construct)m
	// meaning any value on the right will be used if the left is zero according to that definiton.
	for i := range cfg.Builds {
		shallowMerge(&cfg.Builds[i].BuildSettings, cfg.BuildSettings)
		for j := range cfg.Builds[i].Os {
			shallowMerge(&cfg.Builds[i].Os[j].BuildSettings, cfg.Builds[i].BuildSettings)
			for k := range cfg.Builds[i].Os[j].Archs {
				shallowMerge(&cfg.Builds[i].Os[j].Archs[k].BuildSettings, cfg.Builds[i].Os[j].BuildSettings)
			}
		}
	}

	// Merge archive settings.
	// We may have archive settings on all of project > archive.
	for i := range cfg.Archives {
		shallowMerge(&cfg.Archives[i].ArchiveSettings, cfg.ArchiveSettings)
	}

	// Init and alidate archive configs.
	for i := range cfg.Archives {
		if err := cfg.Archives[i].init(); err != nil {
			return *cfg, err
		}
	}

	// Apply some convenient navigation helpers.
	for i := range cfg.Builds {
		for j := range cfg.Builds[i].Os {
			cfg.Builds[i].Os[j].Build = &cfg.Builds[i]
			for k := range cfg.Builds[i].Os[j].Archs {
				cfg.Builds[i].Os[j].Archs[k].Build = &cfg.Builds[i]
				cfg.Builds[i].Os[j].Archs[k].Os = &cfg.Builds[i].Os[j]
			}
		}
	}

	return *cfg, nil
}

// SetEnvVars sets vars on the form key=value in the oldVars slice.
func SetEnvVars(oldVars *[]string, keyValues ...string) {
	for i := 0; i < len(keyValues); i += 2 {
		setEnvVar(oldVars, keyValues[i], keyValues[i+1])
	}
}

func SplitEnvVar(v string) (string, string) {
	name, value, _ := strings.Cut(v, "=")
	return name, value
}

// IsTruthful returns whether in represents a truthful value.
// See IsTruthfulValue
func IsTruthful(in any) bool {
	switch v := in.(type) {
	case reflect.Value:
		return isTruthfulValue(v)
	default:
		return isTruthfulValue(reflect.ValueOf(in))
	}
}

type zeroer interface {
	IsZero() bool
}

func setEnvVar(vars *[]string, key, value string) {
	for i := range *vars {
		if strings.HasPrefix((*vars)[i], key+"=") {
			(*vars)[i] = key + "=" + value
			return
		}
	}
	// New var.
	*vars = append(*vars, key+"="+value)
}

// This is based on "thruthy" function used in the Hugo template system, reused for a slightly different domain.
// The only difference is that this function returns true for empty non-nil slices and maps.
//
// isTruthfulValue returns whether the given value has a meaningful truth value.
// This is based on template.IsTrue in Go's stdlib, but also considers
// IsZero and any interface value will be unwrapped before it's considered
// for truthfulness.
//
// Based on:
// https://github.com/golang/go/blob/178a2c42254166cffed1b25fb1d3c7a5727cada6/src/text/template/exec.go#L306
func isTruthfulValue(val reflect.Value) (truth bool) {
	val = indirectInterface(val)

	if !val.IsValid() {
		// Something like var x interface{}, never set. It's a form of nil.
		return
	}

	if val.Type().Implements(zeroType) {
		return !val.Interface().(zeroer).IsZero()
	}

	switch val.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice:
		return !val.IsNil()
	case reflect.String:
		truth = val.Len() > 0
	case reflect.Bool:
		truth = val.Bool()
	case reflect.Complex64, reflect.Complex128:
		truth = val.Complex() != 0
	case reflect.Chan, reflect.Func, reflect.Ptr, reflect.Interface:
		truth = !val.IsNil()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		truth = val.Int() != 0
	case reflect.Float32, reflect.Float64:
		truth = val.Float() != 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		truth = val.Uint() != 0
	case reflect.Struct:
		truth = true // Struct values are always true.
	default:
		return
	}

	return
}

// Based on: https://github.com/golang/go/blob/178a2c42254166cffed1b25fb1d3c7a5727cada6/src/text/template/exec.go#L931
func indirectInterface(v reflect.Value) reflect.Value {
	if v.Kind() != reflect.Interface {
		return v
	}
	if v.IsNil() {
		return reflect.Value{}
	}
	return v.Elem()
}

func shallowMerge(dst, src any) {
	dstv := reflect.ValueOf(dst)
	if dstv.Kind() != reflect.Ptr {
		panic("dst is not a pointer")
	}

	dstv = reflect.Indirect(dstv)
	srcv := reflect.Indirect(reflect.ValueOf(src))

	for i := 0; i < dstv.NumField(); i++ {
		v := dstv.Field(i)
		if !v.CanSet() {
			continue
		}
		if !isTruthfulValue(v) {
			v.Set(srcv.Field(i))
		}
	}
}
