#!/usr/bin/env bash
# FR-2.4 gate: every handler/validator/writer name in a socket-config template
# must appear as a registered string literal in atlas-login / atlas-channel / libs/atlas-packet.
# Usage: tools/template-symbol-check.sh services/atlas-configurations/seed-data/templates/template_gms_84_1.json
set -euo pipefail
TEMPLATE="${1:?usage: template-symbol-check.sh <template.json>}"
ROOT="$(git rev-parse --show-toplevel)"
SEARCH_PATHS=("$ROOT/services/atlas-login" "$ROOT/services/atlas-channel" "$ROOT/libs/atlas-packet")

names() { # extract distinct values for a JSON key under socket.<group>
  python3 -c "import json,sys; d=json.load(open('$TEMPLATE')); s=d.get('socket',{}); \
print('\n'.join(sorted({h.get('$1','') for h in s.get('$2',[]) if h.get('$1')})))"
}

missing=0
check() {
  local name="$1"
  [ -z "$name" ] && return 0
  if ! grep -rqF "\"$name\"" "${SEARCH_PATHS[@]}" --include='*.go'; then
    echo "DANGLING: $name (no registered string literal found)"
    missing=1
  fi
}

while IFS= read -r n; do check "$n"; done < <(names validator handlers)
while IFS= read -r n; do check "$n"; done < <(names handler  handlers)
while IFS= read -r n; do check "$n"; done < <(names writer   writers)

if [ "$missing" -ne 0 ]; then
  echo "FAIL: template has dangling symbol references"
  exit 1
fi
echo "OK: all template symbols resolve"
