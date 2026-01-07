#!/usr/bin/env bash
set -euo pipefail

# Script to remove JavaScript files from tmp directory that have been converted to JSON

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

CONVERSATIONS_DIR="$PROJECT_ROOT/services/atlas-npc-conversations/conversations"
TMP_DIR="$PROJECT_ROOT/services/atlas-npc-conversations/tmp"

if [[ ! -d "$CONVERSATIONS_DIR" ]]; then
  echo "Error: Conversations directory not found: $CONVERSATIONS_DIR"
  exit 1
fi

if [[ ! -d "$TMP_DIR" ]]; then
  echo "Error: Tmp directory not found: $TMP_DIR"
  exit 1
fi

removed=0
skipped=0

for json_file in "$CONVERSATIONS_DIR"/npc_*.json; do
  [[ -e "$json_file" ]] || continue

  # Extract NPC ID from filename (npc_12345.json -> 12345)
  filename=$(basename "$json_file")
  npc_id="${filename#npc_}"
  npc_id="${npc_id%.json}"

  js_file="$TMP_DIR/${npc_id}.js"

  if [[ -f "$js_file" ]]; then
    echo "Removing: $js_file"
    rm "$js_file"
    ((removed++)) || true
  else
    ((skipped++)) || true
  fi
done

echo ""
echo "Done. Removed $removed JavaScript file(s). $skipped conversion(s) had no corresponding JS file."
