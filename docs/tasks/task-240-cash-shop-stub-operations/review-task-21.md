# Review: Task 21 — derive the equip-slot extension facts

**Range:** `27d1c91e8..9d3026337` (single commit `9d3026337`)
**Deliverable:** `docs/tasks/task-240-cash-shop-stub-operations/derivation-equip-slot.md` (486 lines, sole file)
**Reviewer constraint honored:** no edits, no `go mod`/mutation experiments; verification limited to reading, `grep`, and repo file resolution. No ida-pro-mcp access — IDB claims were checked only for internal consistency and citation specificity, not re-decompiled.

## Method

`git diff --stat 27d1c91e8..9d3026337` confirms exactly one file, 486 insertions,
0 deletions — matches the brief and report. Every repo-side (non-IDB)
`file:line` citation in the document was independently re-resolved against
the current worktree (not trusted from the brief or the report, per the
brief's own "plan-time line numbers are stale" rule and the review's
instruction to re-resolve independently).

## 1. Repo file:line citations — all resolve and say what the document claims

| Citation in doc | Verified against | Result |
|---|---|---|
| `data.go:50` `Timestamp int64` | `sed -n '50p' libs/atlas-packet/character/data.go` | Exact match |
| `data.go:450` `w.WriteInt64(m.Inventory.Timestamp)` under `(t.IsRegion("GMS") && t.MajorAtLeast(79)) \|\| t.Region() == "JMS"` | `grep -n Timestamp data.go` | Exact match, gate text matches doc's paraphrase |
| `data.go:541` decode mirror | same | Exact match |
| `constants.go:25` `shoulder -51`, `:28` `pendant -17`, `:56` `pet2MagicScales -36`, `:83`/`:98` plain map write/read | `cat -n constants.go` | All exact |
| `shop_operation_enable_equip_slot.go:52-53` (encode) / `:68-69` (decode), non-legacy `pointType bool` + `serialNumber uint32` | file read | Exact — and the doc's own correction of the brief's stale `:58-72` citation is itself correct |
| `shop_operation_result_slots.go:186-231` `EnableEquipSlotExtSuccess` (mode+slotIndex+days, two uint16 fields) | file read | Exact — and the doc's correction of the brief's stale `shop_operation_body.go:527` citation is correct |
| `template_gms_95_1.json:2311/4839/4840` | `grep -n ENABLE_EQUIP_SLOT` | Exact match, including numeric mode values (10/117/118) |
| `data_test.go:69` `Timestamp: 94354848000000000` | `grep -n` | Exact match (also appears at 6 other sites in the same file) |
| `data_evan_test.go` exists, uses same `Timestamp: 94354848000000000` | file read | Confirmed |

No fabricated or stale-but-uncorrected citation found among the repo-side
evidence. Both places the report says it caught the brief's own stale line
numbers (`shop_operation_enable_equip_slot.go`, `shop_operation_body.go` →
actually `shop_operation_result_slots.go`) check out.

## 2. The load-bearing E2 claim (`InventoryData.Timestamp` == `aEquipExtExpire[0]`)

Flag, gate and wire position all confirmed identical between the doc's IDA
read (`dbcharFlag & 0x100000`, two `Decode4`s, immediately before the `0x4`
equipment block) and the Atlas encode/decode gate
`(t.IsRegion("GMS") && t.MajorAtLeast(79)) || t.Region() == "JMS"` at
`data.go:449-451` / `:540-542`. This structural claim holds.

**However, the "always zero" / "FILETIME 0" wording is not accurate.** The
write path is:

```
services/atlas-channel/atlas.com/channel/socket/writer/character_data.go:122
    Timestamp: ZeroTime,
services/atlas-channel/atlas.com/channel/socket/writer/set_field.go:45
    ZeroTime int64 = 94354848000000000
```

