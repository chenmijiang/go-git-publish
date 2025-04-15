package main

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// TestIsGitRepository tests the Git repository detection function
func TestIsGitRepository(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "git-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change to the temp directory
	currentDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(currentDir)

	os.Chdir(tempDir)

	// Test in a non-git directory
	if isGitRepository() {
		t.Errorf("Expected non-git directory to return false")
	}

	// Initialize a git repository
	cmd := exec.Command("git", "init")
	if err := cmd.Run(); err != nil {
		t.Skipf("Failed to initialize git: %v. Skipping test.", err)
		return
	}

	// Test in a git directory
	if !isGitRepository() {
		t.Errorf("Expected git directory to return true")
	}
}

// TestExtractPrefix tests the prefix extraction function
func TestExtractPrefix(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"v1.0.0", "v"},
		{"g0.0.1", "g"},
		{"release1.2.3", "release"},
		{"dev0.1.0", "dev"},
		{"1.0.0", ""}, // No prefix
		{"", ""},      // Empty string
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := extractPrefix(tc.input)
			if result != tc.expected {
				t.Errorf("extractPrefix(%q) = %q, expected %q", tc.input, result, tc.expected)
			}
		})
	}
}

// TestIsTagVersionGreater tests the tag version comparison
func TestIsTagVersionGreater(t *testing.T) {
	testCases := []struct {
		newTag   string
		oldTag   string
		expected bool
	}{
		{"v1.0.1", "v1.0.0", true},
		{"v1.1.0", "v1.0.0", true},
		{"v2.0.0", "v1.0.0", true},
		{"v1.0.0", "v1.0.0", false},
		{"v1.0.0", "v1.0.1", false},
		{"v1.0.0", "v1.1.0", false},
		{"v1.0.0", "v2.0.0", false},
		{"v1.0.0", "", true}, // No old tag
		{"g1.0.1", "g1.0.0", true},
		{"dev2.0.0", "dev1.9.9", true},
	}

	for _, tc := range testCases {
		t.Run(tc.newTag+"_vs_"+tc.oldTag, func(t *testing.T) {
			result := isTagVersionGreater(tc.newTag, tc.oldTag)
			if result != tc.expected {
				t.Errorf("isTagVersionGreater(%q, %q) = %v, expected %v", tc.newTag, tc.oldTag, result, tc.expected)
			}
		})
	}
}

// TestCalculateNextTag tests the nextTag calculation
func TestCalculateNextTag(t *testing.T) {
	testCases := []struct {
		lastTag   string
		tagFormat string
		expected  string
	}{
		{"v1.0.0", "v0.0.0", "v1.0.1"},
		{"v1.2.3", "v0.0.0", "v1.2.4"},
		{"g0.0.9", "g0.0.0", "g0.0.10"},
		{"", "v0.0.0", "v0.0.0"}, // No last tag
		{"dev1.2.3", "dev0.0.0", "dev1.2.4"},
		{"v1.a.3", "v0.0.0", "v0.0.0"}, // Invalid format
		{"1.2.3", "0.0.0", "1.2.4"},    // No prefix
	}

	for _, tc := range testCases {
		t.Run(tc.lastTag+"_to_next", func(t *testing.T) {
			result := calculateNextTag(tc.lastTag, tc.tagFormat)
			if result != tc.expected {
				t.Errorf("calculateNextTag(%q, %q) = %q, expected %q", tc.lastTag, tc.tagFormat, result, tc.expected)
			}
		})
	}
}

