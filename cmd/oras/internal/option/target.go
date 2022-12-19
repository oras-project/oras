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
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/pflag"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/oci"
	"oras.land/oras/cmd/oras/internal/errors"
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
	opts.From.refInput = from
	opts.To.refInput = to
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
	opts.refInput = ref
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
	config     map[string]string
	refInput   string
	Type       string
	Path       string
	Compressed bool
	Reference  string
	Tarball    bool
	Remote
}

// check value is cow. If not, use a NewFunction instead
var defaultConfig = map[string]string{
	"path":    "",
	"tarball": "false",
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

func (opts *target) parse() error {
	if l, r, found := strings.Cut(opts.refInput, ":"); found && l == OCILayoutType {
		opts.Type = l
		opts.refInput = r
		opts.Path = opts.config["path"]
		isTarball := opts.config["tarball"]
		if isTarball != "" {
			var err error = nil
			if opts.Tarball, err = strconv.ParseBool(isTarball); err != nil {
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

func (opts *target) NewTarget(reference string, common Common, isSrc bool) (oras.GraphTarget, error) {
	switch opts.Type {
	case OCILayoutType:
		graphTarget, err := oci.New(opts.Path)
		if err != nil {
			return nil, err
		}
		return graphTarget, nil
	case RemoteType:
		repo, err := opts.NewRepository(reference, common)
		if err != nil {
			return nil, err
		}
		if isSrc && repo.Reference.Reference == "" {
			return nil, errors.NewErrInvalidReference(repo.Reference)
		}
		return repo, nil
	}
	return nil, fmt.Errorf("unknown target type: %q", opts.Type)
}
