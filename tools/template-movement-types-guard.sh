#!/usr/bin/env bash
# template-movement-types-guard.sh — enforces that every tenant socket-config
# template gives every MOVE handler AND every MOVE writer a well-formed movement
# `types` table.
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
# The handler carrier set is exactly the six listed below. NPCActionHandle IS a
# move handler despite its name — libs/atlas-packet/npc/serverbound/action.go's
# ActionRequest embeds a model.Movement and decodes it through the same
# options.types lookup as the other five (see action.go's Decode). Conversely
# CharacterInventoryMoveHandle is deliberately NOT a move handler despite its
# name — it is inventory item movement and correctly carries no types.
#
# WRITERS need the table for the same reason, in the other direction. An attr
# code only resolves to a fragment SHAPE through options.types, so a writer
# without it encodes every fragment as the bare Element too. Two live defects
# came from exactly that: the summon/dragon move-path re-serialization added in
# bd3a09003 bails out (and reships the GMS v87 XOffset/YOffset pair the v87
# client never reads) when ReserializeMovePath is handed no table, and
# NormalElement.Encode's FALL_DOWN name check never fires, silently dropping
# fhFallStart from outbound pet movement. Both are invisible server-side: the
# packet still goes out, just wrong on the wire. The writer carrier set is
# DERIVED from libs/atlas-packet below rather than listed here, so a new
# movement-carrying writer is covered the day it is added.
#
# See docs/packets/TEMPLATE_CONVENTIONS.md. Pure shell + python3, no Go setup.
# Run from the repo root; non-empty diagnostics → non-zero exit.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
TEMPLATE_DIR="$ROOT/services/atlas-configurations/seed-data/templates"

python3 - "$TEMPLATE_DIR" "$ROOT/libs/atlas-packet" <<'PY'
import glob, json, os, re, sys

tmpl_dir = sys.argv[1]
packet_dir = sys.argv[2]

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

# A clientbound codec carries a move path if it holds a model.Movement field or
# hands a raw blob to model.ReserializeMovePath (summon/dragon do the latter).
MOVEMENT_FIELD_RE = re.compile(r"^\s*\w+\s+model\.Movement\s*$", re.M)
RESERIALIZE_RE = re.compile(r"\bmodel\.ReserializeMovePath\b")
WRITER_CONST_RE = re.compile(r'^const\s+\w*Writer\s*=\s*"([^"]+)"', re.M)


def derive_move_writers(pkt_dir):
    """Writer names, read out of libs/atlas-packet, whose codec encodes a move path.

    Returns {writer name: repo-relative file} or None if the derivation itself
    looks wrong — a derivation that quietly returns nothing would make the whole
    writer half of this guard pass vacuously.
    """
    writers = {}
    ok = True
    for dirpath, _, filenames in os.walk(pkt_dir):
        if os.path.basename(dirpath) != "clientbound":
            continue
        for fn in sorted(filenames):
            if not fn.endswith(".go") or fn.endswith("_test.go"):
                continue
            path = os.path.join(dirpath, fn)
            src = open(path, encoding="utf-8").read()
            if not (MOVEMENT_FIELD_RE.search(src) or RESERIALIZE_RE.search(src)):
                continue
            rel = os.path.relpath(path, os.path.dirname(pkt_dir))
            names = WRITER_CONST_RE.findall(src)
            if len(names) != 1:
                print("DERIVATION ERROR: %s encodes a move path but declares %d"
                      ' `const ...Writer = "..."` (want exactly 1): %s'
                      % (rel, len(names), names))
                ok = False
                continue
            writers[names[0]] = rel
    if not ok:
        return None
    # Six carriers today (character, monster, npc, pet, summon, dragon). The
    # floor only has to catch "the walk found nothing/almost nothing", e.g. the
    # library moved; it does not need bumping when a carrier is added.
    if len(writers) < 4:
        print("DERIVATION ERROR: only %d movement-carrying writer(s) derived from %s"
              " — expected the full carrier set. The library moved or the codec"
              " pattern changed; refusing to check writers vacuously."
              % (len(writers), pkt_dir))
        return None
    return writers


