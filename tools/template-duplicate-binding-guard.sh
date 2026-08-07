#!/usr/bin/env bash
# template-duplicate-binding-guard.sh — enforces that no seed socket template
# binds the same (implementation name, numeric opCode) pair twice.
#
# A name legitimately bound to SEVERAL DISTINCT opcodes is normal and permanent
# (NoOpHandler is a deliberate sink at four opcodes in gms_95_1). What is a data
# defect is the SAME numeric opcode written twice with different leading-zero
# padding — "0xB8" and "0x0B8" — which makes the dispatch map's last-write-wins
# behaviour decide which entry's options survive.
#
# Mirrors the atlas-configurations socket.Validate ErrDuplicateBinding rule
# (task-194): the server rejects such a document with 400, so seed data that
# contained one could never be saved back through the UI.
#
# Run from the repo root; non-empty diagnostics → non-zero exit.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
TEMPLATE_DIR="$ROOT/services/atlas-configurations/seed-data/templates"

python3 - "$TEMPLATE_DIR" <<'PY'
import glob, json, os, sys, collections

tmpl_dir = sys.argv[1]
bad = 0
checked = 0
for path in sorted(glob.glob(os.path.join(tmpl_dir, "template_*.json"))):
    d = json.load(open(path))
    sock = d.get("socket", {})
    for group, namekey in (("handlers", "handler"), ("writers", "writer")):
        arr = sock.get(group) or []
        checked += 1
        seen = collections.defaultdict(list)
        for e in arr:
            if not isinstance(e, dict) or "opCode" not in e:
                continue
            try:
                code = int(e["opCode"], 16)
            except (TypeError, ValueError):
                print("BAD opCode in %s %s: %r" % (os.path.basename(path), group, e.get("opCode")))
                bad += 1
                continue
            seen[(e.get(namekey), code)].append(e["opCode"])
        for (name, code), raws in sorted(seen.items(), key=lambda kv: (str(kv[0][0]), kv[0][1])):
            if len(raws) > 1:
                print("DUPLICATE BINDING: %s %s: %s @ 0x%02X written %d times as %s"
                      % (os.path.basename(path), group, name, code, len(raws), raws))
                bad += 1

if bad:
    print("")
    print("FAIL: %d duplicate binding(s). One (name, numeric opCode) pair may appear at most once." % bad)
    sys.exit(1)
print("OK: %d template arrays carry no duplicate (name, opCode) binding." % checked)
PY
