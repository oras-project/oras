# Artifact: Next Generation

`oras` is capable to push, discover, pull the artifact of [the next generation specification](https://github.com/notaryproject/artifacts/blob/prototype-2/specs-go/v2/artifact.go).

## Push

Pushing an artifact with the latest spec requires specifying the artifact type by the `--artifact-type` option.
The `--artifact-reference` option accepts both tags and digests, where digests are suggested.

For example, push a signature artifact `hello.jwt` of `application/vnd.cncf.notary.v2` type linking to another artifact `localhost:5000/test:latest`:

```shell
oras push localhost:5000/test \
    --artifact-type application/vnd.cncf.notary.v2 \
    --artifact-reference localhost:5000/test@sha256:c38fe4b80a6c5c23b211365408bdb8deeda5132cd802c988fb4cd0b972ccfb9f \
    hello.jwt:application/vnd.cncf.notary.signature.v2+jwt
```

The output is

```
Uploading 556ec1d63c21 hello.jwt
Pushed localhost:5000/test
Digest: sha256:1b54b023fbbc174ea78aef6e3644c1f74f5b93e634d7e4bda486456bb3386791
```

## Pull

Pulling an artifact is the same as the regular `oras pull`.

For example, pull a signature artifact:

```shell
oras pull --media-type application/vnd.cncf.notary.signature.v2+jwt \
    localhost:5000/test@sha256:1b54b023fbbc174ea78aef6e3644c1f74f5b93e634d7e4bda486456bb3386791
```

The output is

```
Downloaded 556ec1d63c21 hello.jwt
Pulled localhost:5000/test@sha256:1b54b023fbbc174ea78aef6e3644c1f74f5b93e634d7e4bda486456bb3386791
Digest: sha256:1b54b023fbbc174ea78aef6e3644c1f74f5b93e634d7e4bda486456bb3386791
```

## Discover

Discovering artifacts of a certain type linking with the given artifact can be done with the command `oras discover`.

For example, discover artifacts of type `application/vnd.cncf.notary.v2` linked with `localhost:5000/test:latest` (digests are also accepted):

```
oras discover --artifact-type application/vnd.cncf.notary.v2 \
    localhost:5000/test:latest
```

The output is

```
Discovered 1 artifacts referencing localhost:5000/test:latest
Digest: sha256:c38fe4b80a6c5c23b211365408bdb8deeda5132cd802c988fb4cd0b972ccfb9f
Reference: application/vnd.cncf.notary.v2
- sha256:556ec1d63c2153fc9645c6af2d87b0ebe66d356457d95d7cce4617aa9f9ec27f
```