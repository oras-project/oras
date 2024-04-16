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

type speedWindow struct {
	point []int64
	head  int
	size  int
	sum   int64
}

// newSpeedWindow creates a new speed window with a given capacity.
func newSpeedWindow(capacity int) *speedWindow {
	return &speedWindow{
		point: make([]int64, capacity),
	}
}

// Add adds a done workload to the window.
func (w *speedWindow) Add(value int64) {
	if w.size == len(w.point) {
		w.sum -= w.point[w.head]
	} else {
		w.size++
	}
	w.point[w.head] = value
	w.sum += value
	w.head = (w.head + 1) % len(w.point)
}

// Mean returns the average workload done in the window.
func (w *speedWindow) Mean() float64 {
	if w.size == 0 {
		return 0
	}
	return float64(w.sum) / float64(w.size)
}
