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

package progress

import (
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"testing"
)

func Test_Messenger(t *testing.T) {
	var msg *status
	messenger := NewMessenger("Action", "Done")

	messenger.Start()
	select {
	case msg = <-messenger.ch:
		if msg.offset != -1 {
			t.Errorf("Expected start message with offset -1, got %d", msg.offset)
		}
	default:
		t.Error("Expected start message")
	}

	desc := v1.Descriptor{
		Digest: "mouse",
		Size:   100,
	}
	expected := int64(50)
	messenger.Send("Reading", desc, expected)
	select {
	case msg = <-messenger.ch:
		if msg.offset != expected {
			t.Errorf("Expected status message with offset %d, got %d", expected, msg.offset)
		}
		if msg.prompt != "Reading" {
			t.Errorf("Expected status message prompt Reading, got %s", msg.prompt)
		}
	default:
		t.Error("Expected status message")
	}

	messenger.Send("Reading", desc, expected)
	messenger.Send("Read", desc, desc.Size)
	select {
	case msg = <-messenger.ch:
		if msg.offset != desc.Size {
			t.Errorf("Expected status message with offset %d, got %d", expected, msg.offset)
		}
		if msg.prompt != "Read" {
			t.Errorf("Expected status message prompt Read, got %s", msg.prompt)
		}
	default:
		t.Error("Expected status message")
	}
	select {
	case msg = <-messenger.ch:
		t.Errorf("Unexpected status message %v", msg)
	default:
	}

	messenger.SendAction(desc, expected)
	select {
	case msg = <-messenger.ch:
		if msg.offset != expected {
			t.Errorf("Expected status message with offset %d, got %d", expected, msg.offset)
		}
		if msg.prompt != "Action" {
			t.Errorf("Expected status message prompt Action, got %s", msg.prompt)
		}
	default:
		t.Error("Expected status message")
	}

	expected += 1
	messenger.SendDone(desc, expected)
	select {
	case msg = <-messenger.ch:
		if msg.offset != expected {
			t.Errorf("Expected status message with offset %d, got %d", expected, msg.offset)
		}
		if msg.prompt != "Done" {
			t.Errorf("Expected status message prompt Done, got %s", msg.prompt)
		}
	default:
		t.Error("Expected status message")
	}

	expected = int64(-1)
	messenger.Stop()
	select {
	case msg = <-messenger.ch:
		if msg.offset != expected {
			t.Errorf("Expected END status message with offset %d, got %d", expected, msg.offset)
		}
	default:
		t.Error("Expected END status message")
	}

	messenger.Stop()
	select {
	case msg = <-messenger.ch:
		if msg != nil {
			t.Errorf("Unexpected status message %v", msg)
		}
	default:
	}
}
