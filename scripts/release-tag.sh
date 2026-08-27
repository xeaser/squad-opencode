#!/usr/bin/env bash
# Create an annotated vX.Y.Z tag on main only. Does not push.
set -euo pipefail
tag="${1:-${TAG:-}}"
if [ -z "$tag" ]; then
  echo "TAG is required (vX.Y.Z)" >&2
  exit 2
fi
if ! printf '%s\n' "$tag" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+$'; then
  echo "TAG must be vX.Y.Z" >&2
  exit 2
fi
branch="$(git rev-parse --abbrev-ref HEAD)"
if [ "$branch" != "main" ]; then
  echo "checkout main first (on $branch)" >&2
  exit 2
fi
git tag -a "$tag" -m "squad-oc $tag"
echo "created $tag; publish with: task release:push TAG=$tag"
