package option

import "github.com/spf13/pflag"

type Common struct {
	Debug   bool
	Verbose bool
}

func (common Common) ApplyFlagsTo(fs *pflag.FlagSet) {
	fs.BoolVarP(&common.Debug, "debug", "d", false, "debug mode")
	fs.BoolVarP(&common.Verbose, "verbose", "v", false, "verbose output")
}
