#!/usr/bin/env bash
# nvs shell hook. This file is embedded into the nvs binary at compile
# time via //go:embed and served by `nvs hook {bash,zsh}`. The same script
# works for both shells: the dispatcher at the bottom auto-detects the
# current shell via BASH_VERSION / ZSH_VERSION. Do not edit the embedded
# copy that ships with the binary; edit this file and rebuild.

_nvs_find_version_file() {
  local dir="$PWD"
  while [[ "$dir" != "/" ]]; do
    if [[ -f "$dir/.nvs-version" ]]; then
      echo "$dir/.nvs-version"
      return
    fi
    dir="$(dirname "$dir")"
  done

  # Check home directory
  if [[ -f "$HOME/.nvs-version" ]]; then
    echo "$HOME/.nvs-version"
  fi
}

_nvs_hook() {
  local nvs_version_file
  nvs_version_file="$(_nvs_find_version_file)"

  if [[ -n "$nvs_version_file" ]]; then
    local version
    version="$(tr -d '[:space:]' < "$nvs_version_file")"

    # Only switch if version changed
    if [[ "$version" != "$_NVS_CURRENT_VERSION" ]]; then
      if nvs use "$version" --force >/dev/null 2>&1; then
        export _NVS_CURRENT_VERSION="$version"
      fi
    fi
  fi
}

# Add hook to PROMPT_COMMAND (bash) or directory-change hook (zsh)
if [[ -n "$BASH_VERSION" ]]; then
  if [[ ! "$PROMPT_COMMAND" =~ "_nvs_hook" ]]; then
    PROMPT_COMMAND="_nvs_hook${PROMPT_COMMAND:+;$PROMPT_COMMAND}"
  fi
elif [[ -n "$ZSH_VERSION" ]]; then
  autoload -Uz add-zsh-hook
  add-zsh-hook chpwd _nvs_hook
  # Run once on shell start
  _nvs_hook
fi
