# Git Tag Publishing Tool

A command-line tool for managing Git tags across multiple branches and pushing them to remote repositories.

## Technologies

- Go
- Git

## Features

- Configuration file with branch-to-tag format mapping
- Interactive command-line interface
  - Branch selection
  - Tag version suggestion (incrementing the last known tag version)
  - Remote repository selection for pushing
- Validation of tag formats with color highlighting
- No branch switching - creates tags on target branches while staying on the current branch

## How It Works

1. Configuration file processing
   - If not found, creates a default configuration: `[{branch: "master", tag: "v0.0.0"}, {branch: "main", tag: "v0.0.0"}, {branch: "gray", tag: "g0.0.0"}]`
   - If found, uses the configuration (validates format)
2. Command-line interaction
   - Prompts to select a branch to tag from configured branches
   - Suggests the next tag version based on the last tag on the selected branch (e.g., v1.0.0 -> v1.0.1)
   - Validates tag input (shows green for valid format, red for invalid)
   - Prompts to select a remote repository for pushing the tag
3. Tag creation and pushing
   - Creates the tag on the specified branch
   - Optionally pushes the tag to the selected remote repository

## Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/chenmijiang/go-git-publish.git
   cd go-git-publish
   ```

2. Run the installation script:
   ```bash
   ./install.sh
   ```

   This script will:
   - Compile the project
   - Add an alias `git-publish` to your shell configuration file (`.bashrc` or `.zshrc`)
   - Provide instructions on how to apply the changes

3. Apply the changes:
   ```bash
   source ~/.bashrc  # or source ~/.zshrc if you're using zsh
   ```

## Usage

Simply run the `git-publish` command from any Git repository:

```bash
git-publish
```

Then follow the interactive prompts.

## Important Notes

1. The tool operates on configured branches without switching your current branch
2. Tag formats must match the pattern specified in the configuration
3. Environment detection ensures you're in a Git repository
4. Tag versions must be greater than the previous tag version
5. Selection of remote repository for pushing tags
6. Skips remote push if no remote repositories are found
