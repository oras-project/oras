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

package option

import (
	"context"

	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"oras.land/oras/internal/trace"
)

// Common option struct.
type Common struct {
	Debug   bool
	Verbose bool
}

// ApplyFlags applies flags to a command flag set.
func (common *Common) ApplyFlags(fs *pflag.FlagSet) {
	fs.BoolVarP(&common.Debug, "debug", "d", false, "debug mode")
	fs.BoolVarP(&common.Verbose, "verbose", "v", false, "verbose output")
}

// SetLoggerLevel sets up the logger based on common options.
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
