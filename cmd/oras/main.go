package main

import (
	"os"

	"github.com/spf13/cobra"
	"oras.land/oras/cmd/oras/manifest"
	"oras.land/oras/cmd/oras/tag"
)

func main() {
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
		manifest.Cmd(),
		tag.TagCmd(),
	)
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
