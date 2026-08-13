# Completeness Critic — task-188-v48-template-completeness

**Verdict: 4 findings (1 blocking-class: missing manifest; 1 gate-scope finding with sub-parts; 1 matrix-regression cluster; 1 confirmation of clean v48 promotions).**

Branch: `task-188-v48-template-completeness`, HEAD `fe9963017` (merge commit).
Diff base: `BASE=$(git merge-base origin/main HEAD)` = `2b57edccbadd23e129e72a6203baf2fd2793d784`.

## Finding 0 — MISSING coverage-manifest.yaml (highest priority)

`docs/tasks/task-188-v48-template-completeness/` contains no `coverage-manifest.yaml`. Its only docs are `v48-clientbound-map.md`, `v48-writers-resolved.md`, `validate-gms_v48*.md`. Per `docs/packets/PROCESS.md`'s coverage-manifest schema, every packet task must declare `ops` / `versions` / `fields` / `out_of_scope`. This one does not.

This is not a narrow miss: `git log $BASE..HEAD` shows **24 commits** touching codecs, the gms_v48 registry, the export, evidence records, and templates — well beyond "wire up the v48 template." Without a manifest, none of the normal claimed-vs-changed cross-check is mechanically possible; everything below had to be reconstructed from raw `git diff` and `status.json` deltas instead of validated against a declaration.

**Recommendation:** add `docs/tasks/task-188-v48-template-completeness/coverage-manifest.yaml` before merge, enumerating every op touched below, and re-run this critic against it.

## Finding 1 — CHANGED-BUT-UNCLAIMED (gate): non-v48 version gates moved

The task is scoped to v48, but three version gates changed thresholds that affect **other** versions' wire behavior. None of these regressed an already-`verified` cell (checked against the full matrix delta below — no `gms_v61`/`v72`/`v79`/`v83`+ cell changed state), but all three are undeclared scope decisions on versions the task doesn't own.

| kind | file | evidence | recommendation |
|---|---|---|---|
| gate | `libs/atlas-packet/reactor/clientbound/spawn.go` | New `hasReactorName(t)`: `return (t.IsRegion("GMS") && t.MajorAtLeast(83)) \|\| t.Region() == "JMS"`. Previously `w.WriteAsciiString(m.name)` was unconditional for every version. This is the v79/v83 boundary (the exact off-by-one class in memory `bug_v84_opcode_table_shifted_vs_v83` / `bug_majorversion_gt83_is_off_by_one_v87`), **not** the v48/v61 boundary every other change in this branch uses. It changes wire output for `gms_v61`, `gms_v72`, `gms_v79` (all currently `incomplete` in the matrix, unaffected in state but their encode/decode shape silently changed) while leaving `gms_v83`/`v84`/`v87`/`v95`/`jms_v185` (all `verified`, `MajorAtLeast(83)` true) untouched. | Add `reactor/clientbound` (or at least the v79/v83 boundary) to the manifest's declared scope, or split this gate fix into its own task/PR with its own evidence. |
| gate | `libs/atlas-packet/field/clientbound/effect_weather.go` | `t.MajorVersion() < 61` → `t.MajorVersion() == 61`. Before: every version `<61` (i.e. only v48 in the matrix) took `encodeGMSLegacy`; v61 itself took `encodeGMS`. After: only `==61` takes legacy — v61 now routes to `encodeGMSLegacy` (a real behavior change for v61) and v48 now routes to `encodeGMS` (the intended v48 fix). The comment in the diff even says "UNVERIFIED: v72 and v79 have not been checked and currently take encodeGMS" — an open question about a version outside this task's declared scope, admitted in-line but not tracked anywhere. `gms_v61`'s `BLOW_WEATHER` cell stayed `incomplete` before and after, so no verified claim was falsified, but the wire shape for v61 changed without any v61-scoped fixture. | Declare v61 in scope for this op, or gate strictly to v48 only (e.g. `t.Region()=="GMS" && t.MajorVersion()==48`) so the v61 branch is untouched until it's actually verified. |
| gate | `libs/atlas-packet/pet/serverbound/spawn.go` | New `hasPetSpawnLead(t)`: `(t.IsRegion("GMS") && t.MajorAtLeast(61)) \|\| t.Region() == "JMS"`. The in-code comment states: *"v61 is INFERRED, not verified: the v61 IDB has no named equivalent... gms_v61.yaml registers no serverbound SPAWN_PET at all, so no matrix cell depends on this choice."* Self-aware and low-risk (matrix confirms `gms_v61` SPAWN_PET cell is `n-a` before and after), but it's still a v61 boundary decision made inside a v48-scoped task with no v61 evidence. | Note the inferred v61 boundary explicitly in the (missing) manifest's notes/out_of_scope, or leave a `docs/packets/...` cross-reference so a future v61 bring-up doesn't silently trust an unverified guess. |

