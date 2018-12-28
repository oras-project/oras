package main

import (
	"context"
	"fmt"
	"io/ioutil"

	"github.com/containerd/containerd/remotes/docker"
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
	pushContents := make(map[string]oras.Blob)
	pushContents[fileName] = oras.Blob{
		Content: fileContent,
		MediaType: customMediaType,
	}
	fmt.Printf("Pushing %s to %s... ", fileName, ref)
	err := oras.Push(ctx, resolver, ref, pushContents)
	check(err)
	fmt.Println("success!")

	// Pull file(s) from registry and save to disk
	fmt.Printf("Pulling from %s and saving to %s... ", ref, fileName)
	allowedMediaTypes := []string{customMediaType}
	pullContents, err := oras.Pull(ctx, resolver, ref, allowedMediaTypes...)
	check(err)
	err = ioutil.WriteFile(fileName, pullContents[fileName].Content, 0644)
	check(err)
	fmt.Println("success!")
	fmt.Printf("Try running 'cat %s'\n", fileName)
}
