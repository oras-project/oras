package main

import (
	"os"

	"github.com/spf13/cobra"
)

func main() {
	cmd := &cobra.Command{
		Use:          "oras [OPTIONS] COMMAND",
		SilenceUsage: true,
	}
	cmd.AddCommand(pullCmd(), pushCmd())
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
