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
	"errors"
	"reflect"
	"testing"
)

func TestAnnotation_Parse_manifestLevel(t *testing.T) {
	opts := Annotation{
		ManifestAnnotations: []string{"key=value", "foo=bar"},
	}
	if err := opts.Parse(nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]map[string]string{
		AnnotationManifest: {"key": "value", "foo": "bar"},
	}
	if !reflect.DeepEqual(opts.Annotations, want) {
		t.Fatalf("got %v, want %v", opts.Annotations, want)
	}
}

func TestAnnotation_Parse_layerLevel(t *testing.T) {
	opts := Annotation{
		ManifestAnnotations: []string{"file.tar:key=value", "image.bin:env=prod"},
	}
	if err := opts.Parse(nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]map[string]string{
		AnnotationManifest: {},
		"file.tar":         {"key": "value"},
		"image.bin":        {"env": "prod"},
	}
	if !reflect.DeepEqual(opts.Annotations, want) {
		t.Fatalf("got %v, want %v", opts.Annotations, want)
	}
}

func TestAnnotation_Parse_mixed(t *testing.T) {
	opts := Annotation{
		ManifestAnnotations: []string{"manifest-key=mval", "layer.tar:layer-key=lval"},
	}
	if err := opts.Parse(nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]map[string]string{
		AnnotationManifest: {"manifest-key": "mval"},
		"layer.tar":        {"layer-key": "lval"},
	}
	if !reflect.DeepEqual(opts.Annotations, want) {
		t.Fatalf("got %v, want %v", opts.Annotations, want)
	}
}

func TestAnnotation_Parse_multipleAnnotationsSameLayer(t *testing.T) {
	opts := Annotation{
		ManifestAnnotations: []string{"file.tar:k1=v1", "file.tar:k2=v2"},
	}
	if err := opts.Parse(nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]map[string]string{
		AnnotationManifest: {},
		"file.tar":         {"k1": "v1", "k2": "v2"},
	}
	if !reflect.DeepEqual(opts.Annotations, want) {
		t.Fatalf("got %v, want %v", opts.Annotations, want)
	}
}

func TestAnnotation_Parse_emptyAnnotations(t *testing.T) {
	opts := Annotation{}
	if err := opts.Parse(nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]map[string]string{
		AnnotationManifest: {},
	}
	if !reflect.DeepEqual(opts.Annotations, want) {
		t.Fatalf("got %v, want %v", opts.Annotations, want)
	}
}

func TestAnnotation_Parse_valueWithEquals(t *testing.T) {
	opts := Annotation{
		ManifestAnnotations: []string{"key=val=with=equals"},
	}
	if err := opts.Parse(nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.Annotations[AnnotationManifest]["key"] != "val=with=equals" {
		t.Fatalf("unexpected value: %v", opts.Annotations[AnnotationManifest]["key"])
	}
}

func TestAnnotation_Parse_errorMissingEquals(t *testing.T) {
	opts := Annotation{
		ManifestAnnotations: []string{"keyonly"},
	}
	if err := opts.Parse(nil); !errors.Is(err, errAnnotationFormat) {
		t.Fatalf("expected errAnnotationFormat, got: %v", err)
	}
}

func TestAnnotation_Parse_errorDuplicateManifestKey(t *testing.T) {
	opts := Annotation{
		ManifestAnnotations: []string{"key=val1", "key=val2"},
	}
	if err := opts.Parse(nil); !errors.Is(err, errAnnotationDuplication) {
		t.Fatalf("expected errAnnotationDuplication, got: %v", err)
	}
}

func TestAnnotation_Parse_errorDuplicateLayerKey(t *testing.T) {
	opts := Annotation{
		ManifestAnnotations: []string{"file.tar:key=val1", "file.tar:key=val2"},
	}
	if err := opts.Parse(nil); !errors.Is(err, errAnnotationDuplication) {
		t.Fatalf("expected errAnnotationDuplication, got: %v", err)
	}
}

func TestAnnotation_Parse_errorEmptyKeyAfterFilePrefix(t *testing.T) {
	opts := Annotation{
		ManifestAnnotations: []string{"file.tar:=value"},
	}
	if err := opts.Parse(nil); !errors.Is(err, errAnnotationLayerKey) {
		t.Fatalf("expected errAnnotationLayerKey, got: %v", err)
	}
}

func TestAnnotation_Parse_errorEmptyTarget(t *testing.T) {
	opts := Annotation{
		ManifestAnnotations: []string{":key=value"},
	}
	if err := opts.Parse(nil); !errors.Is(err, errAnnotationTarget) {
		t.Fatalf("expected errAnnotationTarget, got: %v", err)
	}
}

func TestAnnotation_Parse_configTarget(t *testing.T) {
	opts := Annotation{
		ManifestAnnotations: []string{"$config:hello=world"},
	}
	if err := opts.Parse(nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]map[string]string{
		AnnotationManifest: {},
		AnnotationConfig:   {"hello": "world"},
	}
	if !reflect.DeepEqual(opts.Annotations, want) {
		t.Fatalf("got %v, want %v", opts.Annotations, want)
	}
}

func TestAnnotation_Parse_manifestTargetMergesWithBareKeys(t *testing.T) {
	opts := Annotation{
		ManifestAnnotations: []string{"foo=bar", "$manifest:baz=qux"},
	}
	if err := opts.Parse(nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]map[string]string{
		AnnotationManifest: {"foo": "bar", "baz": "qux"},
	}
	if !reflect.DeepEqual(opts.Annotations, want) {
		t.Fatalf("got %v, want %v", opts.Annotations, want)
	}
}

func TestAnnotation_Parse_specialAndLayerTargets(t *testing.T) {
	opts := Annotation{
		ManifestAnnotations: []string{
			"top=level",
			"$config:cfg=on",
			"$manifest:mkey=mval",
			"file.tar:lkey=lval",
		},
	}
	if err := opts.Parse(nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]map[string]string{
		AnnotationManifest: {"top": "level", "mkey": "mval"},
		AnnotationConfig:   {"cfg": "on"},
		"file.tar":         {"lkey": "lval"},
	}
	if !reflect.DeepEqual(opts.Annotations, want) {
		t.Fatalf("got %v, want %v", opts.Annotations, want)
	}
}

func TestAnnotation_Parse_errorDuplicateAcrossManifestForms(t *testing.T) {
	opts := Annotation{
		ManifestAnnotations: []string{"key=val1", "$manifest:key=val2"},
	}
	if err := opts.Parse(nil); !errors.Is(err, errAnnotationDuplication) {
		t.Fatalf("expected errAnnotationDuplication, got: %v", err)
	}
}
