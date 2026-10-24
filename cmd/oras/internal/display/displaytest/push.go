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

package displaytest

import (
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras/cmd/oras/internal/display/metadata"
	"oras.land/oras/cmd/oras/internal/display/status"
	"oras.land/oras/cmd/oras/internal/option"
)

// CallRecorder records handler calls in the order they occur.
type CallRecorder struct {
	Calls []string
}

// Record appends a call to the log.
func (r *CallRecorder) Record(call string) {
	r.Calls = append(r.Calls, call)
}

// RecordingPushStatusHandler records push status handler calls.
type RecordingPushStatusHandler struct {
	status.DiscardHandler
	Recorder       *CallRecorder
	FileLoadingErr error
}

// OnFileLoading records the file-loading event and returns the configured error.
func (h *RecordingPushStatusHandler) OnFileLoading(_ string) error {
	h.Recorder.Record("OnFileLoading")
	return h.FileLoadingErr
}

// RecordingPushMetadataHandler records push metadata handler calls.
type RecordingPushMetadataHandler struct {
	metadata.Discard
	Recorder  *CallRecorder
	CopiedErr error
	TaggedErr error
	RenderErr error
}

// OnCopied records the copied event and returns the configured error.
func (h *RecordingPushMetadataHandler) OnCopied(
	_ *option.Target,
	_ ocispec.Descriptor,
) error {
	h.Recorder.Record("OnCopied")
	return h.CopiedErr
}

// OnTagged records the tag event and returns the configured error.
func (h *RecordingPushMetadataHandler) OnTagged(
	_ ocispec.Descriptor,
	tag string,
) error {
	h.Recorder.Record("OnTagged:" + tag)
	return h.TaggedErr
}

// Render records the render call and returns the configured error.
func (h *RecordingPushMetadataHandler) Render() error {
	h.Recorder.Record("Render")
	return h.RenderErr
}
