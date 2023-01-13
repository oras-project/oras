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
	"strings"

	"github.com/spf13/pflag"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/oci"
	"oras.land/oras-go/v2/registry"
)

const (
	RemoteType    = "remote"
	OCILayoutType = "oci"
)

type targetFlag struct {
	config map[string]string
	isOCI  bool
}

// applyFlagsWithPrefix applies flags to a command flag set with a prefix string.
// Commonly used for non-unary remote targets.
func (opts *targetFlag) applyFlagsWithPrefix(fs *pflag.FlagSet, prefix, description string) {
	var (
		flagPrefix string
		noteSuffix string
	)
	if prefix != "" {
		flagPrefix = prefix + "-"
		noteSuffix = "for " + description
	}
	fs.BoolVarP(&opts.isOCI, flagPrefix+"oci", "", false, "Set "+noteSuffix+"target as an OCI-layout")
	fs.StringToStringVarP(&opts.config, flagPrefix+"target", "", map[string]string{"type": "remote"}, "configure target configuration"+noteSuffix)
}

// Unary target option struct.
type Target struct {
	Remote
	FqdnRef   string
	Type      string
	IsTag     bool
	Reference string

	targetFlag
}

// ApplyFlags applies flags to a command flag set for unary target
func (opts *Target) ApplyFlags(fs *pflag.FlagSet) {
	fs.BoolVarP(&opts.PasswordFromStdin, "password-stdin", "", false, "read password or identity token from stdin")
	opts.applyFlagsWithPrefix(fs, "", "")
}

// FullReference returns full printable reference.
func (opts *Target) FullReference() string {
	return fmt.Sprintf("[%s] %s", opts.Type, opts.FqdnRef)
}

// applyFlagsWithPrefix applies flags to a command flag set with a prefix string.
// Commonly used for non-unary remote targets.
func (opts *Target) applyFlagsWithPrefix(fs *pflag.FlagSet, prefix, description string) {
	opts.targetFlag.applyFlagsWithPrefix(fs, prefix, description)
	opts.Remote.ApplyFlagsWithPrefix(fs, prefix, description)
}

// Parse gets target options from user input.
func (opts *Target) Parse() error {
	if opts.isOCI {
		// short flag
		opts.Type = OCILayoutType
		return nil
	} else {
		// long flag
		opts.Type = opts.config["type"]
		switch opts.Type {
		case RemoteType:
			return opts.Remote.Parse()
		case OCILayoutType:
			return nil
		}
	}
	return fmt.Errorf("unknown target type: %q", opts.Type)
}

// NewTarget generates a new target based on opts.
func (opts *Target) NewTarget(common Common) (graphTarget oras.GraphTarget, err error) {
	switch opts.Type {
	case OCILayoutType:
		if idx := strings.LastIndex(opts.FqdnRef, "@"); idx != -1 {
			// `digest` found
			opts.IsTag = false
			graphTarget, err = oci.New(opts.FqdnRef[:idx])
			opts.Reference = opts.FqdnRef[idx+1:]
		} else if idx = strings.LastIndex(opts.FqdnRef, ":"); idx != -1 {
			// `tag` found
			opts.IsTag = true
			graphTarget, err = oci.New(opts.FqdnRef[:idx])
			opts.Reference = opts.FqdnRef[idx+1:]
		} else {
			graphTarget, err = oci.New(opts.FqdnRef)
		}
		return
	case RemoteType:
		repo, err := opts.NewRepository(opts.FqdnRef, common)
		if err != nil {
			return nil, err
		}
		opts.Reference = repo.Reference.Reference
		return repo, nil
	}
	return nil, fmt.Errorf("unknown target type: %q", opts.Type)
}

// Read-only graph target with tag lister.
type ReadOnlyGraphTagFinderTarget interface {
	oras.ReadOnlyGraphTarget
	registry.TagLister
}

// NewReadonlyTargets generates a new read only target based on opts.
func (opts *Target) NewReadonlyTarget(ctx context.Context, common Common) (ReadOnlyGraphTagFinderTarget, error) {
	switch opts.Type {
	case OCILayoutType:
		path := opts.FqdnRef
		if idx := strings.LastIndex(opts.FqdnRef, "@"); idx != -1 {
			// `digest` found
			opts.IsTag = false
			path = opts.FqdnRef[:idx]
			opts.Reference = opts.FqdnRef[idx+1:]
		} else if idx = strings.LastIndex(opts.FqdnRef, ":"); idx != -1 {
			// `tag` found
			opts.IsTag = true
			path = opts.FqdnRef[:idx]
			opts.Reference = opts.FqdnRef[idx+1:]
		}
		var store *oci.ReadOnlyStore
		info, err := os.Stat(path)
		if err != nil {
			return nil, err
		}

		if info.IsDir() {
			store, err = oci.NewFromFS(ctx, os.DirFS(path))
			if err != nil {
				return nil, err
			}
		} else {
			store, err = oci.NewFromTar(ctx, path)
			if err != nil {
				return nil, err
			}
		}
		return store, nil
	case RemoteType:
		repo, err := opts.NewRepository(opts.FqdnRef, common)
		if err != nil {
			return nil, err
		}
		opts.Reference = repo.Reference.Reference
		return repo, nil
	}
	return nil, fmt.Errorf("unknown target type: %q", opts.Type)
}

// Binary target option struct.
type BinaryTarget struct {
	From Target
	To   Target
}

// ApplyFlagsWithPrefix applies flags to a command flag set with a prefix string.
// Commonly used for non-unary remote targets.
func (opts *BinaryTarget) ApplyFlags(fs *pflag.FlagSet) {
	opts.From.applyFlagsWithPrefix(fs, "from", "source")
	opts.To.applyFlagsWithPrefix(fs, "to", "destination")
}

// Parse parses user-provided flags and arguments into option struct.
func (opts *BinaryTarget) Parse() error {
	if err := opts.From.Parse(); err != nil {
		return err
	}
	return opts.To.Parse()
}
