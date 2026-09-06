#!/usr/bin/env bash
# Point flake.nix, package.nix and the install scripts at a released
# version. Binary hashes are read from BUILD_DIR/<asset>.sha256 as
# produced by `just release-ci`. vendorHash is recomputed from go.mod.
#
# Usage: scripts/update-release-refs.sh VERSION [BUILD_DIR]
set -euo pipefail

version="${1:?usage: $0 VERSION [BUILD_DIR]}"
version="${version#v}"
build_dir="${2:-build}"

to_sri() {
  nix hash convert --hash-algo sha256 --to sri "$1"
}

set_hash() {
  local asset="$1" sri="$2"
  perl -0pi -e "s#(/${asset}\)\`\n\s*sha256 = \")sha256-[^\"]+#\${1}${sri}#" package.nix
  grep -q "$sri" package.nix || { echo "failed to set hash for $asset" >&2; exit 1; }
}

for asset in nvs-darwin-arm64 nvs-linux-arm64 nvs-linux-amd64; do
  set_hash "$asset" "$(to_sri "$(cat "$build_dir/$asset.sha256")")"
done

go mod vendor
vendor_hash="$(nix hash path vendor)"
rm -rf vendor
sed -i.bak "s#vendorHash = \"sha256-[^\"]*\"#vendorHash = \"${vendor_hash}\"#" package.nix

sed -i.bak "s#releases/download/v[0-9][^/]*/#releases/download/v${version}/#g" package.nix
sed -i.bak "s#latestVersion = \"[^\"]*\"#latestVersion = \"${version}\"#" flake.nix
sed -i.bak "s#^VERSION=\"[^\"]*\"#VERSION=\"${version}\"#" install.sh
sed -i.bak "s#\$version = \"[^\"]*\"#\$version = \"${version}\"#" install.ps1
rm -f package.nix.bak flake.nix.bak install.sh.bak install.ps1.bak

git --no-pager diff --stat -- flake.nix package.nix install.sh install.ps1
