package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
)

// BranchTagConfig represents the configuration for branch and tag format
type BranchTagConfig struct {
	Branch string `json:"branch"`
	Tag    string `json:"tag"`
}

// Config represents the application configuration
type Config struct {
	BranchTags []BranchTagConfig `json:"branchTags"`
}

// Default configuration
var defaultConfig = Config{
	BranchTags: []BranchTagConfig{
		{Branch: "master", Tag: "v0.0.0"},
		{Branch: "main", Tag: "v0.0.0"},
		{Branch: "gray", Tag: "g0.0.0"},
	},
}

// Variables to allow mocking in tests
var execCommand = exec.Command
var isTagOnBranchFunc = isTagOnBranch

func main() {
	// Check if we're in a git repository
	if !isGitRepository() {
		fmt.Println("Error: Not in a git repository")
		os.Exit(1)
	}

	// Set up colors for better user experience
	green := color.New(color.FgGreen).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()

	// Show initial message
	fmt.Println(cyan("Initializing git-publish..."))

	// Check if remote repository exists early
	remoteURLs := getAllRemoteURLs()
	hasRemote := len(remoteURLs) > 0

	config := readConfig()

	// Filter branches that don't exist in the repository
	fmt.Println("Finding available branches...")
	config = filterExistingBranches(config, hasRemote)

	// Check if any branches remain
	if len(config.BranchTags) == 0 {
		fmt.Println("Error: None of the configured branches exist in this repository")
		os.Exit(1)
	}

	fmt.Println(green("Initialization complete!"))

	// Interactive CLI - now includes tag checking within the selection process
	selectedBranch, tagFormat := selectBranchAndTag(config)

	// Get last tag from the selected branch
	lastTag := getLastTag(selectedBranch, tagFormat)

	// Calculate next tag
	nextTag := calculateNextTag(lastTag, tagFormat)

	if lastTag == "" {
		fmt.Println(cyan("Creating first tag for this branch..."))
	} else {
		fmt.Printf("Last tag: %s, suggested next tag: %s\n", lastTag, green(nextTag))
	}

	// Ask for tag
	tagToCreate := promptForTag(tagFormat, nextTag, lastTag)

	// Ask to push to remote if remotes exist
	if !hasRemote {
		fmt.Println("No remote repositories found. Skipping push step.")

		// Create tag on branch
		createTag(selectedBranch, tagToCreate)

		fmt.Printf("Successfully created tag %s on branch %s\n", green(tagToCreate), green(selectedBranch))
	} else {
		// Ask to push to remote
		pushToRemote, selectedRemote := promptForPushToRemote(remoteURLs)

		// Create tag on branch
		createTag(selectedBranch, tagToCreate)

		// Push to remote if requested
		if pushToRemote {
			fmt.Printf("Pushing tag %s to remote %s...\n", tagToCreate, selectedRemote)
			pushTagToRemote(tagToCreate, selectedRemote)
			fmt.Printf("Successfully created tag %s on branch %s\n", green(tagToCreate), green(selectedBranch))
			fmt.Printf("Tag was pushed to remote: %s\n", green(selectedRemote))
		} else {
			fmt.Printf("Successfully created tag %s on branch %s\n", green(tagToCreate), green(selectedBranch))
		}
	}
}

// isGitRepository checks if the current directory is a git repository
func isGitRepository() bool {
	cmd := execCommand("git", "rev-parse", "--is-inside-work-tree")
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}

// readConfig reads the configuration file
func readConfig() Config {
	configPath := "publish.json"

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Write default config if file doesn't exist
		writeDefaultConfig(configPath)
		return defaultConfig
	}

	// Read config file
	fileContent, err := os.ReadFile(configPath)
	if err != nil {
		fmt.Printf("Error reading config file: %v\n", err)
		fmt.Println("Using default configuration")
		return defaultConfig
	}

	// Parse config file
	var config Config
	if err := json.Unmarshal(fileContent, &config); err != nil {
		fmt.Printf("Error parsing config file: %v\n", err)
		fmt.Println("Using default configuration")
		return defaultConfig
	}

	// Validate config
	if len(config.BranchTags) == 0 {
		fmt.Println("Config file is valid but empty. Using default configuration")
		return defaultConfig
	}

	return config
}

