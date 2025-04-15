//go:build integration
// +build integration

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// setupGitRepo creates a test Git repository
// This function helps set up a test environment for integration tests
func setupGitRepo(t *testing.T) (string, func()) {
	t.Helper()

	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "git-tag-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	// Save the current directory to return to later
	currentDir, err := os.Getwd()
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to get current directory: %v", err)
	}

	// Change to the temp directory
	if err := os.Chdir(tempDir); err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Initialize git repository
	if err := exec.Command("git", "init").Run(); err != nil {
		os.Chdir(currentDir)
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to initialize git repository: %v", err)
	}

	// Set git config
	if err := exec.Command("git", "config", "user.name", "Test User").Run(); err != nil {
		os.Chdir(currentDir)
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to set git user name: %v", err)
	}

	if err := exec.Command("git", "config", "user.email", "test@example.com").Run(); err != nil {
		os.Chdir(currentDir)
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to set git user email: %v", err)
	}

	// Create a test file and commit
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		os.Chdir(currentDir)
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to create test file: %v", err)
	}

	if err := exec.Command("git", "add", "test.txt").Run(); err != nil {
		os.Chdir(currentDir)
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to add test file: %v", err)
	}

	if err := exec.Command("git", "commit", "-m", "Initial commit").Run(); err != nil {
		os.Chdir(currentDir)
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to commit: %v", err)
	}

	// Create branches
	if err := exec.Command("git", "branch", "main").Run(); err != nil {
		os.Chdir(currentDir)
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to create main branch: %v", err)
	}

	if err := exec.Command("git", "branch", "develop").Run(); err != nil {
		os.Chdir(currentDir)
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to create develop branch: %v", err)
	}

	// Create test config file
	configFile := filepath.Join(tempDir, "publish.json")
	configContent := `{
	  "branchTags": [
		{
		  "branch": "master",
		  "tag": "v0.0.0"
		},
		{
		  "branch": "main",
		  "tag": "v0.0.0"
		},
		{
		  "branch": "develop",
		  "tag": "dev0.0.0"
		}
	  ]
	}`
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		os.Chdir(currentDir)
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to create config file: %v", err)
	}

	// Return a cleanup function
	cleanup := func() {
		os.Chdir(currentDir)
		os.RemoveAll(tempDir)
	}

	return tempDir, cleanup
}

// TestIntegrationGitTag demonstrates how to run an integration test for the git tag tool
// This test is marked with the integration build tag and won't run in normal test runs
func TestIntegrationGitTag(t *testing.T) {
	// Skip in normal test runs
	if os.Getenv("RUN_INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test. Set RUN_INTEGRATION_TESTS=true to run")
	}

	// Set up the test environment
	_, cleanup := setupGitRepo(t)
	defer cleanup()

	// Create a tag on master
	cmd := exec.Command("git", "tag", "v0.0.1")
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to create tag: %v", err)
	}

	// Example of verifying a tag exists
	cmd = exec.Command("git", "tag", "-l", "v0.0.1")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to list tags: %v", err)
	}

	if len(output) == 0 {
		t.Errorf("Tag v0.0.1 was not created")
	}

	// The full integration test would now run the main function with mocked stdin/stdout
	// and verify the behavior of the tool
	// This requires more complex mocking of user input and is beyond the scope of this example
}

// Example of a mocked user input scenario for manual testing
/*
func TestScenarioCreateTagOnMaster(t *testing.T) {
	// For this to work, we would need to mock os.Stdin and os.Stdout
	// and implement a way to programmatically provide user input

	dir, cleanup := setupGitRepo(t)
	defer cleanup()

	// Mock stdin with predefined answers:
	// 1 (select master branch)
	// v1.0.0 (enter tag)
	// n (don't push to remote)

	// Call main() with mocked I/O

	// Verify the tag was created
	cmd := exec.Command("git", "tag", "-l", "v1.0.0")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to list tags: %v", err)
	}

	if len(output) == 0 {
		t.Errorf("Tag v1.0.0 was not created")
	}
}
*/
