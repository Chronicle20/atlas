# Review: Task 11 — feed the four encoder sites from the ring processor

Range: `bbb3ad91d..9f65d8ac0` (`678741c15` formatting-only, `9f65d8ac0` Task 11 proper).
Brief: `.superpowers/sdd/plan/task-11-brief.md`. Report: `.superpowers/sdd/plan/task-11-report.md`.

## Scope confirmed

Ten files: `ring/processor.go`, `ring/processor_test.go`,
`socket/writer/character_spawn.go` (+test, new), `character_info.go` (+test),
`character_data.go` (+test), `kafka/consumer/asset/consumer.go` (+test, new).
Matches the brief's four named call sites plus the necessary `ring.Processor`
extension (`GetRingRecords`) and its tests. No out-of-scope files touched.

## Priority 1 — `git checkout` near-miss on `consumer.go`

Read the committed `kafka/consumer/asset/consumer.go` in full
(`consumer.go:415-433`). The ring wiring is present and coherent:

- `var ringProcessorFn = ring.NewProcessor` (`consumer.go:417`) — test seam.
- `updateAppearance` resolves `rings := ringProcessorFn(l, ctx).GetRingSet(c.Id(), c.Equipment())` at `consumer.go:428`, inside `func(c character.Model) model.Operator[session.Model]` but **outside** the returned `func(s session.Model) error`.
- `charcb.NewCharacterAppearanceUpdate(c.Id(), ava, rings)` (`consumer.go:429`) consumes it.
- `packetmodel` import was correctly dropped (no longer needed after the literal `packetmodel.RingSet{}` was replaced).

Ran `go build ./...` and `go test ./kafka/consumer/asset/... -run TestUpdateAppearanceResolvesRingsOnce -v` against the committed tree: **PASS**.

Fault-injected the exact regression the guard claims to catch: moved the
`GetRingSet` call inside the per-session closure (`return func(s session.Model) error { rings := ...; ... }`), rebuilt, reran the same test:

```
consumer_ring_test.go:82: GetRingSet called 3 times across 3 recipient sessions, want 1
--- FAIL: TestUpdateAppearanceResolvesRingsOnce (0.00s)
```

Restored the file from a pre-edit backup; `git status --porcelain` confirmed
clean afterward. **The recovery was complete — nothing was silently lost, and
the guard test is a real regression pin, not a tautology.**

## Priority 2 — PRD §8 hot-path placement

Confirmed at `consumer.go:419-433`: `GetRingSet` is called once per
invocation of `updateAppearance(l)(ctx)(wp)(c)` (i.e., once per broadcasting
character `c`), and the *returned* `model.Operator[session.Model]` — the
thing `ForSessionsInMap` (`consumer.go:381`) invokes once per recipient
session — captures `rings` in its closure rather than recomputing it. The
`TestUpdateAppearanceResolvesRingsOnce` double calls the returned operator
three times against three sessions and asserts exactly one `GetRingSet` call;
verified PASS and fault-injection FAIL above. **Correctly placed outside the
per-recipient hot path.**

## Priority 3 — `GetRingRecords`, the implementer-added method

- Added to the `Processor` interface (`ring/processor.go:62`) and
  `ProcessorImpl` (`ring/processor.go:124-154`).
- Cache-only, confirmed by reading the body: `getRingCache().lookup(...)`,
  debug-log + empty return on miss, no call to `upstreamFn`/`requests.*`.
  Same contract as `GetRingSet` — `Populate` remains the only I/O method in
  the package (unchanged).
- Tested: `TestGetRingRecords` (`ring/processor_test.go`, two subtests) —
  cache-miss returns empty; ACTIVE halves listed regardless of equipped
  state, BROKEN excluded, Couple/Friend correctly split by `ring.Type`,
  `FriendItemId` populated from `ItemTemplateId()`, `Marriage` always empty.
  `go test ./ring/...` — PASS.
- Judgment: the addition is warranted. `character_data.go` needs
  `partnerName`/`partnerCharacterId` (the record block), which `GetRingSet`
  cannot supply (it only returns `PairRing{OwnSN, PartnerSN, ItemId}` — no
  partner identity). No pre-existing exported accessor covered this. Putting
  it on the `Processor` interface (rather than a private helper) is
  consistent with `GetRingSet`'s own shape and is the only way
  `character_data.go` can reach it without depth-reaching into `ring`
  package internals (`halves` is unexported). This is judged in-scope, not
  scope creep, given brief Step 3 explicitly requires
  `ring.Model.PartnerName()` at that site.

## Priority 4 — block confusion at the four call sites

Verified by reading each site directly:

| Site | Accessor | Block | Correct? |
|---|---|---|---|
| `character_spawn.go:60` | `ring.NewProcessor(l, ctx).GetRingSet(c.Id(), c.Equipment())` | avatar (`RingSet`/`PairRing`) | Yes |
| `character_info.go:54` | `ring.NewProcessor(l, ctx).GetRingSet(...)`, uses `rings.Marriage != nil` | avatar | Yes |
| `character_data.go` (after teleport-rock block) | `ring.NewProcessor(l, ctx).GetRingRecords(c.Id())` → `cd.Rings` | record (`RingRecords`/`CoupleRecord`/`FriendRecord`) | Yes |
| `consumer.go:428` (`updateAppearance`) | `ringProcessorFn(l, ctx).GetRingSet(c.Id(), c.Equipment())` | avatar | Yes |

