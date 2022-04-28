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

// WithLoggerLevel returns a context with logrus log entry.
func WithLoggerLevel(ctx context.Context, level logrus.Level) (context.Context, logrus.FieldLogger) {
	logger := logrus.New()
	logger.SetLevel(level)
	entry := logger.WithContext(ctx)
	return context.WithValue(ctx, loggerKey, entry), entry
}

func Logger(ctx context.Context) logrus.FieldLogger {
	logger, ok := ctx.Value(loggerKey).(logrus.FieldLogger)
	if !ok {
		return logrus.StandardLogger()
	}
	return logger
}
