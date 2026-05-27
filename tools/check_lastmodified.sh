#!/bin/sh

# Check that all modified or new JSON files in the given inventory paths have a
# valid lastModified timestamp. Modified files must also have an updated value
# relative to HEAD.
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

for path in "$@"; do
    # tracked files changed vs HEAD (staged or unstaged)
    for file in $(git diff --name-only HEAD -- "${path}/*.json"); do
        count=$((count + 1))
        if ! grep -qE "$LASTMOD_PATTERN" "$file"; then
            invalid="${invalid}  ${file}\n"
        elif ! git diff HEAD -- "$file" | grep -q '^+.*"lastModified"'; then
            stale="${stale}  ${file}\n"
        fi
    done

    # untracked new files — validate format only, no update check required
    for file in $(git ls-files --others --exclude-standard -- "${path}/*.json"); do
        count=$((count + 1))
        if ! grep -qE "$LASTMOD_PATTERN" "$file"; then
            invalid="${invalid}  ${file}\n"
        fi
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
