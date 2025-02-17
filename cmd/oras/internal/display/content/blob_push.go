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

package content

import (
	"encoding/json"
	"os"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras/cmd/oras/internal/output"
)

// blobPush handles raw content output.
type blobPush struct {
	pretty bool
	desc   ocispec.Descriptor
}

// NewBlobPushHandler creates a new handler.
func NewBlobPushHandler(pretty bool, desc ocispec.Descriptor) BlobPushHandler {
	return &blobPush{
		pretty: pretty,
		desc:   desc,
	}
}

// OnBlobPushed is called after a blob is pushed.
func (h *blobPush) OnBlobPushed() error {
	descriptorBytes, err := json.Marshal(h.desc)
	if err != nil {
		return err
	}
	return output.PrintJSON(os.Stdout, descriptorBytes, h.pretty)
}
