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

package display

import (
	"os"
	"testing"

	"oras.land/oras/cmd/oras/internal/option"
)

func TestNewPushHandler(t *testing.T) {
	_, _, err := NewPushHandler(os.Stdout, option.Format{Type: option.FormatTypeText.Name}, os.Stdout, false)
	if err != nil {
		t.Errorf("NewPushHandler() error = %v, want nil", err)
	}
}

func TestNewAttachHandler(t *testing.T) {
	_, _, err := NewAttachHandler(os.Stdout, option.Format{Type: option.FormatTypeText.Name}, os.Stdout, false)
	if err != nil {
		t.Errorf("NewAttachHandler() error = %v, want nil", err)
	}
}

func TestNewPullHandler(t *testing.T) {
	_, _, err := NewPullHandler(os.Stdout, option.Format{Type: option.FormatTypeText.Name}, "", os.Stdout, false)
	if err != nil {
		t.Errorf("NewPullHandler() error = %v, want nil", err)
	}
}
