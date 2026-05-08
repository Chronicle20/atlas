#!/usr/bin/env bash
# Wire this repo's hooks into git by setting core.hooksPath to .githooks.
# Run once per fresh clone.
#
#   ./.githooks/install.sh

set -euo pipefail

repo_root=$(git rev-parse --show-toplevel)
cd "$repo_root"

git config core.hooksPath .githooks
chmod +x .githooks/pre-commit

if ! command -v gitleaks >/dev/null 2>&1; then
  cat <<'EOF'

[install] Hooks installed, but gitleaks is not on PATH.
[install] The pre-commit hook will fail until gitleaks is installed:

    curl -sSfL https://raw.githubusercontent.com/gitleaks/gitleaks/master/install.sh \
      | sh -s -- -b /usr/local/bin

    # or
    brew install gitleaks
    # or
    go install github.com/gitleaks/gitleaks/v8@latest

EOF
else
  echo "[install] Hooks installed. gitleaks: $(gitleaks version 2>&1 | head -n1)"
fi

if [ ! -f .gitleaks.local.toml ]; then
  cat <<'EOF'
[install] No .gitleaks.local.toml found. To scan for your personal
[install] identifiers (username, cluster name, etc), copy the template:
[install]
[install]     cp .gitleaks.local.toml.example .gitleaks.local.toml
[install]     # then edit and replace the placeholders
[install]
[install] The local file is gitignored and will never be committed.
EOF
fi
