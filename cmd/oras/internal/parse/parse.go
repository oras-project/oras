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

package parse

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	oerrors "oras.land/oras/cmd/oras/internal/errors"
)

var ErrMediaTypeNotFound = errors.New(`media type is not specified via the flag "--media-type" nor in the manifest.json`)

// MediaTypeFromJson parses the media type field of bytes content in json format.
func MediaTypeFromJson(cmd *cobra.Command, content []byte) (string, error) {
	var manifest struct {
		MediaType string `json:"mediaType"`
	}
	if err := json.Unmarshal(content, &manifest); err != nil {
		return "", errors.New("not a valid json file")
	}
	if manifest.MediaType == "" {
		return "", &oerrors.Error{
			Err:            ErrMediaTypeNotFound,
			Usage:          fmt.Sprintf("%s %s", cmd.Parent().CommandPath(), cmd.Use),
			Recommendation: `Please specify a valid media type in the manifest JSON or via the "--media-type" flag`,
		}
	}
	return manifest.MediaType, nil
}
