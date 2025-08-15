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
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"github.com/morikuni/aec"
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"gopkg.in/yaml.v3"
	"io"
	"oras.land/oras-go/v2"
	"oras.land/oras/cmd/oras/internal/display/metadata"
	"oras.land/oras/cmd/oras/internal/option"
	"oras.land/oras/internal/tree"
	"os"
	"sort"
	"strings"
)

var (
	artifactTypeColor = aec.LightYellowF
	digestColor       = aec.LightGreenF
	annotationsColor  = aec.LightCyanF
	descriptorColor   = aec.LightBlueF
	layersColor       = aec.LightMagentaF
)

// discoverHandler handles json metadata output for discover events.
type discoverHandler struct {
	out        io.Writer
	path       string
	root       *tree.Node
	nodes      map[digest.Digest]*tree.Node
	verbose    bool
	dumpLayers bool
	tty        *os.File
	context    context.Context
	target     oras.ReadOnlyTarget
	platform   option.Platform
}

// NewDiscoverHandler creates a new handler for discover events.
func NewDiscoverHandler(out io.Writer, path string, root ocispec.Descriptor, verbose bool, dumpLayers bool, tty *os.File, context context.Context, target oras.ReadOnlyTarget, platform option.Platform) metadata.DiscoverHandler {
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
		verbose:    verbose,
		dumpLayers: dumpLayers,
		tty:        tty,
		context:    context,
		target:     target,
		platform:   platform,
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
	if h.verbose {
		descriptorTitle := "[descriptor]"
		if h.tty != nil {
			descriptorTitle = descriptorColor.Apply(descriptorTitle)
		}
		descriptorNode := referrerNode.Add(descriptorTitle)
		descriptorNode.AddKV("Size", referrer.Size)
		descriptorNode.AddKV("MediaType", referrer.MediaType)

		if len(referrer.Annotations) > 0 {
			annotationsTitle := "[annotations]"
			if h.tty != nil {
				annotationsTitle = annotationsColor.Apply(annotationsTitle)
			}
			annotationsNode := referrerNode.Add(annotationsTitle)
			for k, v := range referrer.Annotations {
				annotationsNode.AddKV(k, v)
			}
		}

		if h.dumpLayers {
			if err := addLayersInfo(referrer, referrerNode, h.context, h.target, h.platform, h.tty); err != nil {
				return err
			}
		}
	}
	h.nodes[referrer.Digest] = referrerNode
	return nil
}

// Render implements metadata.DiscoverHandler.
func (h *discoverHandler) Render() error {
	return tree.NewPrinter(h.out).Print(h.root)
}

func sortedKeys[K cmp.Ordered, V any](m map[K]V) []K {
	keys := make([]K, len(m))
	i := 0
	for k := range m {
		keys[i] = k
		i++
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	return keys
}

func addLayersInfo(referrer ocispec.Descriptor, referrerNode *tree.Node, ctx context.Context, target oras.ReadOnlyTarget, platform option.Platform, tty *os.File) error {
	layersTitle := "[layers]"
	if tty != nil {
		layersTitle = layersColor.Apply(layersTitle)
	}

	layersNode := tree.New(layersTitle)
	if err := dumpLayers(referrer.Digest, layersNode, ctx, target, platform, tty); err != nil {
		return err
	} else {
		referrerNode.AddNode(layersNode)
	}
	return nil
}

func dumpLayers(digest digest.Digest, node *tree.Node, ctx context.Context, target oras.ReadOnlyTarget, platform option.Platform, tty *os.File) error {
	fetchOpts := oras.DefaultFetchBytesOptions
	fetchOpts.TargetPlatform = platform.Platform
	_, content, err := oras.FetchBytes(ctx, target, digest.String(), fetchOpts)
	if err != nil {
		return fmt.Errorf("failed to fetch the layers for %q: %w", digest.String(), err)
	}

	var jsonMap map[string]any
	if err = json.Unmarshal(content, &jsonMap); err != nil {
		return err
	}

	if layers, ok := jsonMap["layers"].([]interface{}); ok {
		if len(layers) != 0 {
			for idx, item := range layers {
				layerNode := node.AddPath(fmt.Sprintf("Layer[%d]", idx))

				if layer, ok := item.(map[string]interface{}); ok {

					layerKeys := sortedKeys(layer)
					for _, k := range layerKeys {
						v := layer[k]
						if k == "annotations" {
							if annotationsMap, ok := v.(map[string]interface{}); ok {

								if len(annotationsMap) > 0 {
									annotationsTitle := "[annotations]"
									if tty != nil {
										annotationsTitle = annotationsColor.Apply(annotationsTitle)
									}
									annotationsNode := layerNode.AddPath(annotationsTitle)

									annotationsMapKeys := sortedKeys(annotationsMap)
									for _, k := range annotationsMapKeys {
										v := annotationsMap[k]
										annotationsNode.AddKV(k, v)
									}
								}
							}
						} else {
							bytes, err := yaml.Marshal(map[string]interface{}{k: v})
							if err != nil {
								panic(err)
							}
							layerNode.AddPath(strings.TrimSpace(string(bytes)))
						}
					}
				}
			}
			return nil
		}
	}
	return nil
}
