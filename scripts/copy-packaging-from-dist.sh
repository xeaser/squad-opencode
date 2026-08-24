#!/usr/bin/env bash
set -euo pipefail

DIST="${DIST:-dist}"
REPO="${REPO:?REPO is required (owner/name)}"
TAG="${TAG:?TAG is required (vX.Y.Z)}"
CATALOG="${DIST}/artifacts.json"

if [[ ! -f "${CATALOG}" ]]; then
  echo "missing ${CATALOG}" >&2
  exit 1
fi

copy_one() {
  local typ="$1"
  local dest="$2"
  local path
  path="$(jq -r --arg t "${typ}" '.[] | select(.type == $t) | .path' "${CATALOG}" | head -n 1)"
  if [[ -z "${path}" || "${path}" == "null" ]]; then
    echo "no artifact of type ${typ} in ${CATALOG}" >&2
    exit 1
  fi
  mkdir -p "$(dirname "${dest}")"
  cp "${path}" "${dest}"
}

copy_one "Homebrew Cask" "Casks/squad-oc.rb"

scoop_src="$(jq -r '.[] | select(.type == "Scoop Manifest") | .path' "${CATALOG}" | head -n 1)"
if [[ -z "${scoop_src}" || "${scoop_src}" == "null" ]]; then
  echo "no Scoop Manifest in ${CATALOG}" >&2
  exit 1
fi
mkdir -p bucket
hash_url="https://github.com/${REPO}/releases/download/${TAG}/checksums.txt"
jq --arg url "${hash_url}" '
  .checkver = "github"
  | .autoupdate.hash.url = $url
  | .autoupdate.architecture = (.autoupdate.architecture // .architecture)
  | .autoupdate.architecture["64bit"].url = ((.architecture["64bit"].url // "") | gsub("_[0-9]+\\.[0-9]+\\.[0-9]+_"; "_$version_"))
' "${scoop_src}" > bucket/squad-oc.json

mapfile -t winget_paths < <(jq -r '.[] | select(.type == "Winget Manifest") | .path' "${CATALOG}")
if [[ ${#winget_paths[@]} -eq 0 ]]; then
  echo "no Winget Manifest artifacts" >&2
  exit 1
fi
mkdir -p packaging/winget
rm -f packaging/winget/*.yaml
for p in "${winget_paths[@]}"; do
  cp "${p}" "packaging/winget/$(basename "${p}")"
done
