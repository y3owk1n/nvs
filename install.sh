#!/usr/bin/env bash
set -e

REPO="y3owk1n/nvs"
BIN_NAME="nvs" # Base name for the binary

# Detect OS and Architecture, and set INSTALL_DIR and asset name.
OS="$(uname -s)"
ARCH="$(uname -m)"
ASSET=""
INSTALL_DIR=""

case "$OS" in
Linux)
	INSTALL_DIR="/usr/local/bin"
	case "$ARCH" in
	x86_64)
		ASSET="${BIN_NAME}-linux-amd64"
		;;
	aarch64 | arm64)
		ASSET="${BIN_NAME}-linux-arm64"
		;;
	*)
		echo "Unsupported architecture: $ARCH"
		exit 1
		;;
	esac
	;;
Darwin)
	INSTALL_DIR="/usr/local/bin"
	case "$ARCH" in
	x86_64)
		ASSET="${BIN_NAME}-darwin-amd64"
		;;
	arm64)
		ASSET="${BIN_NAME}-darwin-arm64"
		;;
	*)
		echo "Unsupported architecture: $ARCH"
		exit 1
		;;
	esac
	;;
MINGW* | CYGWIN* | MSYS*)
	# For Windows, use a user-specific directory.
	INSTALL_DIR="$HOME/AppData/Local/Programs"
	mkdir -p "$INSTALL_DIR"
	ASSET="${BIN_NAME}-windows64.exe"
	;;
*)
	echo "Unsupported OS: $OS"
	exit 1
	;;
esac

# Function to check for a download tool.
download_file() {
	local url=$1
	local output=$2
	if command -v curl >/dev/null 2>&1; then
		curl -L -o "$output" "$url"
	elif command -v wget >/dev/null 2>&1; then
		wget -O "$output" "$url"
	else
		echo "Error: Please install curl or wget to download files."
		exit 1
	fi
}

# Construct the download URL for the latest release asset.
DOWNLOAD_URL="https://github.com/${REPO}/releases/latest/download/${ASSET}"
echo "Detected OS: $OS"
echo "Detected Architecture: $ARCH"
echo "Downloading asset: $ASSET"
echo "Download URL: $DOWNLOAD_URL"

# Download the asset to a temporary file.
TMP_FILE=$(mktemp)
download_file "$DOWNLOAD_URL" "$TMP_FILE"

# If not on Windows, make the file executable.
if [[ "$OS" != MINGW* && "$OS" != CYGWIN* && "$OS" != MSYS* ]]; then
	chmod +x "$TMP_FILE"
fi

# Determine final target path. On Windows, rename to include .exe.
TARGET_PATH="${INSTALL_DIR}/${BIN_NAME}"
if [[ "$OS" == MINGW* || "$OS" == CYGWIN* || "$OS" == MSYS* ]]; then
	TARGET_PATH="${INSTALL_DIR}/${BIN_NAME}.exe"
fi

echo "Installing to ${TARGET_PATH}"
if [ ! -w "$INSTALL_DIR" ]; then
	echo "Elevated privileges are required to install to ${INSTALL_DIR}. Prompting for sudo..."
	sudo mv "$TMP_FILE" "$TARGET_PATH"
else
	mv "$TMP_FILE" "$TARGET_PATH"
fi

echo "Installation complete."
echo "You can run the following command and get started"
echo "nvs help"
