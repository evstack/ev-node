#!/bin/bash -e

# This script installs or updates to the latest version of Go.
# Multi-platform (Linux and macOS)
# Multi-architecture (amd64, arm64, arm) support

# if curl is not installed then install it
if ! command -v curl &> /dev/null; then
	echo "curl is not installed. Please install curl and try again."
	exit 1
fi

# if jq is not installed then install it using the script
if ! command -v jq &> /dev/null; then
	echo "jq is not installed. Downloading and executing the script to install jq..."
	curl -sSL https://ev.xyz/install-jq.sh | bash
fi

# Define the Go binary path
GO_BIN_PATH="/usr/local/go/bin"
GO_UNTAR_PATH="/usr/local"

version="${1:-$(curl -sSL 'https://go.dev/dl/?mode=json' | jq -r '.[0].version')}"

current="$($GO_BIN_PATH/go version 2>/dev/null | awk '{print $3}')"
if [[ "$current" == "$version" ]]; then
	echo "Go is already up-to-date at version ${version}"
	exit 0
fi

update_go() {
	local arch="$1"
	local os="$2"

	local go_url="https://golang.org/dl/${version}.${os}-${arch}.tar.gz"

	echo "Downloading Go from ${go_url}"

	curl -so "/tmp/${version}.${os}-${arch}.tar.gz" -L "$go_url"
	if [ $? -eq 0 ]; then
		tar -C $GO_UNTAR_PATH -xzf "/tmp/${version}.${os}-${arch}.tar.gz"
		if [ $? -ne 0 ]; then
			echo "Failed to extract Go. Possibly corrupted download."
			rm "/tmp/${version}.${os}-${arch}.tar.gz"
			exit 1
		else
			echo "Go updated to version ${version}"
		fi
	else
		echo "Failed to download Go from ${go_url}"
		exit 1
	fi

	rm "/tmp/${version}.${os}-${arch}.tar.gz"
}

# Function to add path to the specific shell config file
add_path_to_config() {
    local config_file="$1"

    if ! grep -q "export PATH=.*$GO_BIN_PATH" "$config_file" ; then
	    echo "export PATH=\"\$PATH:$GO_BIN_PATH\"" >> "$config_file"
	    echo "Added $GO_BIN_PATH to $config_file"
    else
	    echo "$GO_BIN_PATH is already in $config_file"
    fi
}

case "$(uname -s)" in
	Linux)
		OS="linux"
		;;
	Darwin)
		OS="darwin"
		;;
	*)
		echo "Unsupported operating system: $(uname -s)" >&2
		exit 1
		;;
esac

# Determine the architecture
case "$(uname -m)" in
	armv6l)
		ARCH="armv6l"
		;;
	armv7l)
		ARCH="armv7l"
		;;
	arm64)
		ARCH="arm64"
		;;
	x86_64)
		ARCH="amd64"
		;;
	*)
		echo "Unsupported architecture: $(uname -m)" >&2
		exit 1
		;;
esac

update_go "$ARCH" "$OS"

# Determine shell and appropriate config file
if [[ -n "$ZSH_VERSION" ]]; then
	# Assuming the user is using Zsh
	CONFIG_FILE="$HOME/.zshenv"
elif [[ -n "$BASH_VERSION" ]]; then
	if [[ -f "$HOME/.bashrc" ]]; then
		CONFIG_FILE="$HOME/.bashrc"
	elif [[ -f "$HOME/.bash_profile" ]]; then
		CONFIG_FILE="$HOME/.bash_profile"
	else
		CONFIG_FILE="$HOME/.profile"
	fi
else
	echo "Unsupported shell. Only Bash and Zsh are supported."
	exit 1
fi

add_path_to_config "$CONFIG_FILE"

$(which go) version

echo "Now run 'source $CONFIG_FILE' to update your environment"
