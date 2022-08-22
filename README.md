[![Tests on Linux, MacOS and Windows](https://github.com/gohugoio/hugoreleaser/workflows/Test/badge.svg)](https://github.com/gohugoio/hugoreleaser/actions?query=workflow%3ATest)
[![Go Report Card](https://goreportcard.com/badge/github.com/gohugoio/hugoreleaser)](https://goreportcard.com/report/github.com/gohugoio/hugoreleaser)
[![codecov](https://codecov.io/gh/gohugoio/hugoreleaser/branch/main/graph/badge.svg?token=OWZ9RCAYWO)](https://codecov.io/gh/gohugoio/hugoreleaser)
[![GoDoc](https://godoc.org/github.com/gohugoio/hugoreleaser?status.svg)](https://godoc.org/github.com/gohugoio/hugoreleaser)

New build script(s) for Hugo. Very much a work in progress.

* [Configuration](#configuration)
    * [Configuration File](#configuration-file)
    * [Template Expansion](#template-expansion)
    * [Environment Variables](#environment-variables)
* [Segments](#segments)
* [Plugins](#plugins)
* [Development of this project](#development-of-this-project)
* [Notes](#notes)

## Configuration

### Configuration File


### Template Expansion

Hugoreleaser supports Go template syntax in all fields with suffix `_template` (e.g. `name_template` used to create archive names).

The data received in the template (e.g. the ".") is:

| Field  | Description |
| ------------- | ------------- |
| Project  | The project name as defined in config.  |
| Tag      | The tag as defined by -flag.  |
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

A `hugoreleaser.env`, if found in the current directory, will be parsed and loaded into the environment of the running process. The format is simple, a text files of key-value-pairs on the form `KEY=value`, empty lines and lines starting with `#` is ignored:

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


## Segments

Both the configuration file and the directory structure inside `/dist` follows the same tree structure: 

* For *builds* and *archives*: `/dist/<project>/<tag>/{builds,archives}/<user-defined-path>/<GOOS>/<GOARCH>`
* For `releases`: `/dist/<project>/<tag>/releases/<user-defined-path>`

Given the above, it's possible to split all the release steps (build, archive, release) into segments. A common setup would probably be to split the build step and let the rest run in one go. That was the prime reason behind why Hugoreleaser was initially created; Hugo needed a way to split the build of its extended builds (C/C++, CGO) across multiple Docker containers.

On both `builds` (groups `builds` and `archives`) and `releases` there is a `path` attribute. Slashes are allowed for more fine grained control (e.g. `unix/bsd`).

```toml
[[builds]]
    path = "unix"

    [[builds.os]]
        goos = "freebsd"
        [[builds.os.archs]]
            goarch = "amd64"
        [[builds.os.archs]]
            goarch = "arm64"
        [[builds.os.archs]]
            goarch = "386"
     [[builds.os]]
        goos = "openbsd"
        [[builds.os.archs]]
            goarch = "amd64"
```

In both the configuration file and in the CLI tool there are `paths`, which represents [Glob patterns](https://github.com/gobwas/glob) applied as filters (with double asterisk support):

```toml
[[builds]]
    path = "unix"
[[archives]]
    paths = "/builds/unix/**"
[[releases]]
    paths = "/archives/**/freebsd/{amd64,386}"
    path = "bsd"
```

The matching starts below `<tag>` in the tree structure described above.

And then running each step in sequence:

```bash
hugoreleaser build -build-paths /builds/**/freebsd/amd64
hugoreleaser build -build-paths /builds/**/freebsd/386
hugoreleaser archive -build-paths /builds/**/freebsd/{amd64,386}
hugoreleaser release -release-paths /releases/bsd
```

## Plugins

Hugoreleaser supports [Go Module](https://go.dev/blog/using-go-modules) plugins to create archives. See the [Deb Plugin](https://github.com/gohugoio/hugoreleaser-archive-plugins/tree/main/deb) for an example.

See [Hugoreleaser Plugins API](https://github.com/gohugoio/hugoreleaser-plugins-api) for API and more detailed information.

A plugin is a server (which will be started on Hugoreleaser startup) implemented as a Go `main` func with a structure like the one below:

```go
func main() {
	server, err := plugins.NewServer(
		func(d plugins.Dispatcher, req archiveplugin.Request) archiveplugin.Response {
			d.Infof("Creating archive %s", req.OutFilename)

			if err := req.Init(); err != nil {
				// ...
			}

			if err := createArchive(req); err != nil {
				// ...
			}
			// Empty response is a success.
			return archiveplugin.Response{}
		},
	)
	if err != nil {
		log.Fatalf("Failed to create server: %s", err)
	}

	if err := server.Start(); err != nil {
		log.Fatalf("Failed to start server: %s", err)
	}

	_ = server.Wait()
}
```


## Development of this project

This project contains some example plugins as sub modules, which makes it convenient to have a `go.work` setup (at least in VS Code).

But we don't want that setup in the production build, so the `go.work*` files are currently in `.gitignore`. Copy the `go.work.dev` file to `go.work` and you should be good. And I agree, this setup leaves something to be desired.

## Notes

* Plugins: Vendoring (`go mod vendor`) or not.
* GOMODCACHE