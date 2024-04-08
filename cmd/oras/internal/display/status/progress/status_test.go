//go:build freebsd || linux || netbsd || openbsd || solaris

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
	"testing"
	"time"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras/cmd/oras/internal/display/status/console"
	"oras.land/oras/cmd/oras/internal/display/status/console/testutils"
	"oras.land/oras/cmd/oras/internal/display/status/progress/humanize"
)

func Test_status_String(t *testing.T) {
	// zero status and progress
	s := newStatus()
	if status, digest := s.String(console.MinWidth); status != zeroStatus || digest != zeroDigest {
		t.Errorf("status.String() = %v, %v, want %v, %v", status, digest, zeroStatus, zeroDigest)
	}

	// not done
	s.Update(&status{
		prompt: "test",
		descriptor: ocispec.Descriptor{
			MediaType: "application/vnd.oci.empty.oras.test.v1+json",
			Size:      2,
			Digest:    "sha256:44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a",
		},
		startTime: time.Now().Add(-time.Minute),
		offset:    0,
		total:     humanize.ToBytes(2),
	})
	// full name
	statusStr, digestStr := s.String(120)
	if err := testutils.OrderedMatch(statusStr+digestStr, " [\x1b[7m\x1b[0m....................]", s.prompt, s.descriptor.MediaType, "0.00/2  B", "0.00%", s.descriptor.Digest.String()); err != nil {
		t.Error(err)
	}
	// partial name
	statusStr, digestStr = s.String(console.MinWidth)
	if err := testutils.OrderedMatch(statusStr+digestStr, " [\x1b[7m\x1b[0m....................]", s.prompt, "application/v.", "0.00/2  B", "0.00%", s.descriptor.Digest.String()); err != nil {
		t.Error(err)
	}
	// done
	s.Update(&status{
		endTime:    time.Now(),
		offset:     s.descriptor.Size,
		descriptor: s.descriptor,
	})
	statusStr, digestStr = s.String(120)
	if err := testutils.OrderedMatch(statusStr+digestStr, "✓", s.prompt, s.descriptor.MediaType, "2/2  B", "100.00%", s.descriptor.Digest.String()); err != nil {
		t.Error(err)
	}
}

func Test_status_String_zeroWitdth(t *testing.T) {
	// zero status and progress
	s := newStatus()
	if status, digest := s.String(console.MinWidth); status != zeroStatus || digest != zeroDigest {
		t.Errorf("status.String() = %v, %v, want %v, %v", status, digest, zeroStatus, zeroDigest)
	}

	// not done
	s.Update(&status{
		prompt: "test",
		descriptor: ocispec.Descriptor{
			MediaType: "application/vnd.oci.empty.oras.test.v1+json",
			Size:      0,
			Digest:    "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
		startTime: time.Now().Add(-time.Minute),
		offset:    0,
		total:     humanize.ToBytes(0),
	})
	// not done
	statusStr, digestStr := s.String(120)
	if err := testutils.OrderedMatch(statusStr+digestStr, " [\x1b[7m\x1b[0m....................]", s.prompt, s.descriptor.MediaType, "0.00/0  B", "0.00%", s.descriptor.Digest.String()); err != nil {
		t.Error(err)
	}
	// done
	s.Update(&status{
		endTime:    time.Now(),
		offset:     s.descriptor.Size,
		descriptor: s.descriptor,
	})
	statusStr, digestStr = s.String(120)
	if err := testutils.OrderedMatch(statusStr+digestStr, "✓", s.prompt, s.descriptor.MediaType, "0/0  B", "100.00%", s.descriptor.Digest.String()); err != nil {
		t.Error(err)
	}
}
func Test_status_durationString(t *testing.T) {
	// zero duration
	s := newStatus()
	if d := s.durationString(); d != zeroDuration {
		t.Errorf("status.durationString() = %v, want %v", d, zeroDuration)
	}

	// not ended
	s.startTime = time.Now().Add(-time.Second)
	if d := s.durationString(); d == zeroDuration {
		t.Errorf("status.durationString() = %v, want not %v", d, zeroDuration)
	}

	// ended: 61 seconds
	s.startTime = time.Now()
	s.endTime = s.startTime.Add(61 * time.Second)
	want := "1m1s"
	if d := s.durationString(); d != want {
		t.Errorf("status.durationString() = %v, want %v", d, want)
	}

	// ended: 1001 Microsecond
	s.startTime = time.Now()
	s.endTime = s.startTime.Add(1001 * time.Microsecond)
	want = "1ms"
	if d := s.durationString(); d != want {
		t.Errorf("status.durationString() = %v, want %v", d, want)
	}

	// ended: 1001 Nanosecond
	s.startTime = time.Now()
	s.endTime = s.startTime.Add(1001 * time.Nanosecond)
	want = "1µs"
	if d := s.durationString(); d != want {
		t.Errorf("status.durationString() = %v, want %v", d, want)
	}
}