No swap found. `GetRingRecords` populates `CoupleRecord{PairCharacterId,
PairCharacterName, OwnSN, PairSN}` — the record block's own field names
(`libs/atlas-packet/model/ring.go:164-169`), not the `GroomId`/`BrideId`
naming, which belongs only to `MarriageRecord` (a block this task correctly
never constructs, since `Marriage` stays empty per the standing ruling).

## Standing rulings — spot confirmations

- FR-15 slot order: `ring/processor.go:172` — `s.Position > bestPosition`
  selects the numerically higher (less negative) sub-slot position first
  (ring1=-12 beats ring2=-13, etc.), tie broken by `h.CashId() < best.CashId()`
  — matches ring1→ring2→ring3→ring4, lower-cashId tiebreak.
- `Marriage` stays nil/empty everywhere in this diff: `GetRingSet` and
  `GetRingRecords` both leave it unset; no call site constructs a
  `MarriageRing`/`MarriageRecord`.
- Ruling 5 (marriage avatar arm's first field is the encoded character's own
  id): unaffected by this task — no site here constructs a `MarriageRing`.
- 20-byte delta (not 18): `character_spawn_test.go:90-97` asserts
  `wantDelta := 20` with a comment tracing the exact codec accounting
  (`byte(1)+int64+int64+uint32 = 21` replacing `byte(0) = 1`). Confirmed by
  reading `libs/atlas-packet/model/ring.go:69-84` (`EncodeField`/`encodePair`)
  — the accounting is correct.

## Test honesty

- `TestUpdateAppearanceResolvesRingsOnce` — verified above via fault
  injection: a **real** regression guard, not a tautology.
- `TestCharacterSpawnBodyCarriesRings/ring_present` — seeds the cache via a
  real `httptest.Server` + `Populate`, asserts a length delta computed from
  first principles (20, not compared against the encoder's own output re-run)
  against two same-equipment characters differing only in cache population.
  Not tautological.
- `TestCharacterSpawnBodyCarriesRings/no_ring` — only asserts determinism of
  two identical encodes; it does not compare against a captured Task-7
  golden byte string, so it can't by itself detect a regression in the
  empty-ring path (a weaker but non-blocking gap — the "ring present" subtest
  and Task 7's own tests carry most of that weight).
- `TestCharacterInfoBodySetsMarriageFlag` — asserts `HasMarriageRing() ==
  false` in the one reachable state (`Marriage` is always nil today). Cannot
  by construction distinguish correct wiring (`rings.Marriage != nil`) from
  Task 7's old hardcoded `false`, since both produce identical output while
  `Marriage` is never non-nil. The report names this limitation itself; it is
  inherent to the standing "Marriage stays nil" ruling, not a fixable defect
  in this task.
- `TestGetRingRecords` — builds its `want` `CoupleRecord` from the same
  fixture's own accessors (`coupleActive.PartnerCharacterId()` etc.), which
  is a reasonable pattern for a straight field-mapping function; it does
  independently pin non-trivial structural behavior (BROKEN exclusion,
  Couple/Friend split, Marriage always empty).

## Non-blocking findings

1. **No writer-level test exercises `character_data.go`'s `cd.Rings` wiring
   directly.** (`socket/writer/character_data.go`, the `cd.Rings =
   ring.NewProcessor(l, ctx).GetRingRecords(c.Id())` line.) The only change to
   `character_data_test.go` in this diff is switching `context.Background()`
   → `pt.CreateContext(...)` in the two pre-existing tests to stop a
   `tenant.MustFromContext` panic — neither test seeds the ring cache or
   asserts anything about `cd.Rings`'s contents. Coverage of the record-block
   mapping itself lives entirely in `ring/processor_test.go`'s
   `TestGetRingRecords`, which tests `GetRingRecords` in isolation, not the
   `character_data.go` call site. A future edit that swapped
   `GetRingRecords()` for `GetRingSet()`-shaped data (or dropped the line
   entirely) at this specific call site would build and pass every existing
   test — the priority-4 "well-formed frame carrying scrambled data" failure
   mode the task explicitly asked to guard against is caught by manual
   reading here, not by a test. Given the brief's own Step 1 test list did
   not name a `character_data.go` test, this was a legitimate area for the
   brief to under-specify, but it is a real, currently-uncovered gap.
2. `TestCharacterSpawnBodyCarriesRings/no_ring` only proves determinism
   between two live encodes, not equality with the actual Task-7 baseline
   byte string (see Test honesty above). Low severity given the codec/logic
   is otherwise directly exercised.

## Not evaluable

None — the full review surface (four call sites, the new `Processor` method,
and the one file with the near-miss) was read and, where relevant,
fault-injected.

## Verdict rationale

No blocking defects found. Priority 1's near-miss resolved cleanly and is
verified by both static reading and fault injection. All four call sites wire
the correct block. `GetRingRecords` is a justified, correctly-scoped,
cache-only addition with its own tests. The one real gap — missing
`character_data.go`-site coverage for the record-block wiring — is a test
completeness finding, not a functional defect (the code itself was read and
confirmed correct), so it is non-blocking.
