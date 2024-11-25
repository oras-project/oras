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

package command

import (
	"context"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"oras.land/oras/cmd/oras/internal/option"
	"oras.land/oras/internal/trace"
)

// GetLogger returns a new FieldLogger and an associated Context derived from command context.
func GetLogger(cmd *cobra.Command, opts *option.Common) (context.Context, logrus.FieldLogger) {
	ctx, logger := trace.NewLogger(cmd.Context(), opts.Debug)
	cmd.SetContext(ctx)
	return ctx, logger
}
