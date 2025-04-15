#!/bin/bash

set -e

# Define colors
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${GREEN}Starting git-publish installation...${NC}"

# Get the directory where the script is located
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]:-$0}" )" &> /dev/null && pwd )"
cd "$SCRIPT_DIR"

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo -e "${RED}Error: Go is not installed. Please install Go first (https://golang.org/doc/install)${NC}"
    exit 1
fi

# Build go-git-publish
echo -e "${YELLOW}Compiling go-git-publish...${NC}"
go build -o go-git-publish

# Ensure the build was successful
if [ ! -f "$SCRIPT_DIR/go-git-publish" ]; then
    echo -e "${RED}Error: Build failed${NC}"
    exit 1
fi

echo -e "${GREEN}Build successful!${NC}"

# Set up the alias
ALIAS_CMD="alias git-publish=\"$SCRIPT_DIR/go-git-publish\""

# Detect which shell is being used
SHELL_TYPE=$(basename "$SHELL")

case "$SHELL_TYPE" in
    bash)
        CONFIG_FILE="$HOME/.bashrc"
        ;;
    zsh)
        CONFIG_FILE="$HOME/.zshrc"
        ;;
    *)
        echo -e "${YELLOW}Unrecognized shell: $SHELL_TYPE${NC}"
        echo -e "${YELLOW}Please manually add the following command to your shell configuration file:${NC}"
        echo -e "${GREEN}$ALIAS_CMD${NC}"
        exit 0
        ;;
esac

# Check if the alias already exists
if grep -q "alias git-publish=" "$CONFIG_FILE"; then
    echo -e "${YELLOW}git-publish alias already exists, updating...${NC}"
    # Detect operating system
    OS_TYPE=$(uname)
    if [ "$OS_TYPE" = "Darwin" ]; then
        # macOS requires different sed syntax
        sed -i '' "s|alias git-publish=.*|$ALIAS_CMD|" "$CONFIG_FILE"
    else
        # Standard sed syntax for Linux
        sed -i "s|alias git-publish=.*|$ALIAS_CMD|" "$CONFIG_FILE"
    fi
else
    # Add the new alias
    echo "" >> "$CONFIG_FILE"
    echo "# git-publish tool alias" >> "$CONFIG_FILE"
    echo "$ALIAS_CMD" >> "$CONFIG_FILE"
fi

echo -e "${GREEN}Installation successful!${NC}"
echo -e "${YELLOW}Please run the following command to activate the alias:${NC}"
echo -e "${GREEN}source $CONFIG_FILE${NC}"
echo ""
echo -e "${YELLOW}Or restart your terminal${NC}"
echo -e "${GREEN}You can now use 'git-publish' command in any Git repository${NC}" 
