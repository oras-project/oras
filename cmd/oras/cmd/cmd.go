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
package cmd

import (
	"github.com/spf13/cobra"
	"oras.land/oras/cmd/oras/blob"
	"oras.land/oras/cmd/oras/manifest"
	"oras.land/oras/cmd/oras/repository"
	"oras.land/oras/cmd/oras/tag"
)

func NewRoot() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "oras [command]",
		SilenceUsage: true,
	}
	cmd.AddCommand(
		pullCmd(),
		pushCmd(),
		loginCmd(),
		logoutCmd(),
		versionCmd(),
		discoverCmd(),
		copyCmd(),
		attachCmd(),
		blob.Cmd(),
		manifest.Cmd(),
		tag.TagCmd(),
		repository.Cmd(),
	)
	return cmd
}