// filterExistingBranches filters out branches that don't exist in the repository
func filterExistingBranches(config Config, hasRemote bool) Config {
	// Extract branch names first
	var branchNames []string
	for _, bt := range config.BranchTags {
		branchNames = append(branchNames, bt.Branch)
	}

	// Only fetch if remote exists
	if hasRemote {
		fetchRemote()
	}

	// Get all available branches (local and now fetched remote)
	availableBranches := getConfiguredBranches(branchNames)

	var filteredBranchTags []BranchTagConfig
	for _, bt := range config.BranchTags {
		if branchExists(bt.Branch, availableBranches) {
			filteredBranchTags = append(filteredBranchTags, bt)
		} else {
			fmt.Printf("Warning: Branch '%s' does not exist in this repository and will be skipped\n", bt.Branch)
		}
	}

	return Config{BranchTags: filteredBranchTags}
}

// getConfiguredBranches gets local and remote branches that match the configured branches
func getConfiguredBranches(configuredBranches []string) []string {
	// Get all local branches
	cmdLocal := execCommand("git", "branch", "--list")
	outputLocal, err := cmdLocal.Output()
	localBranches := []string{}
	if err == nil {
		for _, line := range strings.Split(strings.TrimSpace(string(outputLocal)), "\n") {
			if line != "" {
				// Remove the asterisk and spaces
				branch := strings.TrimSpace(strings.TrimPrefix(line, "*"))
				// Only include branch if it's in the configured branches
				if contains(configuredBranches, branch) {
					localBranches = append(localBranches, branch)
				}
			}
		}
	}

	// Get all remote branches
	cmdRemote := execCommand("git", "branch", "-r")
	outputRemote, err := cmdRemote.Output()
	remoteBranches := []string{}
	if err == nil {
		for _, line := range strings.Split(strings.TrimSpace(string(outputRemote)), "\n") {
			if line != "" {
				// Remove 'origin/' prefix and spaces
				parts := strings.Split(strings.TrimSpace(line), "/")
				if len(parts) >= 2 {
					branch := parts[len(parts)-1]
					// Only include branch if it's in the configured branches
					if contains(configuredBranches, branch) {
						remoteBranches = append(remoteBranches, branch)
					}
				}
			}
		}
	}

	// Combine local and remote branches and remove duplicates
	allBranches := append(localBranches, remoteBranches...)
	return uniqueStrings(allBranches)
}

// contains checks if a string exists in a slice
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// uniqueStrings removes duplicate strings from a slice
func uniqueStrings(slice []string) []string {
	keys := make(map[string]bool)
	var unique []string
	for _, entry := range slice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			unique = append(unique, entry)
		}
	}
	return unique
}

// branchExists checks if a branch exists in the list of available branches
func branchExists(branch string, availableBranches []string) bool {
	for _, b := range availableBranches {
		if b == branch {
			return true
		}
	}
	return false
}

// writeDefaultConfig writes the default configuration to the given path
func writeDefaultConfig(path string) {
	data, err := json.MarshalIndent(defaultConfig, "", "  ")
	if err != nil {
		fmt.Printf("Error creating default config: %v\n", err)
		return
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		fmt.Printf("Error writing default config to %s: %v\n", path, err)
	}
}

// selectBranchAndTag presents a selection of branches from the config
func selectBranchAndTag(config Config) (string, string) {
	// Set up colors for better user experience
	green := color.New(color.FgGreen).SprintFunc()

	// Prepare options for selection
	branchOptions := make([]string, len(config.BranchTags))
	tagFormats := make([]string, len(config.BranchTags))
	for i, bt := range config.BranchTags {
		branchOptions[i] = bt.Branch
		tagFormats[i] = bt.Tag
	}

	// Default to first branch
	defaultBranch := config.BranchTags[0].Branch
	defaultTagFormat := config.BranchTags[0].Tag

	// Display options
	fmt.Println("Select branch for tagging:")
	for i, branch := range branchOptions {
		lastTag := getLastTag(branch, tagFormats[i])
		if lastTag == "" {
			fmt.Printf("%d: %s (No existing tags, format: %s)\n", i+1, branch, tagFormats[i])
		} else {
			fmt.Printf("%d: %s (Last tag: %s)\n", i+1, branch, green(lastTag))
		}
	}

	// Default option as the first one
	fmt.Printf("Enter number (default: 1 for %s): ", defaultBranch)

	// Read user input
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	// Handle default or parse selection
	selectedBranch := defaultBranch
	selectedTagFormat := defaultTagFormat
	if input != "" {
		if idx, err := strconv.Atoi(input); err == nil && idx > 0 && idx <= len(branchOptions) {
			selectedBranch = branchOptions[idx-1]
			selectedTagFormat = tagFormats[idx-1]
		} else {
			fmt.Printf("Invalid selection, using default branch: %s\n", defaultBranch)
		}
	}

	return selectedBranch, selectedTagFormat
}

