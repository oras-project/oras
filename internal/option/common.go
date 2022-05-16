package option

import "github.com/spf13/pflag"

type Common struct {
	Debug   bool
	Verbose bool
}

func (opts *Common) ApplyFlagsTo(fs *pflag.FlagSet) {
	fs.BoolVarP(&opts.Debug, "debug", "d", false, "debug mode")
	fs.BoolVarP(&opts.Verbose, "verbose", "v", false, "verbose output")
}
