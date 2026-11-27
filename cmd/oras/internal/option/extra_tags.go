/*
Copyright The ORAS Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package option

import (
	"context"
	"errors"
	"fmt"
	"strings"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/oras-project/oras-go/v3"
	"github.com/oras-project/oras-go/v3/registry/remote/properties"
	"oras.land/oras/internal/repository"
)

func validateTag(tag string) error {
	ref := properties.Reference{Tag: tag}
	return ref.ValidateTag()
}

// ValidateOCILayoutTag validates an OCI layout tag.
func ValidateOCILayoutTag(tag string) error {
	return validateTag(tag)
}

// ValidateExtraTags validates additional destination tags.
func ValidateExtraTags(tags []string) error {
	for _, tag := range tags {
		if tag == "" {
			return errors.New("empty extra tag")
		}

		if err := validateTag(tag); err != nil {
			return fmt.Errorf("invalid extra tag: %w", err)
		}
	}
	return nil
}

// extraTagPrefix returns the repository prefix ExtraTagTarget would apply to
// additional tags, and whether that prefixing applies. Prefixing happens only
// for a repository-qualified reference with --oci-layout-path, not for
// --oci-layout or a bare --oci-layout-path reference.
func (target *Target) extraTagPrefix() (string, bool) {
	if target.Type != TargetTypeOCILayout || target.IsOCILayout {
		return "", false
	}
	ref, err := repository.ParseReference(target.Reference)
	if err != nil || ref.Repository == "" {
		return "", false
	}

	// Layout references are stored verbatim. Preserve the repository spelling
	// from the primary reference, including any registry port.
	name, _, _ := strings.Cut(target.Reference, "@")
	name = strings.TrimSuffix(name, ":"+ref.Tag)
	return name + ":", true
}

// ScopesExtraTags reports whether ExtraTagTarget would prefix additional tags
// with the destination repository.
func (target *Target) ScopesExtraTags() bool {
	_, ok := target.extraTagPrefix()
	return ok
}

// ExtraTagTarget scopes validated additional tags to the destination repository
// when a repository-qualified reference is used with --oci-layout-path.
func (target *Target) ExtraTagTarget(dst oras.Target) oras.Target {
	prefix, ok := target.extraTagPrefix()
	if !ok {
		return dst
	}
	return &extraTagTarget{Target: dst, prefix: prefix}
}

type extraTagTarget struct {
	oras.Target
	prefix string
}

func (target *extraTagTarget) Tag(ctx context.Context, desc ocispec.Descriptor, tag string) error {
	return target.Target.Tag(ctx, desc, target.prefix+tag)
}