`94354848000000000` (100-ns ticks since 1601-01-01) decodes to
**1900-01-01**, not FILETIME 0 (which would be 1601-01-01). It is a
repo-wide sentinel — the same literal appears in
`libs/atlas-packet/model/asset.go:288,335` (asset `dateExpire` "no expiry"
placeholder) and is exactly the value pinned in every golden test
(`data_test.go`, `data_evan_test.go`). So the document's parenthetical "your
extended slot expired at FILETIME 0" is a specific, checkable claim that is
wrong as stated — the value sent is a distinct non-zero sentinel, not the
literal zero the doc names.

This does not change the substantive, actionable conclusion (the value is
currently a fixed, always-in-the-past placeholder, so the client will always
render the extension as expired, and Task 23 must persist a real expiry and
populate the field from it) — but it is exactly the class of unverified
numeric assertion this review exists to catch, and Task 23 should not go
looking for a literal `int64(0)` write site based on this doc's wording.
**Non-blocking finding**, `derivation-equip-slot.md:394` (and the summary
table repeats it at `:470`).

## 3. IDB claims — internal consistency and citation specificity

Each of K=51 (v79/v83/v84), K=59 (v87/v92/v95), K=36 (jms_v185) is backed by
its own address and its own quoted `lea`/`cmp` pair or decompiled line in
§1.6 — none is extrapolated from a sibling version without its own citation.
The cross-check arithmetic in §1.6 (`v79: 0x293+8×51=0x42B`, etc.) is
internally consistent for each row using the number stated for that row; I
did not re-derive the base literals (`0x293`, `0x2C7`, `0x309`, `0x2D9`,
`0x265`) independently since that requires IDB access, but the arithmetic as
written checks out for every row given those inputs.

**One material disagreement with previously committed, packet-audit-verified
evidence**, found in §1.7 (`derivation-equip-slot.md:281-308`):

