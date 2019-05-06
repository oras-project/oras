# Manifest Annotations

[Annotations](<https://github.com/opencontainers/image-spec/blob/master/annotations.md>), which are supported by [OCI Image Manifest](<https://github.com/opencontainers/image-spec/blob/master/manifest.md#image-manifest>) and [OCI Content Descriptors](<https://github.com/opencontainers/image-spec/blob/master/descriptor.md>), are also supported by `oras`.

## Make Annotations

Making annotations are supported by both the command line tool and the Go package. However, reading the annotations from the remote is limited and only possible with the Go package.

### Command Line Tool

Users can make annotations to the manifest, the config, and individual files (i.e. layers) by the `--manifest-annotations file` option. The annotations file is a JSON file with the following format:

```json
{
  "<filename>": {
    "<annotation_key>": "annotation_value"
  }
}
```

There are two special filenames / entries:
- `$config` is reserved for the annotation of the manifest config.
- `$manifest` is reserved for the annotation of the manifest itself.

For instance, the following annotation file `annotations.json`:
```json
{
  "$config": {
    "hello": "world"
  },
  "$manifest": {
    "foo": "bar"
  },
  "cake.txt": {
    "fun": "more cream"
  }
}
```
Running the following command

```sh
oras push --manifest-annotations annotations.json localhost:5000/club:party cake.txt juice.txt
```

results in

```json
{
  "schemaVersion": 2,
  "config": {
    "mediaType": "application/vnd.oci.image.config.v1+json",
    "digest": "sha256:44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a",
    "size": 2,
    "annotations": {
      "hello": "world"
    }
  },
  "layers": [
    {
      "mediaType": "application/vnd.oci.image.layer.v1.tar",
      "digest": "sha256:22af0898315a239117308d39acd80636326c4987510b0ec6848e58eb584ba82e",
      "size": 6,
      "annotations": {
        "fun": "more cream",
        "org.opencontainers.image.title": "cake.txt"
      }
    },
    {
      "mediaType": "application/vnd.oci.image.layer.v1.tar",
      "digest": "sha256:be6fe11876282442bead98e8b24aca07f8972a763cd366c56b4b5f7bcdd23eac",
      "size": 7,
      "annotations": {
        "org.opencontainers.image.title": "juice.txt"
      }
    }
  ],
  "annotations": {
    "foo": "bar"
  }
}
```

### Go Package

Making annotations in Go is as simple as modifying the `Annotations` field of the [Descriptor](<https://godoc.org/github.com/opencontainers/image-spec/specs-go/v1#Descriptor>) struct objects before passing them to [oras.Push()](https://godoc.org/github.com/deislabs/oras/pkg/oras#Push) with or without the option [oras.WithConfig()](<https://godoc.org/github.com/deislabs/oras/pkg/oras#WithConfig>).

The caller can pass the push option [oras.WithConfigAnnotations()](<https://godoc.org/github.com/deislabs/oras/pkg/oras#WithConfigAnnotations>) to make annotations to the default config. Similarly, the caller can pass the push option [oras.WithManifestAnnotations()](<https://godoc.org/github.com/deislabs/oras/pkg/oras#WithManifestAnnotations>) to make annotations to the manifest.

#### Retrieve Annotations

Retrieving the annotations of individual layers is as simple as reading the `Annotations` field of the [Descriptor](<https://godoc.org/github.com/opencontainers/image-spec/specs-go/v1#Descriptor>) slice returned by [oras.Pull()](https://godoc.org/github.com/deislabs/oras/pkg/oras#Pull).

For example:

```go
_, files, err := oras.Pull(ctx, resolver, ref, store)
if err != nil {
    panic(err)
}
for _, file := range files {
    fmt.Println(file.Annotations)
}
```

Retrieving the annotations of the manifest and/or the config is currently not supported.
