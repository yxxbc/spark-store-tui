#!/usr/bin/env bash
# Mirror the current repository and release to Gitee + CNB using repo-all-sync.
# GitHub releases are handled by .github/workflows/release.yml.
set -euo pipefail

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$root"

VERSION=${VERSION:-$(tr -d '[:space:]' < VERSION)}
DRY_RUN=0
BUILD=0
SKIP_PUSH=0
SKIP_RELEASE=0
GITHUB=0

usage() {
  cat <<'EOF'
Usage: scripts/sync-all.sh [options]

  --build         build dist/ before publishing
  --dry-run       show the operations without changing remote state
  --skip-push     do not mirror source
  --skip-release  do not create releases
  --github        also publish the GitHub release (needs gh)
  --version V     override VERSION
EOF
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --build) BUILD=1 ;;
    --dry-run) DRY_RUN=1 ;;
    --skip-push) SKIP_PUSH=1 ;;
    --skip-release) SKIP_RELEASE=1 ;;
    --github) GITHUB=1 ;;
    --version) VERSION="$2"; shift ;;
    -h|--help) usage; exit 0 ;;
    *) echo "unknown option: $1" >&2; usage >&2; exit 2 ;;
  esac
  shift
done

if command -v repo-all-sync >/dev/null 2>&1; then
  CLI=(repo-all-sync)
else
  CLI=(npx --yes repo-all-sync@0.2.1)
fi

export GITEE_OWNER="${GITEE_OWNER:-spark-store-project}"

"${CLI[@]}" doctor

DRY_FLAG=()
[ "$DRY_RUN" -eq 1 ] && DRY_FLAG=(--dry-run)

if [ "$BUILD" -eq 1 ]; then
  scripts/build-dist.sh
fi

if [ "$SKIP_PUSH" -eq 0 ]; then
  "${CLI[@]}" push "${DRY_FLAG[@]}"
fi

if [ "$SKIP_RELEASE" -eq 0 ]; then
  if [ ! -d dist ]; then
    echo "dist/ is missing; rebuild with --build or run scripts/build-dist.sh" >&2
    exit 1
  fi
  if [ ! -f dist/RELEASE_NOTES.md ]; then
    scripts/release-notes.sh > dist/RELEASE_NOTES.md
  fi
  RELEASE_ARGS=(release --version "$VERSION" --assets dist --notes dist/RELEASE_NOTES.md)
  [ "$GITHUB" -eq 1 ] && RELEASE_ARGS+=(--github)
  RELEASE_ARGS+=("${DRY_FLAG[@]}")
  "${CLI[@]}" "${RELEASE_ARGS[@]}"
fi
