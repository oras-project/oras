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

package repository

import "testing"

func Test_parseRepoPath(t *testing.T) {
	type args struct {
		opts *repositoryOptions
		arg  string
	}
	var testOpts repositoryOptions
	tests := []struct {
		name          string
		args          args
		wantErr       bool
		wantHostname  string
		wantNamespace string
	}{
		{
			name:          "hostname only",
			args:          args{&testOpts, "testregistry.example.io"},
			wantErr:       false,
			wantHostname:  "testregistry.example.io",
			wantNamespace: "",
		},
		{
			name:          "hostname with trailing slash",
			args:          args{&testOpts, "testregistry.example.io/"},
			wantErr:       false,
			wantHostname:  "testregistry.example.io",
			wantNamespace: "",
		},
		{
			name:          "hostname and repo",
			args:          args{&testOpts, "testregistry.example.io/showcase"},
			wantErr:       false,
			wantHostname:  "testregistry.example.io",
			wantNamespace: "showcase/",
		},
		{
			name:          "hostname and repo in a sub-namespace",
			args:          args{&testOpts, "testregistry.example.io/showcase/beta"},
			wantErr:       false,
			wantHostname:  "testregistry.example.io",
			wantNamespace: "showcase/beta/",
		},
		{
			name:          "hostname and repo in a sub-namespace with trailing slash",
			args:          args{&testOpts, "testregistry.example.io/showcase/beta/"},
			wantErr:       false,
			wantHostname:  "testregistry.example.io",
			wantNamespace: "showcase/beta/",
		},
		{
			name:          "error when a tag is provided",
			args:          args{&testOpts, "testregistry.example.io/showcase:latest"},
			wantErr:       true,
			wantHostname:  "",
			wantNamespace: "",
		},
		{
			name:          "error when a digest is provided",
			args:          args{&testOpts, "testregistry.example.io/showcase:sha256:2e0e0fe1fb3edbcdddad941c90d2b51e25a6bcd593e82545441a216de7bfa834"},
			wantErr:       true,
			wantHostname:  "",
			wantNamespace: "",
		},
		{
			name:          "error when a malformed path is provided",
			args:          args{&testOpts, "testregistry.example.io///showcase/"},
			wantErr:       true,
			wantHostname:  "",
			wantNamespace: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := parseRepoPath(tt.args.opts, tt.args.arg)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseRepoPath() error = %v, wantErr %v", err, tt.wantErr)
			} else if testOpts.hostname != tt.wantHostname {
				t.Errorf("got incorrect hostname = %v, want %v", testOpts.hostname, tt.wantHostname)
			} else if testOpts.namespace != tt.wantNamespace {
				t.Errorf("got incorrect hostname = %v, want %v", testOpts.namespace, tt.wantNamespace)
			}
			testOpts.hostname = ""
			testOpts.namespace = ""
		})
	}
}
