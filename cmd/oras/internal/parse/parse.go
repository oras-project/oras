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
)

var ErrMediaTypeNotFound = errors.New(`media type is not specified`)
var ErrInvalidJSON = errors.New("not a valid json file")

// MediaTypeFromJson parses the media type field of bytes content in json format.
func MediaTypeFromJson(content []byte) (string, error) {
	var manifest struct {
		MediaType string `json:"mediaType"`
	}
	if err := json.Unmarshal(content, &manifest); err != nil {
		return "", ErrInvalidJSON
	}
	if manifest.MediaType == "" {
		return "", ErrMediaTypeNotFound
	}
	return manifest.MediaType, nil
}
