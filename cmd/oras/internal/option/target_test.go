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
	"testing"
)

func TestTarget_Parse_shortOCI(t *testing.T) {
	opts := Target{targetFlag: targetFlag{isOCI: true}}

	if err := opts.Parse(); err != nil {
		t.Errorf("Target.Parse() error = %v", err)
	}
	if opts.Type != TargetTypeOCILayout {
		t.Errorf("Target.Parse() failed, got %q, want %q", opts.Type, TargetTypeOCILayout)
	}
}

func TestTarget_Parse_oci(t *testing.T) {
	opts := Target{targetFlag: targetFlag{config: map[string]string{"type": TargetTypeOCILayout}}}
	if err := opts.Parse(); err != nil {
		t.Errorf("Target.Parse() error = %v", err)
	}
	if opts.Type != TargetTypeOCILayout {
		t.Errorf("Target.Parse() failed, got %q, want %q", opts.Type, TargetTypeOCILayout)
	}
}

func TestTarget_Parse_remote(t *testing.T) {
	opts := Target{targetFlag: targetFlag{config: map[string]string{"type": TargetTypeRemote}, isOCI: false}}
	if err := opts.Parse(); err != nil {
		t.Errorf("Target.Parse() error = %v", err)
	}
	if opts.Type != TargetTypeRemote {
		t.Errorf("Target.Parse() failed, got %q, want %q", opts.Type, TargetTypeRemote)
	}
}
