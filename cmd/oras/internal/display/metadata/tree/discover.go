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

package tree

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/morikuni/aec"
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"gopkg.in/yaml.v3"
	"oras.land/oras/cmd/oras/internal/display/metadata"
	"oras.land/oras/internal/tree"
)

var (
	artifactTypeColor = aec.LightYellowF
	digestColor       = aec.GreenF
	annotationsColor  = aec.LightBlackF
)

// discoverHandler handles json metadata output for discover events.
type discoverHandler struct {
	out     io.Writer
	path    string
	root    *tree.Node
	nodes   map[digest.Digest]*tree.Node
	verbose bool
	tty     *os.File
}

// NewDiscoverHandler creates a new handler for discover events.
func NewDiscoverHandler(out io.Writer, path string, root ocispec.Descriptor, verbose bool, tty *os.File) metadata.DiscoverHandler {
	rootDigest := fmt.Sprintf("%s@%s", path, root.Digest)
	if tty != nil {
		rootDigest = digestColor.Apply(rootDigest)
	}
	treeRoot := tree.New(rootDigest)
	return &discoverHandler{
		out:  out,
		path: path,
		root: treeRoot,
		nodes: map[digest.Digest]*tree.Node{
			root.Digest: treeRoot,
		},
		verbose: verbose,
		tty:     tty,
	}
}

// OnDiscovered implements metadata.DiscoverHandler.
func (h *discoverHandler) OnDiscovered(referrer, subject ocispec.Descriptor) error {
	node, ok := h.nodes[subject.Digest]
	if !ok {
		return fmt.Errorf("unexpected subject descriptor: %v", subject)
	}

	// add artifact type and digest to the referrer
	artifactType := referrer.ArtifactType
	if artifactType == "" {
		artifactType = "<unknown>"
	}
	dgst := referrer.Digest.String()
	if h.tty != nil {
		artifactType = artifactTypeColor.Apply(artifactType)
		dgst = digestColor.Apply(dgst)
	}
	referrerNode := node.AddPath(artifactType, dgst)

	// add annotations to the referrer
	if h.verbose && len(referrer.Annotations) > 0 {
		annotationsTitle := "[annotations]"
		if h.tty != nil {
			annotationsTitle = annotationsColor.Apply(annotationsTitle)
		}
		annotationsNode := referrerNode.Add(annotationsTitle)
		for k, v := range referrer.Annotations {
			bytes, err := yaml.Marshal(map[string]string{k: v})
			if err != nil {
				return err
			}
			annotationsNode.AddPath(strings.TrimSpace(string(bytes)))
		}
	}
	h.nodes[referrer.Digest] = referrerNode
	return nil
}

// Render implements metadata.DiscoverHandler.
func (h *discoverHandler) Render() error {
	return tree.NewPrinter(h.out).Print(h.root)
}
