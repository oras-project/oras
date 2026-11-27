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

package option

import (
	"fmt"
)

// ValidateConcurrency rejects a negative --concurrency level.
//
// A negative level is never meaningful. The oras-go copy and tag paths reset a
// limit <= 0 to their default, so the user's value is ignored without a
// warning, and internal/graph.FindPredecessors reads a non-positive limit as
// unbounded. The flag is therefore checked where it is declared rather than in
// a shared hook, since a subcommand that sets its own PersistentPreRunE would
// skip that.
func ValidateConcurrency(level int) error {
	if level >= 0 {
		return nil
	}
	return fmt.Errorf("`--concurrency %d` is invalid: concurrency level must not be negative", level)
}
