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

package warning

import (
	"sync"

	"github.com/sirupsen/logrus"
	"oras.land/oras-go/v2/registry/remote"
)

var (
	globalDeduplicator warningDeduplicator
)

// warningDeduplicator tracks warnings that have been emitted and prevents duplicates.
type warningDeduplicator struct {
	seen sync.Map
}

func GetDeduplicator(registry string, logger logrus.FieldLogger) func(warning remote.Warning) {
	result, _ := globalDeduplicator.seen.LoadOrStore(registry, &sync.Map{})
	warner := result.(*sync.Map)
	return func(warning remote.Warning) {
		if _, loaded := warner.LoadOrStore(warning.WarningValue, true); !loaded {
			logger.WithField("registry", registry).Warn(warning.Text)
		}
	}
}
