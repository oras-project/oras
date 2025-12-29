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
	"strings"
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
	tempDir := t.TempDir()

	// Create a root command
	rootCmd := New()

	// Test markdown generation without headers
	opts := &docsOptions{
		dest:            tempDir,
		docTypeString:   "markdown",
		topCmd:          rootCmd,
		generateHeaders: false,
	}

	err := opts.run()
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
	tempDir := t.TempDir()

	// Create a root command
	rootCmd := New()

	// Test markdown generation with headers
	opts := &docsOptions{
		dest:            tempDir,
		docTypeString:   "md",
		topCmd:          rootCmd,
		generateHeaders: true,
	}

	err := opts.run()
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
	tempDir := t.TempDir()

	// Create a root command
	rootCmd := New()

	// Test man page generation
	opts := &docsOptions{
		dest:          tempDir,
		docTypeString: "man",
		topCmd:        rootCmd,
	}

	err := opts.run()
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
	tempDir := t.TempDir()

	// Create a root command
	rootCmd := New()

	// Test bash completion generation
	opts := &docsOptions{
		dest:          tempDir,
		docTypeString: "bash",
		topCmd:        rootCmd,
	}

	err := opts.run()
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
	tempDir := t.TempDir()

	// Create a root command
	rootCmd := New()

	// Test with invalid type
	opts := &docsOptions{
		dest:          tempDir,
		docTypeString: "invalid",
		topCmd:        rootCmd,
	}

	err := opts.run()
	if err == nil {
		t.Error("Expected error for invalid doc type, got nil")
	}
}

func TestDocsOptions_RunMdown(t *testing.T) {
	tempDir := t.TempDir()

	// Create a root command
	rootCmd := New()

	// Test "mdown" alias for markdown
	opts := &docsOptions{
		dest:            tempDir,
		docTypeString:   "mdown",
		topCmd:          rootCmd,
		generateHeaders: false,
	}

	err := opts.run()
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
	tempDir := t.TempDir()

	// Create a root command and add docs as a subcommand
	rootCmd := New()
	docsCommand := docsCmd()
	rootCmd.AddCommand(docsCommand)

	// Set args to execute the docs subcommand
	rootCmd.SetArgs([]string{"docs", "--dir", tempDir, "--type", "markdown"})

	// Execute the command through RunE
	err := rootCmd.Execute()
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

func TestDocsOptions_RunMarkdownError(t *testing.T) {
	// Create a root command
	rootCmd := New()

	// Test with non-existent directory
	opts := &docsOptions{
		dest:            "/nonexistent/directory/path",
		docTypeString:   "markdown",
		topCmd:          rootCmd,
		generateHeaders: false,
	}

	err := opts.run()
	if err == nil {
		t.Error("Expected error for invalid directory, got nil")
	}
}

func TestDocsOptions_RunMarkdownWithHeadersError(t *testing.T) {
	// Create a root command
	rootCmd := New()

	// Test with non-existent directory and headers enabled
	opts := &docsOptions{
		dest:            "/nonexistent/directory/path",
		docTypeString:   "markdown",
		topCmd:          rootCmd,
		generateHeaders: true,
	}

	err := opts.run()
	if err == nil {
		t.Error("Expected error for invalid directory with headers, got nil")
	}
}

func TestDocsOptions_RunManError(t *testing.T) {
	// Create a root command
	rootCmd := New()

	// Test with non-existent directory
	opts := &docsOptions{
		dest:          "/nonexistent/directory/path",
		docTypeString: "man",
		topCmd:        rootCmd,
	}

	err := opts.run()
	if err == nil {
		t.Error("Expected error for invalid man directory, got nil")
	}
}

func TestDocsOptions_RunBashError(t *testing.T) {
	// Create a root command
	rootCmd := New()

	// Test with non-existent directory
	opts := &docsOptions{
		dest:          "/nonexistent/directory/path",
		docTypeString: "bash",
		topCmd:        rootCmd,
	}

	err := opts.run()
	if err == nil {
		t.Error("Expected error for invalid bash directory, got nil")
	}
}

func TestDocsOptions_RunMarkdownWithHeadersContent(t *testing.T) {
	tempDir := t.TempDir()

	// Create a root command
	rootCmd := New()

	// Test markdown generation with headers
	opts := &docsOptions{
		dest:            tempDir,
		docTypeString:   "markdown",
		topCmd:          rootCmd,
		generateHeaders: true,
	}

	err := opts.run()
	if err != nil {
		t.Fatalf("run() failed: %v", err)
	}

	// Read the oras.md file and verify header content
	orasFile := filepath.Join(tempDir, "oras.md")
	content, err := os.ReadFile(orasFile)
	if err != nil {
		t.Fatalf("Failed to read oras.md: %v", err)
	}

	contentStr := string(content)
	// Check for proper YAML frontmatter
	if !strings.HasPrefix(contentStr, "---\n") {
		t.Errorf("Expected file to start with '---\\n', got: %s", contentStr[:min(20, len(contentStr))])
	}
	if !strings.Contains(contentStr, "title:") {
		t.Error("Expected header to contain 'title:'")
	}
	if !strings.Contains(contentStr, "Oras") {
		t.Error("Expected header to contain capitalized 'Oras'")
	}
}

func TestDocsCmd_ExecuteWithHeaders(t *testing.T) {
	tempDir := t.TempDir()

	// Create a root command and add docs as a subcommand
	rootCmd := New()
	docsCommand := docsCmd()
	rootCmd.AddCommand(docsCommand)

	// Set args to execute the docs subcommand with headers
	rootCmd.SetArgs([]string{"docs", "--dir", tempDir, "--type", "markdown", "--generate-headers"})

	// Execute the command
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Execute() with headers failed: %v", err)
	}

	// Verify files were created with headers
	entries, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("Failed to read output dir: %v", err)
	}
	if len(entries) == 0 {
		t.Error("No files were generated")
	}
}

