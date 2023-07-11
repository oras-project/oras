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
	"testing"
)

func TestPrinter_Print(t *testing.T) {
	tests := []struct {
		name string
		root *Node
		want string
	}{
		{
			name: "single node tree",
			root: &Node{
				Value: "root",
			},
			want: "root\n",
		},
		{
			name: "single child",
			root: &Node{
				Value: "root",
				Nodes: []*Node{
					{
						Value: "hello",
					},
				},
			},
			want: "root\n└── hello\n",
		},
		{
			name: "multiple children",
			root: &Node{
				Value: "root",
				Nodes: []*Node{
					{
						Value: "hello",
					},
					{
						Value: "world",
					},
				},
			},
			want: "root\n├── hello\n└── world\n",
		},
		{
			name: "nested tree (beginning)",
			root: &Node{
				Value: "root",
				Nodes: []*Node{
					{
						Value: "foo",
						Nodes: []*Node{
							{
								Value: "bar",
							},
							{
								Value: 42,
							},
						},
					},
					{
						Value: "hello",
					},
					{
						Value: "world",
					},
				},
			},
			want: "root\n├── foo\n│   ├── bar\n│   └── 42\n├── hello\n└── world\n",
		},
		{
			name: "nested tree (middle)",
			root: &Node{
				Value: "root",
				Nodes: []*Node{
					{
						Value: "hello",
					},
					{
						Value: "foo",
						Nodes: []*Node{
							{
								Value: "bar",
							},
							{
								Value: 42,
							},
						},
					},
					{
						Value: "world",
					},
				},
			},
			want: "root\n├── hello\n├── foo\n│   ├── bar\n│   └── 42\n└── world\n",
		},
		{
			name: "nested tree (end)",
			root: &Node{
				Value: "root",
				Nodes: []*Node{
					{
						Value: "hello",
					},
					{
						Value: "world",
					},
					{
						Value: "foo",
						Nodes: []*Node{
							{
								Value: "bar",
							},
							{
								Value: 42,
							},
						},
					},
				},
			},
			want: "root\n├── hello\n├── world\n└── foo\n    ├── bar\n    └── 42\n",
		},
		{
			name: "double nested tree",
			root: &Node{
				Value: "root",
				Nodes: []*Node{
					{
						Value: "hello",
					},
					{
						Value: "foo",
						Nodes: []*Node{
							{
								Value: "bar",
								Nodes: []*Node{
									{
										Value: 42,
									},
								},
							},
						},
					},
					{
						Value: "world",
					},
				},
			},
			want: "root\n├── hello\n├── foo\n│   └── bar\n│       └── 42\n└── world\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := bytes.NewBuffer(nil)
			printer := NewPrinter(buf)
			if err := printer.Print(tt.root); err != nil {
				t.Fatalf("Printer.Print() error = %v", err)
			}
			if got := buf.String(); got != tt.want {
				t.Errorf("Printer.Print() = %s, want %s", got, tt.want)
			}
		})
	}
}
