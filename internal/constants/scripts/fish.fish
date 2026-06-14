# nvs shell hook for fish. This file is embedded into the nvs binary at
# compile time via //go:embed and served by `nvs hook fish`. Do not edit
# the embedded copy that ships with the binary; edit this file and
# rebuild.

function _nvs_find_version_file
  set -l dir "$PWD"
  while test "$dir" != "/"
    if test -f "$dir/.nvs-version"
      echo "$dir/.nvs-version"
      return
    end
    set dir (dirname -- "$dir")
  end

  # Check home directory
  if test -f "$HOME/.nvs-version"
    echo "$HOME/.nvs-version"
  end
end

function _nvs_hook --on-variable PWD
  set -l nvs_version_file (_nvs_find_version_file)

  if test -n "$nvs_version_file"
    set -l nvs_version (string trim < "$nvs_version_file")

    # Only switch if version changed
    if test "$nvs_version" != "$_NVS_CURRENT_VERSION"
      if nvs use "$nvs_version" --force >/dev/null 2>&1
        set -g _NVS_CURRENT_VERSION "$nvs_version"
      end
    end
  end
end

# Run once on shell start
_nvs_hook
