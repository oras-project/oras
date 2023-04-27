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

package trace

import (
	"context"

	"github.com/sirupsen/logrus"
)

type contextKey int

// loggerKey is the associated key type for logger entry in context.
const loggerKey contextKey = iota

// NewLogger returns a logger.
func NewLogger(ctx context.Context, debug bool, verbose bool) (context.Context, logrus.FieldLogger) {
	var logLevel logrus.Level
	if debug {
		logLevel = logrus.DebugLevel
	} else if verbose {
		logLevel = logrus.InfoLevel
	} else {
		logLevel = logrus.WarnLevel
	}

	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{DisableQuote: true})
	logger.SetLevel(logLevel)
	entry := logger.WithContext(ctx)
	return context.WithValue(ctx, loggerKey, entry), entry
}

// Logger return the logger attached to context or the standard one.
func Logger(ctx context.Context) logrus.FieldLogger {
	logger, ok := ctx.Value(loggerKey).(logrus.FieldLogger)
	if !ok {
		return logrus.StandardLogger()
	}
	return logger
}
