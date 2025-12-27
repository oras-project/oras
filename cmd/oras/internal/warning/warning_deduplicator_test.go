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
	"bytes"
	"strings"
	"sync"
	"testing"

	"github.com/sirupsen/logrus"
	"oras.land/oras-go/v2/registry/remote"
)

func TestWarningDeduplicator_GetDeduplicator(t *testing.T) {
	tests := []struct {
		name      string
		registry  string
		warnings  []remote.Warning
		wantLogs  []string
		wantCount int
	}{
		{
			name:     "single warning",
			registry: "localhost:5000",
			warnings: []remote.Warning{
				{WarningValue: remote.WarningValue{Code: 299, Agent: "oras", Text: "First warning"}},
			},
			wantLogs:  []string{"First warning"},
			wantCount: 1,
		},
		{
			name:     "duplicate warnings same registry",
			registry: "localhost:5000",
			warnings: []remote.Warning{
				{WarningValue: remote.WarningValue{Code: 299, Agent: "oras", Text: "First warning"}},
				{WarningValue: remote.WarningValue{Code: 299, Agent: "oras", Text: "First warning"}},
			},
			wantLogs:  []string{"First warning"},
			wantCount: 1,
		},
		{
			name:     "different warnings same registry",
			registry: "localhost:5000",
			warnings: []remote.Warning{
				{WarningValue: remote.WarningValue{Code: 299, Agent: "oras", Text: "First warning"}},
				{WarningValue: remote.WarningValue{Code: 299, Agent: "oras", Text: "Second warning"}},
			},
			wantLogs:  []string{"First warning", "Second warning"},
			wantCount: 2,
		},
		{
			name:     "empty warning value",
			registry: "localhost:5000",
			warnings: []remote.Warning{
				{WarningValue: remote.WarningValue{Code: 299, Agent: "oras", Text: "Empty value warning"}},
			},
			wantLogs:  []string{"Empty value warning"},
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			globalDeduplicator = warningDeduplicator{}
			var buf bytes.Buffer
			logger := logrus.New()
			logger.SetOutput(&buf)
			logger.SetLevel(logrus.WarnLevel)

			deduplicator := GetDeduplicator(tt.registry, logger)

			for _, warning := range tt.warnings {
				deduplicator(warning)
			}

			output := buf.String()
			logCount := strings.Count(output, "level=warning")

			if logCount != tt.wantCount {
				t.Errorf("Expected %d warning logs, got %d", tt.wantCount, logCount)
			}

			for _, expectedLog := range tt.wantLogs {
				if !strings.Contains(output, expectedLog) {
					t.Errorf("Expected log to contain %q, but it didn't. Output: %s", expectedLog, output)
				}
			}

			if !strings.Contains(output, tt.registry) && tt.wantCount > 0 {
				t.Errorf("Expected log to contain registry %q, but it didn't. Output: %s", tt.registry, output)
			}
		})
	}
}

func TestWarningDeduplicator_GetDeduplicator_DifferentRegistries(t *testing.T) {
	globalDeduplicator = warningDeduplicator{}
	var buf bytes.Buffer
	logger := logrus.New()
	logger.SetOutput(&buf)
	logger.SetLevel(logrus.WarnLevel)

	deduplicator1 := GetDeduplicator("registry1.example.com", logger)
	deduplicator2 := GetDeduplicator("registry2.example.com", logger)

	warning := remote.Warning{WarningValue: remote.WarningValue{Code: 299, Agent: "oras", Text: "Test warning"}}

	deduplicator1(warning)
	deduplicator2(warning)

	output := buf.String()
	logCount := strings.Count(output, "level=warning")

	if logCount != 2 {
		t.Errorf("Expected 2 warning logs for different registries, got %d", logCount)
	}

	if !strings.Contains(output, "registry1.example.com") {
		t.Error("Expected log to contain registry1.example.com")
	}
	if !strings.Contains(output, "registry2.example.com") {
		t.Error("Expected log to contain registry2.example.com")
	}
}

func TestWarningDeduplicator_GetDeduplicator_Concurrency(t *testing.T) {
	globalDeduplicator = warningDeduplicator{}
	var buf bytes.Buffer
	logger := logrus.New()
	logger.SetOutput(&buf)
	logger.SetLevel(logrus.WarnLevel)

	deduplicator := GetDeduplicator("localhost:5000", logger)

	var wg sync.WaitGroup
	numGoroutines := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			warning := remote.Warning{
				WarningValue: remote.WarningValue{Code: 299, Agent: "oras", Text: "Concurrent warning"},
			}
			deduplicator(warning)
		}(i)
	}

	wg.Wait()

	output := buf.String()
	logCount := strings.Count(output, "level=warning")

	if logCount != 1 {
		t.Errorf("Expected exactly 1 warning log despite concurrent calls, got %d", logCount)
	}
}

func TestWarningDeduplicator_GetDeduplicator_MultipleDeduplicatorsForSameRegistry(t *testing.T) {
	globalDeduplicator = warningDeduplicator{}
	var buf bytes.Buffer
	logger := logrus.New()
	logger.SetOutput(&buf)
	logger.SetLevel(logrus.WarnLevel)

	deduplicator1 := GetDeduplicator("localhost:5000", logger)
	deduplicator2 := GetDeduplicator("localhost:5000", logger)

	warning := remote.Warning{WarningValue: remote.WarningValue{Code: 299, Agent: "oras", Text: "Test warning"}}

	deduplicator1(warning)
	deduplicator2(warning)

	output := buf.String()
	logCount := strings.Count(output, "level=warning")

	if logCount != 1 {
		t.Errorf("Expected 1 warning log for same registry with multiple deduplicators, got %d", logCount)
	}
}
