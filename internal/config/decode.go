// Copyright 2026 The Hugoreleaser Authors
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

package config

import (
	"bufio"
	"io"
	"os"
	"reflect"
	"strings"

	"github.com/goccy/go-yaml"

	"github.com/bep/helpers/envhelpers"
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

	s = envhelpers.Expand(s, func(k string) string {
		return os.Getenv(k)
	})

	d := yaml.NewDecoder(strings.NewReader(s),
		yaml.DisallowUnknownField(),
	)

	err = d.Decode(cfg)
	if err != nil {
		return *cfg, err
	}

	// Apply defaults.
	if cfg.GoSettings.GoExe == "" {
		cfg.GoSettings.GoExe = "go"
	}

	if cfg.GoSettings.GoProxy == "" {
		cfg.GoSettings.GoProxy = "https://proxy.golang.org"
	}

	// Merge build settings.
	// We may have build settings on any of Project > Build > Goos > Goarch.
	// Note that this uses the replaces any zero value as defined by IsTruthfulValue (a Hugo construct)m
	// meaning any value on the right will be used if the left is zero according to that definition.
	shallowMerge(&cfg.BuildSettings.GoSettings, cfg.GoSettings)
	for i := range cfg.Builds {
		shallowMerge(&cfg.Builds[i].BuildSettings, cfg.BuildSettings)
		shallowMerge(&cfg.Builds[i].BuildSettings.GoSettings, cfg.BuildSettings.GoSettings)

		for j := range cfg.Builds[i].Os {
			shallowMerge(&cfg.Builds[i].Os[j].BuildSettings, cfg.Builds[i].BuildSettings)
			shallowMerge(&cfg.Builds[i].Os[j].BuildSettings.GoSettings, cfg.Builds[i].BuildSettings.GoSettings)

			for k := range cfg.Builds[i].Os[j].Archs {
				shallowMerge(&cfg.Builds[i].Os[j].Archs[k].BuildSettings, cfg.Builds[i].Os[j].BuildSettings)
				shallowMerge(&cfg.Builds[i].Os[j].Archs[k].BuildSettings.GoSettings, cfg.Builds[i].Os[j].BuildSettings.GoSettings)

			}
		}
	}

	// Merge archive settings.
	// We may have archive settings on all of Project > Archive.
	for i := range cfg.Archives {
		shallowMerge(&cfg.Archives[i].ArchiveSettings, cfg.ArchiveSettings)
	}

	// Merge release settings.
	// We may have release settings on all of Project > Release.
	for i := range cfg.Releases {
		shallowMerge(&cfg.Releases[i].ReleaseSettings, cfg.ReleaseSettings)
		shallowMerge(&cfg.Releases[i].ReleaseSettings.ReleaseNotesSettings, cfg.ReleaseSettings.ReleaseNotesSettings)
	}

	// Init and validate build settings.
	for i := range cfg.Builds {
		if err := cfg.Builds[i].Init(); err != nil {
			return *cfg, err
		}
	}

	// Init and validate archive configs.
	for i := range cfg.Archives {
		if err := cfg.Archives[i].Init(); err != nil {
			return *cfg, err
		}
	}

	// Init and validate release configs.
	for i := range cfg.Releases {
		if err := cfg.Releases[i].Init(); err != nil {
			return *cfg, err
		}
	}

	// Init and validate publish settings.
	if err := cfg.PublishSettings.Init(); err != nil {
		return *cfg, err
	}

	// Init and validate publisher configs.
	for i := range cfg.Publishers {
		if err := cfg.Publishers[i].Init(); err != nil {
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

// LoadEnvFile loads environment variables from text file on the form key=value.
// It ignores empty lines and lines starting with # and lines without an equals sign.
func LoadEnvFile(filename string) (map[string]string, error) {
	fi, err := os.Stat(filename)
	if err != nil || fi.IsDir() {
		return nil, nil
	}

	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	env := make(map[string]string)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		env[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return env, scanner.Err()
}

type zeroer interface {
	IsZero() bool
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
