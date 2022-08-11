New build script(s) for Hugo. Very much a work in progress.


## Development of this project

This project contains some example plugins as sub modules, which makes it convenient to have a `go.work` setup (at least in VS Code).

But we don't want that setup in the production build, so the `go.work*` files are currently in `.gitignore`. Copy the `go.work.dev` file to `go.work` and you should be good. And I agree, this setup leaves something to be desired.