// fetchRemote fetches latest information from remote
func fetchRemote() {
	// Check if there are any remotes first
	// remotes := getAllRemoteURLs()
	// if len(remotes) == 0 {
	// 	// No remotes, skip fetch
	// 	return
	// }

	// Show progress message
	fmt.Println("Fetching branch information from remote, please wait...")

	// Use a channel to track progress with timeout
	done := make(chan bool)
	errCh := make(chan error)

	go func() {
		// First try a simple fetch to update remote refs
		// This avoids issues with specific branches
		cmd := execCommand("git", "fetch", "--no-tags", "origin")
		err := cmd.Run()
		if err != nil {
			// Non-critical error, just log it
			errCh <- fmt.Errorf("warning: initial fetch failed: %v", err)
		}

		// Now try to fetch tags if there are any
		if hasAnyTags() {
			cmd = execCommand("git", "fetch", "--depth=5", "origin", "refs/tags/*:refs/tags/*")
			if err := cmd.Run(); err != nil {
				// Non-critical error, just log it
				errCh <- fmt.Errorf("warning: failed to fetch tags: %v", err)
			}
		}

		// Signal we're done
		close(errCh)
		done <- true
	}()

	// Set a timeout to ensure we don't wait forever
	var fetchErrors []error
	timeoutReached := false

	select {
	case <-done:
		// Fetch completed, collect errors if any
		for err := range errCh {
			fetchErrors = append(fetchErrors, err)
		}
	case <-time.After(5 * time.Second):
		// Timeout reached, continue anyway
		timeoutReached = true
		fmt.Println("Fetch taking longer than expected, continuing...")
	}

	// Print success message unless timeout occurred
	if !timeoutReached {
		fmt.Println("Remote information fetched successfully.")
	}

	// Print any errors we collected
	for _, err := range fetchErrors {
		fmt.Println(err)
	}
}

// hasAnyTags checks if the repository has any tags at all
func hasAnyTags() bool {
	cmd := execCommand("git", "tag", "-l")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	tags := strings.TrimSpace(string(output))
	return tags != ""
}

// getLastTag returns the last tag matching the format on the given branch
func getLastTag(branch string, tagFormat string) string {
	// Check if there are any tags first
	if !hasAnyTags() {
		return ""
	}

	// Extract prefix from tag format (like "v" from "v0.0.0")
	prefix := extractPrefix(tagFormat)

	// Use rev-list to get tags on this branch efficiently
	// This is much faster than listing all tags and checking each one
	cmd := execCommand("git", "tag", "--list", prefix+"*", "--sort=-v:refname")
	output, err := cmd.Output()
	if err != nil {
		fmt.Printf("Error getting tags: %v\n", err)
		return ""
	}

	tags := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(tags) == 0 || (len(tags) == 1 && tags[0] == "") {
		return ""
	}

	// Find the first tag that is on the branch
	for _, tag := range tags {
		// Skip empty tags
		if tag == "" {
			continue
		}

		if isTagOnBranchFunc(tag, branch) {
			// Validate the tag format matches our expected format
			if validateTagFormat(tag, prefix) {
				return tag
			}
		}
	}

	return ""
}

// validateTagFormat checks if the tag matches the expected semantic versioning format
func validateTagFormat(tag, prefix string) bool {
	if !strings.HasPrefix(tag, prefix) {
		return false
	}

	// Extract version numbers
	versionPart := tag[len(prefix):]

	// Split version numbers
	parts := strings.Split(versionPart, ".")
	if len(parts) != 3 {
		return false
	}

	// Validate each part is a number
	for _, part := range parts {
		if _, err := strconv.Atoi(part); err != nil {
			return false
		}
	}

	return true
}

// extractPrefix extracts the prefix from a tag format
func extractPrefix(tagFormat string) string {
	for i, c := range tagFormat {
		if c >= '0' && c <= '9' {
			return tagFormat[:i]
		}
	}
	return tagFormat
}

