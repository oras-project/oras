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
	"bytes"
	"encoding/json"
	"sort"
	"testing"

	"oras.land/oras/cmd/oras/internal/display"
	"oras.land/oras/cmd/oras/internal/option"
	"oras.land/oras/cmd/oras/internal/output"
)

// Let's add a unit test to verify the JSON output doesn't contain the name field
func TestShowTagsJSON(t *testing.T) { // Create a buffer for testing the output
	var buf bytes.Buffer
	p := output.NewPrinter(&buf, &buf)	// Create a handler for JSON format
	format := option.Format{Type: option.FormatTypeJSON.Name}
	handler, err := display.NewTagsHandler(p, format)
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	// Add some test tags
	if err := handler.OnListed("tag1"); err != nil {
		t.Fatalf("Failed to add tag: %v", err)
	}
	if err := handler.OnListed("tag2"); err != nil {
		t.Fatalf("Failed to add tag: %v", err)
	}

	// Render the output
	if err := handler.Render(); err != nil {
		t.Fatalf("Failed to render: %v", err)
	}

	// Parse the JSON output
	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	// Verify the name field doesn't exist
	if _, exists := result["name"]; exists {
		t.Error("JSON output should not contain 'name' field")
	}

	// Verify the tags field exists and contains the right tags
	tags, ok := result["tags"].([]interface{})
	if !ok {
		t.Fatal("JSON output missing or invalid 'tags' field")
	}

	// Convert and sort tags for comparison
	var actualTags []string
	for _, tag := range tags {
		actualTags = append(actualTags, tag.(string))
	}
	sort.Strings(actualTags)

	// Compare with expected tags
	expectedTags := []string{"tag1", "tag2"}
	sort.Strings(expectedTags)

	if len(actualTags) != len(expectedTags) {
		t.Errorf("Expected %d tags, got %d", len(expectedTags), len(actualTags))
	}

	for i, tag := range actualTags {
		if i < len(expectedTags) && tag != expectedTags[i] {
			t.Errorf("Expected tag %s, got %s", expectedTags[i], tag)
		}
	}
}
