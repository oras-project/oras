package option

import (
	"context"

	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"oras.land/oras/internal/trace"
)

type Common struct {
	Debug   bool
	Verbose bool
}

func (common *Common) ApplyFlagsTo(fs *pflag.FlagSet) {
	fs.BoolVarP(&common.Debug, "debug", "d", false, "debug mode")
	fs.BoolVarP(&common.Verbose, "verbose", "v", false, "verbose output")
}

func (common *Common) SetLoggerLevel() (context.Context, logrus.FieldLogger) {
	var logLevel logrus.Level
	if common.Debug {
		logLevel = logrus.DebugLevel
	} else if common.Verbose {
		logLevel = logrus.InfoLevel
	} else {
		logLevel = logrus.WarnLevel
	}
	return trace.WithLoggerLevel(context.Background(), logLevel)
}
