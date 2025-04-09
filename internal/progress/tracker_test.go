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
	"bytes"
	"errors"
	"io"
	"testing"
)

func TestTrackerFunc_Close(t *testing.T) {
	var f TrackerFunc
	if err := f.Close(); err != nil {
		t.Errorf("TrackerFunc.Close() error = %v, wantErr false", err)
	}
}

func TestTrackerFunc_Update(t *testing.T) {
	wantStatus := Status{
		State:  StateTransmitted,
		Offset: 42,
	}
	var wantErr error
	tracker := TrackerFunc(func(status Status, err error) error {
		if status != wantStatus {
			t.Errorf("TrackerFunc status = %v, want %v", status, wantStatus)
		}
		if err != nil {
			t.Errorf("TrackerFunc err = %v, want nil", err)
		}
		return wantErr
	})

	if err := tracker.Update(wantStatus); err != wantErr {
		t.Errorf("TrackerFunc.Update() error = %v, want %v", err, wantErr)
	}

	wantErr = errors.New("fail to track")
	if err := tracker.Update(wantStatus); err != wantErr {
		t.Errorf("TrackerFunc.Update() error = %v, want %v", err, wantErr)
	}
}

func TestTrackerFunc_Fail(t *testing.T) {
	reportErr := errors.New("fail to process")
	var wantStatus Status
	var wantErr error
	tracker := TrackerFunc(func(status Status, err error) error {
		if status != wantStatus {
			t.Errorf("TrackerFunc status = %v, want %v", status, wantStatus)
		}
		if err != reportErr {
			t.Errorf("TrackerFunc err = %v, want %v", err, reportErr)
		}
		return wantErr
	})

	if err := tracker.Fail(reportErr); err != wantErr {
		t.Errorf("TrackerFunc.Fail() error = %v, want %v", err, wantErr)
	}

	wantErr = errors.New("fail to track")
	if err := tracker.Fail(reportErr); err != wantErr {
		t.Errorf("TrackerFunc.Fail() error = %v, want %v", err, wantErr)
	}
}

