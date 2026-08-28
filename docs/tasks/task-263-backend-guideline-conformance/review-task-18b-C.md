# Review: Task 18b-C — final six DOM-04 forgotten-Transform packages

Commit reviewed: `21c1f2367` ("feat(task-263): add Transform for final six DOM-04 forgotten packages")
Brief: `.superpowers/sdd/plan/task-18b-C-brief.md`
Report: `.superpowers/sdd/plan/task-18b-C-report.md`

## Scope

Single commit, 13 files, 487 insertions / 1 deletion, across four modules:
`atlas-messages` (`messages/character`, `messages/data/map`), `atlas-monsters`
(`monster/consumable`, `monster/mobskill`), `atlas-npc-conversations`
(`npc/petdata`), `atlas-npc-shops` (`npc/character`), plus one append to
`handwork-notes.md`. Matches the brief exactly — no scope drift.

Verified independently (not just re-reading the report):
- `git show --stat 21c1f2367` for the file list.
- Read every `model.go`/`rest.go` touched, plus each `rest_test.go` diff, in full.
- `go build ./...` in all four module roots — clean, matching the report.
- `go test ./character/... ./data/map/...` (messages), `./monster/consumable/... ./monster/mobskill/...`
  (monsters), `./petdata/...` (npc-conversations), `./character/...` (npc-shops) — all `ok`.
- `git diff 21c1f2367~1 21c1f2367 -- <each pre-existing rest_test.go>` filtered to `^-` lines to
  confirm no deleted/reworded test content.

## Judgment call 1 — `npc-conversations/petdata`: `Evolutions` left nil

Verified from source, not the report's narrative:
- `model.go:8` — `Model.evolutions` is declared `int`.
- `rest.go` `Extract` — `evolutions: len(rm.Evolutions)`, collapsing the wire slice to a count.
- `model.go` — `IsEvolvable() bool { return m.evolutions > 0 && m.reqItemId != 0 }` is the only
  reader of `m.evolutions`; it needs a boolean-ish count, never `TemplateId`/`Probability`.
- `NewModel(id, name, reqPetLevel, reqItemId, evolutions int)` is the only other constructor and
  also takes a bare `int` — confirmed via `grep -rln petdata services/atlas-npc-conversations/atlas.com/npc`
  that the only production users of this package are `petdata/processor.go` (fetches and `Extract`s
  the upstream resource; never calls `Transform`) and `conversation/operation_executor.go` (uses the
  processor, never touches `RestModel.Evolutions`). `Transform` in this package has **no production
  caller** — it exists purely to close the DOM-04 conformance gap, not to serve a live wire path.
- Conclusion: `Model` genuinely has no evolutions collection to source from, and no consumer reads
  `RestModel.Evolutions` from a value produced by this package's `Transform`. Leaving it `nil` is
  correct, not a silent data gap — there is no live data path shipping a `nil` to a client.
- `rest_test.go` (appended) asserts this honestly: builds `rm` with a non-empty `Evolutions`, round
  trips through `Extract`→`Transform`, asserts `rm2.Evolutions == nil` explicitly (not merely absent
  from a `DeepEqual`), and asserts the resulting `IsEvolvable()` correctly flips to `false` on the
  second pass. No fabrication anywhere.

**Verdict: correct, well-evidenced, not a finding.**

## Judgment call 2 — `messages/data/map`: empty `Transform`

Verified from source:
- `data/map/model.go` — `type Model struct{}`, zero fields, confirmed directly.
- `data/map/processor.go` `GetById` returns `(Model, error)` via `requests.Provider[RestModel, Model]`.
- `messages/map/processor.go:41-42` — the only consumer: `Exists(mapId)` calls
  `_, err := p.dp.GetById(mapId)` and discards the `Model`, keeping only `err`.
- No other file under `services/atlas-messages/atlas.com/messages` imports `data/map` in production
  code (`grep -rln "data/map\"" ... | grep -v _test.go` → only `messages/map/processor.go`).

