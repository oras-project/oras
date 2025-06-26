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

package display

import (
	"oras.land/oras/cmd/oras/internal/display/metadata"
	"oras.land/oras/cmd/oras/internal/display/metadata/json"
	"oras.land/oras/cmd/oras/internal/display/metadata/template"
	"oras.land/oras/cmd/oras/internal/display/metadata/text"
	"oras.land/oras/cmd/oras/internal/errors"
	"oras.land/oras/cmd/oras/internal/option"
	"oras.land/oras/cmd/oras/internal/output"
)

// NewRepoListHandler returns a repo ls handler.
func NewRepoListHandler(printer *output.Printer, format option.Format) (metadata.RepoListHandler, error) {
	var handler metadata.RepoListHandler
	switch format.Type {
	case option.FormatTypeText.Name:
		handler = text.NewRepoListHandler(printer)
	case option.FormatTypeJSON.Name:
		handler = json.NewRepoListHandler(printer)
	case option.FormatTypeGoTemplate.Name:
		handler = template.NewRepoListHandler(printer, format.Template)
	default:
		return nil, errors.UnsupportedFormatTypeError(format.Type)
	}
	return handler, nil
}