func TestStart(t *testing.T) {
	tests := []struct {
		name    string
		t       Tracker
		wantErr bool
	}{
		{
			name: "successful report initialization",
			t: TrackerFunc(func(status Status, err error) error {
				if status.State != StateInitialized {
					t.Errorf("expected state to be StateInitialized, got %v", status.State)
				}
				return nil
			}),
		},
		{
			name: "fail to report initialization",
			t: TrackerFunc(func(status Status, err error) error {
				return errors.New("fail to track")
			}),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := Start(tt.t); (err != nil) != tt.wantErr {
				t.Errorf("Start() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDone(t *testing.T) {
	tests := []struct {
		name    string
		t       Tracker
		wantErr bool
	}{
		{
			name: "successful report initialization",
			t: TrackerFunc(func(status Status, err error) error {
				if status.State != StateTransmitted {
					t.Errorf("expected state to be StateTransmitted, got %v", status.State)
				}
				return nil
			}),
		},
		{
			name: "fail to report initialization",
			t: TrackerFunc(func(status Status, err error) error {
				return errors.New("fail to track")
			}),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := Done(tt.t); (err != nil) != tt.wantErr {
				t.Errorf("Done() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTrackReader(t *testing.T) {
	const bufSize = 6
	content := []byte("hello world")
	t.Run("track io.Reader", func(t *testing.T) {
		var wantStatus Status
		tracker := TrackerFunc(func(status Status, err error) error {
			if status != wantStatus {
				t.Errorf("TrackerFunc status = %v, want %v", status, wantStatus)
			}
			if err != nil {
				t.Errorf("TrackerFunc err = %v, want nil", err)
			}
			return nil
		})
		var reader io.Reader = bytes.NewReader(content)
		reader = io.LimitReader(reader, int64(len(content))) // remove the io.WriterTo interface
		gotReader := TrackReader(tracker, reader)
		if _, ok := gotReader.(*readTracker); !ok {
			t.Fatalf("TrackReader() = %v, want *readTracker", gotReader)
		}

		wantStatus = Status{
			State:  StateTransmitting,
			Offset: bufSize,
		}
		buf := make([]byte, bufSize)
		n, err := gotReader.Read(buf)
		if err != nil {
			t.Fatalf("TrackReader() error = %v, want nil", err)
		}
		if n != bufSize {
			t.Fatalf("TrackReader() n = %v, want %v", n, bufSize)
		}
		if want := content[:bufSize]; !bytes.Equal(buf, want) {
			t.Fatalf("TrackReader() buf = %v, want %v", buf, want)
		}

		wantStatus = Status{
			State:  StateTransmitting,
			Offset: int64(len(content)),
		}
		n, err = gotReader.Read(buf)
		if err != nil {
			t.Fatalf("TrackReader() error = %v, want nil", err)
		}
		if want := len(content) - bufSize; n != want {
			t.Fatalf("TrackReader() n = %v, want %v", n, want)
		}
		buf = buf[:n]
		if want := content[bufSize:]; !bytes.Equal(buf, want) {
			t.Fatalf("TrackReader() buf = %v, want %v", buf, want)
		}
	})

	t.Run("track io.Reader + io.WriterTo", func(t *testing.T) {
		var wantStatus Status
		tracker := TrackerFunc(func(status Status, err error) error {
			if status != wantStatus {
				t.Errorf("TrackerFunc status = %v, want %v", status, wantStatus)
			}
			if err != nil {
				t.Errorf("TrackerFunc err = %v, want nil", err)
			}
			return nil
		})
		var reader io.Reader = bytes.NewReader(content)
		gotReader := TrackReader(tracker, reader)
		if _, ok := gotReader.(*readTrackerWriteTo); !ok {
			t.Fatalf("TrackReader() = %v, want *readTrackerWriteTo", gotReader)
		}

		wantStatus = Status{
			State:  StateTransmitting,
			Offset: bufSize,
		}
		buf := make([]byte, bufSize)
		n, err := gotReader.Read(buf)
		if err != nil {
			t.Fatalf("TrackReader() error = %v, want nil", err)
		}
		if n != bufSize {
			t.Fatalf("TrackReader() n = %v, want %v", n, bufSize)
		}
		if want := content[:bufSize]; !bytes.Equal(buf, want) {
			t.Fatalf("TrackReader() buf = %v, want %v", buf, want)
		}

		wantStatus = Status{
			State:  StateTransmitting,
			Offset: int64(len(content)),
		}
		writeBuf := bytes.NewBuffer(nil)
		wn, err := gotReader.(io.WriterTo).WriteTo(writeBuf)
		if err != nil {
			t.Fatalf("TrackReader() error = %v, want nil", err)
		}
		if want := len(content) - bufSize; wn != int64(want) {
			t.Fatalf("TrackReader() n = %v, want %v", wn, want)
		}
		buf = writeBuf.Bytes()
		if want := content[bufSize:]; !bytes.Equal(buf, want) {
			t.Fatalf("TrackReader() buf = %v, want %v", buf, want)
		}
	})

	t.Run("empty io.Reader", func(t *testing.T) {
		tracker := TrackerFunc(func(status Status, err error) error {
			t.Errorf("TrackerFunc should not be called for empty read")
			return nil
		})
		gotReader := TrackReader(tracker, bytes.NewReader(nil))

		buf := make([]byte, bufSize)
		n, err := gotReader.Read(buf)
		if want := io.EOF; err != want {
			t.Fatalf("TrackReader() error = %v, want %v", err, want)
		}
		if want := 0; n != want {
			t.Fatalf("TrackReader() n = %v, want %v", n, want)
		}

		writeBuf := bytes.NewBuffer(nil)
		wn, err := gotReader.(io.WriterTo).WriteTo(writeBuf)
		if err != nil {
			t.Fatalf("TrackReader() error = %v, want nil", err)
		}
		if want := int64(0); wn != want {
			t.Fatalf("TrackReader() n = %v, want %v", wn, want)
		}
		buf = writeBuf.Bytes()
		if want := []byte{}; !bytes.Equal(buf, want) {
			t.Fatalf("TrackReader() buf = %v, want %v", buf, want)
		}
	})

	t.Run("report failure", func(t *testing.T) {
		var wantStatus Status
		wantErr := errors.New("fail to track")
		trackerMockStage := 0
		tracker := TrackerFunc(func(status Status, err error) error {
			defer func() {
				trackerMockStage++
			}()
			switch trackerMockStage {
			case 0:
				if status != wantStatus {
					t.Errorf("TrackerFunc status = %v, want %v", status, wantStatus)
				}
				if err != nil {
					t.Errorf("TrackerFunc err = %v, want nil", err)
				}
				return wantErr
			case 1:
				var emptyStatus Status
				if wantStatus := emptyStatus; status != wantStatus {
					t.Errorf("TrackerFunc status = %v, want %v", status, wantStatus)
				}
				if err != wantErr {
					t.Errorf("TrackerFunc err = %v, want %v", err, wantErr)
				}
				return nil
			default:
				t.Errorf("TrackerFunc should not be called")
				return nil
			}
		})
		gotReader := TrackReader(tracker, bytes.NewReader(content))

		wantStatus = Status{
			State:  StateTransmitting,
			Offset: bufSize,
		}
		buf := make([]byte, bufSize)
		n, err := gotReader.Read(buf)
		if err != wantErr {
			t.Fatalf("TrackReader() error = %v, want %v", err, wantErr)
		}
		if n != bufSize {
			t.Fatalf("TrackReader() n = %v, want %v", n, bufSize)
		}
		if want := content[:bufSize]; !bytes.Equal(buf, want) {
			t.Fatalf("TrackReader() buf = %v, want %v", buf, want)
		}

		wantStatus = Status{
			State:  StateTransmitting,
			Offset: int64(len(content)),
		}
		trackerMockStage = 0
		writeBuf := bytes.NewBuffer(nil)
		wn, err := gotReader.(io.WriterTo).WriteTo(writeBuf)
		if err != wantErr {
			t.Fatalf("TrackReader() error = %v, want %v", err, wantErr)
		}
		if want := len(content) - bufSize; wn != int64(want) {
			t.Fatalf("TrackReader() n = %v, want %v", wn, want)
		}
		buf = writeBuf.Bytes()
		if want := content[bufSize:]; !bytes.Equal(buf, want) {
			t.Fatalf("TrackReader() buf = %v, want %v", buf, want)
		}
	})

	t.Run("process failure", func(t *testing.T) {
		reportErr := io.ErrClosedPipe
		var wantStatus Status
		var wantErr error
		tracker := TrackerFunc(func(status Status, err error) error {
			if status != wantStatus {
				t.Errorf("TrackerFunc status = %v, want %v", status, wantStatus)
			}
			if err != reportErr {
				t.Errorf("TrackerFunc err = %v, want %v", err, reportErr)
			}
			return wantErr
		})
		pipeReader, pipeWriter := io.Pipe()
		_ = pipeReader.Close()
		_ = pipeWriter.Close()
		gotReader := TrackReader(tracker, pipeReader)

		buf := make([]byte, bufSize)
		n, err := gotReader.Read(buf)
		if err != reportErr {
			t.Fatalf("TrackReader() error = %v, want %v", err, reportErr)
		}
		if want := 0; n != want {
			t.Fatalf("TrackReader() n = %v, want %v", n, want)
		}

		wantErr = errors.New("fail to track")
		n, err = gotReader.Read(buf)
		if err != wantErr {
			t.Fatalf("TrackReader() error = %v, want %v", err, wantErr)
		}
		if want := 0; n != want {
			t.Fatalf("TrackReader() n = %v, want %v", n, want)
		}

		gotReader = TrackReader(tracker, io.MultiReader(pipeReader)) // wrap io.WriteTo
		wantErr = nil
		writeBuf := bytes.NewBuffer(nil)
		wn, err := gotReader.(io.WriterTo).WriteTo(writeBuf)
		if err != reportErr {
			t.Fatalf("TrackReader() error = %v, want %v", err, reportErr)
		}
		if want := int64(0); wn != want {
			t.Fatalf("TrackReader() n = %v, want %v", wn, want)
		}
		buf = writeBuf.Bytes()
		if want := []byte{}; !bytes.Equal(buf, want) {
			t.Fatalf("TrackReader() buf = %v, want %v", buf, want)
		}

		gotReader = TrackReader(tracker, io.MultiReader(pipeReader)) // wrap io.WriteTo
		wantErr = errors.New("fail to track")
		wn, err = gotReader.(io.WriterTo).WriteTo(writeBuf)
		if err != wantErr {
			t.Fatalf("TrackReader() error = %v, want %v", err, wantErr)
		}
		if want := int64(0); wn != want {
			t.Fatalf("TrackReader() n = %v, want %v", wn, want)
		}
		buf = writeBuf.Bytes()
		if want := []byte{}; !bytes.Equal(buf, want) {
			t.Fatalf("TrackReader() buf = %v, want %v", buf, want)
		}
	})
}
