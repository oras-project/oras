# Advanced Push

This documentation explains how advanced push operations can be done with `oras push` CLI.

## Pushing without Tagging

Pushing by digest is the default behavior if a tag is not provided.

```
$ oras push localhost:5000/test hello.txt
Uploading a948904f2f0f hello.txt
Pushed localhost:5000/test
Digest: sha256:c38fe4b80a6c5c23b211365408bdb8deeda5132cd802c988fb4cd0b972ccfb9f
$ rm hello.txt
$ oras pull localhost:5000/test@sha256:c38fe4b80a6c5c23b211365408bdb8deeda5132cd802c988fb4cd0b972ccfb9f
Downloaded a948904f2f0f hello.txt
Pulled localhost:5000/test@sha256:c38fe4b80a6c5c23b211365408bdb8deeda5132cd802c988fb4cd0b972ccfb9f
Digest: sha256:c38fe4b80a6c5c23b211365408bdb8deeda5132cd802c988fb4cd0b972ccfb9f
```

## Dry Run

It is possible to push artifacts in `dry-run` mode by specifying `--dry-run` option.

```
$ oras push localhost:5000/test:latest hello.txt --dry-run
Entered dry-run mode
Uploading a948904f2f0f hello.txt
Pushed localhost:5000/test:latest
Digest: sha256:c38fe4b80a6c5c23b211365408bdb8deeda5132cd802c988fb4cd0b972ccfb9f
```

In `dry-run` mode, everything is computed except sending anything to the actual remote server.

## Export Manifest

The pushed manifest is useful in some scenarios such as signing.
To export the manifest pushed to the remote, a target file name is required to be specified by the `--export-manifest` option in order to save the exported manifest.

For example:

```
$ oras push localhost:5000/test:latest hello.txt --export-manifest manifest.json
Uploading a948904f2f0f hello.txt
Pushed localhost:5000/test:latest
Digest: sha256:c38fe4b80a6c5c23b211365408bdb8deeda5132cd802c988fb4cd0b972ccfb9f
```

After that,  `cat manifest.json | jq .` outputs

```json
{
  "schemaVersion": 2,
  "config": {
    "mediaType": "application/vnd.unknown.config.v1+json",
    "digest": "sha256:44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a",
    "size": 2
  },
  "layers": [
    {
      "mediaType": "application/vnd.oci.image.layer.v1.tar",
      "digest": "sha256:a948904f2f0f479b8f8197694b30184b0d2ed1c1cd2a1ec0fb85d299a192a447",
      "size": 12,
      "annotations": {
        "org.opencontainers.image.title": "hello.txt"
      }
    }
  ]
}
```

Combined with the dry-run mode, it is possible to export the manifest without pushing to the remote.

```
$ oras push localhost:5000/test:latest hello.txt --export-manifest exported-manifest.json --dry-run
Entered dry-run mode
Uploading a948904f2f0f hello.txt
Pushed localhost:5000/test:latest
Digest: sha256:c38fe4b80a6c5c23b211365408bdb8deeda5132cd802c988fb4cd0b972ccfb9f
```

