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

package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"oras.land/oras/cmd/oras/internal/option"
)

type repositoryOptions struct {
	option.Remote
	option.Common
	hostname string
}

// func init() {
// 	rootCmd.AddCommand(versionCmd)
// }

func repositoryCmd() *cobra.Command {
	// in case need option
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version number of Hugo",
		Long:  `All software has versions. This is Hugo's`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Hugo Static Site Generator v0.9 -- HEAD")
			return runRepository()
		},
	}

	// cmd.Flags()

	// option.ApplyFlags()
	return cmd
}

func runRepository(opts repositoryOptions) error {
	// get all repository from the registry
	reg, err := opts.Remote.NewRegistry(opts.hostname, opts.Common)
	fmt.Print(reg.Repositories(), err)
	// list all repository
	return nil
}
