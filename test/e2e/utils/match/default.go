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

package match

// Matching error execution.
var Error = Option{nil, nil, nil, true}

// Matching success execution.
var Success = Option{nil, nil, nil, false}

// ErrorKeywords returns an option for matching stderr keywords in failure
// execution.
func ErrorKeywords(keywords ...string) *Option {
	return NewOption(nil, nil, Keywords(keywords), true)
}

// SuccessKeywords returns an option for matching stdout keywords in success
// execution.
func SuccessKeywords(keywords ...string) *Option {
	return NewOption(nil, Keywords(keywords), nil, false)
}

// SuccessContent returns an option for matching stdout content in success
// execution.
func SuccessContent(s *string) *Option {
	return NewOption(nil, Content{s}, nil, false)
}

func MatchableStatus(cmd string, verbose bool) (opts *StatusOption) {
	opts = NewStatusOption(verbose)

	// prepare edge
	switch cmd {
	case "push", "attach":
		opts.addPath("Uploading", "Uploaded")
		opts.addPath("Existed")
		opts.addPath("Skipped")
	default:
		panic("Unrecognized cmd name " + cmd)
	}
	return opts
}
