# Review — bug-gift-ack-note-and-redisplay (Defects G & H)

Reviewer: task-reviewer (Sonnet 5)
Range reviewed: `dd7e0bbb4..HEAD` — exactly `5ebad82cc` (Defect G) and
`5b9c5be4e` (Defect H).
Brief: `docs/tasks/task-240-cash-shop-stub-operations/bug-gift-ack-note-and-redisplay.md`
Reports: `bug-gift-ack-note-report.md`, `bug-gift-redisplay-report.md`

## Scope confirmed

`git diff --stat dd7e0bbb4..HEAD` matches the two commits claimed: 23 files,
915 insertions / 155 deletions, split cleanly between atlas-channel's
`note_gift_forward.go` + `note_operation.go` (Defect G, commit `5ebad82cc`)
and the `GiftAcknowledged` plumbing across atlas-cashshop and atlas-channel
plus `cash_shop_entry.go` (Defect H, commit `5b9c5be4e`). No scope drift —
the implementer reports' file lists match the actual diff.

## Defect G — gift-forward note (`5ebad82cc`)

`services/atlas-channel/atlas.com/channel/socket/handler/note_operation.go:40-48`
adds the `sp.GiftFlag() == 1` branch before the Note-item gate, delegating to
`handleNoteGiftForward` and returning — the `giftFlag == 0` tamper path below
is untouched (confirmed by diff: only re-commented).

`note_gift_forward.go:79-110` (`handleNoteGiftForward`):
- Loads the sender's own cash-shop `TypeExplorer` compartment (same call as
  `cash_shop_entry.go:75`, TODO left verbatim as instructed).
- `findGiftAsset` (line 57-64) locates the asset by `Item().CashId() ==
  GiftSN`; rejects (log + return, no announce) if absent.
- Rejects unless `GiftFrom() != "" && GiftFrom() == sp.ToName()` (line 94-97)
  — the anti-tamper gate specified by the brief, and it does **not** check
  any acknowledged/unacknowledged state, matching the "Interaction with
  Defect G" requirement exactly.
- Resolves the gifter by name, builds a **single-step** `saga.NoteSend` with
  one `CreateNote` step, `Flag: 0` (`buildGiftForwardSaga`, line 28-52) — no
  `DestroyAsset` step, no receiver-online check, matching the brief's Fix
  item 2 verbatim.
- Announces nothing on any outcome (matches: client already showed SP_2713).

PASS on Fix items 1-2. Build/tests: `go build ./...` and `go test
./socket/handler/...` in `services/atlas-channel/atlas.com/channel` are
green (verified directly, not from the report).

### Test honesty gap (non-blocking)

