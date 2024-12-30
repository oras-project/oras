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

// Package progress tracks the status of descriptors being processed.
package progress

import (
	"io"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// Manager tracks the progress of multiple descriptors.
type Manager interface {
	io.Closer

	// Track starts tracking the progress of a descriptor.
	Track(desc ocispec.Descriptor) (Tracker, error)
}

// ManagerFunc is an adapter to allow the use of ordinary functions as Managers.
// If f is a function with the appropriate signature, ManagerFunc(f) is a
// [Manager] that calls f.
type ManagerFunc func(desc ocispec.Descriptor, status Status, err error) error

// Close closes the manager.
func (f ManagerFunc) Close() error {
	return nil
}

// Track starts tracking the progress of a descriptor.
func (f ManagerFunc) Track(desc ocispec.Descriptor) (Tracker, error) {
	return TrackerFunc(func(status Status, err error) error {
		return f(desc, status, err)
	}), nil
}
