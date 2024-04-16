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

func Test_speedWindow(t *testing.T) {
	w := newSpeedWindow(5)
	for i := range 20 {
		w.Add(int64(i))
	}
	if w.Mean() != 17 {
		t.Errorf("expected mean to be 17, got %f", w.Mean())
	}
}
