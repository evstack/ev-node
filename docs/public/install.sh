#!/bin/bash

# Define colors for output
RED="\033[31m"
GREEN="\033[32m"
YELLOW="\033[33m"
CYAN="\033[36m"
BOLD="\033[1m"
RESET="\033[0m"

# Function to print headers
print_header() {
    echo -e "${CYAN}${BOLD}--> $1${RESET}"
}

# Function to print success messages
print_success() {
    echo -e "${GREEN}${BOLD}$1${RESET}"
}

# Function to print warnings
print_warning() {
    echo -e "${YELLOW}${BOLD}$1${RESET}"
}

# Function to print errors
print_error() {
    echo -e "${RED}${BOLD}$1${RESET}"
}

# Function to compare versions
compare_versions() {
    if [ "$(printf '%s\n' "$1" "$2" | sort -V | head -n1)" = "$1" ]; then
        if [ "$1" = "$2" ]; then
            return 0 # Equal
        else
            return 1 # First is less
        fi
    else
        return 2 # First is greater
    fi
}

print_header "Downloading ev-npde source code..."

if [ -z "$1" ]; then
  print_error "Usage: install.sh <ev-node-tag|branch>"; exit 1
fi

git clone --depth 1 --branch "$1" https://github.com/evstack/ev-node.git || {
  print_error "Failed to clone ev-node at ref '$1'"; exit 1;
}

cd ev-node || { print_error "Failed to find the downloaded repository."; exit 1; }

print_header "Extracting Go version from go.mod..."
go_mod_version=$(grep "^go " go.mod | cut -d' ' -f2)

if [ -z "$go_mod_version" ]; then
    print_error "Error: Could not find a Go version in go.mod."
    exit 1
fi
formatted_go_version="go${go_mod_version}"
echo -e "   Required Go version: ${BOLD}${formatted_go_version}${RESET}\n"

print_header "Checking if Go is installed..."
if ! which go > /dev/null; then
    print_warning "Go is not installed. Attempting to install Go..."
    curl -sL "https://ev.xyz/install-go.sh" | sh -s "$formatted_go_version"
fi

installed_version=$(go version | awk '{print $3}' | sed 's/go//')
echo -e "   Installed Go version: ${BOLD}${installed_version}${RESET}\n"

print_header "Validating installed Go version..."
compare_versions "$installed_version" "$go_mod_version"
comparison_result=$?

if [ $comparison_result -eq 1 ]; then
    print_error "ERROR: The installed Go version ($installed_version) is less than the required version ($go_mod_version)."
    echo "       Please upgrade your version of Go."
    exit 1
elif [ $comparison_result -eq 2 ]; then
    print_warning "INFO: The installed Go version ($installed_version) is greater than the required version ($go_mod_version)."
    echo "      If you run into issues, try downgrading your version of Go."
fi
echo ""

print_header "Building and installing executable..."
make install
print_success "CLI installed successfully!"

cd ..
print_header "Cleaning up downloads..."
rm -rf ev-node

print_success "Installation completed successfully."