**Verdict: correct — `Model` truly carries no data, and independently re-derived the same
single-consumer, error-only-usage claim the report made. Not a finding.**

## Field parity — the other four packages

- `messages/character/rest.go` — 29-field `Transform`, every `RestModel` field sourced from the
  matching `Model` field by name; `Stance` is the one exception, documented: `Model` has no
  `stance` field, only a hardcoded `Stance() byte { return 0 }` stub, and `Transform` assigns the
  literal `0` directly (not `m.Stance()`, respecting D1). Verified against `model.go` field list —
  no dropped or swapped-neighbour field.
- `monsters/monster/consumable/rest.go` — 6-field `Transform`, 1:1 with `Model`'s fields and
  `Extract`'s existing reverse mapping. Clean.
- `monsters/monster/mobskill/rest.go` — 16 scalar fields + `Summons`, all correctly named-neighbour
  mapped; no `Id` field exists on `RestModel` (composite `SkillId`+`Level` key via `GetID()`), so
  there is no `Id`-mapping ambiguity — confirmed this claim by reading `GetID()`.
- `npc-shops/character/rest.go` — 29-field `Transform`, same shape as `messages/character` but with
  real `x`/`y`/`stance` fields on `Model` (not stubs) — `Transform` reads `m.x`, `m.y`, `m.stance`
  directly, all correctly matched to their `RestModel` neighbours (no swap).

No dropped fields, no same-typed-wrong-neighbour assignments found in any of the four.

## `RestModel.Id` mapping and assertion

- `messages/character`: `Id: m.id`; test asserts `rm2.Id != rm.Id`. Pass.
- `messages/data/map`: no `id` exists anywhere on `Model` or `RestModel` usage in this package
  (`RestModel.Id` stays the zero value by design, matching the empty-`Model` finding above); the
  test doesn't fabricate an assertion it can't back. Acceptable given judgment call 2 holds.
- `monsters/monster/consumable`: `Id: m.id`; test asserts `rm2.Id`. Pass.
- `monsters/monster/mobskill`: composite key `SkillId`+`Level`, both mapped and asserted together.
  Pass — no single `Id` field is expected here, verified against `GetID()`.
- `npc-conversations/petdata`: `Id: m.id`; test asserts `rm2.Id`. Pass.
- `npc-shops/character`: `Id: m.id`; test asserts `rm2.Id`. Pass.

No instance of the Task 17 "`Id` left at zero and untested" bug recurred.

## Test rigor

- All six new/appended `TestTransformRoundTrip` tests use distinct, non-zero, in-range fixture
  values per field (byte-width fields checked: `Level byte`≤255, `SkinColor`/`Gender` bytes,
  `Fame int16`, all within range; `mobskill`'s `int32`/`uint32` fields all small positive values).
  No all-zero fixture found.
- All six do a full `reflect.DeepEqual` `Model`→`Model` round trip (or, for `petdata`, an explicit
  field-by-field check plus an explicit `nil` assertion where a full round trip is provably
  impossible — correctly justified, not a workaround for laziness).
- `mobskill`'s test additionally mutates the returned `Summons` slice and asserts the source
  `Model`'s backing slice is unaffected — a real non-aliasing proof, not just a length check.
- One real (but non-blocking, and self-disclosed) blind spot: in `npc-shops/character`, the
  pre-existing `Extract` (unchanged by this diff, `rest.go`) never reads `RestModel.X`/`Y`/`Stance`
  into `Model`. Because the round-trip test builds `m` via `Extract(rm)` first, `m.x`/`m.y`/`m.stance`
  are already `0` before `Transform` ever runs, so the `DeepEqual` on `X`/`Y`/`Stance` specifically
  proves nothing about `Transform`'s correctness for those three fields (it would still pass even
  if `Transform` swapped `X`↔`Y`). I independently re-read `Transform` and confirmed it does map
  `X: m.x, Y: m.y, Stance: m.stance` correctly by name, so there is no actual behavioral bug — only
  a test-rigor gap for 3 of 29 fields, caused by, and inseparable from, the pre-existing `Extract`
  defect. The report discloses this explicitly in `handwork-notes.md` rather than hiding it. Per
  the task's own scoping note (pre-existing `Extract`-drops-a-field defects are out of scope; a
  *new instance* should be noted non-blocking), this is exactly that: a new instance (X/Y/Stance,
  not `spawnPoint`), correctly disclosed. **Non-blocking.**

