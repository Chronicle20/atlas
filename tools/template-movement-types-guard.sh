#!/usr/bin/env bash
# template-movement-types-guard.sh — enforces that every tenant socket-config
# template gives every MOVE handler a well-formed movement `types` table.
#
# Rationale: libs/atlas-packet/model/movement.go decodes a movement fragment by
# reading a one-byte element type and looking it up AS AN ARRAY INDEX in the
# handler's options.types. The entry's `Type` selects the concrete element
# decoder. When `types` is missing the lookup returns ("NOT_FOUND", "DEFAULT"),
# no decoder branch matches DEFAULT, and every fragment falls through to the
# bare 3-byte Element decoder — desyncing the reader against a fragment that is
# 9-15 bytes wide. When a single `Type` value is TYPO'D the same thing happens
# for that one index, silently and with no log line.
#
# This defect has now shipped twice (task-179 fixed v48/61/72/79 but missed
# v48's SummonMoveHandle; the v92/v95 templates were seeded with no types at
# all), which is why this is a permanent guard and not a one-off check.
#
# The carrier set is exactly five handlers. NPCActionHandle IS a move handler
# despite its name — libs/atlas-packet/npc/serverbound/action.go's
# ActionRequest embeds a model.Movement and decodes it through the same
# options.types lookup as the other four (see action.go's Decode). Conversely
# CharacterInventoryMoveHandle is deliberately NOT a move handler despite its
# name — it is inventory item movement and correctly carries no types.
#
# See docs/packets/TEMPLATE_CONVENTIONS.md. Pure shell + python3, no Go setup.
# Run from the repo root; non-empty diagnostics → non-zero exit.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
TEMPLATE_DIR="$ROOT/services/atlas-configurations/seed-data/templates"

python3 - "$TEMPLATE_DIR" <<'PY'
import glob, json, os, sys

tmpl_dir = sys.argv[1]

MOVE_HANDLERS = {
    "CharacterMoveHandle",
    "DragonMoveHandle",
    "MonsterMovementHandle",
    "NPCActionHandle",
    "PetMovementHandle",
    "SummonMoveHandle",
}
VALID_TYPES = {
    "NORMAL", "JUMP", "TELEPORT", "START_FALL_DOWN",
    "FLYING_BLOCK", "STAT_CHANGE", "DEFAULT",
}

bad = 0
checked = 0
contributions = {}  # template name -> move handlers it contributed, for the summary breakdown

paths = sorted(glob.glob(os.path.join(tmpl_dir, "template_*.json")))

# Floor check: a wrong/empty template dir would otherwise make this guard pass
# VACUOUSLY (0 files -> 0 violations -> exit 0), which is the worst outcome for
# a guard. There are 11 templates today; the floor only has to catch "the glob
# found nothing/almost nothing", so it does not need bumping per new template.
if len(paths) < 5:
    print("FAIL: found only %d template_*.json under %s — expected the full set."
          " The template directory moved or the glob is wrong; refusing to pass"
          " vacuously." % (len(paths), tmpl_dir))
    sys.exit(1)

for path in paths:
    name = os.path.basename(path)
    try:
        d = json.load(open(path))
    except Exception as e:
        print("PARSE ERROR: %s: %s" % (name, e))
        bad += 1
        continue

    arrays = {}  # handler -> serialized types, for the intra-template equality check
    contrib = 0  # move handlers THIS template contributed, for the per-template floor below
    for e in d.get("socket", {}).get("handlers", []) or []:
        if not isinstance(e, dict):
            continue
        h = e.get("handler")
        if h not in MOVE_HANDLERS:
            continue
        checked += 1
        contrib += 1
        types = (e.get("options") or {}).get("types")

        # (1) present and non-empty
        if not isinstance(types, list) or len(types) == 0:
            print("MISSING TYPES: %s %s (opCode %s) has no non-empty options.types"
                  % (name, h, e.get("opCode")))
            bad += 1
            continue

        # (2) well-formed entries with a recognized Type
        fall_down = 0
        for i, entry in enumerate(types):
            if not isinstance(entry, dict):
                print("BAD ENTRY: %s %s index %d is %s, want an object"
                      % (name, h, i, type(entry).__name__))
                bad += 1
                continue
            n, ty = entry.get("Name"), entry.get("Type")
            if not isinstance(n, str) or not isinstance(ty, str):
                print("BAD ENTRY: %s %s index %d needs string Name and Type, got %r/%r"
                      % (name, h, i, n, ty))
                bad += 1
                continue
            if ty not in VALID_TYPES:
                print("BAD TYPE: %s %s index %d has Type %r, not one of %s"
                      % (name, h, i, ty, sorted(VALID_TYPES)))
                bad += 1
            if n == "FALL_DOWN":
                fall_down += 1

        # (3) at most one FALL_DOWN (the only Name the decoder branches on)
        if fall_down > 1:
            print("DUPLICATE FALL_DOWN: %s %s has %d entries named FALL_DOWN, want at most 1"
                  % (name, h, fall_down))
            bad += 1

        arrays[h] = json.dumps(types, sort_keys=True)

    # (4) all move-handler arrays within one template are identical
    if len(set(arrays.values())) > 1:
        print("DIVERGENT TYPES: %s move handlers disagree: %s"
              % (name, ", ".join("%s(%d entries)" % (h, len(json.loads(v)))
                                 for h, v in sorted(arrays.items()))))
        bad += 1

    # (5) per-template contribution floor. The directory-level floor above only
    # catches "the glob found nothing/almost nothing" — it is blind to a
    # PRESENT template whose contribution silently drops to 0 through a
    # different corruption path: a deleted "socket" key, a null
    # socket.handlers, or all five move handlers renamed all make the loop
    # above simply never match an entry, so no per-handler check above ever
    # fires and the file is never named in any diagnostic. That is the exact
    # failure mode this guard exists to prevent (movement config silently
    # absent), reached around the per-handler checks instead of through them.
    #
    # Every template contributes at least 4 move handlers today
    # (template_gms_12_1.json legitimately has no PetMovementHandle, so it
    # contributes 4; every other template contributes 4 today (gms_92_1,
    # gms_95_1, pre-fix) or 5). A floor of 3 leaves one handler of headroom
    # over today's minimum without being brittle, while still catching a
    # PARTIAL rename (1-2 contributed) that a bare zero-check would miss.
    contributions[name] = contrib
    if contrib == 0:
        print("NO MOVE HANDLERS: %s contributed 0 move handlers — socket.handlers is missing, null,"
              " or contains none of the five move handlers" % name)
        bad += 1
    elif contrib < 3:
        print("TOO FEW MOVE HANDLERS: %s contributed only %d move handler(s) (want >= 3) —"
              " socket.handlers may be missing most of the five move handlers" % (name, contrib))
        bad += 1

if bad:
    print("")
    print("FAIL: %d movement-types violation(s). Every move handler needs a well-formed,"
          " template-consistent options.types. See docs/packets/TEMPLATE_CONVENTIONS.md." % bad)
    sys.exit(1)
print("OK: %d move handlers across %d templates carry a valid movement types table."
      % (checked, len(paths)))
for name in sorted(contributions):
    print("  %s: %d move handler(s)" % (name, contributions[name]))
PY
