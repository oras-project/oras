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

package progress

import "testing"

func Test_spinner_symbol(t *testing.T) {
	var s spinner
	for i := 0; i < len(spinnerSymbols); i++ {
		if s.symbol() != spinnerSymbols[i] {
			t.Errorf("symbol() = %v, want %v", s.symbol(), spinnerSymbols[i])
		}
	}
	if s.symbol() != spinnerSymbols[0] {
		t.Errorf("symbol() = %v, want %v", s.symbol(), spinnerSymbols[0])
	}
}
