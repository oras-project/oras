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
Reference: sha256:9ac59685f09eee88a1294e6872d13ba83fe44b934ae5992645ef5952d590d29e
```

The verbose output with the `--verbose` option is available as:

```json
Discovered 1 artifacts referencing localhost:5000/test:latest
Digest: sha256:c38fe4b80a6c5c23b211365408bdb8deeda5132cd802c988fb4cd0b972ccfb9f
Reference: sha256:9ac59685f09eee88a1294e6872d13ba83fe44b934ae5992645ef5952d590d29e
{
  "schemaVersion": 2,
  "mediaType": "application/vnd.oci.artifact.manifest.v1+json",
  "artifactType": "application/vnd.cncf.notary.v2",
  "blobs": [
    {
      "mediaType": "application/vnd.oci.image.layer.v1.tar",
      "digest": "sha256:556ec1d63c2153fc9645c6af2d87b0ebe66d356457d95d7cce4617aa9f9ec27f",
      "size": 2373,
      "annotations": {
        "org.opencontainers.image.title": "hello.jwt"
      }
    }
  ],
  "manifests": [
    {
      "mediaType": "application/vnd.oci.image.manifest.v1+json",
      "digest": "sha256:c38fe4b80a6c5c23b211365408bdb8deeda5132cd802c988fb4cd0b972ccfb9f",
      "size": 395
    }
  ]
}
```

Since the JSON output in the verbose mode is human-friendly but not user-friendly, `oras` also provides a `--output-json` option to output in JSON format:

```json
{
  "digest": "sha256:c38fe4b80a6c5c23b211365408bdb8deeda5132cd802c988fb4cd0b972ccfb9f",
  "links": [
    {
      "digest": "sha256:9ac59685f09eee88a1294e6872d13ba83fe44b934ae5992645ef5952d590d29e",
      "manifest": {
        "schemaVersion": 2,
        "mediaType": "application/vnd.oci.artifact.manifest.v1+json",
        "artifactType": "application/vnd.cncf.notary.v2",
        "blobs": [
          {
            "mediaType": "application/vnd.oci.image.layer.v1.tar",
            "digest": "sha256:556ec1d63c2153fc9645c6af2d87b0ebe66d356457d95d7cce4617aa9f9ec27f",
            "size": 2373,
            "annotations": {
              "org.opencontainers.image.title": "hello.jwt"
            }
          }
        ],
        "manifests": [
          {
            "mediaType": "application/vnd.oci.image.manifest.v1+json",
            "digest": "sha256:c38fe4b80a6c5c23b211365408bdb8deeda5132cd802c988fb4cd0b972ccfb9f",
            "size": 395
          }
        ]
      }
    }
  ]
}
```