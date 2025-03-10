#!/usr/bin/env bash
set -e

# ANSI color codes for styling output
RED='\033[1;31m'
GREEN='\033[1;32m'
BLUE='\033[1;34m'
CYAN='\033[1;36m'
YELLOW='\033[1;33m'
RESET='\033[0m'

# Function to display a header banner.
function header() {
	echo -e "${CYAN}========================================${RESET}"
	echo -e "${CYAN}          NVS Uninstaller             ${RESET}"
	echo -e "${CYAN}========================================${RESET}"
}

# Functions for printing messages with colors.
function info() {
	echo -e "${BLUE}[INFO]${RESET} $1"
}

function success() {
	echo -e "${GREEN}[SUCCESS]${RESET} $1"
}

function error() {
	echo -e "${RED}[ERROR]${RESET} $1"
}

# Print header at the start.
header

BIN_NAME="nvs" # Base name for the binary

# Detect OS and set the installation directory.
OS="$(uname -s)"
INSTALL_DIR=""

case "$OS" in
Linux | Darwin)
	INSTALL_DIR="/usr/local/bin"
	;;
MINGW* | CYGWIN* | MSYS*)
	INSTALL_DIR="$HOME/AppData/Local/Programs"
	;;
*)
	error "Unsupported OS: $OS"
	exit 1
	;;
esac

# Set target path; on Windows, the binary includes .exe.
TARGET_PATH="${INSTALL_DIR}/${BIN_NAME}"
if [[ "$OS" == MINGW* || "$OS" == CYGWIN* || "$OS" == MSYS* ]]; then
	TARGET_PATH="${INSTALL_DIR}/${BIN_NAME}.exe"
fi

info "Removing installed binary at ${YELLOW}${TARGET_PATH}${RESET}..."
if [ -f "${TARGET_PATH}" ]; then
	if [ ! -w "${INSTALL_DIR}" ]; then
		info "Elevated privileges required. Prompting for sudo..."
		sudo rm -f "${TARGET_PATH}"
	else
		rm -f "${TARGET_PATH}"
	fi
	success "Uninstallation complete."
	echo -e "${CYAN}To verify, run:${RESET} ${YELLOW}nvs help${RESET}"
else
	info "No installed binary found at ${YELLOW}${TARGET_PATH}${RESET}."
fi