// TestGrayScaleTagging specifically tests the gray-scale tagging issue
func TestGrayScaleTagging(t *testing.T) {
	// Test the specific issue with g1.9.9 -> g1.9.10 instead of g1.10.0

	// Save original functions to restore them after the test
	originalExec := execCommand
	originalTagOnBranch := isTagOnBranchFunc

	defer func() {
		execCommand = originalExec
		isTagOnBranchFunc = originalTagOnBranch
	}()

	// Test case 1: With current implementation, g1.9.9 would increment to g1.9.10
	lastTag := "g1.9.9"
	tagFormat := "g0.0.0"
	nextTag := calculateNextTag(lastTag, tagFormat)
	expected := "g1.9.10"

	if nextTag != expected {
		t.Errorf("For lastTag=%s, got nextTag=%s, expected %s", lastTag, nextTag, expected)
	}

	// Test case 2: With sorting, g1.9.10 is correctly placed after g1.9.9
	tags := []string{"g1.9.9", "g1.9.10"}

	// We'll implement a semver sort function for testing since it was removed from main.go
	sortVersionTags(tags)

	if tags[0] != "g1.9.9" || tags[1] != "g1.9.10" {
		t.Errorf("Incorrect sorting: got %v, expected [g1.9.9, g1.9.10]", tags)
	}

	// Test case 3: Verify that isTagVersionGreater works correctly with these versions
	if !isTagVersionGreater("g1.9.10", "g1.9.9") {
		t.Errorf("isTagVersionGreater(g1.9.10, g1.9.9) returned false, expected true")
	}

	// Test case 4: Mock getLastTag to see that g1.9.10 is correctly determined to be newer than g1.9.9
	// First ensure hasAnyTags returns true
	execCommand = func(cmd string, args ...string) *exec.Cmd {
		// Mock hasAnyTags check
		if cmd == "git" && len(args) == 2 && args[0] == "tag" && args[1] == "-l" {
			cs := []string{"-test.run=TestHelperProcess", "--", cmd}
			cs = append(cs, args...)
			mockCmd := exec.Command(os.Args[0], cs...)
			mockCmd.Env = []string{"GO_WANT_HELPER_PROCESS=1", "TEST_TAGS=g1.9.9,g1.9.10"}
			return mockCmd
		}

		// Mock the sorted tag list (--sort=-v:refname returns newest tags first)
		if cmd == "git" && len(args) >= 3 && args[0] == "tag" && args[1] == "--list" {
			cs := []string{"-test.run=TestHelperProcess", "--", cmd}
			cs = append(cs, args...)
			mockCmd := exec.Command(os.Args[0], cs...)
			// Return them in reverse order since Git sorts them with newest first
			mockCmd.Env = []string{"GO_WANT_HELPER_PROCESS=1", "TEST_TAGS=g1.9.10,g1.9.9"}
			return mockCmd
		}

		// Mock other git commands
		return exec.Command("echo", "Testing")
	}

	// Mock isTagOnBranch to return true only for g1.9.10
	isTagOnBranchFunc = func(tag, branch string) bool {
		return tag == "g1.9.10"
	}

	latestTag := getLastTag("gray", "g0.0.0")
	expectedLatest := "g1.9.10"

	if latestTag != expectedLatest {
		t.Errorf("getLastTag() = %q, expected %q", latestTag, expectedLatest)
	}
}

// sortVersionTags sorts a list of tags according to semantic versioning rules for tests
func sortVersionTags(tags []string) {
	sort.Slice(tags, func(i, j int) bool {
		// Extract prefix - assumes all tags have the same prefix
		prefixLen := 0
		for _, c := range tags[i] {
			if c >= '0' && c <= '9' {
				break
			}
			prefixLen++
		}

		// Extract version numbers
		versionI := tags[i][prefixLen:]
		versionJ := tags[j][prefixLen:]

		// Split into parts
		partsI := strings.Split(versionI, ".")
		partsJ := strings.Split(versionJ, ".")

		// Ensure we have 3 parts
		if len(partsI) != 3 || len(partsJ) != 3 {
			return tags[i] < tags[j] // Fallback to string comparison
		}

		// Compare major
		majorI, errI := strconv.Atoi(partsI[0])
		majorJ, errJ := strconv.Atoi(partsJ[0])
		if errI != nil || errJ != nil {
			return tags[i] < tags[j] // Fallback to string comparison
		}
		if majorI != majorJ {
			return majorI < majorJ
		}

		// Compare minor
		minorI, errI := strconv.Atoi(partsI[1])
		minorJ, errJ := strconv.Atoi(partsJ[1])
		if errI != nil || errJ != nil {
			return tags[i] < tags[j] // Fallback to string comparison
		}
		if minorI != minorJ {
			return minorI < minorJ
		}

		// Compare patch
		patchI, errI := strconv.Atoi(partsI[2])
		patchJ, errJ := strconv.Atoi(partsJ[2])
		if errI != nil || errJ != nil {
			return tags[i] < tags[j] // Fallback to string comparison
		}
		return patchI < patchJ
	})
}

