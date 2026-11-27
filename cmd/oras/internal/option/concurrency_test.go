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

package option_test

import (
	"testing"

	"github.com/spf13/cobra"
	"oras.land/oras/cmd/oras/internal/option"
)

func TestConcurrency_Parse(t *testing.T) {
	tests := []struct {
		name        string
		concurrency int
		wantErr     bool
	}{
		{
			name:        "negative concurrency",
			concurrency: -1,
			wantErr:     true,
		},
		{
			name:        "zero concurrency",
			concurrency: 0,
			wantErr:     true,
		},
		{
			name:        "minimum valid concurrency",
			concurrency: 1,
			wantErr:     false,
		},
		{
			name:        "positive concurrency",
			concurrency: 5,
			wantErr:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &option.Concurrency{
				Concurrency: tt.concurrency,
			}
			err := opts.Parse(&cobra.Command{})
			if (err != nil) != tt.wantErr {
				t.Fatalf("Concurrency.Parse() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err.Error() != "concurrency value should be at least 1" {
				t.Fatalf("Concurrency.Parse() error = %q, want %q", err.Error(), "concurrency value should be at least 1")
			}
		})
	}
}

func TestConcurrency_ApplyFlags(t *testing.T) {
	type commandOpts struct {
		option.Concurrency
	}

	t.Run("uses fallback default when unset", func(t *testing.T) {
		cmd := &cobra.Command{}
		var opts commandOpts
		option.ApplyFlags(&opts, cmd.Flags())
		if err := option.Parse(cmd, &opts); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if opts.Concurrency.Concurrency != 3 {
			t.Fatalf("got concurrency %d, want 3", opts.Concurrency.Concurrency)
		}
	})

	t.Run("preserves configured default", func(t *testing.T) {
		cmd := &cobra.Command{}
		opts := commandOpts{
			Concurrency: option.Concurrency{Concurrency: 5},
		}
		option.ApplyFlags(&opts, cmd.Flags())
		f := cmd.Flags().Lookup("concurrency")
		if f == nil || f.DefValue != "5" {
			t.Fatalf("expected --concurrency default 5, got %v", f)
		}
		if err := option.Parse(cmd, &opts); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if opts.Concurrency.Concurrency != 5 {
			t.Fatalf("got concurrency %d, want 5", opts.Concurrency.Concurrency)
		}
	})

	t.Run("rejects non-positive flag value through option.Parse", func(t *testing.T) {
		for _, val := range []string{"0", "-2"} {
			cmd := &cobra.Command{}
			opts := commandOpts{
				Concurrency: option.Concurrency{Concurrency: 5},
			}
			option.ApplyFlags(&opts, cmd.Flags())
			if err := cmd.Flags().Set("concurrency", val); err != nil {
				t.Fatalf("failed to set flag: %v", err)
			}
			if err := option.Parse(cmd, &opts); err == nil {
				t.Fatalf("expected error for --concurrency %s, got nil", val)
			}
		}
	})
}
