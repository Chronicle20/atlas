# Review — Task 11: route Cash/0543 to the Maple Life dialog by classification

Commit range: `b9581c675..e2d87afd9` (single commit `e2d87afd9`).
Brief: `.superpowers/sdd/plan/task-11-brief.md` (reviewed against the
**Controller addendum**, which overrides the base brief body).
Report: `.superpowers/sdd/plan/task-11-report.md`.

## Scope

```
services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go       | 39 ++++
services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use_maple_life_test.go | 302 ++
services/atlas-channel/atlas.com/channel/socket/handler/maple_life_open.go                | 52 ++++
services/atlas-channel/atlas.com/channel/socket/handler/maple_life_open_test.go           | 125 ++
4 files changed, 518 insertions(+), 0 deletions(-)
```

All four files are additive-only, all inside
`services/atlas-channel/atlas.com/channel/socket/handler/`. No other file in
the repo changed in this range (confirmed via `git diff --stat` scoped to
`libs/`, `docs/packets/`, `services/atlas-login/`,
`services/atlas-character-factory/`, `deploy/`, and each of the six
out-of-scope templates — all empty diffs). `maplelife/registry.go` is
untouched. Matches the stated unit exactly.

## 1. Classification-first property (control-flow read, not spot-check)

Read `character_cash_item_use.go` in full for `CharacterCashItemUseHandleFunc`.
Sequence of every `it ==` comparison and the classification computation:

- `it := GetCashSlotItemType(t)(itemId)` — line 67.
- Comparisons against `it` run from line 69 through line 774 (pet-consumable,
  pet-skill, chalkboard, kite, field-effect, note, teleport-rock, item-tag,
  `CashSlotItemTypeSeal || sealTimedCashSlotItemType(t)` (268), karma-scissors
  (336), expiration-extender (458), incubator (559), cube (682),
  vegas-spell (688), point-reset (709), pet-name-tag (716), currency-sack
  (723), `viciousHammerCashSlotItemType(t)` (732), store-search (739),
  `nameChangeCashSlotItemType(t)` (770), `worldTransferCashSlotItemType(t)`
  (774)).
- `category := item.GetClassification(itemId)` — line 783 (pre-existing).
- `if category == item.ClassificationRemoteMerchant` — 787.
- **New 543 arm** — line 800, `if category == item.ClassificationCharacterCreation`.
- `if category == item.ClassificationMegaphones …` — 813.

For a Maple Life item (classification 543), `GetCashSlotItemType`'s
`ClassificationCharacterCreation` branch (`:1496-1506`) returns 65 (v83) or 66
(v95) for `itemId/1000-5431 <= 1`, and 57/58 for the far-range ids. I checked
every `it ==` comparison between lines 69–774 against those four values on
both a v83 and a v95 tenant:

- `sealTimedCashSlotItemType(t)` returns 64 on v83 / 65 on v95 — **tenant-
  scoped**, so it never equals 65 on the v83 tenant where Maple Life's `it`
  is also 65 (Maple Life is 65 on v83, sealTimed is 64 on v83: no collision on
  the SAME tenant). On v95, sealTimed=65 but Maple Life=66: no collision.
- `viciousHammerCashSlotItemType(t)` returns 66 on v83 / 67 on v95 — also
  tenant-scoped. On v83 Maple Life's `it`=65 vs vicious hammer's 66: no
  collision. On v95 Maple Life's `it`=66 vs vicious hammer's 67: no collision.
- No `it ==` comparison anywhere in lines 69–774 uses the raw literals 57 or
  58 (`grep -n "CashSlotItemType(57)\|CashSlotItemType(58)"` only matches
  inside `GetCashSlotItemType` itself, never in a handler dispatch
  comparison).

So every pre-existing `it ==` arm that could numerically collide with a Maple
Life value is itself version-scoped through a `xCashSlotItemType(t)` helper
that never returns the same numeric value Maple Life's own tenant-scoped
lookup returns for the same tenant. The new arm's own body (`:800-812`)
contains zero `it ==` / `CashSlotItemType(N)` comparisons — it gates purely
on `category`. **Verified as a property, not a spot-check: PASS.**

Also confirmed the ownership check (`cashItemInSlotFunc`, lines 60-64) runs
before `it` is even computed, so FR-5.3 (slot ownership) is enforced before
any arm, including the new one.

## 2. Neighbour-collision regression suite

