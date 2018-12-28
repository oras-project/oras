# Developer Guide

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
the image `ocistorage/oras`.
