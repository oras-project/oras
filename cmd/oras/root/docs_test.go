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

package root

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDocsCmd(t *testing.T) {
	// Test that the command can be created
	cmd := docsCmd()
	if cmd == nil {
		t.Fatal("docsCmd() returned nil")
	}
	if cmd.Use != "docs" {
		t.Errorf("Expected Use to be 'docs', got %s", cmd.Use)
	}
	if !cmd.Hidden {
		t.Error("Expected docs command to be hidden")
	}
}

func TestDocsOptions_RunMarkdown(t *testing.T) {
	// Create a temporary directory for output
	tempDir, err := os.MkdirTemp("", "oras-docs-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create a root command
	rootCmd := New()

	// Test markdown generation without headers
	opts := &docsOptions{
		dest:            tempDir,
		docTypeString:   "markdown",
		topCmd:          rootCmd,
		generateHeaders: false,
	}

	err = opts.run()
	if err != nil {
		t.Fatalf("run() failed for markdown: %v", err)
	}

	// Check that markdown files were created
	entries, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("Failed to read output dir: %v", err)
	}
	if len(entries) == 0 {
		t.Error("No markdown files were generated")
	}

	// Verify at least one .md file exists
	foundMd := false
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == ".md" {
			foundMd = true
			break
		}
	}
	if !foundMd {
		t.Error("No .md files found in output directory")
	}
}

func TestDocsOptions_RunMarkdownWithHeaders(t *testing.T) {
	// Create a temporary directory for output
	tempDir, err := os.MkdirTemp("", "oras-docs-test-headers-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create a root command
	rootCmd := New()

	// Test markdown generation with headers
	opts := &docsOptions{
		dest:            tempDir,
		docTypeString:   "md",
		topCmd:          rootCmd,
		generateHeaders: true,
	}

	err = opts.run()
	if err != nil {
		t.Fatalf("run() failed for markdown with headers: %v", err)
	}

	// Check that markdown files were created
	entries, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("Failed to read output dir: %v", err)
	}
	if len(entries) == 0 {
		t.Error("No markdown files were generated")
	}

	// Read one of the markdown files and verify it has a header
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == ".md" {
			content, err := os.ReadFile(filepath.Join(tempDir, entry.Name()))
			if err != nil {
				t.Fatalf("Failed to read markdown file: %v", err)
			}
			contentStr := string(content)
			if len(contentStr) < 4 || contentStr[:3] != "---" {
				t.Errorf("Expected markdown file to start with '---', got: %s", contentStr[:min(20, len(contentStr))])
			}
			break
		}
	}
}

func TestDocsOptions_RunMan(t *testing.T) {
	// Create a temporary directory for output
	tempDir, err := os.MkdirTemp("", "oras-docs-test-man-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create a root command
	rootCmd := New()

	// Test man page generation
	opts := &docsOptions{
		dest:          tempDir,
		docTypeString: "man",
		topCmd:        rootCmd,
	}

	err = opts.run()
	if err != nil {
		t.Fatalf("run() failed for man: %v", err)
	}

	// Check that man files were created
	entries, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("Failed to read output dir: %v", err)
	}
	if len(entries) == 0 {
		t.Error("No man files were generated")
	}
}

func TestDocsOptions_RunBash(t *testing.T) {
	// Create a temporary directory for output
	tempDir, err := os.MkdirTemp("", "oras-docs-test-bash-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create a root command
	rootCmd := New()

	// Test bash completion generation
	opts := &docsOptions{
		dest:          tempDir,
		docTypeString: "bash",
		topCmd:        rootCmd,
	}

	err = opts.run()
	if err != nil {
		t.Fatalf("run() failed for bash: %v", err)
	}

	// Check that completions.bash was created
	completionFile := filepath.Join(tempDir, "completions.bash")
	if _, err := os.Stat(completionFile); os.IsNotExist(err) {
		t.Error("completions.bash file was not generated")
	}
}

func TestDocsOptions_RunInvalidType(t *testing.T) {
	// Create a temporary directory for output
	tempDir, err := os.MkdirTemp("", "oras-docs-test-invalid-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create a root command
	rootCmd := New()

	// Test with invalid type
	opts := &docsOptions{
		dest:          tempDir,
		docTypeString: "invalid",
		topCmd:        rootCmd,
	}

	err = opts.run()
	if err == nil {
		t.Error("Expected error for invalid doc type, got nil")
	}
}

func TestDocsOptions_RunMdown(t *testing.T) {
	// Create a temporary directory for output
	tempDir, err := os.MkdirTemp("", "oras-docs-test-mdown-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create a root command
	rootCmd := New()

	// Test "mdown" alias for markdown
	opts := &docsOptions{
		dest:            tempDir,
		docTypeString:   "mdown",
		topCmd:          rootCmd,
		generateHeaders: false,
	}

	err = opts.run()
	if err != nil {
		t.Fatalf("run() failed for mdown: %v", err)
	}

	// Check that markdown files were created
	entries, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("Failed to read output dir: %v", err)
	}
	if len(entries) == 0 {
		t.Error("No markdown files were generated")
	}
}

func TestDocsCmd_Execute(t *testing.T) {
	// Create a temporary directory for output
	tempDir, err := os.MkdirTemp("", "oras-docs-test-execute-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create a root command and add docs as a subcommand
	rootCmd := New()
	docsCommand := docsCmd()
	rootCmd.AddCommand(docsCommand)

	// Set args to execute the docs subcommand
	rootCmd.SetArgs([]string{"docs", "--dir", tempDir, "--type", "markdown"})

	// Execute the command through RunE
	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("Execute() failed: %v", err)
	}

	// Verify files were created
	entries, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("Failed to read output dir: %v", err)
	}
	if len(entries) == 0 {
		t.Error("No files were generated")
	}
}

func TestDocsCmd_FlagCompletion(t *testing.T) {
	// Create the docs command
	cmd := docsCmd()

	// Get the flag completion function for "type"
	flag := cmd.Flag("type")
	if flag == nil {
		t.Fatal("type flag not found")
	}

	// Test the completion function
	completions, directive := flag.Value.(interface {
		Type() string
	}), 0

	// Since we can't directly access the completion function,
	// we verify the flag exists and is properly configured
	if flag.Name != "type" {
		t.Errorf("Expected flag name 'type', got %s", flag.Name)
	}
	if flag.DefValue != "markdown" {
		t.Errorf("Expected default value 'markdown', got %s", flag.DefValue)
	}

	_ = completions
	_ = directive
}

func TestDocsCmd_Flags(t *testing.T) {
	cmd := docsCmd()

	// Verify all flags are present
	dirFlag := cmd.Flag("dir")
	if dirFlag == nil {
		t.Fatal("dir flag not found")
	}
	if dirFlag.DefValue != "./" {
		t.Errorf("Expected dir default './', got %s", dirFlag.DefValue)
	}

	typeFlag := cmd.Flag("type")
	if typeFlag == nil {
		t.Fatal("type flag not found")
	}
	if typeFlag.DefValue != "markdown" {
		t.Errorf("Expected type default 'markdown', got %s", typeFlag.DefValue)
	}

	headersFlag := cmd.Flag("generate-headers")
	if headersFlag == nil {
		t.Fatal("generate-headers flag not found")
	}
	if headersFlag.DefValue != "false" {
		t.Errorf("Expected generate-headers default 'false', got %s", headersFlag.DefValue)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
