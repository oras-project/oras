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

package manifest

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"

	"oras.land/oras/cmd/oras/internal/errors"
	"oras.land/oras/cmd/oras/internal/option"
)

type fetchOptions struct {
	option.Common
	option.Remote
	option.Platform

	targetRef string
	pretty    bool
	indent    int
	mediaType string
}

func fetchManifest(opts fetchOptions) error {
	ctx, _ := opts.SetLoggerLevel()
	repo, err := opts.NewRepository(opts.targetRef, opts.Common)
	if err != nil {
		return err
	}
	if repo.Reference.Reference == "" {
		return errors.NewErrInvalidReference(repo.Reference)
	}
	if opts.mediaType != "" {
		repo.ManifestMediaTypes = []string{opts.mediaType}
	}

	// Fetch and output
	manifest, err := opts.Platform.FetchManifest(ctx, repo, opts.targetRef)
	if err != nil {
		return err
	}
	var out bytes.Buffer
	if opts.pretty {
		json.Indent(&out, manifest, "", strings.Repeat(" ", opts.indent))
	} else {
		out = *bytes.NewBuffer(manifest)
	}
	out.WriteTo(os.Stdout)
	return nil
}
