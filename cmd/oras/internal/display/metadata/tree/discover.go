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
	"gopkg.in/yaml.v3"
	"io"
	"oras.land/oras-go/v2"
	"oras.land/oras/cmd/oras/internal/option"
	"sort"
	"strconv"
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
func (h *discoverHandler) OnDiscovered(referrer, subject ocispec.Descriptor, ctx context.Context, target oras.ReadOnlyTarget, platform option.Platform) error {
	node, ok := h.nodes[subject.Digest]
	if !ok {
		return fmt.Errorf("unexpected subject descriptor: %v", subject)
	}
	if referrer.ArtifactType == "" {
		referrer.ArtifactType = "<unknown>"
	}
	referrerNode := node.AddPath(referrer.ArtifactType, referrer.Digest)
	if h.verbose {
		addDescriptorInfo(referrer, referrerNode)

		if err := addAnnotationsInfo(referrer, referrerNode); err != nil {
			return err
		}

		if err := addLayersInfo(referrer, referrerNode, ctx, target, platform); err != nil {
			return err
		}
	}
	h.nodes[referrer.Digest] = referrerNode
	return nil
}

func addDescriptorInfo(referrer ocispec.Descriptor, referrerNode *tree.Node) {
	descriptorNode := referrerNode.AddPath("Descriptor")
	descriptorNode.AddPath(strings.TrimSpace("MediaType: " + referrer.MediaType))
	descriptorNode.AddPath(strings.TrimSpace("Size: " + strconv.FormatInt(referrer.Size, 10)))
}

func addAnnotationsInfo(referrer ocispec.Descriptor, referrerNode *tree.Node) error {
	if len(referrer.Annotations) != 0 {
		annotationsNode := referrerNode.AddPath("Annotations")
		annotationsKeys := sortedKeys(referrer.Annotations)
		for _, k := range annotationsKeys {
			v := referrer.Annotations[k]
			bytes, err := yaml.Marshal(map[string]string{k: v})
			if err != nil {
				return err
			}
			annotationsNode.AddPath(strings.TrimSpace(string(bytes)))
		}
	}
	return nil
}

func addLayersInfo(referrer ocispec.Descriptor, referrerNode *tree.Node, ctx context.Context, target oras.ReadOnlyTarget, platform option.Platform) error {
	layersNode := tree.New("Layers")
	layersDumped, ok := dumpLayers(referrer.Digest, layersNode, ctx, target, platform)
	if ok != nil {
		return ok
	} else if layersDumped {
		referrerNode.AddNode(layersNode)
	}
	return nil
}

func dumpLayers(digest digest.Digest, node *tree.Node, ctx context.Context, target oras.ReadOnlyTarget, platform option.Platform) (bool, error) {
	fetchOpts := oras.DefaultFetchBytesOptions
	fetchOpts.TargetPlatform = platform.Platform
	_, content, err := oras.FetchBytes(ctx, target, digest.String(), fetchOpts)
	if err != nil {
		return false, fmt.Errorf("failed to fetch the content of %q: %w", "", err)
	}

	var jsonMap map[string]any
	if err = json.Unmarshal(content, &jsonMap); err != nil {
		return false, err
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
							annotationsNode := layerNode.AddPath(k)
							if annotationsMap, ok := v.(map[string]interface{}); ok {
								annotationsMapKeys := sortedKeys(annotationsMap)
								for _, k := range annotationsMapKeys {
									v := annotationsMap[k]
									bytes, err := yaml.Marshal(map[string]interface{}{k: v})
									if err != nil {
										continue
									}
									annotationsNode.AddPath(strings.TrimSpace(string(bytes)))
								}
							}
						} else {
							bytes, err := yaml.Marshal(map[string]interface{}{k: v})
							if err != nil {
								continue
							}
							layerNode.AddPath(strings.TrimSpace(string(bytes)))
						}
					}
				}
			}
			return true, nil
		}
	}

	return false, nil

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

// OnCompleted implements metadata.DiscoverHandler.
func (h *discoverHandler) OnCompleted() error {
	return tree.NewPrinter(h.out).Print(h.root)
}
