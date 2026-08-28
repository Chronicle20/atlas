# Review: Task 11 — atlas-inventory explicit-stat asset creation

Range: `3ba8a3aff..b36fff470` (single commit `b36fff470`)
Brief: `.superpowers/sdd/plan/task-11-brief.md`
Report: `.superpowers/sdd/plan/task-11-report.md`

## Scope

`git diff --stat 3ba8a3aff..b36fff470`:

```
asset/processor.go                                   |  93 ++++++++++++++--
asset/processor_test.go                              | 122 +++++++++++++++++++++
compartment/mock/processor.go                        |  18 +--
compartment/processor.go                              |  90 +++++++++++++--
kafka/consumer/compartment/consumer.go               |  19 +++-
kafka/message/compartment/kafka.go                   |  16 +++
kafka/message/compartment/kafka_test.go              |  60 ++++++++++
```

Matches the brief's file list plus one file the brief did not name
(`compartment/mock/processor.go`), which is addressed under finding 3 below.
All work is confined to `services/atlas-inventory/atlas.com/inventory`, as
the brief specifies. Scope is confirmed — no drift.

## 1. Does a test actually assert the seam?

Yes, and it asserts the correct thing. `asset/processor_test.go`:

- `TestCreateAssetAppliesExplicitStats` (new) calls `p.Create(mb)(...)` with
  `CreateOptions{Slots:7, Strength:3, WeaponAttack:4, WeaponDefense:6, HP:15}`
  and asserts on the returned `asset.Model` accessors — `a.Strength()`,
  `a.WeaponAttack()`, `a.WeaponDefense()`, `a.Hp()`, `a.Slots()` — i.e. on the
  **persisted asset**, not on the options struct. This is the load-bearing
  assertion the task exists to produce.
- `TestCreateAssetExplicitStatsTakePrecedenceOverAverage` repeats the same
  call with `UseAverageStats: true` **and** explicit stats set, and asserts
  the explicit values still land on the asset — pinning the precedence rule
  the brief asked the implementer to decide and document.
- `TestCreateAssetWithoutExplicitStatsIsUnchanged` is the regression guard:
  no stat fields set, `UseAverageStats: true`, and it stands up a real
  `httptest` stat-service double so the pre-existing `statProcessor.GetById`
  path is exercised, not skipped. Asserts the template's stats land verbatim
  — the old behaviour is provably unchanged.

Verified by reading `asset/processor.go`'s new `hasExplicitStats` /
`applyExplicitStats` helpers: `hasExplicitStats` is checked before the
`UseAverageStats` branch inside the `TypeValueEquip` case of `Create`
(`asset/processor.go:556-579` post-change), so explicit stats really do take
precedence, and the explicit path is entered only when at least one stat or
`Slots` is non-zero — a zero-everything craft falls through to the
pre-existing atlas-data lookup, exactly matching
`TestCreateAssetWithoutExplicitStatsIsUnchanged`.

I additionally traced the full chain that the brief calls the seam
(`CreateAssetCommandBody → handleCreateAssetCommand → CreateAssetAndEmit →
CreateAssetAndLock → CreateAsset → asset.CreateOptions`) by hand:

- `consumer.go:240-256` builds `compartment.EquipStats{...}` directly from
  `c.Body`'s new fields and passes it as the sole variadic argument.
- `compartment/processor.go`'s three methods thread `stats...` straight
  through to `firstEquipStats`, which is applied into **both**
  `asset.CreateOptions{...}` literal sites inside `CreateAsset`
  (`:1360-1380`, the stack-merge branch, and `:1394-1414`, the normal
  branch) — every one of the 16 stat fields is copied field-for-field, no
  drops.

No test exercises this middle hop (`compartment.Processor` or the consumer
handler) directly with a non-zero `EquipStats` — the brief's own Step 1 test
list only asked for kafka-wire and `asset.Create` level tests, and the
compartment/consumer layers are pure pass-through with no branching logic on
the new fields, so hand-verification is adequate here. Noted as **not
evaluable via a dedicated test**, but not a blocking gap since the brief did
not ask for one and the code at that hop has no conditional logic to hide a
bug.

