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
	"iter"
	"reflect"

	"github.com/spf13/cobra"
)

// FlagParser parses flags in an option.
type FlagParser interface {
	Parse(cmd *cobra.Command) error
}

// Parse parses applicable fields of the passed-in option pointer and returns
// error during parsing.
func Parse(cmd *cobra.Command, optsPtr any) error {
	for parser := range fields[FlagParser](optsPtr) {
		if err := parser.Parse(cmd); err != nil {
			return err
		}
	}
	return nil
}

// fields returns an iterator that yields all fields of the given struct that
// implement the given interface.
func fields[T any](ptr any) iter.Seq[T] {
	return func(yield func(T) bool) {
		v := reflect.ValueOf(ptr).Elem()
		for i := range v.NumField() {
			f := v.Field(i)
			if f.CanSet() {
				if opts, ok := f.Addr().Interface().(T); ok {
					if !yield(opts) {
						return
					}
				}
			}
		}
	}
}
