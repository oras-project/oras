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

package main

import (
	"os"

	"oras.land/oras/cmd/oras/repository"
)

func main() {
	cmd := repository.Cmd()
	cmd.SetArgs([]string{"tags", "195.17.139.162:30003/curated-packages/prometheus/prometheus", "--ca-file", "/Users/tlhowe/Library/Application Support/mkcert/rootCA.pem"})
	err := cmd.Execute()
	if err != nil {
		os.Exit(-1)
	}
	os.Exit(0)
}
