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

package match

type output struct {
	content []byte
}

func NewOutput() *output {
	return &output{}
}

// Write captures p into the content.
func (w *output) Write(p []byte) (n int, err error) {
	w.content = append(w.content, p...)
	return len(p), nil
}

func (w *output) readAll() []byte {
	return w.content
}
