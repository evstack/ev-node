#!/bin/bash

if [[ "$OSTYPE" == "darwin"* ]]; then
	echo "Detected macOS. Installing jq..."
	if ! command -v brew &> /dev/null; then
		echo "Homebrew is not installed. Installing Homebrew..."
		/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
		# adding /opt/homebrew/bin to the $PATH variable based on the shell
		if [[ -f "$HOME/.bash_profile" ]]; then
			echo "export PATH=\"/opt/homebrew/bin:\$PATH\"" >> "$HOME/.bash_profile"
			source "$HOME/.bash_profile"
		elif [[ -f "$HOME/.bashrc" ]]; then
			echo "export PATH=\"/opt/homebrew/bin:\$PATH\"" >> "$HOME/.bashrc"
			source "$HOME/.bashrc"
		elif [[ -f "$HOME/.zshrc" ]]; then
			echo "export PATH=\"/opt/homebrew/bin:\$PATH\"" >> "$HOME/.zshrc"
			source "$HOME/.zshrc"
		else
			echo "Unsupported shell. Please add /opt/homebrew/bin to your PATH manually."
			exit 1
		fi
	fi  # Closing the brew installation check
	brew install jq
	echo "jq has been installed successfully."
elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
	echo "Detected Linux. Installing jq..."
	if command -v apt &> /dev/null; then
		sudo apt update
		sudo apt install -y jq
	elif command -v yum &> /dev/null; then
		sudo yum install -y epel-release
		sudo yum install -y jq
	else
		echo "Unsupported package manager. Please install jq manually."
		exit 1
	fi
	echo "jq has been installed successfully."
else
	echo "Unsupported operating system."
	exit 1
fi  # Closing the OS type check