// isTagOnBranch checks if the given tag is on the specified branch
func isTagOnBranch(tag, branch string) bool {
	// First check if the tag exists
	cmdTagExists := execCommand("git", "show-ref", "--tags", tag)
	if err := cmdTagExists.Run(); err != nil {
		return false
	}

	// Get the commit hash for the tag
	cmdTagCommit := execCommand("git", "rev-list", "-n", "1", tag)
	tagCommit, err := cmdTagCommit.Output()
	if err != nil {
		return false
	}
	tagCommitStr := strings.TrimSpace(string(tagCommit))

	// Try local branch first
	cmdBranchCommit := execCommand("git", "rev-parse", "--verify", branch)
	branchCommit, err := cmdBranchCommit.Output()

	// If local branch doesn't exist, try remote branch
	if err != nil {
		cmdBranchCommit = execCommand("git", "rev-parse", "--verify", "origin/"+branch)
		branchCommit, err = cmdBranchCommit.Output()
		if err != nil {
			// Neither local nor remote branch exists
			return false
		}
	}
	branchCommitStr := strings.TrimSpace(string(branchCommit))

	// Fast check: if tag is the branch tip, return true
	if tagCommitStr == branchCommitStr {
		return true
	}

	// Check if the tag commit is an ancestor of branch commit
	// This is much faster than the previous checks
	cmdMergeBase := execCommand("git", "merge-base", "--is-ancestor", tagCommitStr, branchCommitStr)
	return cmdMergeBase.Run() == nil
}

// calculateNextTag calculates the next tag based on the last tag
func calculateNextTag(lastTag, tagFormat string) string {
	if lastTag == "" {
		return tagFormat // Use the format directly if no last tag
	}

	// Extract prefix and numbers
	prefix := extractPrefix(tagFormat)
	versionPart := lastTag[len(prefix):]

	// Split version numbers
	parts := strings.Split(versionPart, ".")
	if len(parts) != 3 {
		return tagFormat // Fallback to format if version is not in x.y.z format
	}

	// Validate each part is a number
	for _, part := range parts {
		if _, err := strconv.Atoi(part); err != nil {
			// If any part is not a valid number, return the tag format
			return tagFormat
		}
	}

	// Increment the last part
	lastPart, _ := strconv.Atoi(parts[2]) // We already checked this is valid
	parts[2] = strconv.Itoa(lastPart + 1)

	// Combine parts back
	return prefix + strings.Join(parts, ".")
}

// isTagVersionGreater checks if newTag is greater than oldTag
func isTagVersionGreater(newTag, oldTag string) bool {
	if oldTag == "" {
		return true
	}

	// Extract prefix from tags
	prefix := extractPrefix(newTag)

	// Extract version numbers
	newVersion := newTag[len(prefix):]
	oldVersion := oldTag[len(prefix):]

	// Parse version numbers
	newParts := strings.Split(newVersion, ".")
	oldParts := strings.Split(oldVersion, ".")

	// Check if both have three parts
	if len(newParts) != 3 || len(oldParts) != 3 {
		return false
	}

	// Compare major version
	newMajor, _ := strconv.Atoi(newParts[0])
	oldMajor, _ := strconv.Atoi(oldParts[0])
	if newMajor > oldMajor {
		return true
	} else if newMajor < oldMajor {
		return false
	}

	// Compare minor version
	newMinor, _ := strconv.Atoi(newParts[1])
	oldMinor, _ := strconv.Atoi(oldParts[1])
	if newMinor > oldMinor {
		return true
	} else if newMinor < oldMinor {
		return false
	}

	// Compare patch version
	newPatch, _ := strconv.Atoi(newParts[2])
	oldPatch, _ := strconv.Atoi(oldParts[2])
	return newPatch > oldPatch
}

