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
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/pflag"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/oci"
)

const RemoteType = "remote"
const OCILayoutType = "oci"

// target option struct.
type BinaryTarget struct {
	From target
	To   target
}

// ApplyFlagsWithPrefix applies flags to a command flag set with a prefix string.
// Commonly used for non-unary remote targets.
func (opts *BinaryTarget) ApplyFlags(fs *pflag.FlagSet) {
	opts.From.ApplyFlagsWithPrefix(fs, "from", "source")
	opts.To.ApplyFlagsWithPrefix(fs, "to", "destination")
}

func (opts *BinaryTarget) SetReferenceInput(from, to string) error {
	opts.From.Fqdn = from
	opts.To.Fqdn = to
	return nil
}

func (opts *BinaryTarget) Parse() error {
	if err := opts.From.parse(); err != nil {
		return err
	}
	return opts.To.parse()
}

// Unary target option struct.
type Target struct {
	target
}

func (opts *Target) SetReferenceInput(ref string) error {
	opts.Fqdn = ref
	return nil
}

// ApplyFlagsWithPrefix applies flags to a command flag set with a prefix string.
// Commonly used for non-unary remote targets.
func (opts *Target) ApplyFlags(fs *pflag.FlagSet) {
	opts.target.applyFlags(fs)
}

func (opts *Target) Parse() error {
	if err := opts.target.parse(); err != nil {
		return err
	}
	return opts.target.Remote.Parse()
}

// target option struct.
type target struct {
	config  map[string]string
	Fqdn    string
	Type    string
	tarball bool
	Remote

	// need new reference in oras-go
	isTag     bool
	Reference string
}

func (opts *target) applyFlags(fs *pflag.FlagSet) {
	fs.BoolVarP(&opts.PasswordFromStdin, "password-stdin", "", false, "read password or identity token from stdin")
	opts.ApplyFlagsWithPrefix(fs, "", "")
}

func (opts *target) ApplyFlagsWithPrefix(fs *pflag.FlagSet, prefix, description string) {
	var (
		flagPrefix string
		noteSuffix string
	)
	if prefix != "" {
		flagPrefix = prefix + "-"
		noteSuffix = "for " + description
	}

	fs.StringToStringVarP(&opts.config, flagPrefix+"target", "", defaultConfig, "configure target configuration"+noteSuffix)
	opts.Remote.ApplyFlagsWithPrefix(fs, prefix, description)
}

// check value is cow. If not, use a NewFunction instead
var defaultConfig = map[string]string{
	"tarball": "false",
}

func (opts *target) parse() error {
	if l, r, found := strings.Cut(opts.Fqdn, ":"); found && l == OCILayoutType {
		opts.Type = l
		opts.Fqdn = r
		isTarball := opts.config["tarball"]
		if isTarball != "" {
			var err error = nil
			if opts.tarball, err = strconv.ParseBool(isTarball); err != nil {
				return err
			}
		}
	} else {
		opts.Type = RemoteType
	}

	switch opts.Type {
	case OCILayoutType, RemoteType:
		return nil
	}
	return fmt.Errorf("unknown target type: %q", opts.Type)
}

func (opts *target) NewTarget(common Common) (graphTarget oras.GraphTarget, err error) {
	switch opts.Type {
	case OCILayoutType:
		if idx := strings.Index(opts.Fqdn, "@"); idx != -1 {
			// `digest` found
			opts.isTag = false
			graphTarget, err = oci.New(opts.Fqdn[:idx])
			opts.Reference = opts.Fqdn[idx+1:]
		} else if idx = strings.Index(opts.Fqdn, ":"); idx != -1 {
			// `tag` found
			opts.isTag = true
			graphTarget, err = oci.New(opts.Fqdn[:idx])
			opts.Reference = opts.Fqdn[idx+1:]
		} else {
			graphTarget, err = oci.New(opts.Fqdn)
		}
		return
	case RemoteType:
		repo, err := opts.NewRepository(opts.Fqdn, common)
		if err != nil {
			return nil, err
		}
		opts.Reference = repo.Reference.Reference
		return repo, nil
	}
	return nil, fmt.Errorf("unknown target type: %q", opts.Type)
}

func (opts *target) NewReadonlyTarget(ctx context.Context, common Common) (oras.ReadOnlyGraphTarget, error) {
	switch opts.Type {
	case OCILayoutType:
		var path string
		if idx := strings.Index(opts.Fqdn, "@"); idx != -1 {
			// `digest` found
			opts.isTag = false
			path = opts.Fqdn[:idx]
			opts.Reference = opts.Fqdn[idx+1:]
		} else if idx = strings.Index(opts.Fqdn, ":"); idx != -1 {
			// `tag` found
			opts.isTag = true
			path = opts.Fqdn[:idx]
			opts.Reference = opts.Fqdn[idx+1:]
		}
		var graphTarget *oci.ReadOnlyStore
		var err error
		if opts.tarball {
			graphTarget, err = oci.NewFromTar(ctx, path)
			if err != nil {
				return nil, err
			}
		} else {
			graphTarget, err = oci.NewFromFS(ctx, os.DirFS(path))
			if err != nil {
				return nil, err
			}
		}
		return graphTarget, nil
	case RemoteType:
		repo, err := opts.NewRepository(opts.Fqdn, common)
		if err != nil {
			return nil, err
		}
		opts.Reference = repo.Reference.Reference
		return repo, nil
	}
	return nil, fmt.Errorf("unknown target type: %q", opts.Type)
}
