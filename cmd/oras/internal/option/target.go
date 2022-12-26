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
	"oras.land/oras-go/v2/registry"
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
	opts.From.ApplyShortFlagsWithPrefix(fs, "from", "source")
	opts.From.ApplyShortFlagsWithPrefix(fs, "to", "destination")
	opts.From.ApplyFlagsWithPrefix(fs, "from", "source")
	opts.To.ApplyFlagsWithPrefix(fs, "to", "destination")
}

func (opts *BinaryTarget) SetReferenceInput(from, to string) {
	opts.From.Fqdn = from
	opts.To.Fqdn = to
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

func (opts *Target) SetReferenceInput(ref string) {
	opts.Fqdn = ref
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

type OCI struct {
	config  map[string]string
	Fqdn    string
	Type    string
	tarball bool

	isOCIFolder  bool
	isOCITarball bool
}

func (opts *OCI) ApplyShortFlagsWithPrefix(fs *pflag.FlagSet, prefix, description string) {
	var (
		flagPrefix string
		notePrefix string
	)
	if prefix != "" {
		flagPrefix = prefix + "-"
		notePrefix = description + " "
	}
	fs.BoolVarP(&opts.isOCIFolder, flagPrefix+"oci", "", false, "Set "+notePrefix+"target as an OCI-layout folder")
	fs.BoolVarP(&opts.isOCITarball, flagPrefix+"ocitar", "", false, "Set "+notePrefix+"target as an OCI-layout tarball")
}

// target option struct.
type target struct {
	OCI
	Remote

	// might design need new reference in oras-go for oci-layout target
	isTag     bool
	Reference string
}

func (opts *target) applyFlags(fs *pflag.FlagSet) {
	fs.BoolVarP(&opts.PasswordFromStdin, "password-stdin", "", false, "read password or identity token from stdin")
	opts.ApplyFlagsWithPrefix(fs, "", "")
	opts.ApplyShortFlagsWithPrefix(fs, "", "")
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
	"type":    "remote",
	"tarball": "false",
}

func (opts *target) parse() error {
	// short flag
	if opts.isOCIFolder {
		opts.Type = OCILayoutType
		opts.tarball = false
		return nil
	} else if opts.isOCITarball {
		opts.Type = OCILayoutType
		opts.tarball = true
		return nil
	}

	opts.Type = opts.config["type"]
	switch opts.Type {
	case OCILayoutType:
		isTarball := opts.config["tarball"]
		if isTarball != "" {
			var err error = nil
			if opts.tarball, err = strconv.ParseBool(isTarball); err != nil {
				return err
			}
		}
		return nil
	case RemoteType:
		return nil
	}

	return fmt.Errorf("unknown target type: %q", opts.Type)
}

func (opts *target) NewTarget(common Common) (graphTarget oras.GraphTarget, err error) {
	switch opts.Type {
	case OCILayoutType:
		if idx := strings.LastIndex(opts.Fqdn, "@"); idx != -1 {
			// `digest` found
			opts.isTag = false
			graphTarget, err = oci.New(opts.Fqdn[:idx])
			opts.Reference = opts.Fqdn[idx+1:]
		} else if idx = strings.LastIndex(opts.Fqdn, ":"); idx != -1 {
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

type ReadOnlyGraphTagFinderTarget interface {
	oras.ReadOnlyGraphTarget
	registry.TagFinder
}

func (opts *target) NewReadonlyTarget(ctx context.Context, common Common) (ReadOnlyGraphTagFinderTarget, error) {
	switch opts.Type {
	case OCILayoutType:
		path := opts.Fqdn
		if idx := strings.LastIndex(opts.Fqdn, "@"); idx != -1 {
			// `digest` found
			opts.isTag = false
			path = opts.Fqdn[:idx]
			opts.Reference = opts.Fqdn[idx+1:]
		} else if idx = strings.LastIndex(opts.Fqdn, ":"); idx != -1 {
			// `tag` found
			opts.isTag = true
			path = opts.Fqdn[:idx]
			opts.Reference = opts.Fqdn[idx+1:]
		}
		var store *oci.ReadOnlyStore
		var err error
		if opts.tarball {
			store, err = oci.NewFromTar(ctx, path)
			if err != nil {
				return nil, err
			}
		} else {
			store, err = oci.NewFromFS(ctx, os.DirFS(path))
			if err != nil {
				return nil, err
			}
		}
		return store, nil
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