`character_cash_item_use_maple_life_test.go:71-165`,
`TestCashItemNeighbourArmsUnaffectedByMapleLife`. Mechanism: drives the real
`CharacterCashItemUseHandleFunc` with a hand-built wire payload (common
`ItemUse` prefix + the correct sub-body shape per neighbour) via
`installCashItemInSlotSeam`, then observes arm selection by checking
`maplelife.GetRegistry().Get(sessTen, accountId)` — **presence of a registry
entry is the sole observation of "the Maple Life arm was reached."** This is
a real behavioural probe, not a re-derivation of `GetCashSlotItemType`
(that's a separate, prior assertion in the same subtest at line 103,
correctly labelled "structural check" in the test's own comment).

All four rows (pet multi-consumable/5460000, sealing lock/5061000, vicious
hammer/5570000, maple life/5431000) run against both GMS v83 and GMS v95
(`t.Run(c.name+"/"+v.name, …)`, 8 subtests total) — confirmed by running
`go test -run 'TestCashItemNeighbourArmsUnaffectedByMapleLife' -v`, all 8
subtests present and PASS. `TestMapleLifeArmUsesNoBareCashSlotTypeComparison`
additionally source-greps the isolated arm body for the four forbidden
`CashSlotItemType(N)` literals (FR-2.2 made executable). **PASS — non-vacuous
on both tenants for all four rows.**

## 3. v84 exclusion

`mapleLifeSupported` (`character_cash_item_use.go:1114-1131`):

```go
func mapleLifeSupported(t tenant.Model) bool {
	return t.IsRegion("GMS") && t.MajorAtLeast(83) && t.MajorVersion() != 84
}
```

Explicit `!= 84` exclusion, not merely `MajorAtLeast`. Doc comment names the
Task 2 ruling. `TestMapleLifeSupported` table row `{"GMS 84", false}`
(`character_cash_item_use_maple_life_test.go:230`) — confirmed by running the
test, `GMS_84` subtest PASSes against `want=false`.
`TestMapleLifeUnsupportedVersionWritesNothing` covers both v79 and v84
(`:258-293`), each asserting zero writer calls and no registry entry.

`git diff --stat` over the full range confirms zero changes to any of the six
out-of-scope templates (`template_gms_{84,48,61,72,79}_1.json`,
`template_jms_185_1.json` — found only under
`services/atlas-configurations/seed-data/templates/`, untouched) and zero
changes under `libs/`, `docs/packets/`, `services/atlas-login/`,
`services/atlas-character-factory/`, `deploy/`. **PASS.**

## 4. `TestMapleLifeUnsupportedVersionWritesNothing` — all three halves

- No writer call: `wpCallCounter` asserts `calls.n == 0`. PASS.
- No registry entry: `maplelife.GetRegistry().Get(ten, accountId)` asserts
  `!ok`. PASS.
- **Reader left unconsumed past the common prefix: NOT asserted.** The test
  sends only the common `ItemUse` prefix bytes (no Maple Life sub-body
  bytes) and never checks the reader's position or that `sp.Decode` was not
  called. This is exactly the checklist's predicted risk ("the reader-
  position half is the one most likely to have been dropped").

  Control-flow reading confirms the *code* is correct regardless: the arm
  returns (`character_cash_item_use.go:801-804`) before `sp.Decode(l, ctx)(r,
  readerOptions)` is ever reached when `!mapleLifeSupported(t)`, so no reader
  consumption is structurally possible on an unsupported version. But the
  brief explicitly asked for this to be a **test-asserted** property (FR-2.4),
  and it is not — an assertion like "no extra bytes remain in the reader" or
  a mock that fails on any further `Read*` call would have made this
  regression-proof rather than merely true-by-current-code-shape. **Finding
  (non-blocking): the reader-position half of this test is missing**, not
  tautological — it is simply absent.

## 5. `beginMapleLife` — session-sourced account/world, idempotency, Phase

`maple_life_open.go:38-52`: `beginMapleLife`'s returned closure takes
`(s session.Model, itemId item.Id, source slot.Position, updateTime uint32)`
— there is no packet-supplied account or world anywhere in the signature, so
FR-4.2 is enforced by the type signature, not merely by convention.
`s.AccountId()` / `s.WorldId()` are the only sources used in the `Put` call.

`TestBeginMapleLifeRecordsPending` (`maple_life_open_test.go:46-81`) asserts
`CharacterId`, `WorldId`, `ItemId`, `Slot`, `UpdateTime`, and
`Phase == maplelife.PhaseOpen` all match the exact input values — genuinely
exercised, not a subset.

`TestBeginMapleLifeIsIdempotent` (`:83-107`) calls `beginMapleLife` twice for
the same account with **different** item/slot/updateTime values
(5431000/-3/1, then 5432000/-4/2) and asserts the registry holds exactly the
second call's values. This is a non-vacuous idempotency test — a `Put` that
silently no-oped on the second call, or that merged fields instead of
replacing, would fail this test. **PASS.**

## 6. `updateTime` handling across the version split

`libs/atlas-packet/cash/serverbound/item_use_maple_life.go`'s own doc
(`:37-92`) establishes that this sub-body's trailing `update_time` is written
**unconditionally on every version** (IDA-verified two independent call sites
on v87+, one on v83) — `updateTimeFirst` is accepted for signature parity
only and is not read inside `Encode`/`Decode`.

The handler arm:

```go
sp := cashsb.NewItemUseMapleLife(updateTimeFirst)
sp.Decode(l, ctx)(r, readerOptions)
if !updateTimeFirst {
    updateTime = sp.UpdateTime()
}
```

`updateTimeFirst := cashsb.UpdateTimeFirst(t)` is computed once at the top of
`CharacterCashItemUseHandleFunc` (`:56`) from the tenant, and passed through
correctly. On v83 (`updateTimeFirst == false`), the header carries no leading
copy, so `updateTime` is overwritten from the sub-body's (only) trailing
decode — correct, matches the codec's documented single-occurrence shape. On
v87+ (`updateTimeFirst == true`), the header's leading copy (already in
`updateTime` from `p.UpdateTime()`, line 57) is kept, and the sub-body's
always-present but redundant second copy is correctly discarded. **No swapped
boolean — PASS.**

