New build script(s) for Hugo. Very much a work in progress.


## Configuration

## Configuration File

## Environment Variables

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

## Development of this project

This project contains some example plugins as sub modules, which makes it convenient to have a `go.work` setup (at least in VS Code).

But we don't want that setup in the production build, so the `go.work*` files are currently in `.gitignore`. Copy the `go.work.dev` file to `go.work` and you should be good. And I agree, this setup leaves something to be desired.

