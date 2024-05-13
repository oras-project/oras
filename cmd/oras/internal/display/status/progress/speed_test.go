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

import (
	"testing"
	"time"
)

func Test_speedWindow(t *testing.T) {
	w := newSpeedWindow(3)
	if s := w.Mean(); s != 0 {
		t.Errorf("expected 0, got %f", s)
	}

	now := time.Now()
	w.Add(now, 100)
	if s := w.Mean(); s != 0 {
		t.Errorf("expected 0, got %f", s)
	}

	w.Add(now.Add(1*time.Second), 200)
	if s := w.Mean(); s != 100 {
		t.Errorf("expected 100, got %f", s)
	}

	w.Add(now.Add(4*time.Second), 900)
	if s := w.Mean(); s != 200 {
		t.Errorf("expected 200, got %f", s)
	}

	w.Add(now.Add(5*time.Second), 1400)
	if s := w.Mean(); s != 300 {
		t.Errorf("expected 300, got %f", s)
	}
}