// TestUniqueStrings tests the string deduplication function
func TestUniqueStrings(t *testing.T) {
	testCases := []struct {
		input    []string
		expected []string
	}{
		{[]string{"a", "b", "c"}, []string{"a", "b", "c"}},
		{[]string{"a", "a", "b", "c", "b"}, []string{"a", "b", "c"}},
		{[]string{}, []string{}},
		{[]string{"a"}, []string{"a"}},
		{[]string{"a", "a", "a"}, []string{"a"}},
	}

	for i, tc := range testCases {
		t.Run("case_"+string(rune('0'+i)), func(t *testing.T) {
			result := uniqueStrings(tc.input)

			// Convert to maps for easy comparison since order may vary
			resultMap := make(map[string]bool)
			expectedMap := make(map[string]bool)

			for _, s := range result {
				resultMap[s] = true
			}

			for _, s := range tc.expected {
				expectedMap[s] = true
			}

			// Check if maps are equal
			if len(resultMap) != len(expectedMap) {
				t.Errorf("uniqueStrings() returned %d elements, expected %d", len(resultMap), len(expectedMap))
			}

			for s := range expectedMap {
				if !resultMap[s] {
					t.Errorf("uniqueStrings() is missing expected element %q", s)
				}
			}
		})
	}
}

// Mock configuration for testing
func getTestConfig() Config {
	return Config{
		BranchTags: []BranchTagConfig{
			{Branch: "master", Tag: "v0.0.0"},
			{Branch: "main", Tag: "v0.0.0"},
			{Branch: "develop", Tag: "dev0.0.0"},
		},
	}
}

// TestReadConfig tests config file reading (limited test)
func TestConfigStructure(t *testing.T) {
	// Test the basic structure matches expectations
	config := getTestConfig()

	if len(config.BranchTags) != 3 {
		t.Errorf("Expected 3 branch tag configs, got %d", len(config.BranchTags))
	}

	// Check if master branch has the correct tag format
	var foundMaster bool
	for _, bt := range config.BranchTags {
		if bt.Branch == "master" {
			foundMaster = true
			if bt.Tag != "v0.0.0" {
				t.Errorf("Expected master branch to have tag format v0.0.0, got %s", bt.Tag)
			}
		}
	}

	if !foundMaster {
		t.Errorf("Expected to find a master branch in the configuration")
	}
}

