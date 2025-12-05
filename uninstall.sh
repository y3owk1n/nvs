# For Windows users: Please use uninstall.ps1 instead
#!/usr/bin/env bash
set -euo pipefail

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
	echo -e "${CYAN}            NVS Uninstaller           ${RESET}"
	echo -e "${CYAN}========================================${RESET}"
}
header

BIN_NAME="nvs"
OS="$(uname -s)"
INSTALL_DIR=""

case "$OS" in
Linux | Darwin)
	INSTALL_DIR="/usr/local/bin"
	;;
*)
	log_error "Unsupported OS: $OS"
	exit 1
	;;
esac

TARGET_PATH="${INSTALL_DIR}/${BIN_NAME}"

log_info "Removing installed binary at ${YELLOW}$TARGET_PATH${RESET}..."
if [ -f "${TARGET_PATH}" ]; then
	if [ ! -w "${INSTALL_DIR}" ]; then
		log_info "Elevated privileges required. Prompting for sudo..."
		sudo rm -f "${TARGET_PATH}"
	else
		rm -f "${TARGET_PATH}"
	fi
	log_success "Uninstallation complete."
	echo -e "${CYAN}To verify, run: ${YELLOW}nvs help${RESET}"
else
	log_info "No installed binary found at ${YELLOW}$TARGET_PATH${RESET}."
fi