**Verdict on point 1: satisfied.** The seam test is real, asserts on the
created asset, and would fail without the change (confirmed structurally —
`CreateOptions` has no `Strength`/`Slots` fields before this commit, so the
test literal would not compile against the pre-change struct).

## 2. The variadic `EquipStats ...` API change

Judged on the merits, per the review brief's explicit ask.

**What happens when the variadic is empty:** `firstEquipStats` returns a
zero-valued `EquipStats{}` (`compartment/processor.go`, new).
`hasExplicitStats` in `asset/processor.go` treats an all-zero `CreateOptions`
stat block as "no explicit stats" and falls back to the pre-existing
roll/average path. This means every one of the ~25 pre-existing call sites
in `compartment/processor_test.go`, plus the two internal `CreateAsset`
calls inside the stack-merge/split logic at (post-change) `:1620` and
`:1636`, keep their exact pre-change behaviour with zero source changes.
Confirmed: `grep -c "CreateAsset(" compartment/processor_test.go` → 23, none
of them touched by this diff, and `go build ./...` / targeted `go test`
scoped to `atlas-inventory/compartment` pass clean.

**What happens if more than one `EquipStats` is passed:** `firstEquipStats`
takes `stats[0]` and silently drops every subsequent element
(`compartment/processor.go`, `firstEquipStats`). This compiles without any
diagnostic — `CreateAsset(mb)(..., es1, es2)` is valid Go and es2 vanishes
with no warning, log line, or panic. This is exactly the "hides a required
argument" failure mode the review brief asked me to check for, and it is
real, but the current diff does not exercise it: there is exactly one
production call site that passes an `EquipStats` value
(`consumer.go:240`), and it passes exactly one.

**Would I keep it?** I would not have chosen the variadic if the choice were
mine going forward, but I would not block this commit on it. An optional
pointer (`stats *EquipStats`, nil = absent) expresses "0 or 1, and nothing
more" at the type level, which the variadic does not — the variadic's true
contract ("only index 0 is ever read") is documented only in a doc comment
on `firstEquipStats`, not enforced by the compiler. The tradeoff the
implementer names — 16 more positional `uint16` parameters vs. touching 25
call sites vs. this variadic — is real, and a pointer would have solved it
with the identical "zero call sites touched" property (`nil` is the zero
value for a pointer parameter, just as omitting a variadic is for a slice)
while closing the "more than one silently dropped" hole entirely. This is a
**non-blocking design note**, not a defect: nothing today can trigger the
multi-argument case, and the doc comment on `EquipStats` and
`firstEquipStats` is honest about the intended contract.

## 3. `compartment/mock/processor.go` — necessary, not scope creep