Gates confirmed **in-scope** (only affect the v48-vs-≥61 boundary where v48 is the only matrix version below 61, so no other version's behavior changes): `field/clientbound/set_field.go` (`>28`→`MajorAtLeast(61)`, gated separately from the untouched seed-count check), `login/clientbound/server_list_entry.go` (`>12`→`MajorAtLeast(61)`), `model/monster.go` (`>12`→`MajorAtLeast(61)`, plus new `mobLegacyV12Tail` for GMS≤12 which replaces a `// TODO proper temp stat encoding` — a TODO removal, good), `npc/clientbound/spawn.go` + `spawn_request_controller.go` (`hasEnabledFlag`, GMS<61 excluded), `cash/serverbound/shop_operation_increase_storage.go` (`!IsRegion(GMS) || MajorAtLeast(61)`, v48-only exclusion).

## Finding 2 — matrix regression cluster: 6 gms_v48 cash-shop cells demoted verified→partial, 1 verified→n-a

Full `gms_v48` column diff (`docs/packets/audits/status.json`, `BASE` vs `HEAD`) shows every non-`n-a`-origin state change. Alongside ~20 legitimate `n-a → verified`/`incomplete`/`partial` promotions (new opcode wiring — expected for a template bring-up), the following **regressed**:

| op / packet | gms_v48: before → after | evidence |
|---|---|---|
| `USE_CASH_ITEM` / `cash/serverbound/CashItemUseMegaphone` | `verified` → `partial` | `libs/atlas-packet/cash/serverbound/v48_test.go` rewritten (commit `79a0d16e6`) — old fixtures for `ShopOperationBuyCouple`, `ShopOperationBuyPackage`, `ShopOperationBuy`, `ShopOperationGift` etc. deleted outright, not re-derived. |
| `cash/serverbound/CashShopOperationBuyCouple` (op unresolved to a stable name in this row) | `verified` → `partial` | same commit; old `TestShopOperationBuyCoupleBytesV48` (`packet-audit:verify ... ida=0x44b4c1`) removed with no replacement test for that op. |
| `cash/serverbound/CashShopOperationBuyFriendship` | `verified` → `partial` | same commit; corresponding v48 fixture removed. |
| `cash/serverbound/CashShopOperationBuyPackage` | `verified` → `partial` | same commit; old `TestShopOperationBuyPackageBytesV48` (`ida=0x44b837`) removed. |
| `cash/serverbound/CashShopOperationBuy` | `verified` → `partial` | same commit; old `TestShopOperationBuyBytesV48` (`ida=0x44b0cf`) removed. |
| `cash/serverbound/CashShopOperationGift` | `verified` → `partial` | same commit; old `TestShopOperationGiftBytesV48` (`ida=0x44ba5d`) removed. |
| `FIELD_OBSTACLE_ONOFF_LIST` / `field/clientbound/FieldFieldObstacleOnOffList` | `verified` → `n-a` | not traced to a specific commit in the sampled log; flagged for the author to confirm intent (legitimately inapplicable to v48, or a lost fixture). |

`git diff $BASE...HEAD -- libs/atlas-packet/cash/serverbound/v48_test.go` shows the file was substantially rewritten, not incrementally extended: several `packet-audit:verify` fixtures for ops that were apparently unverified guesses got deleted rather than corrected, which is defensible (removing false-verified evidence is correct behavior), but it means this branch did more than "fill in v48 template gaps" — it re-litigated existing v48 cash-shop verification without declaring that as in-scope anywhere.

**Recommendation:** confirm each demotion is intentional (stale/incorrect prior fixture, now correctly downgraded pending real IDA re-derivation) rather than an accidental loss, and record the intent in the manifest's `notes`/`out_of_scope` so a future pass knows these were deliberately un-verified rather than simply missed.

## Finding 3 — CLAIMED-BUT-UNVERIFIED: not applicable (no manifest to claim against)

Since there is no `coverage-manifest.yaml`, there is no `ops × versions` claim set to check cells against per Step 3 of the standard procedure. This section is empty by construction — see Finding 0.

## Positive confirmation (for completeness, not a finding)

Cross-checked every `gms_v48` cell that transitioned to `verified` in this branch (`SPAWN_NPC`, `SPAWN_NPC_REQUEST_CONTROLLER`, `SPAWN_MONSTER`, `SPAWN_PET`, `MOB_AFFECTED`, `REACTOR_HIT`, `REACTOR_DESTROY`, `SPAWN_DOOR`, `REMOVE_DOOR`, `SPAWN_KITE`, `REMOVE_KITE`, `CANNOT_SPAWN_KITE`, `MOB_DROP_PICKUP_REQUEST`, `USE_UPGRADE_SCROLL`, plus several `cash/serverbound` ops) against `docs/packets/audits/gms_v48/*.json`/`*.md` evidence and `packet-audit:verify` markers in the corresponding `*_v48_test.go` files (e.g. `libs/atlas-packet/npc/clientbound/spawn_v48_test.go:39` — `packet-audit:verify packet=npc/clientbound/NpcSpawn version=gms_v48 ida=0x56d527`). Every sampled promotion has a pinned evidence record and a marker; no orphaned promotion found. Also confirmed via the full `status.json` cell diff that **no non-`gms_v48` cell changed state** anywhere in the matrix — the gate-scope changes in Finding 1 altered wire behavior for other versions without falsifying any of their existing verified/partial/incomplete classifications.

## Method notes

- `git diff --name-only $BASE...HEAD -- 'libs/atlas-packet' | grep '\.go$' | grep -v '_test\.go$'` → 9 non-test codec files changed (28 total including tests): `cash/serverbound/shop_operation_increase_storage.go`, `field/clientbound/effect_weather.go`, `field/clientbound/set_field.go`, `login/clientbound/server_list_entry.go`, `model/monster.go`, `npc/clientbound/spawn.go`, `npc/clientbound/spawn_request_controller.go`, `pet/serverbound/spawn.go`, `reactor/clientbound/spawn.go`.
- Version-gate hunks extracted via `git diff $BASE...HEAD -- 'libs/atlas-packet' | grep -E '^[+-].*(MajorVersion|MajorAtLeast|IsRegion|Region\(\))'` and attributed per-file above.
- Matrix delta computed by parsing `status.json` `rows[].cells[<version>].state` at `BASE` vs `HEAD` (via `git show $BASE:docs/packets/audits/status.json`) keyed on `(op, packet, direction)`, diffed per version column — not via raw `git diff` text, which is dominated by row-reordering noise in this file.
