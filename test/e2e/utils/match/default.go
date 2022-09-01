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

// ErrorKeywords returns a result to matching failure execution with keywords
// in stderr.
func ErrorKeywords(keywords []string) *Result {
	return NewResult(nil, nil, Keyword(keywords), true)
}

// SuccessKeywords returns a result to matching success execution with keywords
// in stdout.
func SuccessKeywords(keywords []string) *Result {
	return NewResult(nil, Keyword(keywords), nil, false)
}

// SuccessContent returns a result to match success execution with stdout
// content.
func SuccessContent(content string) *Result {
	return NewResult(nil, Content(content), nil, false)
}
