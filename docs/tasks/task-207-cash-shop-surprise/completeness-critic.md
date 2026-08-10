# Packet Completeness Critic — task-207-cash-shop-surprise

**Verdict: CLEAN.** 0 CHANGED-BUT-UNCLAIMED findings, 0 CLAIMED-BUT-UNVERIFIED findings. One process finding (missing manifest, informational only).

## Process finding: no coverage-manifest.yaml

`docs/tasks/task-207-cash-shop-surprise/` contains `prd.md`, `design.md`, `plan.md`, `context.md`, `re-checklist.md` — no `coverage-manifest.yaml` per the schema in `docs/packets/PROCESS.md`. Per the packet-completeness-critic playbook, a missing manifest is normally the top-priority finding and would stop the audit. Per the dispatching instructions for this run, the claim set was substituted from `plan.md` (Global Constraints, Tasks 1–3/16/19), `context.md` §3/§4, and `.superpowers/sdd/plan/progress.md`. **Recommendation:** add `docs/tasks/task-207-cash-shop-surprise/coverage-manifest.yaml` before merge so future automated completeness checks (which require the file) don't hard-stop on this task.

## CHANGED-BUT-UNCLAIMED

None found.

| kind | file-or-packet | evidence | recommendation |
|---|---|---|---|
| — | — | — | — |

Detail of what was checked:

- **Codec diff** (`git diff 1e0a321b8...HEAD -- libs/atlas-packet`, non-test `.go` files): exactly two new files — `libs/atlas-packet/cash/clientbound/item_gachapon_result.go` and `libs/atlas-packet/cash/serverbound/item_gachapon_button.go`. Both map directly to the two declared ops (`cash/clientbound/CashItemGachaponResult`, `cash/serverbound/CashItemGachaponButton`). No other codec file under `libs/atlas-packet` was touched.
- **Consumed-but-should-be-unmodified check:** `git diff --name-only $BASE...HEAD -- libs/atlas-packet/cash/clientbound/shop_inventory.go` returned empty — `CashInventoryItem` (consumed by the new `CashItemGachaponSuccess.newItem` field) genuinely was not modified, confirming the new codec only reads the existing type rather than mutating its shared decode contract.
- **Version gates:** `git diff $BASE...HEAD -- libs/atlas-packet | grep -E '^[+-].*(MajorVersion|MajorAtLeast|IsRegion|Region\(\))'` returned one line — `ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)` inside `item_gachapon_button_test.go`'s fixture harness, not a version-gate branch in the codec itself. Neither new codec contains a `MajorAtLeast`/`MajorVersion>`/region gate; both files' own doc comments assert the wire shape is identical across all in-scope versions (mode-byte resolution goes through `atlas_packet.WithResolvedCode("operations", ...)`, config-driven, not a hardcoded gate) — consistent with DOM-25 and with there being no gate to declare.
- **Matrix delta** (`docs/packets/audits/status.json`, row-by-row cell-state diff): exactly three op rows changed —
  - `CASHSHOP_CASH_ITEM_GACHAPON_RESULT` (clientbound): v83/v84/v87/v92/v95/jms_v185 incomplete→verified, v48/v61/v72 incomplete→n-a. All in the declared claim set.
  - `CASH_ITEM_GACHAPON_BUTTON` (serverbound): added `packet` field `cash/serverbound/CashItemGachaponButton`, v79/v83/v84/v87/v92/v95/jms_v185 verified, v48/v61/v72 n-a. All in the declared claim set.
  - `CASHSHOP_CASH_GACHAPON_OPEN_RESULT` (clientbound, the alias row): only the v83 cell changed, incomplete→n-a — matches the documented "v83's alias cell was corrected from `0x14E ❌` to `n-a`" note. Confirmed the v84/v87/v92/v95 cells on this row are byte-identical to base (still `incomplete`/`no audit report`) — the declared "deliberately left at ❌" decision was not silently altered.
  - No other row in `status.json` changed state between `1e0a321b8` and `HEAD`.
