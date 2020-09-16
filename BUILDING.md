# Developer Guide

## Running unit tests

To run all unit tests, run `make test`.

This will make a coverage report available, which can be viewed
in your web browser by running `make covhtml`.

All code under the `pkg/` directory should be thoroughly tested.

## Building binary

Use `make build` to build all platform binaries to the `bin/` directory.

Mac:

```bash
# builds to bin/darwin/amd64/oras
make build-mac
```

Linux:

```bash
# builds to bin/linux/amd64/oras
make build-linux
```

Linux  ARM64:

```bash
# builds to bin/linux/amd64/oras
make build-linux
```


Windows:

```bash
# builds to bin/windows/amd64/oras.exe
make build-windows
```

## Cleaning workspace

To remove all files not manged by git, run `make clean` (be careful!)

## Managing dependencies

[Using Go Modules](https://blog.golang.org/using-go-modules) to manage dependencies.

To update or add new dependencies, run `go get <package name>`.

## Cutting a new release

Example of releasing `v0.1.0`:
```
git tag -a v0.1.0 -m "Release v0.1.0"
git push origin v0.1.0
```

A Codefresh pipeline will pick up the GitHub tag event
and run [.codefresh/release.yml](.codefresh/release.yml).

This will result in running [goreleaser](https://goreleaser.com/)
to upload release artiacts, as well as push a tag to Docker Hub for
the image `orasbot/oras`.
