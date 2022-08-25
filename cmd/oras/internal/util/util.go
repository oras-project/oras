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

package util

import (
	"fmt"
	"strings"
)

func AskForComfirmation() (bool, error) {
	var response string
	_, err := fmt.Scanln(&response)
	if err != nil {
		return false, err
	}

	switch strings.ToLower(response) {
	case "y", "yes":
		return true, nil
	case "n", "no":
		return false, nil
	default:
		fmt.Println("please type (y)es or (n)o and then press enter")
		return AskForComfirmation()
	}
}
