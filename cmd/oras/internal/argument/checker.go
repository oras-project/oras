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

package argument

import "fmt"

// Exactly checks if the number of arguments is exactly cnt.
func Exactly(cnt int) func(args []string) (bool, string) {
	return func(args []string) (bool, string) {
		return len(args) == cnt, fmt.Sprintf("exactly %d argument", cnt)
	}
}

// AtLeast checks if the number of arguments is larger or equal to cnt.
func AtLeast(cnt int) func(args []string) (bool, string) {
	return func(args []string) (bool, string) {
		return len(args) >= cnt, fmt.Sprintf("at least %d argument", cnt)
	}
}