func TestDocsCmd_ExecuteMan(t *testing.T) {
	tempDir := t.TempDir()

	// Create a root command and add docs as a subcommand
	rootCmd := New()
	docsCommand := docsCmd()
	rootCmd.AddCommand(docsCommand)

	// Set args to execute the docs subcommand for man pages
	rootCmd.SetArgs([]string{"docs", "--dir", tempDir, "--type", "man"})

	// Execute the command
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Execute() for man failed: %v", err)
	}

	// Verify man files were created
	entries, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("Failed to read output dir: %v", err)
	}
	if len(entries) == 0 {
		t.Error("No man files were generated")
	}
}

func TestDocsCmd_ExecuteBash(t *testing.T) {
	tempDir := t.TempDir()

	// Create a root command and add docs as a subcommand
	rootCmd := New()
	docsCommand := docsCmd()
	rootCmd.AddCommand(docsCommand)

	// Set args to execute the docs subcommand for bash
	rootCmd.SetArgs([]string{"docs", "--dir", tempDir, "--type", "bash"})

	// Execute the command
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Execute() for bash failed: %v", err)
	}

	// Verify bash completion file was created
	completionFile := filepath.Join(tempDir, "completions.bash")
	if _, err := os.Stat(completionFile); os.IsNotExist(err) {
		t.Error("completions.bash file was not generated")
	}
}

func TestDocsOptions_InvalidTypeErrorMessage(t *testing.T) {
	tempDir := t.TempDir()

	// Create a root command
	rootCmd := New()

	// Test with invalid type
	opts := &docsOptions{
		dest:          tempDir,
		docTypeString: "pdf",
		topCmd:        rootCmd,
	}

	err := opts.run()
	if err == nil {
		t.Fatal("Expected error for invalid doc type, got nil")
	}

	// Verify error message contains the invalid type
	errMsg := err.Error()
	if !strings.Contains(errMsg, "pdf") {
		t.Errorf("Expected error message to contain 'pdf', got: %s", errMsg)
	}
	if !strings.Contains(errMsg, "unknown doc type") {
		t.Errorf("Expected error message to contain 'unknown doc type', got: %s", errMsg)
	}
}

func TestDocsCmd_TypeFlagCompletionFunc(t *testing.T) {
	// Create the docs command
	cmd := docsCmd()

	// Get completions for the type flag
	completionFunc, ok := cmd.GetFlagCompletionFunc("type")
	if !ok {
		t.Fatal("Expected completion function for 'type' flag, but none found")
	}

	completions, directive := completionFunc(cmd, []string{}, "")

	// Verify the completions
	expectedCompletions := []string{"bash", "man", "markdown"}
	if len(completions) != len(expectedCompletions) {
		t.Errorf("Expected %d completions, got %d", len(expectedCompletions), len(completions))
	}

	for i, expected := range expectedCompletions {
		if i < len(completions) && completions[i] != expected {
			t.Errorf("Expected completion[%d] to be %s, got %s", i, expected, completions[i])
		}
	}

	// Verify the directive
	if directive != 0 { // cobra.ShellCompDirectiveDefault is 0
		t.Errorf("Expected directive to be ShellCompDirectiveDefault (0), got %d", directive)
	}
}
