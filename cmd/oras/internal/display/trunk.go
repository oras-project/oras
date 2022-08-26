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

/*
REF
golang: https://go.dev/blog/slices-intro
client:
	https://docs.microsoft.com/en-us/azure/governance/resource-graph/concepts/work-with-data#paging-results
	https://www.eternalsoftsolutions.com/blog/what-is-aws-cli-pagination/
server: https://github.com/Azure/azure-resource-manager-rpc/blob/master/v1.0/resource-api-reference.md#pagination
*/

package display

// filter process the input with the given keywords
func Filter(input []string, startwith, endwith, contains string) []string {
	return nil
}

// cut help size the input
func Cut(input []string, first, skip int) []string {
	if skip >= cap(input) {
		return []string{}
	}
	if skip+first >= cap(input) {
		return input[skip:]
	}
	return input[skip : skip+first]
}

// pagination the large slice input
func Pagination(input []string, nextIndex int) ([]string, int) {
	return nil, 0
}
