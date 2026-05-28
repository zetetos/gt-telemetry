#!/bin/sh

# Check that all modified or new JSON files in the given inventory paths have a
# valid lastModified timestamp. Modified files must also have an updated value
# relative to the previous commit or HEAD (whichever diff is applicable).
#
# Checks two sources of modifications:
#   - Files changed in the latest commit (HEAD~1..HEAD), for CI
#   - Files with uncommitted changes vs HEAD, for local development
#
# Usage: check_lastmodified.sh <path>...

set -eu

LASTMOD_PATTERN='"lastModified" *: *"[0-9]{4}-(0[1-9]|1[0-2])-(0[1-9]|[12][0-9]|3[01])T([01][0-9]|2[0-3]):[0-5][0-9]:[0-5][0-9]Z"'

if [ "$#" -lt 1 ]; then
  echo "Usage: $0 <path>..." >&2
  exit 1
fi

stale=""
invalid=""
count=0
processed=""

has_parent_commit() {
    git rev-parse --verify HEAD~1 >/dev/null 2>&1
}

lastmod_updated() {
    file="$1"

    if git diff HEAD -- "$file" | grep -q '^+.*"lastModified"'; then
        return 0
    fi

    if has_parent_commit && git diff HEAD~1 HEAD -- "$file" | grep -q '^+.*"lastModified"'; then
        return 0
    fi

    return 1
}

validate() {
    file="$1"
    check_stale="$2"

    # skip if already processed (deduplicate across sources)
    case "$processed" in
        *"|${file}|"*) return ;;
    esac
    processed="${processed}|${file}|"
    count=$((count + 1))

    if ! grep -qE "$LASTMOD_PATTERN" "$file"; then
        invalid="${invalid}  ${file}\n"
        return
    fi

    if [ "$check_stale" = "true" ] && ! lastmod_updated "$file"; then
        stale="${stale}  ${file}\n"
    fi
}

for path in "$@"; do
    # files changed in the latest commit — catches CI pushes
    if has_parent_commit; then
        for file in $(git diff --name-only HEAD~1 HEAD -- "${path}/*.json"); do
            validate "$file" "true"
        done
    fi

    # tracked files with uncommitted changes vs HEAD — catches local development
    # note: word-split on filenames with spaces would break these loops
    for file in $(git diff --name-only HEAD -- "${path}/*.json"); do
        validate "$file" "true"
    done

    # untracked new files — validate format only, no update check required
    for file in $(git ls-files --others --exclude-standard -- "${path}/*.json"); do
        validate "$file" "false"
    done
done

errors=""

if [ -n "$invalid" ]; then
    errors="${errors}Error: the following files have a missing or invalid lastModified timestamp:\n${invalid}"
fi

if [ -n "$stale" ]; then
    errors="${errors}Error: the following files were modified without updating lastModified:\n${stale}"
fi

if [ -n "$errors" ]; then
    printf '%b' "$errors"
    exit 1
fi

if [ "$count" -eq 0 ]; then
    echo "No modified inventory files found."
else
    echo "${count} inventory file(s) are ready for release."
fi
