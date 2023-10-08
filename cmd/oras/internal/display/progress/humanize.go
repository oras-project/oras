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
	"math"
)

var (
	units = []string{"B", "kB", "MB", "GB", "TB"}
	base  = 1000.0
)

type bytes struct {
	size float64
	unit string
}

// ToBytes converts size in bytes to human readable format.
func ToBytes(sizeInBytes int64) bytes {
	f := float64(sizeInBytes)
	if f < base {
		return bytes{f, "B"}
	}
	e := math.Floor(math.Log(f) / math.Log(base))
	p := f / math.Pow(base, e)
	return bytes{RoundTo(p), units[int(e)]}
}

// RoundTo makes length of the size string to less than or equal to 4.
func RoundTo(size float64) float64 {
	if size < 10 {
		return math.Round(size*100) / 100
	} else if size < 100 {
		return math.Round(size*10) / 10
	}
	return math.Round(size)
}