`note_gift_forward_test.go:100-113`, `TestFindGiftAsset_GiftFromMismatch`,
does **not** test the mismatch-rejection path the brief's Fix item 3
requires ("`giftFlag == 1` whose asset's `GiftFrom` differs from `ToName`
creates nothing"). The test body only asserts `findGiftAsset`'s returned
`giftFrom` ("ActualGifter") is not literally the string `"SomeoneElse"` — it
never calls `handleNoteGiftForward` with a mismatched `ToName`, and never
asserts no saga is created. This test passes regardless of whether the
mismatch gate at `note_gift_forward.go:94-97` exists at all (it would still
pass if that `if` block were deleted, since it never exercises it). The gate
itself is correct by direct code reading (`file:line` above), and no live or
full-dispatch test in this diff exercises "matching SN, mismatched
`GiftFrom`" end-to-end — that specific scenario is untested, only
code-reviewed. Not blocking (the production code is correct and the
mismatch predicate is a single trivial comparison), but the implementer's
report overstates this test's coverage ("pins ... the important invariant ...
is pinned by this test"); it does not.

### Anti-tamper gate soundness

The gate (own-compartment asset lookup by `GiftSN` + exact `GiftFrom ==
ToName` match) is sound against the specific tamper the brief names: a
client cannot mint a note to an arbitrary character, or claim an item it
does not hold, or claim a gift it did not receive. It has one residual gap
that is a **design property, not an implementer deviation**: nothing in this
path is a one-shot consumption — a client that emits the same
`NOTE_ACTION SEND giftFlag=1 GiftSN=X ToName=Y` packet twice (bypassing the
UI, which normally only sends once per modal OK) can mint two notes from the
same gift, because the gift-forward handler performs no state transition on
success (deliberately: it does not touch `GiftAcknowledged`, and Defect H's
flag is drained on announce, not on send). Defect H's flag prevents the
*modal* from re-firing on honest clients, but a `NOTE_ACTION SEND` packet is
never itself deduplicated or tied to a consumed marker. This is exactly the
design specified in "Interaction with Defect G" (no unacknowledged check),
so it is not a defect in this unit's implementation of the brief — it is
worth surfacing as a residual property of the merged design, not a blocking
finding here.

## Defect H — stop re-announcing acknowledged gifts (`5b9c5be4e`)

Round trip entity → model → JSON:API → channel model, traced by hand:

- `atlas-cashshop` `entity.go:54-65` adds `GiftAcknowledged bool` with
  `gorm:"not null;default:false"`; `Make()` (`entity.go` bottom) carries it
  into the domain model via `SetGiftAcknowledged`.
- `atlas-cashshop` `model.go` — field, getter, `Clone` (both `Clone(m)` and
  `ModelBuilder.Build()`) all carry `giftAcknowledged` — confirmed by full
  diff read, no drop.
- `atlas-cashshop` `rest.go` — `RestModel.GiftAcknowledged bool
  \`json:"giftAcknowledged"\``; `Transform` and `Extract` both carry it.
  `rest_test.go`'s `TestTransformExtractRoundTripGiftAcknowledged` pins the
  round trip and would fail if either direction dropped the field.
- `atlas-channel` `cashshop/inventory/asset/{model.go,builder.go,rest.go}` —
  same triad (getter, `CloneModel`, REST `Transform`/`Extract`), independently
  confirmed by diff read — no drop on the channel side either.

**Seam 1 — Kafka body byte-identity**: independently re-verified (not just
hand-checked per the report) by reading both copies side by side
(`services/atlas-cashshop/.../kafka/message/cashshop/kafka.go` and
`services/atlas-channel/.../kafka/message/cashshop/kafka.go`):
`AcknowledgeGiftsCommandBody{ AccountId uint32 \`json:"accountId"\`; CashIds
[]int64 \`json:"cashIds"\` }` — identical field names, types, and JSON tags
on both sides. PASS.

**Seam 3 — ordering / no unacknowledged dependency**: `cash_shop_entry.go`
announces `LOAD_GIFT_SUCCESS` at line 107 (`session.Announce(...)`, `return`
on error at 108-110) and only *after* that succeeds does it build `cashIds`
and call `AcknowledgeGifts` (lines 112-126) — confirmed ordering matches the
brief. Cross-checked against Defect G's handler: `handleNoteGiftForward`
validates ownership + `GiftFrom == ToName` only (see Defect G section above)
— no dependency on `GiftAcknowledged` anywhere in that file. PASS on both
halves of the interaction requirement.

**Seam 4 — `loadGiftDoneConfigured` gate**: the entire drain block (lines
112-126) sits inside `if loadGiftDoneConfigured(l, ctx) { ... }`
(`cash_shop_entry.go:105-127`). A tenant with no `LOAD_GIFT_SUCCESS` key
never reaches the announce or the drain. PASS.

**Seam 5 — empty list emits no command**: `if len(gifts) > 0 { ... }`
(`cash_shop_entry.go:118`) wraps the `AcknowledgeGifts` call; an empty
`gifts` slice never calls the processor. PASS. (No handler-level test
asserts this directly — see Test honesty note below — but
`TestAcknowledgeGiftsAndEmitEmpty` covers the same "empty is a no-op, not an
unbounded update" property one layer down, at the atlas-cashshop processor.)

### buildGiftListEntries filter

`cash_shop_entry.go:173`: `if as.GiftFrom() == "" || as.GiftAcknowledged() {
continue }` — correctly skips both non-gifts and already-presented gifts.
`TestBuildGiftListEntriesSkipsAcknowledged`
(`cash_shop_entry_test.go`) constructs one acknowledged and one
unacknowledged gift asset and asserts only the unacknowledged one survives —
this test genuinely fails without the `|| as.GiftAcknowledged()` clause
(verified by reading the diff: the clause is exactly what was added). PASS,
test is honest.

### atlas-cashshop consumer/processor wiring

`consumer.go` registers `handleCommandAcknowledgeGifts` guarded on
`c.Type != cashshop.CommandTypeAcknowledgeGifts`; delegates to
`AcknowledgeGiftsAndEmit` (`processor_gift_ack.go`), which resolves all of
the account's compartments via `compartment.Processor.GetByAccountId` and
calls `asset.Processor.AcknowledgeGifts(ccm.Id(), cashIds)` per compartment.
The per-compartment SQL update (`administrator.go`'s
`updateGiftAcknowledged`) is scoped by both `compartment_id` and `cash_id IN
(?)`, so resolving *all* the account's compartments (not just
`TypeExplorer`) is safe — a `cashId` only matches rows that actually belong
to it. `TestHandleAcknowledgeGiftsInvokesProcessor` /
`TestAcknowledgeGiftsAndEmit` both assert the named cash id flips and an
unrelated one in the same compartment does not — genuine, would fail
without the fix. `TestAcknowledgeGiftsAndEmitEmpty` /
`TestHandleAcknowledgeGiftsIgnoresOtherCommandTypes` cover the no-op and
type-guard paths. PASS.

### Migration

`asset.Migration` is bare `AutoMigrate(&Entity{})`; `GiftAcknowledged bool
\`gorm:"not null;default:false"\`` follows the exact same pattern as the
pre-existing `GiftFrom`/`GiftMessage` columns, so the "existing rows land at
false" claim is consistent with established convention in this file, not a
new risk. Not independently verified against a live migration run (no DB
diff tool invoked), but this is the established pattern for this struct, so
low risk. Recorded under Not evaluable.

## Not yet answered / concerns carried forward (informational, not this unit's scope)

- The bug file's own "Not yet answered" section (locker→locker copy paths
  not carrying `GiftAcknowledged`) is out of scope for Defect H and correctly
  left untouched.