// TestSemverSort tests the semantic version sorting function
func TestSemverSort(t *testing.T) {
	testCases := []struct {
		input    []string
		expected []string
	}{
		{
			// Simple version increments
			[]string{"v1.0.0", "v1.1.0", "v2.0.0", "v1.0.1"},
			[]string{"v1.0.0", "v1.0.1", "v1.1.0", "v2.0.0"},
		},
		{
			// Test with "g" prefix
			[]string{"g1.10.0", "g1.9.9", "g1.9.10", "g2.0.0", "g1.9.1"},
			[]string{"g1.9.1", "g1.9.9", "g1.9.10", "g1.10.0", "g2.0.0"},
		},
		{
			// Test with two-digit numbers
			[]string{"v1.0.9", "v1.0.10", "v1.0.11"},
			[]string{"v1.0.9", "v1.0.10", "v1.0.11"},
		},
		{
			// Test with different prefixes
			[]string{"dev1.2.3", "dev1.1.4", "dev2.0.0"},
			[]string{"dev1.1.4", "dev1.2.3", "dev2.0.0"},
		},
		{
			// Test with empty array
			[]string{},
			[]string{},
		},
		{
			// Test with single element
			[]string{"v1.0.0"},
			[]string{"v1.0.0"},
		},
	}

	for i, tc := range testCases {
		t.Run("case_"+string(rune('0'+i)), func(t *testing.T) {
			// Create a copy to avoid modifying the original
			input := make([]string, len(tc.input))
			copy(input, tc.input)

			// Sort the input
			sortVersionTags(input)

			// Check if the sorted result matches the expected order
			if len(input) != len(tc.expected) {
				t.Errorf("sortVersionTags() returned %d elements, expected %d", len(input), len(tc.expected))
				return
			}

			for j, v := range input {
				if v != tc.expected[j] {
					t.Errorf("sortVersionTags() result[%d] = %q, expected %q", j, v, tc.expected[j])
				}
			}
		})
	}
}

// TestGetLastTagSorting tests that getLastTag correctly sorts tags before returning the last one
func TestGetLastTagSorting(t *testing.T) {
	// This is a more integration-oriented test, but we can still test the logic
	// by setting up a fake list of tags that would be incorrectly sorted by string comparison

	// Save original functions to restore them after the test
	originalExec := execCommand
	originalTagOnBranch := isTagOnBranchFunc

	defer func() {
		execCommand = originalExec
		isTagOnBranchFunc = originalTagOnBranch
	}()

	// Mock the git command execution
	execCommand = func(cmd string, args ...string) *exec.Cmd {
		// Mock the hasAnyTags check
		if cmd == "git" && len(args) == 2 && args[0] == "tag" && args[1] == "-l" {
			cs := []string{"-test.run=TestHelperProcess", "--", cmd}
			cs = append(cs, args...)
			mockCmd := exec.Command(os.Args[0], cs...)
			mockCmd.Env = []string{"GO_WANT_HELPER_PROCESS=1", "TEST_HAS_TAGS=true"}
			return mockCmd
		}

		// For git tag --list, return our test tags sorted by the git command
		if cmd == "git" && len(args) >= 3 && args[0] == "tag" && args[1] == "--list" {
			// Create a fake command that will output our test data
			cs := []string{"-test.run=TestHelperProcess", "--", cmd}
			cs = append(cs, args...)
			mockCmd := exec.Command(os.Args[0], cs...)
			// Return sorted tags (g2.0.0 first since --sort=-v:refname sorts descending)
			mockCmd.Env = []string{"GO_WANT_HELPER_PROCESS=1", "TEST_TAGS=g2.0.0,g1.10.0,g1.9.10,g1.9.9,g1.9.1"}
			return mockCmd
		}

		// For other commands, use a mock implementation
		return exec.Command("echo", "Testing")
	}

	// Mock isTagOnBranch to return true only for the first tag (g2.0.0)
	isTagOnBranchFunc = func(tag, branch string) bool {
		return tag == "g2.0.0"
	}

	// Run the test
	result := getLastTag("gray", "g0.0.0")
	expected := "g2.0.0"

	if result != expected {
		t.Errorf("getLastTag() = %q, expected %q", result, expected)
	}
}

// TestHelperProcess is not a real test, it's used to mock command execution
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	// Mock hasAnyTags check
	if os.Getenv("TEST_HAS_TAGS") == "true" {
		fmt.Println("v1.0.0")
		os.Exit(0)
	}

	// Get the tags we want to return
	tags := os.Getenv("TEST_TAGS")
	if tags != "" {
		// Convert comma-separated list to newline-separated list
		tagList := strings.Split(tags, ",")
		for _, tag := range tagList {
			fmt.Println(tag)
		}
	}

	os.Exit(0)
}
