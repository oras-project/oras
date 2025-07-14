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
	"oras.land/oras-go/v2"
)

func TestBinaryTarget_ModifyError(t *testing.T) {
	testCases := []struct {
		name         string
		target       *BinaryTarget
		err          error
		wantModified bool
		wantPrefix   string
		wantErr      error
	}{
		{
			name: "CopyError with Source origin sets prefix",
			target: &BinaryTarget{
				From: Target{
					Type:         "registry",
					RawReference: "localhost:5000/test:v1",
				},
				To: Target{
					Type:         "oci-layout",
					RawReference: "oci-dir:v1",
				},
			},
			err:          &oras.CopyError{Origin: oras.CopyErrorOriginSource, Err: errors.New("source error")},
			wantModified: true,
			wantPrefix:   `Error from source registry for "localhost:5000/test:v1":`,
			wantErr:      errors.New("source error"),
		},
		{
			name: "CopyError with Destination origin sets prefix",
			target: &BinaryTarget{
				From: Target{
					Type:         "registry",
					RawReference: "localhost:5000/test:v1",
				},
				To: Target{
					Type:         "oci-layout",
					RawReference: "oci-dir:v1",
				},
			},
			err:          &oras.CopyError{Origin: oras.CopyErrorOriginDestination, Err: errors.New("destination error")},
			wantModified: true,
			wantPrefix:   `Error from destination oci-layout for "oci-dir:v1":`,
			wantErr:      errors.New("destination error"),
		},
		{
			name: "CopyError with unknown origin",
			target: &BinaryTarget{
				From: Target{
					Type:         "registry",
					RawReference: "localhost:5000/test:v1",
				},
				To: Target{
					Type:         "oci-layout",
					RawReference: "oci-dir:v1",
				},
			},
			err:          &oras.CopyError{Origin: oras.CopyErrorOrigin(-1), Err: errors.New("unknown error")},
			wantPrefix:   "Error:",
			wantModified: true,
			wantErr:      errors.New("unknown error"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			err, modified := tc.target.ModifyError(cmd, tc.err)
			if modified != tc.wantModified {
				t.Errorf("ModifyError() modified = %v, want %v", modified, tc.wantModified)
			}
			if modified && cmd.ErrPrefix() != tc.wantPrefix {
				t.Errorf("ModifyError() cmd.ErrPrefix() = %q, want %q", cmd.ErrPrefix(), tc.wantPrefix)
			}
			if err.Error() != tc.wantErr.Error() {
				t.Errorf("ModifyError() error = %q, want %q", err.Error(), tc.wantErr.Error())
			}
		})
	}
}