- `tools/lint.sh` (golangci-lint, flagless) was not run to completion by
  either implementer or this review (binary not available in this
  environment). `gofmt -l` on every touched file is clean (independently
  re-verified). Recorded under Not evaluable — a golangci-lint-only finding
  cannot be ruled out.

## Not evaluable

1. Live re-verification of the migration path against a real
   Postgres/sqlite upgrade of an existing row (only inspected the gorm tag
   convention, matching pre-existing columns).
2. `tools/lint.sh` full run (golangci-lint binary unavailable in this
   sandbox).

## Verdict rationale

Both fixes satisfy every enumerated Fix item and the "Interaction with
Defect G" contract precisely: the anti-tamper gate does not gate on
acknowledgement, the drain lives inside `loadGiftDoneConfigured`, the empty
list emits nothing, the Kafka body is byte-identical, and the
`giftAcknowledged` field round-trips cleanly on both services' full
entity→model→REST→channel-model chain, verified by direct diff reading, not
by trusting the reports. Build and tests are green in both modules
(independently re-run). The one finding — `TestFindGiftAsset_GiftFromMismatch`
does not actually exercise the mismatch-rejection behavior it is named for —
is a test-honesty gap, not a production defect (the gate itself was read and
confirmed correct at `note_gift_forward.go:94-97`), so it does not block.

verdict: APPROVED_WITH_FINDINGS
