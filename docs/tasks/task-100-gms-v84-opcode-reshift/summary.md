# task-100 — gms_v84 opcode-table reshift (summary)

## Outcome
`docs/packets/registry/gms_v84.yaml` now has **zero duplicate `(opcode, direction)` pairs** and every reshifted opcode is read from the v84 IDB dispatch (not v83-offset-derived). `tools/packet-audit matrix --check` exits 0. Folded into PR #794 (CField packet family + complete gms_v84 opcode-table reshift).

## What changed
Starting from the task-096 base (560 v84 rows), the v84 registry was reconciled against the v84 IDB (`GMS_v84.1_U_DEVM.i64`) and the v95 PDB reference:

- **188 rows reshifted** — opcode corrected from the stale v83-seeded value to the IDA-confirmed v84 value. The shift is non-uniform per neighborhood (+2/+3/+4/+6/+7), so each was **read from the IDB dispatcher (clientbound) or `COutPacket(N)` send-site (serverbound)**, never offset-derived.
- **1 row added** — `POTION_DISCOUNT_RATE_CHANGED` (clientbound, 0x60/96).
- **9 rows deleted** — version-absent or unidentifiable phantoms:
  - `OPEN_GATE` (serverbound) — Mechanic-job op, post-v84; absent in v84.
  - `MESO_BAG_MESSAGE` (clientbound) — confirmed version-absent in v84.
  - `ALLIANCE_REQUEST` (serverbound) — folds into `ALLIANCE_OPERATION`.
  - `CLICK_GUIDE` (serverbound) — mis-fnamed and absent in v84.
  - `UNNAMED_R245 / R296 / R297 / R299 / R349` (serverbound) — csv-import placeholders with no fname, unknown identity, UNVERIFIED opcode. Each occupied a slot that turned out to belong to an IDA-verified op; with no fname there is no way to locate them in the IDB to verify any alternative opcode, so they were deleted rather than parked at a guessed "free" slot.

Result: 560 → **552 rows**.

## v84 IDB naming
~80 previously-unnamed v84 functions were renamed to their canonical demangled names (body-verified against the v95 PDB counterpart, not positional dispatcher-order mapping) and `idb_save`'d, so the v84 IDB is now a properly-named reference like the v95 PDB and future exports resolve by name.

## Merge with main (#746 player-summons) — conflict resolution
Main advanced past the branch point with `a2207e7c7 feat: player summons (#746)`, which touched the same packet artifacts. `origin/main` was merged into the branch; resolution:

- **IDA exports** (gms_v84/v87/v95/jms_185) — both sides are pure additions to the `functions` map with **zero base-key modifications and zero added-key collisions**. Unioned mechanically (ours + main's 9 summon fnames each).
- **Seed templates** (gms_84/87/95/jms_185) — additive writer/handler unions. My task-096 writers (MTS/ContiMove/Pyramid/Snowball/Coconut/etc.) + main's `Summon*` writers (0xB3–0xB8 in v84) and 3 summon handlers. Disjoint names and opcodes; no new duplicate opcodes introduced (the pre-existing 0x00/0x0a shared-opcode login/serverlist writers are unchanged).
- **registry/gms_v84.yaml** — the 6 conflict hunks were all summon rows. **Took main's (#746) version** for all of them:
  - *Clientbound* summon rows (179–184): both sides independently reached the **same opcodes** — a cross-validation. Main's rows carry the richer summon-specific metadata (actual handler fnames as primary, dispatcher as `fname_alts`, skill/damage-swap notes), so theirs wins with no opcode change.
  - *Serverbound* summon rows: the two sides **disagreed** — mine used the client `COutPacket(N)` send-table literal (MOVE=180/ATTACK=181/DAMAGE=182); main used the **server recv-handler opcode** (178/179/180). The live v84 template routes `SummonMoveHandle`→0xB2, `SummonAttackHandle`→0xB3, `SummonDamageHandle`→0xB4, confirming **main is correct** (the registry serverbound opcode must match the server's recv-dispatch table, which differs from the client send table). My reshift had conflated the two tables here; main's deployed values are authoritative.
  - Taking main's serverbound `SUMMON_ATTACK`=179 collided with `UNNAMED_R296` (the fname-less phantom my reshift had parked at 179 assuming it was free). R296 was deleted per the phantom-deletion rationale above.
- **STATUS.md / status.json** — generated; regenerated via `packet-audit matrix` after all other files were resolved.

## Verification
- `python3` dup scan on `gms_v84.yaml` → `[]` (552 rows).
- `go run ./tools/packet-audit matrix --check` → exit 0.
- `tools/packet-audit` `go test ./...` → all pass; `go build` clean for packet-audit, atlas-channel, atlas-configurations.
