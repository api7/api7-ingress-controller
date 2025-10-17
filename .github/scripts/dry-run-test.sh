#!/usr/bin/env bash

# Dry-run cherry-pick test for backport commits.

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

die() {
  echo -e "${RED}$1${NC}" >&2
  exit "${2:-1}"
}

if [[ $# -ne 1 ]]; then
  die "Usage: $0 <commit-sha>"
fi

COMMIT_SHA="$1"

if ! [[ "$COMMIT_SHA" =~ ^[0-9a-f]{40}$ ]]; then
  die "Invalid commit SHA: $COMMIT_SHA"
fi

SOURCE_REPO="${SOURCE_REPO:-apache/apisix-ingress-controller}"
TARGET_BRANCH="${TARGET_BRANCH:-master}"
SHORT_SHA="${COMMIT_SHA:0:7}"

[[ "$TARGET_BRANCH" =~ ^[A-Za-z0-9._/-]+$ ]] || die "Invalid TARGET_BRANCH: $TARGET_BRANCH"

echo -e "${YELLOW}Running dry-run cherry-pick for ${SHORT_SHA}${NC}"

if ! git cat-file -e "${COMMIT_SHA}^{commit}" 2>/dev/null; then
  die "Commit $COMMIT_SHA is not available locally - fetch upstream before running this script"
fi

COMMIT_TITLE="$(git log --format='%s' -n 1 "$COMMIT_SHA")"
COMMIT_AUTHOR="$(git log --format='%an <%ae>' -n 1 "$COMMIT_SHA")"

echo -e "${YELLOW}Title: ${COMMIT_TITLE}${NC}"
echo -e "${YELLOW}Author: ${COMMIT_AUTHOR}${NC}"

TEMP_BRANCH="dry-run-test-${SHORT_SHA}"

git fetch origin "$TARGET_BRANCH" --quiet
git checkout "$TARGET_BRANCH" --quiet
git reset --hard "origin/$TARGET_BRANCH" --quiet

if git rev-parse --verify "$TEMP_BRANCH" >/dev/null 2>&1; then
  git branch -D "$TEMP_BRANCH" --quiet
fi

git checkout -b "$TEMP_BRANCH" --quiet

PARENT_COUNT="$(git rev-list --parents -n 1 "$COMMIT_SHA" | awk '{print NF-1}')"

echo -e "${YELLOW}Testing cherry-pick...${NC}"

success=false
if [[ "$PARENT_COUNT" -gt 1 ]]; then
  echo -e "${YELLOW}Merge commit detected; using -m 1${NC}"
  if git cherry-pick -x -m 1 "$COMMIT_SHA" --no-commit 2>/dev/null; then
    success=true
  else
    git cherry-pick --abort 2>/dev/null || git reset --hard HEAD --quiet
  fi
else
  if git cherry-pick -x "$COMMIT_SHA" --no-commit 2>/dev/null; then
    success=true
  else
    git cherry-pick --abort 2>/dev/null || git reset --hard HEAD --quiet
  fi
fi

git reset --hard HEAD --quiet
git checkout "$TARGET_BRANCH" --quiet
git branch -D "$TEMP_BRANCH" --quiet 2>/dev/null || true

if [[ "$success" == "true" ]]; then
  echo -e "${GREEN}Dry-run successful. Commit ${SHORT_SHA} can be backported cleanly.${NC}"
  exit 0
fi

echo -e "${RED}Dry-run failed. Commit ${SHORT_SHA} requires manual conflict resolution.${NC}"
exit 1

