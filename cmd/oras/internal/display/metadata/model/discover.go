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

package model

import (
	"fmt"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// Discover is a model for discovered referrers.
type Discover struct {
	name  string
	nodes map[digest.Digest]*Node
	Root  *Node
}

// Node represents a node in the discovered reference tree.
type Node struct {
	Descriptor
	Referrers []*Node `json:"referrers,omitempty"`
}

// Add adds a node to the discovered referrers tree.
func (d *Discover) Add(referrer, subject ocispec.Descriptor) error {
	to, ok := d.nodes[subject.Digest]
	if !ok {
		return fmt.Errorf("unexpected subject descriptor: %v", subject)
	}
	from := NewNode(d.name, referrer)
	d.nodes[from.Digest] = from
	to.Referrers = append(to.Referrers, from)
	return nil
}

// NewDiscover creates a new discover model.
func NewDiscover(path string, root ocispec.Descriptor) Discover {
	treeRoot := NewNode(path, root)
	return Discover{
		name: path,
		nodes: map[digest.Digest]*Node{
			root.Digest: treeRoot,
		},
		Root: treeRoot,
	}
}

// NewNode creates a new discover model.
func NewNode(name string, desc ocispec.Descriptor) *Node {
	return &Node{
		Descriptor: FromDescriptor(name, desc),
	}
}
