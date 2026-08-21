# Review: Task 3 — MESO_AWARDED event and Reserve wiring

Scope: commit range `47dd62e02..2ef8dd600` (single commit `2ef8dd600`), package
`services/atlas-drops/atlas.com/drops`. Reviewed against
`.superpowers/sdd/plan/task-3-brief.md` and the implementer report
`.superpowers/sdd/plan/task-3-report.md`.

## Scope confirmation

`git log --oneline 47dd62e02..2ef8dd600` shows exactly one commit. `git diff
--stat 47dd62e02..2ef8dd600` and the supplied review-package diff both list the
same four files the brief named:

- `services/atlas-drops/atlas.com/drops/drop/processor.go`
- `services/atlas-drops/atlas.com/drops/drop/processor_test.go`
- `services/atlas-drops/atlas.com/drops/drop/producer.go`
- `services/atlas-drops/atlas.com/drops/kafka/message/drop/kafka.go`

No `go.mod`/`go.sum` touched (`git diff 47dd62e02..2ef8dd600 -- '**/go.mod'
'**/go.sum'` empty). `services/atlas-drops/atlas.com/drops/drop/mock/processor.go`
has no diff, confirming the report's claim it was deliberately left alone
(accepted per task instructions — no `Processor` assertion, no callers).
Scope matches the brief exactly.

## Requirement-by-requirement

1. **`RESERVED` body unchanged** — `kafka/message/drop/kafka.go:110-116`
   (`StatusEventReservedBody`) has zero diff lines; the new
   `StatusEventMesoAwardedBody` (`kafka.go:118-125`) is added below it, not
   inside it. PASS.

2. **New status type/body** —
   `kafka.go:540` adds `StatusEventTypeMesoAwarded = "MESO_AWARDED"` to the
   existing const block; `kafka.go:121-125` adds
   `StatusEventMesoAwardedBody{CharacterId uint32 \`json:"characterId"\`;
   Amount uint32 \`json:"amount"\`; Picker bool \`json:"picker"\`}`. Tags match
   the brief's `{characterId, amount, picker}` literally — checked character
   by character, not paraphrased. PASS.

3. **Envelope** — `StatusEvent[E]` (`kafka.go:57-66`, untouched by this diff)
   carries `TransactionId, WorldId, ChannelId, MapId, Instance, DropId, Type,
   Body`. `mesoAwardedEventStatusProvider` (`drop/producer.go:496-513`)
   populates all seven envelope fields plus `Body`, matching
   `reservedEventStatusProvider`'s shape one-for-one. PASS.

4. **Keying** — `producer.go:497`: `key := producer.CreateKey(int(dropId))`,
   identical pattern to every other status provider in the file (verified by
   diffing against the untouched `consumedEventStatusProvider` just below it
   at `producer.go:515-516`). PASS.

5. **Same buffer as `RESERVED`, only on successful reservation of `meso > 0`**
   — `processor.go:163-186` (`Reserve`): on `err != nil` returns immediately
   after buffering `RESERVATION_FAILURE`, never reaching the split branch
   (`processor.go:164-167`). On success, `RESERVED` is buffered
   (`processor.go:170`), then `if d.Meso() == 0 { return d, nil }`
   (`processor.go:171-173`) skips the party lookup and award loop entirely for
   item drops. Both `RESERVED` and every `MESO_AWARDED` go into the same
   `msgBuf` parameter — no second buffer is created. PASS.

6. **Party-lookup error degrades to single full-amount award, never fails
   pickup** — `resolveMembers` (`processor.go:229-241`) swallows the error from
   `p.pp.GetByMemberId`, logs it, and returns `nil`. `splitMeso`
   (`drop/split.go:26-49`, read-only Task 2 code) with `members == nil` still
   seeds `ids := []uint32{pickerId}` and the loop over `nil` members is a
   no-op, so the recipient set collapses to exactly `{pickerId}` at full
   `meso`. `Reserve` never inspects or propagates the lookup error — the
   function signature of `resolveMembers` returns no error, so there is no
   code path that could accidentally fail the pickup on an atlas-parties
   outage. PASS. Confirmed by test
   `TestProcessor_Reserve_PartyLookupError_AwardsFullAmountToPicker`
   (`processor_test.go` diff lines 370-402), which returns
   `party.Model{}, errors.New("unreachable")` from the mock and asserts
   exactly one award `{12345, 100, true}`.

7. **Zero-share suppression applies to non-pickers only; picker's award
   emitted even at `Amount: 0`** — `processor.go:178-181`:
   `if r.Amount == 0 && !r.Picker { continue }`. The picker is never skipped
   regardless of amount. PASS. Confirmed by
   `TestProcessor_Reserve_ZeroShareSuppressesNonPickersOnly`
   (`processor_test.go` diff lines 443-480): `SetMeso(2)` with 3 co-located
   online members forces `share = 2/3 = 0` (integer division in
   `split.go:45`); the test asserts exactly one award, `{12345, 0, true}`, and
   no award for 22222/33333.

