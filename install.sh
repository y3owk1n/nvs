#!/usr/bin/env bash
set -euo pipefail

# Global variable for temporary file.
TMP_FILE=""

# Cleanup function to remove temporary files.
cleanup() {
	if [[ -n "${TMP_FILE:-}" && -f "$TMP_FILE" ]]; then
		rm -f "$TMP_FILE"
	fi
}
trap cleanup EXIT INT TERM ERR

# ANSI color codes.
RED='\033[1;31m'
GREEN='\033[1;32m'
BLUE='\033[1;34m'
CYAN='\033[1;36m'
YELLOW='\033[1;33m'
RESET='\033[0m'

# Logging functions with timestamps.
log_info() {
	echo -e "$(date +"%Y-%m-%d %H:%M:%S") ${BLUE}[INFO]${RESET} $1"
}
log_success() {
	echo -e "$(date +"%Y-%m-%d %H:%M:%S") ${GREEN}[SUCCESS]${RESET} $1"
}
log_error() {
	echo -e "$(date +"%Y-%m-%d %H:%M:%S") ${RED}[ERROR]${RESET} $1"
}

# Header banner.
header() {
	echo -e "${CYAN}========================================${RESET}"
	echo -e "${CYAN}           NVS Installer              ${RESET}"
	echo -e "${CYAN}========================================${RESET}"
}
header

# Dependency check: require curl or wget.
if ! command -v curl >/dev/null 2>&1; then
	log_error "curl is required to download files. Please install curl."
	exit 1
fi

REPO="y3owk1n/nvs"
BIN_NAME="nvs" # Base name for the binary

# Detect OS and architecture.
OS="$(uname -s)"
ARCH="$(uname -m)"
ASSET=""
INSTALL_DIR=""

log_info "Detecting operating system and architecture..."
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
		log_error "Unsupported architecture: $ARCH"
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
		log_error "Unsupported architecture: $ARCH"
		exit 1
		;;
	esac
	;;
MINGW* | CYGWIN* | MSYS*)
	INSTALL_DIR="$HOME/AppData/Local/Programs"
	mkdir -p "$INSTALL_DIR"
	ASSET="${BIN_NAME}-windows64.exe"
	;;
*)
	log_error "Unsupported OS: $OS"
	exit 1
	;;
esac

log_info "Detected OS: ${YELLOW}$OS${RESET}"
log_info "Detected Architecture: ${YELLOW}$ARCH${RESET}"
log_info "Preparing to download asset: ${YELLOW}$ASSET${RESET}"

DOWNLOAD_URL="https://github.com/${REPO}/releases/latest/download/${ASSET}"
log_info "Download URL: ${YELLOW}$DOWNLOAD_URL${RESET}"

# Function to download files.
download_file() {
	local url="$1"
	local output="$2"
	log_info "Downloading binary from: ${YELLOW}$url${RESET}"
	curl -L --progress-bar -o "$output" "$url"
}

# Download the asset to a temporary file.
TMP_FILE=$(mktemp)
download_file "$DOWNLOAD_URL" "$TMP_FILE"

# --- Checksum Verification ---
CHECKSUM_URL="https://github.com/${REPO}/releases/latest/download/${ASSET}.sha256"
TMP_CHECKSUM=$(mktemp)
log_info "Downloading checksum from: ${YELLOW}$CHECKSUM_URL${RESET}"
# Attempt to download the checksum file using curl with --fail.
if curl -L --fail --progress-bar -o "$TMP_CHECKSUM" "$CHECKSUM_URL"; then
	log_info "Extracting expected checksum from the checksum file..."

	EXPECTED_CHECKSUM=$(awk '{ print $1 }' "$TMP_CHECKSUM")
	log_info "Expected checksum: ${YELLOW}$EXPECTED_CHECKSUM${RESET}"

	log_info "Computing checksum of the downloaded asset..."
	if command -v sha256sum >/dev/null 2>&1; then
		COMPUTED_CHECKSUM=$(sha256sum "$TMP_FILE" | awk '{ print $1 }')
	else
		COMPUTED_CHECKSUM=$(shasum -a 256 "$TMP_FILE" | awk '{ print $1 }')
	fi
	log_info "Computed checksum: ${YELLOW}$COMPUTED_CHECKSUM${RESET}"

	if [[ "$EXPECTED_CHECKSUM" != "$COMPUTED_CHECKSUM" ]]; then
		log_error "Checksum verification failed! The downloaded file may be corrupted."
		rm -f "$TMP_CHECKSUM"
		exit 1
	else
		log_success "Checksum verification passed."
	fi
	rm -f "$TMP_CHECKSUM"
else
	log_error "Checksum file not found at ${YELLOW}$CHECKSUM_URL${RESET}. Aborting installation."
	rm -f "$TMP_CHECKSUM"
	exit 1
fi
# --- End Checksum Verification ---

# Make executable if not on Windows.
if [[ "$OS" != MINGW* && "$OS" != CYGWIN* && "$OS" != MSYS* ]]; then
	chmod +x "$TMP_FILE"
fi

# Set final target path.
TARGET_PATH="${INSTALL_DIR}/${BIN_NAME}"
if [[ "$OS" == MINGW* || "$OS" == CYGWIN* || "$OS" == MSYS* ]]; then
	TARGET_PATH="${INSTALL_DIR}/${BIN_NAME}.exe"
fi

log_info "Installing to ${YELLOW}$TARGET_PATH${RESET}"
if [ ! -w "$INSTALL_DIR" ]; then
	log_info "Elevated privileges required to install to ${INSTALL_DIR}. Prompting for sudo..."
	sudo mv "$TMP_FILE" "$TARGET_PATH"
else
	mv "$TMP_FILE" "$TARGET_PATH"
fi

log_success "Installation complete!"
echo -e "${CYAN}You can now run: ${YELLOW}nvs help${RESET}"
