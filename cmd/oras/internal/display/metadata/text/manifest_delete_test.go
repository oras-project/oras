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

package text

import (
	"bytes"
	"testing"

	"oras.land/oras/cmd/oras/internal/option"
	"oras.land/oras/cmd/oras/internal/output"
)

func TestManifestDeleteHandler_OnManifestMissing(t *testing.T) {
	mockPrinter := output.NewPrinter(bytes.NewBuffer(nil), bytes.NewBuffer(nil), false)
	mockTarget := &option.Target{
		RawReference: "test-repo:missing-tag",
	}
	handler := NewManifestDeleteHandler(mockPrinter, mockTarget)
	
	err := handler.OnManifestMissing()
	if err != nil {
		t.Errorf("OnManifestMissing() error = %v, want nil", err)
	}
}

func TestManifestDeleteHandler_OnManifestDeleted(t *testing.T) {
	mockPrinter := output.NewPrinter(bytes.NewBuffer(nil), bytes.NewBuffer(nil), false)
	mockTarget := &option.Target{
		RawReference: "test-repo:deleted-tag",
	}
	handler := NewManifestDeleteHandler(mockPrinter, mockTarget)
	
	err := handler.OnManifestDeleted()
	if err != nil {
		t.Errorf("OnManifestDeleted() error = %v, want nil", err)
	}
}

func TestNewManifestDeleteHandler(t *testing.T) {
	mockPrinter := output.NewPrinter(bytes.NewBuffer(nil), bytes.NewBuffer(nil), false)
	mockTarget := &option.Target{
		RawReference: "test-repo:test-tag",
	}
	
	handler := NewManifestDeleteHandler(mockPrinter, mockTarget)
	if handler == nil {
		t.Error("NewManifestDeleteHandler() returned nil")
	}
	
	concreteHandler, ok := handler.(*ManifestDeleteHandler)
	if !ok {
		t.Error("NewManifestDeleteHandler() did not return *ManifestDeleteHandler")
	}
	
	if concreteHandler.printer != mockPrinter {
		t.Error("NewManifestDeleteHandler() printer not set correctly")
	}
	
	if concreteHandler.target != mockTarget {
		t.Error("NewManifestDeleteHandler() target not set correctly")
	}
}
