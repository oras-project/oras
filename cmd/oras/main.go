package main

import (
	"os"

	"github.com/spf13/cobra"
)

func main() {
	cmd := &cobra.Command{
		Use:          "oras [command]",
		SilenceUsage: true,
		Version:      "v0.4.0",
	}
	cmd.AddCommand(pullCmd(), pushCmd(), loginCmd(), logoutCmd())
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