- **`seed_fname.go` tie-break change** (`tools/packet-audit/cmd/seed_fname.go`): now resolves an ambiguous `(direction, opcode)` group by matching the template entry's already-bound implementation's `Operation()` constant, falling back to lexicographic-first only when no bound match exists. This is a generic mechanism, not scoped to jms 167, so it could in principle re-resolve any other ambiguous opcode group tree-wide. Checked for out-of-scope fallout two ways:
  - `git diff --name-only $BASE...HEAD -- services/atlas-configurations/seed-data/templates` lists all 11 templates (all touched, expected — regeneration runs over every template), but `git diff ... | grep '"fname"'` shows every changed `fname` line is either `CUICashItemGachapon::OnButtonClicked` or `CCashShop::OnCashItemGachaponResult` — no other fname in any template was re-resolved.
  - The `CASHSHOP_SURPRISE` row (the other op sharing jms opcode 167) is byte-identical between BASE and HEAD in `status.json` (verified by direct row comparison) — the dual-row ruling at jms 167 produced no matrix-state fallout for the sibling op.
- **v48/v61/v72 `CASHSHOP_CASH_ITEM_GACHAPON_RESULT` row removals:** each removed registry entry (opcode 257/256/292 respectively) was the sole occupant of its `(direction, opcode)` key in that file — the diffs show no sibling entry sharing the opcode, so removal cannot have silently re-attributed some other op's fname via the tie-break path. Each removal is backed by an IDA re-derivation comment citing a verified 3-line stub address (0x4536b9 / 0x46128a / 0x470d24) and grades the cell to `n-a` via registry absence, consistent with `internal/matrix/grade.go`'s StateNA rule cited in the removal comments.

## CLAIMED-BUT-UNVERIFIED

None found.

| op | version | actual state | recommendation |
|---|---|---|---|
| — | — | — | — |

All 13 declared "verified" cells read `verified` in HEAD `status.json`:
- `CASHSHOP_CASH_ITEM_GACHAPON_RESULT` (clientbound): gms_v83, v84, v87, v92, v95, jms_v185 — all `verified`.
- `CASH_ITEM_GACHAPON_BUTTON` (serverbound): gms_v79, v83, v84, v87, v92, v95, jms_v185 — all `verified`.

All 7 declared "n-a with proof" cells read `n-a` in HEAD `status.json`:
- v48/v61/v72 × both directions (6 cells) — all `n-a`, each backed by a registry-removal comment (see above) plus corresponding `feature-na-evidence.yaml` entries.
- v79 clientbound (`CASHSHOP_CASH_ITEM_GACHAPON_RESULT`) — `n-a` (this cell was already `n-a` pre-branch; the branch did not need to newly prove it, and did not disturb it).

The `CASHSHOP_CASH_GACHAPON_OPEN_RESULT` alias row's v84/v87/v92/v95 cells remain `incomplete` (`no audit report`) exactly as the "explicit coverage decision" describes — not silently promoted, not silently claimed as covered. Its v83 cell is `n-a`, matching the documented dispatcher-has-no-0x14E-case correction.

## Summary of checks performed

1. Confirmed `coverage-manifest.yaml` absent; substituted plan.md/context.md/progress.md as claim source per dispatch instructions.
2. Diffed `libs/atlas-packet` non-test `.go` files against the claim set — 2 files, both claimed.
3. Confirmed `shop_inventory.go` untouched despite being consumed by the new codec.
4. Grepped for version-gate idioms in the packet diff — one hit, in a test harness, not a codec gate.
5. Row-by-row diffed `docs/packets/audits/status.json` (Python key-indexed comparison, not raw JSON-diff, to avoid false positives from key reordering) — exactly 3 op rows changed, all within declared scope; every declared verified/n-a cell confirmed present in HEAD.
6. Reviewed the `seed_fname.go` tie-break rewrite and confirmed (a) no fname outside the two new ops changed in any regenerated template, and (b) the sibling op at the shared jms-167 opcode (`CASHSHOP_SURPRISE`) has a byte-identical matrix row before/after.
7. Confirmed each of the three v48/v61/v72 registry-row removals had no sibling entry at the same opcode, ruling out an unclaimed re-attribution side effect.
