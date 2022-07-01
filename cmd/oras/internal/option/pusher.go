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
	"context"
	"os"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/pflag"
	"oras.land/oras-go/v2/content"
)

// Pusher option struct.
type Pusher struct {
	ManifestExport string
}

// ApplyFlags applies flags to a command flag set.
func (opts *Pusher) ApplyFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&opts.ManifestExport, "export-manifest", "", "", "export the pushed manifest")
}

// ExportManifest saves the pushed manifest to a local file.
func (opts *Pusher) ExportManifest(ctx context.Context, pushed ocispec.Descriptor, pushedTo content.Fetcher) error {
	if opts.ManifestExport == "" {
		return nil
	}
	manifestBytes, err := content.FetchAll(ctx, pushedTo, pushed)
	if err != nil {
		return err
	}
	return os.WriteFile(opts.ManifestExport, manifestBytes, 0666)
}