MOVE_WRITERS = derive_move_writers(packet_dir)
if MOVE_WRITERS is None:
    sys.exit(1)

bad = 0
checked = 0  # move handler entries checked
checked_w = 0  # move writer entries checked
contributions = {}  # template name -> (move handlers, move writers) it contributed

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

    arrays = {}  # label -> serialized types, for the intra-template equality check
    contrib = 0  # move handlers THIS template contributed, for the per-template floor below
    contrib_w = 0  # move writers THIS template contributed

    entries = []
    for e in d.get("socket", {}).get("handlers", []) or []:
        if isinstance(e, dict) and e.get("handler") in MOVE_HANDLERS:
            entries.append((e.get("handler"), e))
    for e in d.get("socket", {}).get("writers", []) or []:
        if isinstance(e, dict) and e.get("writer") in MOVE_WRITERS:
            entries.append(("%s (writer)" % e.get("writer"), e))

    for h, e in entries:
        if h.endswith(" (writer)"):
            checked_w += 1
            contrib_w += 1
        else:
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

    # (4) all move-handler AND move-writer arrays within one template are
    # identical. Index position IS the attr code and CMovePath is one shared
    # client function, so the outbound table is by construction the same table
    # that parsed the blob inbound: a writer whose table drifts from its
    # handler's is a bug in either direction.
    if len(set(arrays.values())) > 1:
        print("DIVERGENT TYPES: %s move handlers/writers disagree: %s"
              % (name, ", ".join("%s(%d entries)" % (h, len(json.loads(v)))
                                 for h, v in sorted(arrays.items()))))
        bad += 1

    # (5) per-template contribution floor. The directory-level floor above only
    # catches "the glob found nothing/almost nothing" — it is blind to a
    # PRESENT template whose contribution silently drops to 0 through a
    # different corruption path: a deleted "socket" key, a null
    # socket.handlers/socket.writers, or all the move entries renamed all make
    # the loop above simply never match an entry, so no per-entry check above
    # ever fires and the file is never named in any diagnostic. That is the
    # exact failure mode this guard exists to prevent (movement config silently
    # absent), reached around the per-entry checks instead of through them.
    #
    # Every template contributes at least 4 move handlers today (gms_12_1 has
    # neither PetMovementHandle nor DragonMoveHandle, so it contributes 4;
    # gms_48/61/72/79 have no DragonMoveHandle and contribute 5; the rest
    # contribute 6) and at least 4 move writers (gms_12_1 has no
    # DragonMove/PetMovement writer; gms_48/61/72/79 have no DragonMove and
    # contribute 5; the rest contribute 6). A floor of 3 leaves headroom over today's
    # minimum without being brittle, while still catching a PARTIAL rename
    # (1-2 contributed) that a bare zero-check would miss.
    contributions[name] = (contrib, contrib_w)
    for kind, key, n in (("HANDLERS", "socket.handlers", contrib),
                         ("WRITERS", "socket.writers", contrib_w)):
        if n == 0:
            print("NO MOVE %s: %s contributed 0 move %s — %s is missing, null, or"
                  " contains none of the movement carriers"
                  % (kind, name, kind.lower(), key))
            bad += 1
        elif n < 3:
            print("TOO FEW MOVE %s: %s contributed only %d move %s (want >= 3) — %s may be"
                  " missing most of the movement carriers"
                  % (kind, name, n, kind.lower(), key))
            bad += 1

if bad:
    print("")
    print("FAIL: %d movement-types violation(s). Every move handler and every move writer"
          " needs a well-formed, template-consistent options.types."
          " See docs/packets/TEMPLATE_CONVENTIONS.md." % bad)
    sys.exit(1)
print("OK: %d move handlers and %d move writers across %d templates carry a valid movement"
      " types table." % (checked, checked_w, len(paths)))
print("  writer carrier set derived from libs/atlas-packet: %s"
      % ", ".join(sorted(MOVE_WRITERS)))
for name in sorted(contributions):
    print("  %s: %d move handler(s), %d move writer(s)"
          % (name, contributions[name][0], contributions[name][1]))
PY