- `libs/atlas-packet/cash/serverbound/shop_operation_enable_equip_slot_test.go:63-71`
  carries a `packet-audit:verify ... ida=0x468e43` marker (landed under
  task-085, commit history confirms) asserting that v72 address `0x468e43` is
  an **IDA-mislabeled** `CCashShop::OnEnableEquipSlotExt` (size `0x407`,
  body matches v79's `0x469fa9`), i.e. that the *equip-slot-extension send*
  does exist on v72.
- Task 21's doc, decompiling the same address independently, reports it as
  genuinely `CCashShop::OnIncCharacterSlotCount` (mode = `(itemId/1000==9110)+6`,
  the slot-count purchase), directly contradicting the existing pinned
  annotation.

The doc states its own reading as settled fact and moves on to conclude "no
effect on v72," without flagging that this reverses a previously
`packet-audit:verify`-pinned claim from a prior task. I have no IDA access to
adjudicate which read is correct. It doesn't change E1/E2/E3's actionable
conclusions for the v95 target, and the v72 "no effect" conclusion is also
independently supported by the separate `func_query`/`search_text` sweep for
`OnCashItemResEnableEquipSlotExtDone` (unaffected by which function
`0x468e43` really is). But a doc whose entire premise is "flag every
disagreement with existing evidence rather than silently overriding it"
should have called this out as needing reconciliation, not stated it as a
plain correction. **Non-blocking finding**, worth a follow-up ticket/note
before anyone relies on the v72 fixture's `packet-audit:verify` marker again.

## 4. Step 3 constants recommendation — verified against the actual file

- `-59` is free: scanning all 51 entries in `constants.go`, no `Position`
  value below `-51` exists except the pet-prefixed `-30..-48` range; `-59`
  does not collide with anything.
- `-51` is taken by `shoulder` (`constants.go:25`) — confirmed.
- `-36` is taken by `pet2MagicScales` (`constants.go:56`) — confirmed.
- `positionToSlot` (`constants.go:71-86`) is a plain `map[Position]Slot`
  populated by a single unconditional loop with no duplicate-key check —
  confirmed; a second entry at an already-used `Position` would silently
  overwrite.

All four sub-claims in the Step 3 recommendation check out against the
current file.

**Tree-state note (not a Task-21 defect):** at review time,
`libs/atlas-constants/inventory/slot/constants.go` has an *uncommitted*
working-tree diff adding exactly `{Type: "pendant2", Position: -59}` after
`pet3ItemIgnore`. This is not part of task-21's commit — `git diff --stat
27d1c91e8..9d3026337` shows only the derivation doc changed, and `git diff`
against `HEAD` (which is `9d3026337`, the task-21 commit) shows this as a
separate, currently-uncommitted modification. It matches the doc's own
recommendation exactly, so it is most likely stray in-progress work from a
later plan task (22/23) sharing this worktree, not evidence of a defect in
task 21's own commit. Flagged here only because the review brief calls for
confirming tree cleanliness; task 21's own commit is clean.

## 5. UNRESOLVED items — honest, not rounded off

- gms_v72/v61/v48 "no effect half": recorded with what was tried
  (`func_query` result counts, `search_text` over a specific address range,
  the empty-`session_id` reason v61/v48 were skipped) — not silently
  resolved, not silently dropped. (§1.7, summary §4)
- gms_v92 struct offset: explicitly flagged as not pinned, with the reason
  (read window didn't reach the store instruction) and why it doesn't block
  any server-side decision. (§1.6, §4)
- Conversely, nothing that IS resolved (E1, E1-b, E2, E3) is hedged past what
  Task 23 can act on — each has a concrete numeric answer plus its own
  citation.

## 6. Scope

Exactly one file changed by the commit under review; no code touched by this
commit. The working tree carries the pre-existing untracked `review-task-*.md`
/ `agent-ledger.tsv` files and modified `go.work.sum` as expected, plus the
unrelated uncommitted `constants.go` diff noted in §4 (not part of this
commit).

## Verdict rationale

The two load-bearing derivations this task exists to settle — E1 (slot = v95
body part 59, Atlas position −59, wire value always 0) and E2 (relog
propagation via `CharacterData::aEquipExtExpire`, matching Atlas's existing
but mis-named `InventoryData.Timestamp` field with the exact same flag/gate/
position) — are each backed by a specific address and quoted decompilation,
and every repo-side citation supporting them resolves exactly as claimed.
The Step 3 constants recommendation is verified correct against the actual
file on both the "−59 is free" and "−51/−36 are taken" halves, and the
"plain map, no duplicate detection" characterization is accurate. Two
findings are real but non-blocking: an imprecise "FILETIME 0" characterization
of a named non-zero sentinel constant, and an unflagged reversal of a
previously packet-audit-verified pin for an address that doesn't affect the
document's actionable conclusions.

---

verdict: APPROVED_WITH_FINDINGS
artifact: docs/tasks/task-240-cash-shop-stub-operations/review-task-21.md
scope_confirmed: single-commit doc-only diff (27d1c91e8..9d3026337, derivation-equip-slot.md, 486 lines); all repo file:line citations re-resolved independently; IDB claims checked for internal consistency and citation specificity only (no ida-pro access)
blocking: 0
non_blocking: 2
  - docs/tasks/task-240-cash-shop-stub-operations/derivation-equip-slot.md:394 (and :470) — "the field is currently always zero (FILETIME 0)" is imprecise; the actual write is the named sentinel `ZeroTime = 94354848000000000` (1900-01-01) at services/atlas-channel/atlas.com/channel/socket/writer/set_field.go:45, not literal FILETIME 0. Functional conclusion (always-past placeholder) still holds; the specific wording does not match the code.
  - docs/tasks/task-240-cash-shop-stub-operations/derivation-equip-slot.md:281-308 — §1.7's re-decompilation of v72 address 0x468e43 as `OnIncCharacterSlotCount` directly contradicts a previously `packet-audit:verify`-pinned claim in libs/atlas-packet/cash/serverbound/shop_operation_enable_equip_slot_test.go:63-71 (landed under task-085) asserting the same address is an IDA-mislabeled `OnEnableEquipSlotExt`. The doc states its own reading as settled without flagging the disagreement for reconciliation; doesn't change E1/E2/E3's actionable conclusions but should be surfaced as a follow-up.
not_evaluable: 0