`compartment/processor.go`'s `Processor` interface is widened
(`CreateAssetAndEmit`, `CreateAssetAndLock`, `CreateAsset` all gain
`stats ...EquipStats`). `ProcessorMock` in `compartment/mock/processor.go`
implements that same interface — every implementer of `compartment.Processor`
must update in lockstep or the package fails to compile. Confirmed the mock
diff only touches the three widened signatures and their trivial pass-throughs
(`compartment/mock/processor.go:53-65`, `:366`, `:437-449`), nothing else.
`go build ./...` (module-scoped) is clean. This is a required consequence of
the interface change, correctly in scope, exactly as the brief's Step 5
anticipates ("If any of these is an interface method, update its mock in the
same step").

## 4. The `omitempty` asymmetry

Verified the implementer's reasoning directly rather than taking it at face
value.

- `CreateAssetCommandBody` (inventory copy, `kafka/message/compartment/kafka.go:114-131`)
  has `omitempty` on all 15 stat fields, none on `Slots` — matches the
  brief's Step 3 code block verbatim.
- `AwardCraftedAssetPayload` (Task 10, `libs/atlas-saga/payloads.go:1078-1099`,
  already landed and reviewed at `3ba8a3aff`) has **no** `omitempty` on any
  field, `Slots` included, by explicit design choice documented in its own
  comment ("Slots deliberately has no omitempty: a zero-slot craft is
  meaningful").
- JSON key **names** match character-for-character between the two structs
  (`slots`, `strength`, `dexterity`, ... `jump`) — confirmed by diffing both
  field lists side by side. This is the part that actually matters for wire
  compatibility.
- Confirmed by grep that `CreateAssetCommandBody` is **never marshaled**
  anywhere in `atlas-inventory` — the only uses are the struct declaration,
  the consumer's `c.Body` field access (decode-only, since Sarama delivers
  bytes that `message.Handler` unmarshals), and the two new round-trip
  tests. `atlas-inventory` is purely a consumer of this command; it never
  produces one. `omitempty` is an encode-only directive
  (`encoding/json`'s `Marshal`); it has zero effect on `Unmarshal`. So the
  implementer's claim — "safe because omitempty affects only encoding, not
  decoding, and this service never encodes this struct" — is correct and
  verified against actual usage, not just asserted.

**For Task 12, not yet dispatched:** the orchestrator-side copy of
`CreateAssetCommandBody` that Task 12 will add is the one that actually
*encodes* this struct onto the wire (it is the producer). Its own choice of
`omitempty` tags governs what bytes are actually sent — if the orchestrator's
struct also omits `omitempty` on some field this inventory struct expects to
default to zero when absent, that is still fine on decode (missing key →
Go zero value regardless of the *receiver's* tag). The one thing that
**would** matter is if Task 12's struct uses different **JSON key names**
than this one (e.g. `weaponAtt` vs `weaponAttack`) — that would silently
drop a field on decode with no error, the exact two-copy trap the brief
warns about. Task 12 should assert its own struct's field names against
this inventory struct's names (not against `AwardCraftedAssetPayload`'s
`omitempty` tags, which are irrelevant to the seam). I am flagging this as
guidance for Task 12's brief/review, not as a defect in Task 11 — Task 11's
own reasoning and code are correct as they stand.

## Other checks

- `go build ./...` and `go test ./kafka/message/compartment/... ./asset/...
  ./compartment/... -count=1`, run from
  `services/atlas-inventory/atlas.com/inventory` only (module-scoped, no
  `-race`, no repo-wide gate): all packages pass.
  ```
  ok  	atlas-inventory/kafka/message/compartment	0.062s
  ok  	atlas-inventory/asset	0.078s
  ok  	atlas-inventory/compartment	0.641s
  ```
- `database.ApplyOnce` idempotency guard (task-208) in
  `handleCreateAssetCommand` is untouched by this diff — confirmed by
  reading the full consumer.go diff, which only inserts the `EquipStats{}`
  literal into the existing `CreateAssetAndEmit(...)` call, no
  restructuring.
- `applyExplicitStats` sets exactly the same 16 `Builder` setters that
  `applyEquipStats` sets (`Strength` through `Jump`, plus `Slots`) —
  confirmed by reading both functions side by side in the diff; no stat the
  rolled path assigns is missing from the explicit path.
- No `TODO`/stub/placeholder introduced. Pre-existing `TODO` above the equip
  branch is untouched.
- Test naming, Builder-pattern usage (`NewProcessor`, `testDatabase`,
  `testContext`), and package conventions match the surrounding file; no
  `*_testhelpers.go` file added.

## Not evaluable

- No dedicated test exercises the `compartment.Processor` / consumer hop
  with a non-zero `EquipStats` (see point 1). Assessed by hand-tracing
  instead; the brief did not request such a test and the hop has no
  conditional logic on the new fields.

## Findings

Non-blocking:
1. `compartment/processor.go`'s `firstEquipStats` silently drops any
   `EquipStats` argument past the first — no compiler or runtime diagnostic.
   Not exercised by any current call site; worth a comment or a switch to
   `*EquipStats` if a second caller is ever added. (`compartment/processor.go`,
   `firstEquipStats`)
2. Task 12 (not yet dispatched) should validate its orchestrator-side
   `CreateAssetCommandBody` copy against this inventory struct's **JSON key
   names**, not against `AwardCraftedAssetPayload`'s `omitempty` tags — the
   latter is irrelevant to wire compatibility since `atlas-inventory` never
   encodes this struct. Flag this explicitly in Task 12's brief so it is not
   re-litigated as a blocking issue there.

No blocking findings.

## Verdict

APPROVED_WITH_FINDINGS
