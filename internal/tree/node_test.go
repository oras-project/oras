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
	"bytes"
	"reflect"
	"testing"
)

func TestNode_Add(t *testing.T) {
	root := &Node{
		Value: "root",
	}

	nodeNil := root.Add(nil)
	want := &Node{}
	if !reflect.DeepEqual(nodeNil, want) {
		t.Errorf("Node.Add() = %v, want %v", nodeNil, want)
	}

	nodeFoo := root.Add("foo")
	want = &Node{
		Value: "foo",
	}
	if !reflect.DeepEqual(nodeFoo, want) {
		t.Errorf("Node.Add() = %v, want %v", nodeFoo, want)
	}
	nodeBar := nodeFoo.Add("bar")
	want = &Node{
		Value: "bar",
	}
	if !reflect.DeepEqual(nodeBar, want) {
		t.Errorf("Node.Add() = %v, want %v", nodeBar, want)
	}

	node42 := root.Add(42)
	want = &Node{
		Value: 42,
	}
	if !reflect.DeepEqual(node42, want) {
		t.Errorf("Node.Add() = %v, want %v", node42, want)
	}

	buf := bytes.NewBuffer(nil)
	printer := NewPrinter(buf)
	if err := printer.Print(root); err != nil {
		t.Fatalf("Printer.Print() error = %v", err)
	}
	gotPrint := buf.String()
	// root
	// ├── <nil>
	// ├── foo
	// │   └── bar
	// └── 42
	wantPrint := "root\n├── <nil>\n├── foo\n│   └── bar\n└── 42\n"
	if gotPrint != wantPrint {
		t.Errorf("Node = %s, want %s", gotPrint, wantPrint)
	}
}

func TestNode_AddPath(t *testing.T) {
	root := &Node{
		Value: "root",
	}

	nodeNil := root.AddPath()
	var want *Node
	if !reflect.DeepEqual(nodeNil, want) {
		t.Errorf("Node.AddPath() = %v, want %v", nodeNil, want)
	}

	nodeBar := root.AddPath("foo", "bar")
	want = &Node{
		Value: "bar",
	}
	if !reflect.DeepEqual(nodeBar, want) {
		t.Errorf("Node.AddPath() = %v, want %v", nodeBar, want)
	}
	nodeBar2 := root.AddPath("foo", "bar2")
	want = &Node{
		Value: "bar2",
	}
	if !reflect.DeepEqual(nodeBar2, want) {
		t.Errorf("Node.AddPath() = %v, want %v", nodeBar2, want)
	}

	node42 := root.AddPath(42)
	want = &Node{
		Value: 42,
	}
	if !reflect.DeepEqual(node42, want) {
		t.Errorf("Node.AddPath() = %v, want %v", node42, want)
	}

	buf := bytes.NewBuffer(nil)
	printer := NewPrinter(buf)
	if err := printer.Print(root); err != nil {
		t.Fatalf("Printer.Print() error = %v", err)
	}
	gotPrint := buf.String()
	// root
	// ├── foo
	// │   ├── bar
	// │   └── bar2
	// └── 42
	wantPrint := "root\n├── foo\n│   ├── bar\n│   └── bar2\n└── 42\n"
	if gotPrint != wantPrint {
		t.Errorf("Node = %s, want %s", gotPrint, wantPrint)
	}
}

func TestNode_Find(t *testing.T) {
	root := &Node{
		Value: "root",
		Nodes: []*Node{
			{
				Value: "foo",
				Nodes: []*Node{
					{
						Value: "bar",
					},
				},
			},
			{
				Value: 42,
			},
		},
	}
	tests := []struct {
		name  string
		value any
		want  *Node
	}{
		{
			name:  "find existing node",
			value: 42,
			want:  root.Nodes[1],
		},
		{
			name:  "find non-existing node",
			value: "hello",
			want:  nil,
		},
		{
			name:  "find non-existing node but it is a grand child",
			value: "bar",
			want:  nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := root.Find(tt.value); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Node.Find() = %v, want %v", got, tt.want)
			}
		})
	}
}
