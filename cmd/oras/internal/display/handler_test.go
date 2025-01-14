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
	"reflect"
	"testing"

	"oras.land/oras/internal/testutils"

	"oras.land/oras/cmd/oras/internal/display/metadata/text"
	"oras.land/oras/cmd/oras/internal/display/status"
	"oras.land/oras/cmd/oras/internal/option"
	"oras.land/oras/cmd/oras/internal/output"
)

func TestNewPushHandler(t *testing.T) {
	mockFetcher := testutils.NewMockFetcher()
	printer := output.NewPrinter(os.Stdout, os.Stderr)
	_, _, err := NewPushHandler(printer, option.Format{Type: option.FormatTypeText.Name}, os.Stdout, mockFetcher.Fetcher)
	if err != nil {
		t.Errorf("NewPushHandler() error = %v, want nil", err)
	}
}

func TestNewAttachHandler(t *testing.T) {
	mockFetcher := testutils.NewMockFetcher()
	printer := output.NewPrinter(os.Stdout, os.Stderr)
	_, _, err := NewAttachHandler(printer, option.Format{Type: option.FormatTypeText.Name}, os.Stdout, mockFetcher.Fetcher)
	if err != nil {
		t.Errorf("NewAttachHandler() error = %v, want nil", err)
	}
}

func TestNewPullHandler(t *testing.T) {
	printer := output.NewPrinter(os.Stdout, os.Stderr)
	_, _, err := NewPullHandler(printer, option.Format{Type: option.FormatTypeText.Name}, "", os.Stdout)
	if err != nil {
		t.Errorf("NewPullHandler() error = %v, want nil", err)
	}
}

func TestNewCopyHandler(t *testing.T) {
	printer := output.NewPrinter(os.Stdout, os.Stderr)
	copyHandler, copyMetadataHandler := NewCopyHandler(printer, os.Stdout, nil)
	if _, ok := copyHandler.(*status.TTYCopyHandler); !ok {
		t.Errorf("expected *status.TTYCopyHandler actual %v", reflect.TypeOf(copyHandler))
	}
	if _, ok := copyMetadataHandler.(*text.CopyHandler); !ok {
		t.Errorf("expected metadata.CopyHandler actual %v", reflect.TypeOf(copyMetadataHandler))
	}
	copyHandler, copyMetadataHandler = NewCopyHandler(printer, nil, nil)
	if _, ok := copyHandler.(*status.TextCopyHandler); !ok {
		t.Errorf("expected *status.TextCopyHandler actual %v", reflect.TypeOf(copyHandler))
	}
	if _, ok := copyMetadataHandler.(*text.CopyHandler); !ok {
		t.Errorf("expected metadata.CopyHandler actual %v", reflect.TypeOf(copyMetadataHandler))
	}
}
