# Developer Guide

## Cutting a new release

Use [goreleaser](https://goreleaser.com/):

Example of releasing `v0.1.0`:
```
git tag -a v0.1.0 -m "Release v0.1.0"
git push origin v0.1.0
goreleaser
```

