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

package testutils

import (
	"context"
	"fmt"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"io"
)

// ErrorFetcher implements content.Fetcher.
type ErrorFetcher struct {
	ExpectedError error
}

// NewErrorFetcher create and error fetcher
func NewErrorFetcher() *ErrorFetcher {
	return &ErrorFetcher{
		ExpectedError: fmt.Errorf("expected error"),
	}
}

// Fetch returns an error.
func (f *ErrorFetcher) Fetch(context.Context, ocispec.Descriptor) (io.ReadCloser, error) {
	return nil, f.ExpectedError
}
