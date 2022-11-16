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

	"github.com/spf13/pflag"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/oci"
	"oras.land/oras/cmd/oras/internal/errors"
)

const RemoteType = "remote"
const OCILayoutType = "oci"

// Target option struct.
type Target struct {
	config  map[string]string
	Type    string
	Path    string
	Tarball bool
	Remote
}

// check value is cow. If not, use a NewFunction instead
var defaultConfig = map[string]string{
	"type": "remote",
	"path": "",
}

// ApplyFlags applies flags to a command flag set.
func (opts *Target) ApplyFlags(fs *pflag.FlagSet) {
	opts.ApplyFlagsWithPrefix(fs, "", "")
}

// ApplyFlagsWithPrefix applies flags to a command flag set with a prefix string.
// Commonly used for non-unary remote targets.
func (opts *Target) ApplyFlagsWithPrefix(fs *pflag.FlagSet, prefix, description string) {
	var (
		flagPrefix string
		noteSuffix string
	)
	if prefix != "" {
		flagPrefix = prefix + "-"
		noteSuffix = "for " + description
	}

	fs.StringToStringVarP(&opts.config, flagPrefix+"target", "", defaultConfig, "configure target configuration"+noteSuffix)
	opts.Type = opts.config["type"]
	opts.Path = opts.config["path"]
	if opts.Type == RemoteType {
		opts.Remote.ApplyFlagsWithPrefix(fs, prefix, description)
	}
}

func (opts *Target) ParseFlags() error {
	var err error = nil
	if opts.Tarball, err = strconv.ParseBool(opts.config["tarball"]); err != nil {
		return err
	}

	switch opts.Type {
	case OCILayoutType, RemoteType:
		return nil
	}
	return fmt.Errorf("unknown target type: %s", opts.Type)
}

func (opts *Target) NewTarget(reference string, common Common, isSrc bool) (oras.GraphTarget, error) {
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
	return nil, fmt.Errorf("unknown target type: %s", opts.Type)
}
