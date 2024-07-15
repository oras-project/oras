package index

import (
	"github.com/spf13/cobra"
)

func Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "index [command]",
		Short: "Index operations",
	}

	cmd.AddCommand(
		createCmd(),
	)
	return cmd
}
