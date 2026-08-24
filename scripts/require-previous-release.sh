#!/usr/bin/env bash
# Refuse to publish a newer stable tag while the previous stable tag has no GitHub Release.
set -euo pipefail

current="${1:-}"
if [[ -z "${current}" ]]; then
  echo "usage: require-previous-release.sh vX.Y.Z" >&2
  exit 2
fi
if [[ ! "${current}" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "skip previous-release check for non-stable tag ${current}"
  exit 0
fi

mapfile -t tags < <(git tag --list 'v*' --sort=v:refname | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$' || true)
prev=""
for t in "${tags[@]}"; do
  if [[ "${t}" == "${current}" ]]; then
    break
  fi
  prev="${t}"
done
if [[ -z "${prev}" ]]; then
  echo "no previous stable tag before ${current}"
  exit 0
fi
if gh release view "${prev}" --json tagName >/dev/null 2>&1; then
  echo "previous ${prev} has a GitHub Release"
  exit 0
fi
echo "::error::previous tag ${prev} has no GitHub Release; retry ${prev} (Actions → release → workflow_dispatch) instead of skipping to ${current}"
exit 1
