package main

import (
	"context"
	"fmt"

	"github.com/shizhMSFT/oras/pkg/content"

	"github.com/containerd/containerd/remotes/docker"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
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
