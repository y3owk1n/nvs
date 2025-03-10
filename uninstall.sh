#!/usr/bin/env bash
set -e

BIN_NAME="nvs" # Base name for the binary

# Detect OS and set installation directory.
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
	echo "Unsupported OS: $OS"
	exit 1
	;;
esac

# Set target path; on Windows, the binary includes .exe.
TARGET_PATH="${INSTALL_DIR}/${BIN_NAME}"
if [[ "$OS" == MINGW* || "$OS" == CYGWIN* || "$OS" == MSYS* ]]; then
	TARGET_PATH="${INSTALL_DIR}/${BIN_NAME}.exe"
fi

echo "Removing installed binary at ${TARGET_PATH}..."
if [ -f "${TARGET_PATH}" ]; then
	if [ ! -w "${INSTALL_DIR}" ]; then
		echo "Elevated privileges required. Prompting for sudo..."
		sudo rm -f "${TARGET_PATH}"
	else
		rm -f "${TARGET_PATH}"
	fi
	echo "Uninstallation complete."
	echo "Try running the following to verify the uninstallation, the binary should be removed from bin."
	echo "nvs help"
else
	echo "No installed binary found at ${TARGET_PATH}."
fi
