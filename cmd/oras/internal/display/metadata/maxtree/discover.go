package maxtree

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

import (
	"fmt"
	"io"
	"strings"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras/cmd/oras/internal/display/metadata"
	"oras.land/oras/internal/tree"
)

// discoverHandler handles json metadata output for discover events.
type discoverHandler struct {
	out     io.Writer
	path    string
	root    *tree.Node
	nodes   map[digest.Digest]*tree.Node
	verbose bool
}

// NewDiscoverHandler creates a new handler for discover events.
func NewDiscoverHandler(out io.Writer, path string, root ocispec.Descriptor, verbose bool) metadata.DiscoverHandler {
	treeRoot := tree.New(fmt.Sprintf("%s@%s", path, root.Digest))
	return &discoverHandler{
		out:  out,
		path: path,
		root: treeRoot,
		nodes: map[digest.Digest]*tree.Node{
			root.Digest: treeRoot,
		},
		verbose: verbose,
	}
}

// MultiLevelSupported implements metadata.DiscoverHandler.
func (h *discoverHandler) MultiLevelSupported() bool {
	return true
}

// OnDiscovered implements metadata.DiscoverHandler.
func (h *discoverHandler) OnDiscovered(referrer, subject ocispec.Descriptor) error {
	node, ok := h.nodes[subject.Digest]
	if !ok {
		return fmt.Errorf("unexpected subject descriptor: %v", subject)
	}

	referrerNode := node.AddPath(fmt.Sprintf("ArtifactType: %s", referrer.ArtifactType), fmt.Sprintf("Digest: %s", referrer.Digest))

	referrerNode.AddPath(fmt.Sprintf("MediaType: %s", referrer.MediaType))
	referrerNode.AddPath(fmt.Sprintf("Size: %d bytes", referrer.Size))
	if len(referrer.URLs) > 0 {
		referrerNode.AddPath(fmt.Sprintf("URLs: %s", strings.Join(referrer.URLs, ", ")))
	}
	if len(referrer.Data) > 0 {
		referrerNode.AddPath(fmt.Sprintf("Data: %s", string(referrer.Data)))
	}
	if referrer.Platform != nil {
		referrerNode.AddPath(fmt.Sprintf("Platform: OS=%s, Architecture=%s, OSVersion=%s, Variant=%s",
			referrer.Platform.OS, referrer.Platform.Architecture, referrer.Platform.OSVersion, referrer.Platform.Variant))
		if len(referrer.Platform.OSFeatures) > 0 {
			referrerNode.AddPath(fmt.Sprintf("Platform OSFeatures: %s", strings.Join(referrer.Platform.OSFeatures, ", ")))
		}
	}
	for k, v := range referrer.Annotations {
		referrerNode.AddPath(fmt.Sprintf("Annotation: %s = %s", k, v))
	}

	h.nodes[referrer.Digest] = referrerNode
	return nil
}

// OnCompleted implements metadata.DiscoverHandler.
func (h *discoverHandler) OnCompleted() error {
	return tree.NewPrinter(h.out).Print(h.root)
}
