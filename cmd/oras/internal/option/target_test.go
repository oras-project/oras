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
	"testing"

	"github.com/spf13/cobra"
)

func TestTarget_Parse_oci(t *testing.T) {
	opts := Target{IsOCILayout: true}

	if err := opts.Parse(); err != nil {
		t.Errorf("Target.Parse() error = %v", err)
	}
	if opts.Type != TargetTypeOCILayout {
		t.Errorf("Target.Parse() failed, got %q, want %q", opts.Type, TargetTypeOCILayout)
	}
}

func TestTarget_Parse_remote(t *testing.T) {
	opts := Target{IsOCILayout: false}
	if err := opts.Parse(); err != nil {
		t.Errorf("Target.Parse() error = %v", err)
	}
	if opts.Type != TargetTypeRemote {
		t.Errorf("Target.Parse() failed, got %q, want %q", opts.Type, TargetTypeRemote)
	}
}

func Test_parseOCILayoutReference(t *testing.T) {
	type args struct {
		raw string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		want1   string
		wantErr bool
	}{
		{"Empty input", args{raw: ""}, "", "", true},
		{"Empty path and tag", args{raw: ":"}, "", "", true},
		{"Empty path and digest", args{raw: "@"}, "", "", false},
		{"Empty digest", args{raw: "path@"}, "path", "", false},
		{"Empty tag", args{raw: "path:"}, "path", "", false},
		{"path and digest", args{raw: "path@digest"}, "path", "digest", false},
		{"path and tag", args{raw: "path:tag"}, "path", "tag", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := parseOCILayoutReference(tt.args.raw)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseOCILayoutReference() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseOCILayoutReference() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("parseOCILayoutReference() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestTarget_Modify_ociLayout(t *testing.T) {
	errClient := errors.New("client error")
	opts := &Target{
		IsOCILayout: true,
	}
	got, modified := opts.Modify(&cobra.Command{}, errClient)

	if modified {
		t.Errorf("expect error not to be modified but received true")
	}
	if got != errClient {
		t.Errorf("unexpected output from Target.Process() = %v", got)
	}
}

// func TestTarget_Process_hint(t *testing.T) {
// 	type fields struct {
// 		Remote       Remote
// 		RawReference string
// 		Type         string
// 		Reference    string
// 		Path         string
// 		IsOCILayout  bool
// 	}
// 	type args struct {
// 		err      error
// 		callPath string
// 	}
// 	errs := errcode.Errors{
// 		errcode.Error{
// 			Code:    "000",
// 			Message: "mocked message",
// 			Detail:  map[string]string{"mocked key": "mocked value"},
// 		},
// 	}
// 	tests := []struct {
// 		name   string
// 		fields fields
// 		args   args
// 		want   *oerrors.Error
// 	}{
// 		{
// 			"namespace already exists",
// 			fields{RawReference: "docker.io/library/alpine:latest"},
// 			args{
// 				err: &errcode.ErrorResponse{
// 					URL:        &url.URL{Host: "registry-1.docker.io"},
// 					StatusCode: http.StatusUnauthorized,
// 					Errors:     errs,
// 				},
// 			},
// 			&oerrors.Error{Err: errs},
// 		},
// 		{
// 			"no namespace",
// 			fields{RawReference: "docker.io"},
// 			args{
// 				err: &errcode.ErrorResponse{
// 					URL:        &url.URL{Host: "registry-1.docker.io"},
// 					StatusCode: http.StatusUnauthorized,
// 					Errors:     errs,
// 				},
// 			},
// 			&oerrors.Error{Err: errs},
// 		},
// 		{
// 			"not 401",
// 			fields{RawReference: "docker.io"},
// 			args{
// 				err: &errcode.ErrorResponse{
// 					URL:        &url.URL{Host: "registry-1.docker.io"},
// 					StatusCode: http.StatusConflict,
// 					Errors:     errs,
// 				},
// 			},
// 			&oerrors.Error{Err: errs},
// 		},
// 		{
// 			"should hint",
// 			fields{
// 				RawReference: "docker.io/alpine",
// 				Path:         "oras test",
// 			},
// 			args{
// 				err: &errcode.ErrorResponse{
// 					URL:        &url.URL{Host: "registry-1.docker.io"},
// 					StatusCode: http.StatusUnauthorized,
// 					Errors:     errs,
// 				},
// 				callPath: "oras test",
// 			},
// 			&oerrors.Error{
// 				Err:            errs,
// 				Recommendation: "Namespace is missing, do you mean `oras test docker.io/library/alpine`?",
// 			},
// 		},
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			opts := &Target{
// 				Remote:       tt.fields.Remote,
// 				RawReference: tt.fields.RawReference,
// 				Type:         tt.fields.Type,
// 				Reference:    tt.fields.Reference,
// 				Path:         tt.fields.Path,
// 				IsOCILayout:  tt.fields.IsOCILayout,
// 			}
// 			got := opts.Process(tt.args.err, tt.args.callPath)
// 			if !reflect.DeepEqual(got, tt.want) {
// 				t.Errorf("Target.Process() = %v, want %v", got, tt.want)
// 			}
// 		})
// 	}
// }
