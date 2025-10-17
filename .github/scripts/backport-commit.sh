#!/usr/bin/env bash

# Safe backport helper. Creates a PR in the current repository that cherry-picks a commit from upstream.

set -euo pipefail

# ANSI colors for readability
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

die() {
  echo -e "${RED}$1${NC}" >&2
  exit "${2:-1}"
}

require_env() {
  local name="$1"
  local value="${!name:-}"
  if [[ -z "$value" ]]; then
    die "Environment variable $name is required"
  fi
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
GITHUB_REPO="${GITHUB_REPOSITORY:-}"

require_env SOURCE_REPO
require_env TARGET_BRANCH
require_env GH_TOKEN

[[ "$SOURCE_REPO" =~ ^[A-Za-z0-9._-]+/[A-Za-z0-9._-]+$ ]] || die "Invalid SOURCE_REPO: $SOURCE_REPO"
[[ "$TARGET_BRANCH" =~ ^[A-Za-z0-9._/-]+$ ]] || die "Invalid TARGET_BRANCH: $TARGET_BRANCH"

if [[ -z "$GITHUB_REPO" ]]; then
  GITHUB_REPO="$(gh repo view --json nameWithOwner -q '.nameWithOwner')"
fi

[[ "$GITHUB_REPO" =~ ^[A-Za-z0-9._-]+/[A-Za-z0-9._-]+$ ]] || die "Invalid target repo: $GITHUB_REPO"

echo -e "${YELLOW}Backporting commit ${COMMIT_SHA} from ${SOURCE_REPO}${NC}"

ORIGINAL_REF=""
ORIGINAL_COMMIT=""
if ORIGINAL_REF=$(git symbolic-ref --quiet HEAD 2>/dev/null); then
  ORIGINAL_REF=${ORIGINAL_REF#refs/heads/}
else
  ORIGINAL_COMMIT=$(git rev-parse HEAD)
fi

restore_original_ref() {
  if [[ -n "$ORIGINAL_REF" ]]; then
    git checkout "$ORIGINAL_REF" >/dev/null 2>&1 || true
  elif [[ -n "$ORIGINAL_COMMIT" ]]; then
    git checkout --detach "$ORIGINAL_COMMIT" >/dev/null 2>&1 || true
  fi
}

if ! git cat-file -e "${COMMIT_SHA}^{commit}" 2>/dev/null; then
  die "Commit $COMMIT_SHA is not available locally - fetch upstream before running this script"
fi

COMMIT_TITLE="$(git log --format='%s' -n 1 "$COMMIT_SHA")"
COMMIT_AUTHOR="$(git log --format='%an <%ae>' -n 1 "$COMMIT_SHA")"
COMMIT_URL="https://github.com/${SOURCE_REPO}/commit/${COMMIT_SHA}"
SHORT_SHA="${COMMIT_SHA:0:7}"
if [[ -z "$COMMIT_TITLE" ]]; then
  COMMIT_TITLE="Backport ${SHORT_SHA} from ${SOURCE_REPO}"
fi
TITLE_SUFFIX=" (${SHORT_SHA})"
if [[ "$COMMIT_TITLE" == *"$SHORT_SHA"* ]]; then
  TITLE_SUFFIX=""
fi
BRANCH_NAME="backport/${SHORT_SHA}-to-${TARGET_BRANCH}"

[[ "$BRANCH_NAME" =~ ^[A-Za-z0-9._/-]+$ ]] || die "Generated branch name is unsafe: $BRANCH_NAME"

echo -e "${YELLOW}Generated branch name: ${BRANCH_NAME}${NC}"

SEARCH_QUERY="${COMMIT_URL} in:body"
EXISTING_PR="$(gh pr list --state all --search "$SEARCH_QUERY" --json url --jq '.[0].url' 2>/dev/null || true)"
if [[ -n "$EXISTING_PR" ]]; then
  echo -e "${GREEN}PR already exists: ${EXISTING_PR}. Skipping duplicate.${NC}"
  exit 0
fi

git fetch origin "$TARGET_BRANCH" --quiet
git checkout -B "$TARGET_BRANCH" "origin/$TARGET_BRANCH"

if git rev-parse --verify "$BRANCH_NAME" >/dev/null 2>&1; then
  git checkout "$BRANCH_NAME"
  git reset --hard "origin/$TARGET_BRANCH"
else
  git checkout -b "$BRANCH_NAME"
fi

PARENT_COUNT="$(git rev-list --parents -n 1 "$COMMIT_SHA" | awk '{print NF-1}')"
HAS_CONFLICTS=false

echo -e "${YELLOW}Running cherry-pick...${NC}"

cherry_pick() {
  if [[ "$PARENT_COUNT" -gt 1 ]]; then
    git cherry-pick -x -m 1 "$COMMIT_SHA"
  else
    git cherry-pick -x "$COMMIT_SHA"
  fi
}

if ! cherry_pick; then
  echo -e "${YELLOW}Cherry-pick reported conflicts; leaving markers for manual resolution.${NC}"
  HAS_CONFLICTS=true
  git add .
  git -c core.editor=true cherry-pick --continue || true
fi

echo -e "${YELLOW}Pushing branch to origin...${NC}"
if ! git push -u origin "$BRANCH_NAME"; then
  echo -e "${YELLOW}Push failed, trying force-with-lease...${NC}"
  git fetch origin "$BRANCH_NAME" || true
  git branch --set-upstream-to="origin/$BRANCH_NAME" "$BRANCH_NAME" || true
  git push -u origin "$BRANCH_NAME" --force-with-lease || {
    git checkout "$TARGET_BRANCH"
    git branch -D "$BRANCH_NAME" || true
    restore_original_ref
    die "Unable to push branch ${BRANCH_NAME}"
  }
fi

echo -e "${YELLOW}Creating pull request...${NC}"

if [[ "$HAS_CONFLICTS" == "true" ]]; then
  PR_TITLE="conflict: ${COMMIT_TITLE}${TITLE_SUFFIX}"
  PR_BODY=$(cat <<EOF
## ⚠️ Backport With Conflicts

- Upstream commit: ${COMMIT_URL}
- Original title: ${COMMIT_TITLE}
- Original author: ${COMMIT_AUTHOR}

This PR contains unresolved conflicts. Please resolve them before merging.

### Suggested workflow
1. \`git fetch origin ${BRANCH_NAME}\`
2. \`git checkout ${BRANCH_NAME}\`
3. Resolve conflicts, commit, and push updates.

> Created automatically by backport-bot.
EOF
)
  LABEL_FLAGS=(--label backport --label automated --label needs-manual-action --label conflicts)
else
  PR_TITLE="${COMMIT_TITLE}${TITLE_SUFFIX}"
  PR_BODY=$(cat <<EOF
## 🔄 Automated Backport

- Upstream commit: ${COMMIT_URL}
- Original title: ${COMMIT_TITLE}
- Original author: ${COMMIT_AUTHOR}

Please review and run the relevant validation before merging.

> Created automatically by backport-bot.
EOF
)
  LABEL_FLAGS=(--label backport --label automated)
fi

set +e
PR_RESPONSE="$(gh pr create \
  --title "$PR_TITLE" \
  --body "$PR_BODY" \
  --head "$BRANCH_NAME" \
  --base "$TARGET_BRANCH" \
  --repo "$GITHUB_REPO" \
  "${LABEL_FLAGS[@]}" 2>&1)"
PR_EXIT_CODE=$?
set -e

if [[ $PR_EXIT_CODE -ne 0 ]]; then
  echo -e "${RED}Failed to create PR:${NC}\n${PR_RESPONSE}"
  if grep -q "already exists" <<<"$PR_RESPONSE"; then
    echo -e "${YELLOW}Detected existing PR, assuming success.${NC}"
    git checkout "$TARGET_BRANCH"
    restore_original_ref
    exit 0
  fi
  git checkout "$TARGET_BRANCH"
  git push origin --delete "$BRANCH_NAME" || true
  git branch -D "$BRANCH_NAME" || true
  restore_original_ref
  die "PR creation failed"
fi

echo -e "${GREEN}Pull request created successfully:${NC} ${PR_RESPONSE}"

restore_original_ref

echo -e "${GREEN}Backport finished for ${COMMIT_SHA}.${NC}"
