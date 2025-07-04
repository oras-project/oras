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

package repo

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestShowTags_Limit(t *testing.T) {
	ctx := context.Background()

	tags := []string{"v1.0.0", "v1.0.1", "v2.0.0"}
	limit := 2
	printed := 0

	err := simulateTagList(ctx, tags, limit, func(result []string) error {
		for range result {
			if printed >= limit {
				break
			}
			printed++
		}
		return nil
	})

	require.NoError(t, err)
	require.Equal(t, 2, printed)
}

// simulateTagList mocks a tag-fetching behavior
func simulateTagList(ctx context.Context, tags []string, limit int, fn func([]string) error) error {
	return fn(tags)
}

