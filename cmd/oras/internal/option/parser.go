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

package option

import (
	"reflect"
)

// FlagParser parses flags in an option.
type FlagParser interface {
	Parse() error
}

// Parse parses applicable fields of the passed-in option pointer and returns
// error during parsing.
func Parse(optsPtr interface{}) error {
	return rangeFields(optsPtr, func(fp FlagParser) error {
		return fp.Parse()
	})
}

// rangeFields goes through all fields of ptr, optionally run fn if a field is
// public AND typed T.
func rangeFields[T any](ptr any, fn func(T) error) error {
	v := reflect.ValueOf(ptr).Elem()
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		if f.CanSet() {
			iface := f.Addr().Interface()
			if opts, ok := iface.(T); ok {
				if err := fn(opts); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
