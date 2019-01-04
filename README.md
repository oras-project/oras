# OCI Registry As Storage

[![Codefresh build status](https://g.codefresh.io/api/badges/pipeline/shizh/shizhMSFT%2Foras%2Fmaster?type=cf-1)](https://g.codefresh.io/public/accounts/shizh/pipelines/shizhMSFT/oras/master)
[![Go Report Card](https://goreportcard.com/badge/github.com/shizhMSFT/oras)](https://goreportcard.com/report/github.com/shizhMSFT/oras)
[![GoDoc](https://godoc.org/github.com/shizhMSFT/oras?status.svg)](https://godoc.org/github.com/shizhMSFT/oras)

`oras` can push/pull any files to/from any registry with OCI image support.

Registries with known support:

- [Distribution](https://github.com/docker/distribution) (open source, version 2.7+)
- [Azure Container Registry](https://azure.microsoft.com/en-us/services/container-registry/)

For more background on this topic, please see
[this post](https://www.opencontainers.org/blog/2018/10/11/oci-image-support-comes-to-open-source-docker-registry).

## CLI

`oras` is a CLI that allows you to push and pull files from
any registry with OCI image support.

### Pushing files to remote registry
```
oras push localhost:5000/hello:latest hi.txt
```

The default media type for all files is `application/vnd.oci.image.layer.v1.tar`.

The push a custom media type, use the format `filename[:type]`:
```
oras push localhost:5000/hello:latest hi.txt:application/vnd.me.hi
```

To push multiple files with different media types:
```
oras push localhost:5000/hello:latest hi.txt:application/vnd.me.hi bye.txt:application/vnd.me.bye
```

### Pulling files from remote registry
```
oras pull localhost:5000/hello:latest
```

By default, only blobs with media type `application/vnd.oci.image.layer.v1.tar` will be downloaded.

To specify which media types to download, use the `--media-type`/`-t` flag:
```
oras pull localhost:5000/hello:latest -t application/vnd.me.hi
```

Or to allow all media types, use the `--allow-all`/`-a` flag:
```
oras pull localhost:5000/hello:latest -a
```

### Login Credentials
`oras` uses the local docker credential by default. Please run `docker login` in advance for any private registries.

`oras` also accepts explicit credentials via options. For example,
```
oras pull -u username -p password myregistry.io/myimage:latest
```

### Run in Docker

Public image is available on [Docker Hub](https://hub.docker.com/r/ocistorage/oras) at `ocistorage/oras`

#### Run on Mac/Linux
```
docker run --rm -it -v $(pwd):/workplace ocistorage/oras:v0.2.0 \
  pull myregistry.io/hello:latest
```

#### Run on Windows PowerShell
```
docker run --rm -it -v ${pwd}:/workplace ocistorage/oras:v0.2.0 \
  pull myregistry.io/hello:latest
```

#### Run on Windows Commands
```
docker run --rm -it -v %cd%:/workplace ocistorage/oras:v0.2.0 \
  pull myregistry.io/hello:latest
```

### Install the binary

Install from latest release (v0.2.0):

```
# on Linux
curl -LO https://github.com/shizhMSFT/oras/releases/download/v0.2.0/oras_0.2.0_linux_amd64.tar.gz

# on macOS
curl -LO https://github.com/shizhMSFT/oras/releases/download/v0.2.0/oras_0.2.0_darwin_amd64.tar.gz

# on Windows
curl -LO https://github.com/shizhMSFT/oras/releases/download/v0.2.0/oras_0.2.0_windows_amd64.tar.gz

mkdir -p oras/
tar -zxf oras_0.2.0_*.tar.gz -C oras/
mv oras/bin/oras /usr/local/bin/
rm -rf oras_0.2.0_*.tar.gz oras/
```

Then, to run:

```
oras help
```

The checksums for the `.tar.gz` files above can be found [here](https://github.com/shizhMSFT/oras/releases/tag/v0.2.0).


## Go Module

The package `github.com/shizhMSFT/oras/pkg/oras` can quickly be imported in other Go-based tools that
wish to benefit from the ability to store arbitrary content in container registries.

Example:

[Source](examples/simple_push_pull.go)

```go
package main

import (
	"context"
	"fmt"

	"github.com/containerd/containerd/remotes/docker"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/shizhMSFT/oras/pkg/content"
	"github.com/shizhMSFT/oras/pkg/oras"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func main() {
	ref := "localhost:5000/oras:test"
	fileName := "hello.txt"
	fileContent := []byte("Hello World!\n")
	customMediaType := "my.custom.media.type"

	ctx := context.Background()
	resolver := docker.NewResolver(docker.ResolverOptions{})

	// Push file(s) w custom mediatype to registry
	memoryStore := content.NewMemoryStore()
	desc := memoryStore.Add(fileName, customMediaType, fileContent)
	pushContents := []ocispec.Descriptor{desc}
	fmt.Printf("Pushing %s to %s... ", fileName, ref)
	err := oras.Push(ctx, resolver, ref, memoryStore, pushContents)
	check(err)
	fmt.Println("success!")

	// Pull file(s) from registry and save to disk
	fmt.Printf("Pulling from %s and saving to %s... ", ref, fileName)
	fileStore := content.NewFileStore("")
	allowedMediaTypes := []string{customMediaType}
	_, err = oras.Pull(ctx, resolver, ref, fileStore, allowedMediaTypes...)
	check(err)
	fmt.Println("success!")
	fmt.Printf("Try running 'cat %s'\n", fileName)
}
```
