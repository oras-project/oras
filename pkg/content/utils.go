package content

import ocispec "github.com/opencontainers/image-spec/specs-go/v1"

func resolveName(desc ocispec.Descriptor) (string, bool) {
	name, ok := desc.Annotations[ocispec.AnnotationTitle]
	return name, ok
}