## 7. Registry untouched, no scope creep

`git diff b9581c675..e2d87afd9 -- services/atlas-channel/atlas.com/channel/maplelife/`
is empty. No `*_testhelpers.go` added (new test constructors live inside the
`_test.go` files themselves, following the package-var/builder precedent
already in the package). No TODO/FIXME/placeholder/stub text anywhere in the
diff (`git diff | grep -ni "TODO\|FIXME\|placeholder\|not implemented\|stub"`
— no output). `gofmt -l` and `go vet ./socket/handler/...` both clean on the
touched/added files. **PASS.**

## Addendum items confirmed absent (not findings, per instructions)

- No `MapleLifeUseHandleFunc`, no `maplelifesb` import anywhere in
  `socket/handler/` (`grep -rn` returns no matches).
- `libs/atlas-packet/maplelife/serverbound/` contains only `check_name.go`
  and `serverbound_test.go` — no `use.go` was created.
- No `CashItemUseMapleLife` coverage-matrix cell or `candidatesFromFName`
  case added; no packet-audit artifact touched.

## Test-run verification (independent of the report)

```
cd services/atlas-channel/atlas.com/channel
go build ./...                                          # clean
go test ./socket/handler/... -run 'MapleLife' -v        # all PASS (21 cases)
gofmt -l <4 files>                                       # clean
go vet ./socket/handler/...                               # clean
```

**One anomaly, not treated as a finding**: a single ad-hoc run of
`go test ./socket/handler/... ./maplelife/...` (no `-run` filter, default
cache) produced 3 failures in `TestBeginMapleLife{RecordsPending,
IsIdempotent, UsesSessionAccountAndWorld}` ("expected a pending entry"). This
did not reproduce over 15 subsequent attempts, including 10 with
`-count=1` immediately after `go clean -testcache`, nor with `-v` covering
the full package. No `t.Parallel()` exists anywhere in the package (grep
confirmed), and `mustTenant` mints a fresh `uuid.New()`-keyed tenant per
call, so there is no structural pollution vector between these tests and any
other test in the package that I could locate. Given 15/15 clean reruns
against 1/16 failing, and no design in the new code that plausibly explains
sequential-run flakiness, I attribute the single failure to sandbox
environment noise (this is a resource-constrained WSL2 shell) rather than a
defect in the unit under review. Flagging it here for the record rather than
silently dropping it, per "never suppress a real concern."

## Findings

**Blocking:** none.

**Non-blocking:**
1. `character_cash_item_use_maple_life_test.go:258-293` —
   `TestMapleLifeUnsupportedVersionWritesNothing` does not assert the
   "reader left unconsumed past the common prefix" half of FR-2.4 (checklist
   item 4). The property is true by construction (verified by direct code
   read — the guard `return`s before `sp.Decode` is ever called), but the
   brief asked for it to be test-proven, and no such assertion exists. Not
   blocking because I independently proved the property from the control
   flow; a future refactor that moved the `mapleLifeSupported` check after
   `sp.Decode` would break FR-2.4 with no test catching it.
2. One non-reproducible full-package test-run anomaly, documented above —
   noted for the record, not attributed to this unit's code.

**Not evaluable:** none — this task's surface (a handler package,
consuming a stable, unmodified upstream registry interface) was fully
within the review's reach.

## Verdict

APPROVED_WITH_FINDINGS. The task's core reason for existing — classification-
first dispatch that structurally cannot collide with the pet-multi-
consumable/sealing-lock/vicious-hammer neighbours — is correctly implemented
and genuinely (non-vacuously) tested on both v83 and v95. The v84 exclusion,
the session-sourced account/world, the idempotent registry write, and the
`updateTime` leading/trailing split are all correct. The only gap is a missing
test assertion (not a missing behaviour) for the reader-position half of the
unsupported-version guard.