8. **Failed reservation emits no awards** — already covered by point 5's
   early return; independently confirmed by
   `TestProcessor_Reserve_FailedReservation_EmitsNoAwards`
   (`processor_test.go` diff lines 404-441), which reserves the same drop
   twice, asserts the second call errors, `awardedFrom` on that call's buffer
   is empty, and the party-mock `calls` counter is `0` — proving the party
   lookup itself never runs on a failed reservation, not just that awards
   are absent.

## Test honesty

All six tests from the brief's table are present verbatim by name and match
the table's expected values:

- `TestProcessor_Reserve_MesoDrop_SplitsAmongCoLocatedPartyMembers` — 3 members,
  `SetMeso(100)`, share `100/3=33` per recipient (remainder 1 discarded per
  `split.go`'s documented integer-division behavior), ascending-id order
  `12345,22222,33333`, exactly one `Picker: true` on `12345`. Matches table.
- `TestProcessor_Reserve_MesoDrop_ExcludesMembersNotCoLocated` — offline member
  and off-map member both excluded, single award `{12345,100,true}`. Matches.
- `TestProcessor_Reserve_ItemDrop_MakesNoPartyLookup` — `SetItem(1000000,10)`,
  asserts `calls==0`, no awards, `RESERVED` still buffered. Matches.
- `TestProcessor_Reserve_PartyLookupError_AwardsFullAmountToPicker` — covered
  above.
- `TestProcessor_Reserve_FailedReservation_EmitsNoAwards` — covered above.
- `TestProcessor_Reserve_ZeroShareSuppressesNonPickersOnly` — covered above.

Each test would fail without the change: before this diff, `WithPartyProcessor`,
`With`, and `StatusEventTypeMesoAwarded` do not exist, so the file would not
compile — a hard failure, not a pass-either-way test. Verified by reading the
pre-diff `processor.go` (no `With`/`ProcessorOption`/`pp` field) and confirming
the new symbols are introduced only in this commit.

## Correctness of the change itself

- **Immutability / value receivers / Builder** — `With` (`processor.go:187-194`)
  clones the struct value (`clone := *p`) rather than mutating the receiver,
  consistent with the existing pattern copied from
  `atlas-mounts/mount/processor.go:48-63`. `Recipient` (`split.go:12-16`,
  Task 2 code, not touched here) is a plain value struct built by `splitMeso`.
  No new domain type/alias/constant was invented outside
  `libs/atlas-constants/`; `StatusEventTypeMesoAwarded` is a local Kafka event
  string, the same category as the six sibling constants already in that
  block, not a domain type needing a shared constant.
- **No stubs/TODOs** — `grep -n "TODO\|FIXME"` against the diff returns
  nothing. `resolveMembers` and the `Reserve` split branch are both fully
  implemented, not placeholders.
- **`go build ./...`** in the module passes clean (re-run here only as a
  cheap sanity check, not a re-run of the implementer's test suite).
- **No other `Processor` implementer breaks** from the additive `With`
  interface method — `grep -rn "drop.Processor\b"` outside `_test.go` finds no
  external implementers; the only one is `ProcessorImpl`, and the build
  already confirms `var _ Processor = (*ProcessorImpl)(nil)` still holds.
- **Cosmetic-only rename** — the `Reserve` closure's parameter was renamed
  from `field` to `f` to avoid shadowing the `field` package inside the loop
  body; the outer function signature (`processor.go:161`) is untouched, so
  `ReserveAndEmit`'s call site needs no change and none was made. Confirmed
  no other call sites were touched in the diff.

## Cross-service seam (Task 4, not yet landed)

Task 4 (atlas-character) is not implemented on this branch, so there is no
consumer test to trace an emit into yet — this is expected at this stage. The
producer-side contract (topic, type string, body field names/tags, envelope
shape, keying) is exactly what the brief specifies for Task 4 to consume, and
is verified literally above rather than assumed.

## Not evaluable

- `split.go`'s own correctness (integer division, colocation matching,
  picker-always-included) is Task 2 scope, already landed and out of this
  unit's diff; it was read here only to verify Task 3's *call* into it is
  consistent with its documented contract, which it is.
- Task 4's real consumer behavior cannot be checked — it does not exist yet
  on this branch.

## Verdict rationale

Every brief requirement is implemented exactly as specified, with literal
field/tag verification rather than "looks right." All six required tests are
present, match the table's expected values, and are genuine (would fail to
compile pre-change). No scope creep, no stray files, no go.mod change, no
stub/TODO, `RESERVED` body untouched. No blocking or non-blocking findings.

Both the spec-compliance and task-quality verdicts are APPROVED.
