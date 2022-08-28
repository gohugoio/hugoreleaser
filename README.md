[![Tests on Linux, MacOS and Windows](https://github.com/gohugoio/hugoreleaser/workflows/Test/badge.svg)](https://github.com/gohugoio/hugoreleaser/actions?query=workflow%3ATest)
[![Go Report Card](https://goreportcard.com/badge/github.com/gohugoio/hugoreleaser)](https://goreportcard.com/report/github.com/gohugoio/hugoreleaser)
[![codecov](https://codecov.io/gh/gohugoio/hugoreleaser/branch/main/graph/badge.svg?token=OWZ9RCAYWO)](https://codecov.io/gh/gohugoio/hugoreleaser)
[![GoDoc](https://godoc.org/github.com/gohugoio/hugoreleaser?status.svg)](https://godoc.org/github.com/gohugoio/hugoreleaser)

* [Configuration](#configuration)
    * [Configuration File](#configuration-file)
    * [Template Expansion](#template-expansion)
    * [Environment Variables](#environment-variables)
* [Glob Matching](#glob-matching)
* [Partitions](#partitions)
    * [Manual Partitioning](#manual-partitioning)
    * [Parallelism](#parallelism)
* [Plugins](#plugins)
* [Release Notes](#release-notes)
* [Why another Go release tool?](#why-another-go-release-tool)

## Configuration

### Configuration File

Hugoreleaser reads its main configuration from a file named `hugoreleaser.toml` in the working directory. See [this project's configuration](./hugoreleaser.toml) for an annotated example.

### Template Expansion

Hugoreleaser supports Go template syntax in all fields with suffix `_template` (e.g. `name_template` used to create archive names).

The data received in the template (e.g. the ".") is:

| Field  | Description |
| ------------- | ------------- |
| Project  | The project name as defined in config.  |
| Tag      | The tag as defined by the -tag flag.  |
| Goos     | The current GOOS.  |
| Goarch   | The current GOARCH.  |

In addition to Go's [built-ins](https://pkg.go.dev/text/template#hdr-Functions), we have added a small number of convenient template funcs:

* `upper`
* `lower`
* `replace` (uses `strings.ReplaceAll`)
* `trimPrefix`
* `trimSuffix`

With that, a name template may look like this:

```toml
name_template = "{{ .Project }}_{{ .Tag | trimPrefix `v` }}_{{ .Goos }}-{{ .Goarch }}"
```

### Environment Variables

The order of presedence for environment variables/flags:

1. Flags (e.g. `-tag`)
2. OS environment variables.
3. Environment variables defined in `hugoreleaser.env`.

A `hugoreleaser.env` file will, if found in the current directory, be parsed and loaded into the environment of the running process. The format is simple, a text files of key-value-pairs on the form `KEY=value`, empty lines and lines starting with `#` is ignored:

Environment variable expressions in `hugoreleaser.toml` on the form `${VAR}` will be expanded before it's parsed.

An example `hugoreleaser.env` with the enviromnent for the next release may look like this:

```
HUGORELEASER_TAG=v1.2.3
HUGORELEASER_COMMITISH=main
MYPROJECT_RELEASE_NAME=First Release!
MYPROJECT_RELEASE_DRAFT=false
```

In the above, the variables prefixed `HUGORELEASER_` will be used to set the flags when running the `hugoreleaser` commands.

The other custom variables can be used in `hugoreleaser.toml`, e.g:

```toml
[release_settings]
    name                           = "${MYPROJECT_RELEASE_NAME}"
    draft                          = "${MYPROJECT_RELEASE_DRAFT@U}"
```

Note the special `@U` (_Unquoute_) syntax. The field `draft` is a boolean and cannot be quouted, but this would create ugly validation errors in TOML aware editors. The construct above signals that the quoutes (single or double) should be removed before doing any variable expansion.

## Glob Matching

Hugo releaser supports the Glob rules as defined in [Gobwas Glob](https://github.com/gobwas/glob) with one additional rule: Glob patterns can be negated with a `!` prefix.

The CLI `-paths` flag is a slice an, if repeated for a given prefix, will be ANDed together, e.g.:

```
hugoreleaser build  -paths "builds/**" -paths "!builds/**/arm64"
```

The above will build everything, expect the ARM64 `GOARCH`.

## Partitions

### Manual Partitioning

The configuration file and the (mimics the directory structure inside `/dist`) creates a simple tree structure that can be used to partition a build/release. All commands takes one or more `-paths` flag values. This is a [Glob Path](#glob-matching) matching builds to cover or releases to release (the latter is only relevant for the last step). Hugo has partitioned its builds using a container name as the first path element. With that, releasing may look something like this:

```
# Run this in container1
hugoreleaser build --paths "builds/container1/**"
# Run this in container2, using the same /dist as the first step.
hugoreleaser build --paths "builds/container2/**"
hugoreleaser archive
hugoreleaser release
```

### Parallelism

The build command takes the optional `-chunks` and `-chunk-index` which could be used to automatically split the builds to speed up pipelines., e.g. using [Circle CI's Job Splitting](https://circleci.com/docs/parallelism-faster-jobs#using-environment-variables-to-split-tests).

See [Hugo v0.102.0 Release Notes](https://github.com/gohugoio/hugo/releases/tag/v0.102.0) for more information.

## Plugins

Hugoreleaser supports [Go Module](https://go.dev/blog/using-go-modules) plugins to create archives. See the [Deb Plugin](https://github.com/gohugoio/hugoreleaser-archive-plugins/tree/main/deb) for an example.

See the [Hugoreleaser Plugins API](https://github.com/gohugoio/hugoreleaser-plugins-api) for API and more information.

## Release Notes

The config map `release_notes_settings` has 3 options for how to handle release notes:

1. Set a `filename`
2. Set `generate_on_host=true` and let GitHub do it.
3. Set `generate=true` and let Hugoreleaser do it.

There are more details about change grouping etc. in this [this project's configuration](./hugoreleaser.toml).

For the third option, you can set a custom release notes template to use in `template_filename`. See the default template in [staticfiles/templates/release-notes.gotmpl](./staticfiles/templates/release-notes.gotmpl) for an example.

## Why another Go release tool?

If you need a Go build/release tool with all the bells and whistles, check out [GoReleaser](https://github.com/goreleaser/goreleaser). This project was created because [Hugo](https://github.com/gohugoio/hugo) needed some features not on the road map of that project. 

Hugo used this tool for its [v0.102.0 Release](https://github.com/gohugoio/hugo/releases/tag/v0.102.0). If you also want to use this tool and have a complex setup, you may want to wait for a conclusion to [Issue #27](https://github.com/gohugoio/hugoreleaser/issues/27) first.
