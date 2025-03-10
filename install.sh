#!/usr/bin/env bash
set -e

# ANSI color codes for styling output
RED='\033[1;31m'
GREEN='\033[1;32m'
BLUE='\033[1;34m'
CYAN='\033[1;36m'
YELLOW='\033[1;33m'
RESET='\033[0m'

# Display a header banner
function header() {
	echo -e "${CYAN}========================================${RESET}"
	echo -e "${CYAN}           NVS Installer              ${RESET}"
	echo -e "${CYAN}========================================${RESET}"
}

# Functions for printing messages with colors
function info() {
	echo -e "${BLUE}[INFO]${RESET} $1"
}

function success() {
	echo -e "${GREEN}[SUCCESS]${RESET} $1"
}

function error() {
	echo -e "${RED}[ERROR]${RESET} $1"
}

# Print header at the start
header

REPO="y3owk1n/nvs"
BIN_NAME="nvs" # Base name for the binary

# Detect OS and Architecture, and set INSTALL_DIR and asset name.
OS="$(uname -s)"
ARCH="$(uname -m)"
ASSET=""
INSTALL_DIR=""

info "Detecting operating system and architecture..."
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
		error "Unsupported architecture: $ARCH"
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
		error "Unsupported architecture: $ARCH"
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
	error "Unsupported OS: $OS"
	exit 1
	;;
esac

info "Detected OS: ${YELLOW}${OS}${RESET}"
info "Detected Architecture: ${YELLOW}${ARCH}${RESET}"
info "Preparing to download asset: ${YELLOW}${ASSET}${RESET}"

# Construct the download URL for the latest release asset.
DOWNLOAD_URL="https://github.com/${REPO}/releases/latest/download/${ASSET}"
info "Download URL: ${YELLOW}${DOWNLOAD_URL}${RESET}"

# Function to check for a download tool and perform the download.
download_file() {
	local url=$1
	local output=$2
	if command -v curl >/dev/null 2>&1; then
		info "Using curl to download..."
		curl -L --progress-bar -o "$output" "$url"
	elif command -v wget >/dev/null 2>&1; then
		info "Using wget to download..."
		wget -O "$output" "$url"
	else
		error "Please install curl or wget to download files."
		exit 1
	fi
}

# Download the asset to a temporary file.
TMP_FILE=$(mktemp)
download_file "$DOWNLOAD_URL" "$TMP_FILE"

# Make the file executable (if not on Windows)
if [[ "$OS" != MINGW* && "$OS" != CYGWIN* && "$OS" != MSYS* ]]; then
	chmod +x "$TMP_FILE"
fi

# Determine final target path. On Windows, add the .exe extension.
TARGET_PATH="${INSTALL_DIR}/${BIN_NAME}"
if [[ "$OS" == MINGW* || "$OS" == CYGWIN* || "$OS" == MSYS* ]]; then
	TARGET_PATH="${INSTALL_DIR}/${BIN_NAME}.exe"
fi

info "Installing to ${YELLOW}${TARGET_PATH}${RESET}"
if [ ! -w "$INSTALL_DIR" ]; then
	info "Elevated privileges are required to install to ${INSTALL_DIR}. Prompting for sudo..."
	sudo mv "$TMP_FILE" "$TARGET_PATH"
else
	mv "$TMP_FILE" "$TARGET_PATH"
fi

success "Installation complete!"
echo -e "${CYAN}You can now run:${RESET} ${YELLOW}nvs help${RESET}"
