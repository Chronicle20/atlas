# task-130 Vega's Spell — Rollout Notes

## Wired versions (seed templates updated — new tenants only)

The `VegaScroll` writer (opcode + per-version `operations` mode-byte table) and,
where the handler was previously absent, the `CharacterCashItemUseHandle`
handler entry (`LoggedInValidator`) were added to the seed templates below.
These values are IDA-verified (Task 4 campaign) and are the post-merge source
of truth — copy them verbatim when patching a live tenant.

| version | template | writer opCode | handler opCode | handler added by this task? |
|---|---|---|---|---|
| gms_83 | `template_gms_83_1.json` | `0x166` | `0x4F` | no — already present |
| gms_84 | `template_gms_84_1.json` | `0x170` | `0x4F` | no — already present |
| gms_87 | `template_gms_87_1.json` | `0x17B` | `0x52` | yes |
| gms_95 | `template_gms_95_1.json` | `0x1AD` | `0x55` | yes |
| jms_185 | `template_jms_185_1.json` | `0x183` | `0x47` | yes |

Per-version `operations` values (decimal; hex in parens), copied from the seed
templates:

| version | START_SUCCESS | START_FAILURE | RESULT_SUCCESS | RESULT_FAILURE | INVALID |
|---|---|---|---|---|---|
| gms_83 | 64 (0x40) | 69 (0x45) | 65 (0x41) | 67 (0x43) | 66 (0x42) |
| gms_84 | 64 (0x40) | 69 (0x45) | 65 (0x41) | 67 (0x43) | 66 (0x42) |
| gms_87 | 66 (0x42) | 71 (0x47) | 67 (0x43) | 69 (0x45) | 68 (0x44) |
| gms_95 | 68 (0x44) | 73 (0x49) | 69 (0x45) | 71 (0x47) | 66 (0x42) |
| jms_185 | 59 (0x3B) | 64 (0x40) | 60 (0x3C) | 62 (0x3E) | 61 (0x3D) |

Note the v83 correction from the original design hypothesis: every version
selects the success/fail popup by the START byte value itself, so
START_FAILURE is a distinct opcode from START_SUCCESS on every version,
including v83 (69, not 64 as originally hypothesized in design §2.3).

## Live tenants (seed templates only affect NEW tenants)

For every live tenant on a wired version (gms_83 / gms_84 / gms_87 / gms_95 /
jms_185), PATCH the tenant socket config:

1. Add the `VegaScroll` writer entry (opcode + operations per version — copy
   from the matching seed template above, which is the source of truth
   post-merge).
2. Ensure the `CharacterCashItemUseHandle` handler entry exists with
   `LoggedInValidator` (required for gms_87/95/jms tenants created before
   task-126/task-130 landed; gms_83 tenants already have it).
3. Restart the tenant's atlas-channel pods — handlers/writers do NOT
   hot-reload from config changes (known gotcha).

Symptom of a missed patch: using a Vega's Spell logs the handler fall-through
warn (missing handler) or "Property [operations] missing ... defaulting to 99"
(missing writer options); the item no-ops or the dialog shows "This item
cannot be used."

## Parked versions

- **gms_v84** — WIRED (Task 4b, 2026-07-03). Previously blocked in Task 4 for
  lack of a live IDB; now IDA-verified against the live `GMS_v84.1_U_DEVM` IDB.
  `CField::OnPacket` routes clientbound 367-370 → `CField::OnVega` →
  `CUIVega::OnPacket` (`sub_858D68`) case 368 → `CUIVega::OnVegaResult`
  (`sub_858D7E`, 0x858d7e). Writer opcode is **0x170/368** — the prior 0x166/358
  csv-import carryover was confirmed WRONG (below the v84 dispatch range). Mode
  bytes are byte-identical to v83 (verified via Draw `sub_857734`:
  `this[34]==0x40`→success popup, `0x45`→fail). The registry row is promoted to
  `ida-discovered`, the `VegaScroll` writer entry is in `template_gms_84_1.json`
  (handler `CharacterCashItemUseHandle` at 0x4F was already present), and the
  `cash/clientbound/CashVegaScroll × gms_v84` matrix cell is ✅.
- **gms_v92** — PARKED, no artifacts at all: no IDB, no packet registry row,
  and no `USE_CASH_ITEM` handler exists for this version — the entire
  cash-item-use path is inert on gms_v92 (design §2.6). No template entry was
  added; `template_gms_92_1.json` is untouched. CSV hint for a future IDB:
  `VEGA_SCROLL 0x1A0` (UNVERIFIED).
