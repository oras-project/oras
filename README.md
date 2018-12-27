# OCI Registry As Storage
`oras` can push/pull any files from/to any registry with OCI image support.

Registries with known support:

- [Distribution](https://github.com/docker/distribution) (open source, version 2.7+)
- [Azure Container Registry](https://azure.microsoft.com/en-us/services/container-registry/)

For more backgound on this topic, please see
[this post](https://www.opencontainers.org/blog/2018/10/11/oci-image-support-comes-to-open-source-docker-registry).

## CLI

`oras` is a CLI that allows you to push and pull files from
any registry with OCI image support.

### Installation

Install from latest release (v0.1.0):

```
# on Linux
curl -LO https://github.com/shizhMSFT/oras/releases/download/v0.1.0/oras_0.1.0_linux_amd64.tar.gz

# on macOS
curl -LO https://github.com/shizhMSFT/oras/releases/download/v0.1.0/oras_0.1.0_darwin_amd64.tar.gz

# on Windows
curl -LO https://github.com/shizhMSFT/oras/releases/download/v0.1.0/oras_0.1.0_windows_amd64.tar.gz

mkdir -p oras/
tar -zxf oras_0.1.0_*.tar.gz -C oras/
mv oras/bin/oras /usr/local/bin/
rm -rf oras_0.1.0_*.tar.gz oras/
```

The checksums for the `.tar.gz` files can be found [here](https://github.com/shizhMSFT/oras/releases/tag/v0.1.0).

### Push files to remote registry
```
oras push localhost:5000/hello:latest hello.txt
```

### Pull files from remote registry
```
oras pull localhost:5000/hello:latest
```

### Login Credentials
`oras` uses the local docker credential by default. Therefore, please run `docker login` in advance for any private registries.

`oras` also accepts explicit credentials via options. For example,
```
oras pull -u username -p password myregistry.io/myimage:latest
```

### Running in Docker
#### Build the image
```
docker build -t oras .
```

#### Run on Linux
```
docker run --rm -it -v $(pwd):/workplace oras pull localhost:5000/hello:latest
```

#### Run on Windows PowerShell
```
docker run --rm -it -v ${pwd}:/workplace oras pull localhost:5000/hello:latest
```

#### Run on Windows Commands
```
docker run --rm -it -v %cd%:/workplace oras pull localhost:5000/hello:latest
```

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
	"github.com/shizhMSFT/oras/pkg/oras"
	"io/ioutil"
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

	ctx := context.Background()
	resolver := docker.NewResolver(docker.ResolverOptions{})

	// Push file(s) to registry
	pushContents := make(map[string][]byte)
	pushContents[fileName] = fileContent
	fmt.Printf("Pushing %s to %s... ", fileName, ref)
	err := oras.Push(ctx, resolver, ref, pushContents)
	check(err)
	fmt.Println("success!")

	// Pull file(s) from registry and save to disk
	fmt.Printf("Pulling from %s and saving to %s... ", ref, fileName)
	pullContents, err := oras.Pull(ctx, resolver, ref)
	check(err)
	err = ioutil.WriteFile(fileName, pullContents[fileName], 0644)
	check(err)
	fmt.Println("success!")
	fmt.Printf("Try running 'cat %s'\n", fileName)
}
```
