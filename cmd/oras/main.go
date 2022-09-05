package main

import (
	"os"

	"github.com/spf13/cobra"
	"oras.land/oras/cmd/oras/manifest"
<<<<<<< HEAD
	"oras.land/oras/cmd/oras/tag"
=======
	"oras.land/oras/cmd/oras/repository"
>>>>>>> efd765928b9adfdc0df27cabd8ef6b9138d192b9
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
<<<<<<< HEAD
		tag.TagCmd(),
=======
		repository.Cmd(),
>>>>>>> efd765928b9adfdc0df27cabd8ef6b9138d192b9
	)
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
