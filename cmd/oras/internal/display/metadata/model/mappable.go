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

package model

import (
	"reflect"
	"strings"
)

// ToMappable converts an reflect value into a map[string]any with json tags as
// key.
func ToMappable(v reflect.Value) any {
	switch v.Kind() {
	case reflect.Struct:
		ret := make(map[string]any)
		addToMap(ret, v)
		return ret
	case reflect.Slice, reflect.Array:
		ret := make([]any, v.Len())
		for i := 0; i < v.Len(); i++ {
			ret[i] = ToMappable(v.Index(i))
		}
		return ret
	case reflect.Ptr, reflect.Interface:
		if v.IsNil() {
			return nil
		}
		elem := ToMappable(v.Elem())
		return &elem
	}
	return v.Interface()
}

func addToMap(ret map[string]any, v reflect.Value) {
	t := v.Type()
	numField := t.NumField()
	for i := 0; i < numField; i++ {
		fv := v.Field(i)
		ft := t.Field(i)
		tag := ft.Tag.Get("json")
		if tag == "" {
			addToMap(ret, fv)
		} else {
			key, _, _ := strings.Cut(tag, ",")
			ret[key] = ToMappable(fv)
		}
	}
}
