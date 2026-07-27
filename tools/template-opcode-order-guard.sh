#!/usr/bin/env bash
# template-opcode-order-guard.sh — enforces that every tenant socket-config
# template lists its handlers and writers in STRICTLY ASCENDING opcode order.
#
# Rationale: the handler/writer arrays are read into an opcode-keyed dispatch
# map, so order is functionally irrelevant to the server — which is exactly why
# it drifts. Keeping both arrays sorted by opCode makes templates diffable,
# makes "is opcode 0xNN already routed?" answerable by eye, and keeps merges
# from silently shuffling entries. New handlers/writers MUST be inserted at
# their sorted position, never appended next to a semantically-related entry.
#
# See docs/packets/TEMPLATE_CONVENTIONS.md. Pure shell + python3, no Go setup.
# Run from the repo root; non-empty diagnostics → non-zero exit.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
TEMPLATE_DIR="$ROOT/services/atlas-configurations/seed-data/templates"

python3 - "$TEMPLATE_DIR" <<'PY'
import glob, json, os, sys

tmpl_dir = sys.argv[1]
bad = 0
checked = 0
for path in sorted(glob.glob(os.path.join(tmpl_dir, "template_*.json"))):
    try:
        d = json.load(open(path))
    except Exception as e:
        print("PARSE ERROR: %s: %s" % (os.path.basename(path), e))
        bad += 1
        continue
    sock = d.get("socket", {})
    for group in ("handlers", "writers"):
        arr = sock.get(group)
        if not arr:
            continue
        checked += 1
        prev = None
        prev_label = None
        for e in arr:
            if not isinstance(e, dict) or "opCode" not in e:
                continue
            try:
                code = int(e["opCode"], 16)
            except (TypeError, ValueError):
                print("BAD opCode in %s %s: %r" % (os.path.basename(path), group, e.get("opCode")))
                bad += 1
                continue
            label = e.get("handler") or e.get("writer") or "?"
            if prev is not None and code < prev:
                print("OUT-OF-ORDER: %s %s: 0x%02X (%s) follows 0x%02X (%s)"
                      % (os.path.basename(path), group, code, label, prev, prev_label))
                bad += 1
            prev = code
            prev_label = label

if bad:
    print("")
    print("FAIL: %d ordering violation(s). Handlers/writers must be sorted by ascending opCode." % bad)
    sys.exit(1)
print("OK: %d template arrays are in ascending opcode order." % checked)
PY
