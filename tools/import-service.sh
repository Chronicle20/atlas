#!/usr/bin/env bash
set -euo pipefail

# Imports a new Chronicle20/atlas-<name> service repo into
# services/atlas-<name>. After this script completes, manually:
#   1. Append a "{name,type:go-service,path,module_path,docker_image,
#      docker_context}" row to .github/config/services.json under
#      .services[].
#   2. Append "    ./services/atlas-<name>/atlas.com/<inner>" to
#      /go.work's `use (...)` block (where <inner> is the inner module
#      directory the imported repo uses).
#   3. Run `docker buildx bake atlas-<name>` to verify the shared
#      Dockerfile builds it.
#
# No per-service Dockerfile is needed — the shared repo-root Dockerfile
# parameterized by ARG SERVICE handles it. See CLAUDE.md
# "Build & Verification" for the post-task-074 workflow.


# -------------------------------
# Configuration
# -------------------------------
GITHUB_ORG="Chronicle20"
TARGET_DIR="services"
WORKDIR="$(mktemp -d)"


# -------------------------------
# Input validation
# -------------------------------
if [[ $# -ne 1 ]]; then
  echo "Usage: $0 <repo-name>"
  exit 1

fi

REPO_NAME="$1"
REPO_URL="git@gh-chronicle20:${GITHUB_ORG}/${REPO_NAME}.git"
TARGET_PATH="${TARGET_DIR}/${REPO_NAME}"
REMOTE_NAME="${REPO_NAME}-rewrite"

echo "=== Importing ${REPO_NAME} into ${TARGET_PATH} ==="

# -------------------------------
# Ensure clean monorepo state
# -------------------------------
if [[ -n "$(git status --porcelain)" ]]; then
  echo "ERROR: Monorepo has uncommitted changes."
  exit 1
fi

# -------------------------------
# Clone repo into temp dir
# -------------------------------

echo "Cloning ${REPO_URL}"
git clone "${REPO_URL}" "${WORKDIR}/${REPO_NAME}"
cd "${WORKDIR}/${REPO_NAME}"

# -------------------------------
# Detect default branch (main/master/etc.)
# -------------------------------
DEFAULT_BRANCH="$(git symbolic-ref --short refs/remotes/origin/HEAD | sed 's@^origin/@@')"
echo "Detected default branch: ${DEFAULT_BRANCH}"

# -------------------------------
# Rewrite history into subdirectory
# -------------------------------

echo "Rewriting history into ${TARGET_PATH}"
git filter-repo --to-subdirectory-filter "${TARGET_PATH}"

# -------------------------------
# Merge into monorepo
# -------------------------------
cd - >/dev/null
git remote add "${REMOTE_NAME}" "${WORKDIR}/${REPO_NAME}"
git fetch "${REMOTE_NAME}"

git merge --allow-unrelated-histories \
  "${REMOTE_NAME}/${DEFAULT_BRANCH}" \
  -m "Import service ${REPO_NAME}"

git remote remove "${REMOTE_NAME}"

# -------------------------------
# Cleanup

# -------------------------------
rm -rf "${WORKDIR}"

echo "=== Successfully imported ${REPO_NAME} ==="