// promptForTag asks the user for the tag to create
func promptForTag(tagFormat, defaultTag, lastTag string) string {
	// Compile regex for tag validation
	prefix := extractPrefix(tagFormat)
	patternStr := "^" + regexp.QuoteMeta(prefix) + "\\d+\\.\\d+\\.\\d+$"
	pattern := regexp.MustCompile(patternStr)

	// Set up colors
	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()

	fmt.Printf("Enter tag (format: %s, default: %s):\n", tagFormat, green(defaultTag))
	fmt.Print("> ")

	// Read user input
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	// If empty, use default
	if input == "" {
		input = defaultTag
	}

	// Validate input format and version
	for {
		// First check format
		if !pattern.MatchString(input) {
			fmt.Printf("Invalid format! Tag should match %s\n", tagFormat)
			fmt.Print("> ")
			input, _ = reader.ReadString('\n')
			input = strings.TrimSpace(input)
			continue
		}

		// Then check if version is greater than the last tag
		// Skip this check if there's no last tag
		if lastTag != "" && !isTagVersionGreater(input, lastTag) {
			fmt.Printf("%s New tag must be greater than the last tag: %s\n", red("Error:"), lastTag)
			fmt.Print("> ")
			input, _ = reader.ReadString('\n')
			input = strings.TrimSpace(input)
			continue
		}

		// If we get here, the tag is valid
		fmt.Printf("Valid tag: %s\n", green(input))
		break
	}

	return input
}

// getAllRemoteURLs gets all remote repository URLs
func getAllRemoteURLs() map[string]string {
	// Get all remotes
	cmd := execCommand("git", "remote")
	output, err := cmd.Output()
	if err != nil {
		return map[string]string{}
	}

	remotes := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(remotes) == 0 || (len(remotes) == 1 && remotes[0] == "") {
		return map[string]string{}
	}

	// Get URL for each remote
	remoteURLs := make(map[string]string)
	for _, remote := range remotes {
		cmd = execCommand("git", "config", "--get", fmt.Sprintf("remote.%s.url", remote))
		url, err := cmd.Output()
		if err == nil {
			remoteURLs[remote] = strings.TrimSpace(string(url))
		}
	}

	return remoteURLs
}

// promptForPushToRemote asks if the tag should be pushed to remote and which remote to use
func promptForPushToRemote(remoteURLs map[string]string) (bool, string) {
	// Ask if user wants to push
	fmt.Print("Do you want to push tag to remote? (Y/n): ")
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.ToLower(strings.TrimSpace(input))

	// If user doesn't want to push, return false
	if input != "" && input != "y" && input != "yes" {
		return false, ""
	}

	// If there's only one remote, use it without asking
	if len(remoteURLs) == 1 {
		for name, url := range remoteURLs {
			fmt.Printf("Using remote: %s (%s)\n", name, url)
			return true, name
		}
	}

	// If there are multiple remotes, let the user choose
	fmt.Println("Select remote to push to:")
	remoteNames := make([]string, 0, len(remoteURLs))
	for name := range remoteURLs {
		remoteNames = append(remoteNames, name)
	}

	// Sort remote names for consistent display
	sort.Strings(remoteNames)

	// Display options
	for i, name := range remoteNames {
		fmt.Printf("%d: %s (%s)\n", i+1, name, remoteURLs[name])
	}

	// Default to first remote
	defaultRemote := remoteNames[0]
	fmt.Printf("Enter number (default: 1 for %s): ", defaultRemote)

	// Read user selection
	input, _ = reader.ReadString('\n')
	input = strings.TrimSpace(input)

	// Handle default or parse selection
	selectedRemote := defaultRemote
	if input != "" {
		if idx, err := strconv.Atoi(input); err == nil && idx > 0 && idx <= len(remoteNames) {
			selectedRemote = remoteNames[idx-1]
		} else {
			fmt.Printf("Invalid selection, using default remote: %s\n", defaultRemote)
		}
	}

	return true, selectedRemote
}

// pushTagToRemote pushes the tag to the specified remote
func pushTagToRemote(tag, remote string) {
	cmd := execCommand("git", "push", remote, tag)
	if err := cmd.Run(); err != nil {
		fmt.Printf("Error pushing tag %s to remote %s: %v\n", tag, remote, err)
		os.Exit(1)
	}
}

// createTag creates a tag on the specified branch
func createTag(branch, tag string) {
	// Get commit hash from branch
	cmd := execCommand("git", "rev-parse", branch)
	commitHash, err := cmd.Output()
	if err != nil {
		fmt.Printf("Error getting commit hash for branch %s: %v\n", branch, err)
		os.Exit(1)
	}

	// Create tag
	cmd = execCommand("git", "tag", tag, strings.TrimSpace(string(commitHash)))
	if err := cmd.Run(); err != nil {
		fmt.Printf("Error creating tag %s: %v\n", tag, err)
		os.Exit(1)
	}
}
