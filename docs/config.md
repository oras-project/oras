# Manifest Config

According to [OCI Image Manifest Specification](<https://github.com/opencontainers/image-spec/blob/master/manifest.md#image-manifest-property-descriptions>), the property `config` is required by an image manifest. Since `oras` does not make use of the configuration object, an empty JSON object `{}` is used by default when pushing, and never being fetched when pulling.

The descriptor of the default configuration object is fixed as follows.

```json
{
    "mediaType": "application/vnd.oci.image.config.v1+json",
    "digest": "sha256:44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a",
    "size": 2
}
```

## Customize Config

Projects may leverage `oras` to push their own artifacts to a remote registry. In that case, users may compose their own configuration object with an alternative media type to identify their artifacts. The configuration object may follow the [OCI Image Configuration](<https://github.com/opencontainers/image-spec/blob/master/config.md#properties>) spec.

Config customization is achievable via the command line tool `oras` or the Go package. 

### Command Line Tool

Users can customize the configuration object by the `--manifest-config file[:type]` option. To push a file `hi.txt` with the custom manifest config file `config.json`, run

```sh
oras push --manifest-config config.json localhost:5000/hello:latest hi.txt
```

The media type of the config is set to the default value `application/vnd.oci.image.config.v1+json`. 

Similar to the file reference, it is possible to change the media type of the manifest config. To push a file `hi.txt` with the custom manifest config file  `config.json` with the custom media type `application/vnd.oras.config.v1+json`, run

```sh
oras push --manifest-config config.json:application/vnd.oras.config.v1+json localhost:5000/hello:latest hi.txt
```

In addition, it is possible to pass a null device `/dev/null` (`NUL` on Windows) to `oras` for an empty config file.

```sh
oras push --manifest-config /dev/null:application/vnd.oras.config.v1+json localhost:5000/hello:latest hi.txt
```

### Go Package

Customizing the configuration object in Go is as simple as passing [oras.WithConfig()](<https://godoc.org/github.com/deislabs/oras/pkg/oras#WithConfig>) option to [oras.Push()](https://godoc.org/github.com/deislabs/oras/pkg/oras#Push).

Suppose there is a descriptor `configDesc` referencing the config file in the content provider `store`.

```go
configDesc := ocispec.Descriptor{
    MediaType: mediaType, // config media type
    Digest:    digest,    // sha256 digest of the config file
    Size:      size,      // config file size
}
```

To push with custom config, execute

```go
_, err := oras.Push(ctx, resolver, ref, store, contents, oras.WithConfig(configDesc))
```

If the caller wants to customize the config media type only, pass the [oras.WithConfigMediaType()](<https://godoc.org/github.com/deislabs/oras/pkg/oras#WithConfigMediaType>) option to [oras.Push()](https://godoc.org/github.com/deislabs/oras/pkg/oras#Push).

```go
_, err := oras.Push(ctx, resolver, ref, store, contents,
                    oras.WithConfigMediaType("application/vnd.oras.config.v1+json"))
```

## Docker Behaviors

The config used by `oras` is not a real config. Therefore, the pushed image cannot be recognized or pulled by `docker` as expected. In this section, docker behaviors are shown given various configs.

### Empty Config File

```
$ oras push --manifest-config /dev/null localhost:5000/hello:latest hi.txt
Uploading a948904f2f0f hi.txt
Pushed localhost:5000/hello:latest
Digest: sha256:733074abcfb2bcaac28893bb05e7b6783ae25722407bd742d728dbb330e98683
$ docker pull localhost:5000/hello:latest
latest: Pulling from hello
a948904f2f0f: Extracting [==================================================>]      12B/12B
unexpected end of JSON input
```

### Empty JSON Object

```
$ cat config.json
{}
$ oras push --manifest-config config.json localhost:5000/hello:latest hi.txt
Uploading a948904f2f0f hi.txt
Pushed localhost:5000/hello:latest
Digest: sha256:f04b8a748bcecbf326503687cfd8ff6709669e73a120df9d4592c56ed193d128
$ docker pull localhost:5000/hello:latest
latest: Pulling from hello
a948904f2f0f: Pulling fs layer
invalid rootfs in image configuration
```

### Arbitrary OS

```
$ cat config.json
{
    "architecture": "cloud",
    "os": "oras"
}
$ oras push --manifest-config config.json localhost:5000/hello:latest hi.txt
Uploading a948904f2f0f hi.txt
Pushed localhost:5000/hello:latest
Digest: sha256:4732e48d7c9ccbc096292070c143c92dd19a163c93abccea7f1fa7517ef70a22
$ docker pull localhost:5000/hello:latest
latest: Pulling from hello
a948904f2f0f: Extracting [==================================================>]      12B/12B
operating system is not supported
```

### Arbitrary Config Media Type

```
$ oras push --manifest-config /dev/null:application/vnd.oras.config.v1+json localhost:5000/hello:latest hi.txt
Uploading a948904f2f0f hi.txt
Pushed localhost:5000/hello:latest
Digest: sha256:5d8ea018049870aab566350660b9a003c646a7f955f9996d35cc0c71bf41b3d0
$ docker pull localhost:5000/hello:latest
Error response from daemon: Encountered remote "application/vnd.oras.config.v1+json"(unknown) when fetching
```

Note: Layers are not pulled in this case. Thus **it is encouraged to specify customized config media type**.
