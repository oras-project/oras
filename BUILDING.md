# Developer Guide

## Running unit tests

To run all unit tests, run `make test`.

This will make a coverage report available, which can be viewed
in your web browser by running `make covhtml`.

All code under the `pkg/` directory should be thoroughly tested.

## Building binary

Use `make test` to build all platofrm binaries to the `bin/` directory.

Mac:

```
# builds to bin/darwin/amd64/oras
make build-mac
```

Linux:

```
# builds to bin/linux/amd64/oras
make build-linux
```

Windows:

```
# builds to bin/windows/amd64/oras.exe
make build-windows
```

## Cleaning workspace

To remove all files not manged by git, run `make clean` (be careful!)

## Adding/updating dependencies

Requires [dep](https://golang.github.io/dep/).

Add new dependencies directly to `Gopkg.toml` and run `dep ensure`.

To update all dependencies, run `make update-deps`.

After doing any action above, please make sure to run `make fix-deps`,
which will fix some issue with how dependencies are resolved.

Please check in all changes in the `vendor/` directory.

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
