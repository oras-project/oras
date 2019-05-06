# Content Store

The [oras.content](<https://godoc.org/github.com/deislabs/oras/pkg/content>) package provides the following two major content stores, which implement the [Ingester](<https://godoc.org/github.com/containerd/containerd/content#Ingester>) and the [Provider](<https://godoc.org/github.com/containerd/containerd/content#Provider>) interfaces, in order to be used in [oras.Pull()](https://godoc.org/github.com/deislabs/oras/pkg/oras#Pull) and [oras.Push()](https://godoc.org/github.com/deislabs/oras/pkg/oras#Push).

- [FileStore](<https://godoc.org/github.com/deislabs/oras/pkg/content#FileStore>) to use the file system as the content store.
- [MemoryStore](<https://godoc.org/github.com/deislabs/oras/pkg/content#Memorystore>) to use the memory as the content store with the purpose of metadata cache or testing.

Most documentations are available at [GoDoc](<https://godoc.org/github.com/deislabs/oras/pkg/content>). In this article, some best practices and advanced usage of this package is documented.

## FileStore

`FileStore` provides contents from the file system. Since its [Add()](<https://godoc.org/github.com/deislabs/oras/pkg/content#FileStore.Add>) method may require temporary file creation (e.g. add a directory other than a regular file), it is a good practice to call [Close()](<https://godoc.org/github.com/deislabs/oras/pkg/content#FileStore.Close>) after use.

```go
store := content.NewFileStore(".")
defer store.Close()
```

### Saving Files to Alternative Paths

By default, the files are saved to the relative path specified by its name (i.e. [AnnotationTitle](<https://godoc.org/github.com/opencontainers/image-spec/specs-go/v1#pkg-constants>), which can be obtained by [ResolveName()](<https://godoc.org/github.com/deislabs/oras/pkg/content#ResolveName>) from a [Descriptor](<https://godoc.org/github.com/opencontainers/image-spec/specs-go/v1#Descriptor>)). For example, a file of name `hi.txt` is saved to `hi.txt`. If the caller knows the file name in advance, it can invoke the thread-safe method [MapPath()](<https://godoc.org/github.com/deislabs/oras/pkg/content#FileStore.MapPath>) to relocate the path to store the file. For example, `MapPath("hi.txt", "hello.txt")` will save the file of name `hi.txt` to `hello.txt`.

It is also possible to relocate the path when pulling with the [oras.WithPullBaseHandler()](<https://godoc.org/github.com/deislabs/oras/pkg/oras#WithPullBaseHandler>) option. For example:

```go
_, _, err = oras.Pull(ctx, resolver, ref, store,
	oras.WithPullBaseHandler(images.HandlerFunc(func(ctx context.Context, desc ocispec.Descriptor) ([]ocispec.Descriptor, error) {
		if desc.MediaType == "the.desired.media.type" {
			name, _ := content.ResolveName(desc)
			store.MapPath(name, "desired/path")
		}
		return nil, nil
	})))
```

## Hybrid Store

[FileStore](<https://godoc.org/github.com/deislabs/oras/pkg/content#FileStore>) and [MemoryStore](<https://godoc.org/github.com/deislabs/oras/pkg/content#Memorystore>) can be combined to create many other advanced stores.

For instance, a layered store with a writable [MemoryStore](<https://godoc.org/github.com/deislabs/oras/pkg/content#Memorystore>) on the top and a read-only [FileStore](<https://godoc.org/github.com/deislabs/oras/pkg/content#FileStore>) at the bottom is handy if the caller wants to push additional content (e.g. custom config or layers) to the remote without tampering the existing file system.

```go
package example

import (
	"context"

	"github.com/containerd/containerd/content"
	orascontent "github.com/deislabs/oras/pkg/content"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// LayeredStore has a writable cache on the top and a provider at the bottom
type LayeredStore struct {
	*orascontent.Memorystore
	provider content.Provider
}

// NewLayeredStore create a new layered store
func NewLayeredStore(provider content.Provider) *LayeredStore {
	return &LayeredStore{
		Memorystore: orascontent.NewMemoryStore(),
		provider:    provider,
	}
}

// ReaderAt reads from the cache first and fallback to the provider
func (s *LayeredStore) ReaderAt(ctx context.Context, desc ocispec.Descriptor) (content.ReaderAt, error) {
	readerAt, err := s.Memorystore.ReaderAt(ctx, desc)
	if err == nil {
		return readerAt, nil
	}
	if s.provider != nil {
		return s.provider.ReaderAt(ctx, desc)
	}
	return nil, err
}
```
