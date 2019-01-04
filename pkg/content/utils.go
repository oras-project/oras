package content

import ocispec "github.com/opencontainers/image-spec/specs-go/v1"

// ResolveName resolves name from descriptor
func ResolveName(desc ocispec.Descriptor) (string, bool) {
	name, ok := desc.Annotations[ocispec.AnnotationTitle]
	return name, ok
}
