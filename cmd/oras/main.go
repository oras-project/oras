package main

import (
	"os"

	"github.com/spf13/cobra"
	"oras.land/oras/cmd/oras/manifest"
	"oras.land/oras/cmd/oras/repository"
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
		repository.Cmd(),
	)
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
