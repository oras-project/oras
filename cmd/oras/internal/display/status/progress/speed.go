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

import "time"

type speedPoint struct {
	time   time.Time
	offset int64
}

type speedWindow struct {
	point []speedPoint
	next  int
	size  int
}

// newSpeedWindow creates a new speed window with a given capacity.
func newSpeedWindow(capacity int) *speedWindow {
	return &speedWindow{
		point: make([]speedPoint, capacity),
	}
}

// Add adds a done workload to the window.
func (w *speedWindow) Add(time time.Time, offset int64) {
	if w.size != len(w.point) {
		w.size++
	}
	w.point[w.next] = speedPoint{
		time:   time,
		offset: offset,
	}
	w.next = (w.next + 1) % len(w.point)
}

// Mean returns the mean speed of the window with unit of byte per second.
func (w *speedWindow) Mean() float64 {
	if w.size < 2 {
		// no speed diplayed for first read
		return 0
	}

	begin := (w.next - w.size + len(w.point)) % len(w.point)
	end := (begin - 1 + w.size) % w.size

	return float64(w.point[end].offset-w.point[begin].offset) / w.point[end].time.Sub(w.point[begin].time).Seconds()
}