## Pre-existing test files — purely additive?

Verified independently with `git diff 21c1f2367~1 21c1f2367 -- <file>` filtered to `^-` lines for
every touched pre-existing `rest_test.go`:
- `messages/character/rest_test.go` — no deleted lines (only an import-line→import-block change,
  which is additive in effect).
- `monsters/monster/consumable/rest_test.go` — exactly one `-` line: `-import "testing"`, replaced
  by a multi-line import block adding `reflect`. No test function touched.
- `npc-conversations/petdata/rest_test.go` — no deleted lines.

No prior test function was dropped, renamed, or reworded. The report's claim holds.

## Collection copying — `mobskill.Summons`

`rest.go`:
```go
summons := make([]uint32, len(m.summons))
copy(summons, m.summons)
```
Confirmed `make`+`copy`, not aliasing, in `Transform`. (The pre-existing `Extract` does alias
`rm.Summons` into `m.summons` directly — unchanged by this diff, out of scope, and does not
compound with `Transform`'s new copy since it's a distinct direction.)

## Design decision D1 — direct field access

`grep`-verified across the whole commit's `Transform` functions: no `m.<ExportedGetter>()` call
appears in any of the six `Transform` bodies (the only `m.X`/capital matches in the diff are inside
unrelated pre-existing `Extract` functions reading `rm.X`). All six use direct unexported-field
access (`m.id`, `m.name`, ...). No new accessors were added (confirmed no `model.go` file appears
in the diff). Compliant.

## Not evaluable

None. The full review surface (six `rest.go`/`rest_test.go` pairs, the two `model.go` files whose
contract the judgment calls depend on, and the two named consumers) was read and independently
verified; nothing in scope was left unchecked.

## Summary

Both hard judgment calls (`petdata.Evolutions` nil, `messages/data/map` empty `Transform`) are
independently verified correct, with production-consumer tracing that confirms no data gap ships to
a real caller. Field parity, `Id` mapping/assertion, non-aliasing, D1 compliance, and pre-existing
test preservation all check out. One test-rigor gap exists (`npc-shops/character`'s X/Y/Stance
round trip is inert due to a pre-existing, out-of-scope `Extract` bug) but it is self-disclosed,
does not indicate an actual `Transform` defect (verified by direct read), and matches the class of
finding the task brief explicitly rules non-blocking.

---

verdict: APPROVED_WITH_FINDINGS
artifact: docs/tasks/task-263-backend-guideline-conformance/review-task-18b-C.md
scope_confirmed: reviewed commit 21c1f2367 in full — all 6 rest.go/rest_test.go pairs, both model.go files underlying the two judgment calls, both named production consumers (messages/map/processor.go, petdata/processor.go + conversation/operation_executor.go), and handwork-notes.md; independently rebuilt and re-tested all four affected modules
blocking: 0
non_blocking: 1
  - services/atlas-npc-shops/atlas.com/npc/character/rest_test.go (TestTransformRoundTrip) — the X/Y/Stance round trip is inert (both m and m2 land at 0) because pre-existing Extract (rest.go) never populates Model.x/y/stance from RestModel.X/Y/Stance; Transform itself is correct (X: m.x, Y: m.y, Stance: m.stance verified by direct read), but the test provides no signal for those 3 fields. Already disclosed by the implementer in handwork-notes.md; matches the task's own "new instance, note non-blocking" carve-out for pre-existing Extract field-drop defects.
not_evaluable: 0
