# Review — Task 7: Route Maple Life handlers/writers in templates

**Range reviewed:** `ddef3d665..5f427eb3d` (`7ab439050` route commit + `5f427eb3d` reorder-fix commit)
**Brief:** `.superpowers/sdd/plan/task-7-brief.md` (Controller addendum authoritative)
**Report:** `.superpowers/sdd/plan/task-7-report.md`

## Scope confirmed

Diff touches exactly the four in-scope templates
(`template_gms_{83,87,92,95}_1.json`), 156 insertions across the two commits'
squashed effect (net), matching `git diff --stat ddef3d665..5f427eb3d`. No
other file changed. This matches the brief's addended scope exactly.

## Findings

### 1. Out-of-scope templates untouched — PASS

```
$ git diff --stat ddef3d665..5f427eb3d -- .../template_gms_84_1.json \
    .../template_gms_48_1.json .../template_gms_61_1.json \
    .../template_gms_72_1.json .../template_gms_79_1.json \
    .../template_jms_185_1.json
(empty)
```

`template_gms_84_1.json` and the five other do-not-open templates show zero
diff across the full range. Correct per Controller addendum point 1 (v84
struck from scope).

### 2. Opcodes match the committed registry files — PASS

Cross-checked every `opCode` written against `docs/packets/registry/gms_v{83,87,92,95}.yaml`:

| version | op | registry decimal | registry→hex | template opCode |
|---|---|---|---|---|
| v83 | MAPLELIFE_RESULT | 349 | 0x15D | `0x15D` ✓ |
| v83 | MAPLELIFE_ERROR | 350 | 0x15E | `0x15E` ✓ |
| v83 | MAPLELIFE_CHECK_NAME | 256 | 0x100 | `0x100` ✓ |
| v87 | MAPLELIFE_RESULT | 370 | 0x172 | `0x172` ✓ |
| v87 | MAPLELIFE_ERROR | 371 | 0x173 | `0x173` ✓ |
| v87 | MAPLELIFE_CHECK_NAME | 270 | 0x10E | `0x10E` ✓ |
| v92 | MAPLELIFE_RESULT | 404 | 0x194 | `0x194` ✓ |
| v92 | MAPLELIFE_ERROR | 405 | 0x195 | `0x195` ✓ |
| v92 | MAPLELIFE_CHECK_NAME | 301 | 0x12D | `0x12D` ✓ |
| v95 | MAPLELIFE_RESULT | 413 | 0x19D | `0x19D` ✓ |
| v95 | MAPLELIFE_ERROR | 414 | 0x19E | `0x19E` ✓ |
| v95 | MAPLELIFE_CHECK_NAME | 311 | 0x137 | `0x137` ✓ |

All twelve opcodes (registry file, line-verified via `docs/packets/registry/gms_v{83,87,92,95}.yaml:1805-1811/3470-3472` etc.) and their hex conversions verified independently by hand (e.g. 349 = 1·256+5·16+13 = 0x15D) — every one correct, not merely accepted from the report.

### 3. `operations` arm-key completeness — PASS (full enumeration, not spot-check)

Codec constants enumerated directly from source:

- `libs/atlas-packet/maplelife/clientbound/result.go:42-52` — `AVAILABLE`, `TAKEN`, `UNKNOWN_ERROR`
- `libs/atlas-packet/maplelife/clientbound/error.go:51-60` — `SUCCESS`, `NAME_TAKEN_AT_SUBMIT`, `UNKNOWN_ERROR`

Checked programmatically against every one of the four in-scope templates at
`5f427eb3d` (loaded and parsed each template's `MapleLifeResult`/`MapleLifeError`
writer entries):

