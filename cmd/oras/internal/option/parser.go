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

type FlagParser interface {
	ParseFlags() error
}

func ParseFlags(optsPtr interface{}) error {
	v := reflect.ValueOf(optsPtr).Elem()
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		if f.CanSet() {
			iface := f.Addr().Interface()
			if a, ok := iface.(FlagParser); ok {
				if err := a.ParseFlags(); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