```
v83 MapleLifeResult ops={'AVAILABLE': 0, 'TAKEN': 1, 'UNKNOWN_ERROR': 255}
v83 MapleLifeError  ops={'SUCCESS': 52, 'NAME_TAKEN_AT_SUBMIT': 54, 'UNKNOWN_ERROR': 255}
v87 MapleLifeResult ops={'AVAILABLE': 0, 'TAKEN': 1, 'UNKNOWN_ERROR': 255}
v87 MapleLifeError  ops={'SUCCESS': 54, 'NAME_TAKEN_AT_SUBMIT': 56, 'UNKNOWN_ERROR': 255}
v92 MapleLifeResult ops={'AVAILABLE': 0, 'TAKEN': 1, 'UNKNOWN_ERROR': 255}
v92 MapleLifeError  ops={'SUCCESS': 55, 'NAME_TAKEN_AT_SUBMIT': 57, 'UNKNOWN_ERROR': 255}
v95 MapleLifeResult ops={'AVAILABLE': 0, 'TAKEN': 1, 'UNKNOWN_ERROR': 255}
v95 MapleLifeError  ops={'SUCCESS': 56, 'NAME_TAKEN_AT_SUBMIT': 58, 'UNKNOWN_ERROR': 255}
```

All three arm keys present on every version for both writers — no missing key,
so no template resolution can hit `ResolveCode`'s 99-sentinel trap for this
writer pair. `SUCCESS`/`NAME_TAKEN_AT_SUBMIT` literals are correctly
per-version distinct (52/54, 54/56, 55/57, 56/58) and `UNKNOWN_ERROR`=255
never collides with a real per-version literal (max real literal across all
four versions is 58, confirmed above) — this satisfies the controller's
already-resolved non-collision check independently.

### 4. No `USE_MAPLELIFE` / `MapleLifeUseHandle` / `CashItemUseMapleLife` reference — PASS

```
$ git diff ddef3d665..5f427eb3d | grep -i -E "USE_MAPLELIFE|MapleLifeUseHandle|CashItemUseMapleLife"
(no output, exit 1)
$ grep -rl -E "MapleLifeUseHandle|USE_MAPLELIFE|CashItemUseMapleLife" services/atlas-configurations/seed-data/templates/
(no output)
```

Neither the diff nor the full templates directory contains any of the three
forbidden strings. Correct per Controller addendum points 2 and 4.

### 5. Ordering-fix commit is pure reordering — PASS (independently verified, not accepted from report)

Loaded `handlers`/`writers` arrays for all four templates at both `7ab439050`
and `5f427eb3d`, and compared as multisets (sorted by canonical JSON):

```
v83 array keys match: True
  order changed (content same) in /socket/handlers len 149
  order changed (content same) in /socket/writers   len 232
v87 array keys match: True
  order changed (content same) in /socket/handlers len 143
  order changed (content same) in /socket/writers   len 230
v95 array keys match: True
  unchanged /socket/handlers
  order changed (content same) in /socket/writers   len 238
v92 array keys match: True
  unchanged /socket/handlers
  unchanged /socket/writers
```

Every array's content-set (ignoring order) is byte-identical between the two
commits — confirms `5f427eb3d` changed only element positions, no field
value, for v83/v87/v95. `template_gms_92_1.json` is genuinely unchanged by
the fix commit (`git diff --stat` shows only three files touched:
`template_gms_{83,87,95}_1.json`, 102 insertions / 102 deletions), confirming
the report's claim that v92 was already correctly sorted before the fix.

Final-state ordering check (all four templates, `5f427eb3d`):

```
v83 handlers: sorted=True len=149   v83 writers: sorted=True len=232
v87 handlers: sorted=True len=143   v87 writers: sorted=True len=230
v92 handlers: sorted=True len=87    v92 writers: sorted=True len=147
v95 handlers: sorted=True len=150   v95 writers: sorted=True len=238
```

All eight arrays are in strict ascending `opCode` order at HEAD of this range.

## Not evaluable

None — all five checklist items were fully evaluable within the diff surface
plus the two codec files and four registry files the diff's correctness
genuinely depends on.

## Summary

This is a clean, mechanical routing change that matches the brief (as
overridden by the controller addendum) exactly: correct opcodes independently
verified against the registry, complete arm-key coverage on both writers
across all four in-scope templates (the highest-value check per the task
instructions), correct exclusion of v84/other out-of-scope templates and of
the deliberately-absent `USE_MAPLELIFE`/`CashItemUseMapleLife` paths, and a
reorder-only fix commit verified by direct content-set comparison rather than
accepted on the implementer's word. No blocking or non-blocking findings.
