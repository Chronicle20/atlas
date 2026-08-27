# Review — Task 23: ENABLE_EQUIP_SLOT (mode 9/10) end to end

Range reviewed: `7b887ee0e..8cfbc1200` (4 commits, both implementer halves as one unit).

- `202a574d2` style(atlas-packet): gofumpt fix
- `460875a7a` feat(cashshop,character): purchase transaction + POST write route
- `4c5e6bdff` feat(channel): outbound request forwarding
- `8cfbc1200` feat(channel): announce EQUIP_SLOT_INCREASED and propagate the slot expiry

Scope confirmed: the diff matches the two briefs (`task-23-brief.md` + `task-23-brief-cont.md`)
and the report's own accounting. No surprise files. `git diff --stat` matches the
27-file, 981/21 shape reported.

## 1. R1 — the wire-vs-canonical seam

Traced `SlotIndex` end to end:

- `EquipSlotIncreasedBody.SlotIndex` is `int16` in both atlas-cashshop
  (`kafka/message/cashshop/kafka.go:393`) and its atlas-channel mirror
  (`kafka/message/cashshop/kafka.go:405`) — never `uint16`, so no silent
  widening conversion is even representable at that type.
- Producer side (atlas-cashshop `cashshop/equipslot.go:51,124`): `slotIndex :=
  int16(pendant2.Position)` (−59) is carried through `ExtendEquipSlot` (the
  persistence call) and into `EquipSlotIncreasedStatusEventProvider` — both
  correctly get −59. Test asserts this literally:
  `equipslot_test.go:121` `require.Equal(t, int16(-59), evs[0].Body.SlotIndex, ...)`.
- Consumer side (atlas-channel `kafka/consumer/cashshop/consumer.go:544`):
  `cashpkt.CashShopEnableEquipSlotExtSuccessBody(0, e.Body.Days)` — the
  literal `0`, never `e.Body.SlotIndex`. Confirmed by grep: `e.Body.SlotIndex`
  appears nowhere as an argument to a packet-body builder anywhere in the
  diff.
- No `int16`→`uint16` cast of a `SlotIndex`-shaped value exists anywhere in
  the diff (grepped `SlotIndex` across both changed modules — every read site
  either stays `int16` or is asserted/commented as the canonical value).

**Byte test does not actually cover the consumer.go call site (blocking).**
`socket/handler/cash_shop_equip_slot_test.go`'s
`TestCashShopEnableEquipSlotExtSuccessBodyEncodes` calls
`cashcb.CashShopEnableEquipSlotExtSuccessBody(0, 30)` directly — it never
invokes `handleStatusEventEquipSlotIncreased` or feeds a `StatusEvent` with
`SlotIndex: -59` through the consumer. It pins the *encoder's* byte layout for
a slot index of 0, which is necessary but not sufficient. If a future edit
changed `consumer.go:544` to pass `e.Body.SlotIndex` instead of the literal
`0`, this test would keep passing unchanged — it does not touch that line at
all. The report's own Step 2 write-up in `task-23-report.md` claims "this test
directly covers a regression that starts forwarding `e.Body.SlotIndex` ...
instead" and the in-file doc comment repeats the same claim
(`cash_shop_equip_slot_test.go:14-20`) — both overstate what the test proves.
There is no test anywhere in `kafka/consumer/cashshop/*_test.go` that
constructs a `cashshop2.StatusEvent[cashshop2.EquipSlotIncreasedBody]` and
drives it through `handleStatusEventEquipSlotIncreased` (confirmed:
`grep -n "EquipSlot" kafka/consumer/cashshop/*_test.go` returns nothing).
The production code at `consumer.go:544` is correct today, and R1's
highest-risk hazard is currently *avoided*, but the regression guard the plan
asked for at that specific call site does not exist. A future edit could
reintroduce the −59→65477 bug and every test in this diff would stay green.

## 2. R4 — the FILETIME conversion

- `packetmodel.MsTime` (`libs/atlas-packet/model/ms_time.go:9-14`):
  `t.Unix()*10000000 + 116444736000000000` (zero time → `-1`). Confirmed by
  direct read, not by trusting the report.
- It is genuinely an established production path: used at
  `character_data.go:73` (`Skills[].Expiration`) and `:97`
  (`CompletedQuests[].CompletedAt`) inside this same file, in addition to the
  other four sites the report names (not independently re-verified beyond
  this file, but this file's own two prior uses are sufficient corroboration
  that it is the codec's real convention, not an invented one for this task).
- Hand-checked the fixture: `1704067200 * 10_000_000 = 17,040,672,000,000,000`;
  `+ 116,444,736,000,000,000 = 133,485,408,000,000,000`. Matches
  `character_data_test.go:79` exactly. Arithmetic is correct.
- No-active-extension case: `equipSlotExtExpireFor(nil)` and `([]RestModel{})`
  both return `ZeroTime` (`character_data.go:189-190`), asserted by
  `TestEquipSlotExtExpireFor_NoActiveExtension`. `ZeroTime = 94354848000000000`
  read at `set_field.go:45` (unchanged, read-only) — the sentinel is preserved
  correctly, not replaced with a zero. `MsTime`'s own `-1` sentinel for a zero
  `time.Time` is correctly *not* used here — the code takes an explicit
  separate branch, exactly as R4 required.

Confirmed independently: R4 discharged correctly.

## 3. The new atlas-channel REST client (`character/equipslot/`)

- Shape mirrors `pendingchange`/`teleportrock` conventions: `rest.go` (RestModel
  + `GetName`/`GetID`/`SetID` + `Transform`), `requests.go` (`Resource` const,
  `getBaseRequest`, `requestActiveByCharacterId` via `requests.GetRequest`),
  `processor.go` (`Processor` interface + `ProcessorImpl` + `NewProcessor` +
  `requests.SliceProvider`) — consistent with `teleportrock/processor.go` and
  `pendingchange`'s file split.
- Tenant scoping: `requests.GetRequest[A any](url string)` (`libs/atlas-rest/requests/decorated.go:9-15`)
  unconditionally attaches `TenantHeaderDecorator(ctx)`, `SpanHeaderDecorator(ctx)`,
  and `EnvHeaderDecorator(ctx)` regardless of whether the URL itself was built
  via `RootUrlFor(ctx, ...)` or the ctx-less `RootUrl(...)` — so the client is
  correctly tenant-scoped via headers even though it builds its base URL with
  `RootUrlFor` (equipslot) rather than the plain `RootUrl` (teleportrock/
  pendingchange use); both forms end up tenant-scoped through the shared
  request pipeline. No defect here.
- One inconsistency worth naming (non-blocking): `equipslot/requests.go` uses
  `requests.RootUrlFor(ctx, "CHARACTERS")` (can return an error) while
  `teleportrock`/`pendingchange` use the ctx-less `requests.RootUrl("CHARACTERS")`.
  Both compile and both work; this is a style variance, not a defect — flagging
  only because the report describes this client as "mirroring" those two, and
  the base-URL resolution is the one place it does not.

## 4. Fail-open behavior on the equip-slot fetch (character_data.go)

Verified the concern is real, not hypothetical, by running the flagged tests:

```
=== RUN   TestBuildCharacterData_MonsterBook
level=warning msg="Failed calling [GET] on [characters/99/equip-slot-extensions], will retry."  (x3)
level=error   msg="Unable to successfully call [GET] ..." error="after 3 attempts..."
level=warning msg="Unable to fetch equip-slot extensions for character [99]; sending ZeroTime."
--- PASS: TestBuildCharacterData_MonsterBook (0.45s)
--- PASS: TestBuildCharacterData_TeleportMaps (0.48s)
```

Judged on the merits, not the precedent: `EquipSlotExtExpire` is read by the
client at `CharacterData::Decode` time (channel change / login,
derivation-equip-slot.md §2.1) and directly gates whether the client's
`IsEquipSlotExpired`/`WearEquipItem`/`get_real_equip` treat the paid pendant2
slot as active for that session load (derivation §1.4). `ZeroTime`
(1900-01-01) is unambiguously "expired." So a transient failure of the new
equip-slot GET on a channel change **does** cause the client to momentarily
treat an active, paid-for extension as expired — precisely the risk named in
the task brief — until the next successful `SET_FIELD` re-fetch (next map/
channel change) corrects it. This is not "silently dropped forever": nothing
persists the wrong state server-side (`Extend`'s stored row is untouched), so
it self-heals on the next successful fetch. But it is a real, if transient,
degradation of a feature the player paid real currency for, and it is worse
in kind than the `teleportrock` precedent it cites: an empty saved-map list on
a transient failure is a minor UX gap, whereas a false "expired" reading on a
purchased item is closer to a paid-feature regression, however brief.
Fail-open is still the right call versus fail-closed (blocking `SET_FIELD`
entirely would be strictly worse — the player couldn't load into the game at
all), and there is no cached prior value to fall back to in this design. I
am not treating this as blocking because: (a) the alternative (block set-field)
is worse, (b) the window is one bad fetch's worth of one session-load, self-
healing on the next, and (c) this is consistent with an existing, reviewed
precedent in the same file. But it is a genuine architectural gap — retries
inside `requests.SliceProvider` (3x) are the only mitigation, and there is no
option today to prefer a cached "last known good" state over a hard
ZeroTime on transient failure. Flagging as non-blocking for the controller's
awareness, not accepting the precedent citation uncritically as the brief
asked.

## 5. Purchase-transaction correctness (atlas-cashshop)

`cashshop/equipslot.go` — `PurchaseEquipSlotAndEmit`:

- **Replay/idempotency**: `ledger.Claim` is called first, before any read or
  write; `ErrAlreadyProcessed` short-circuits to `return nil` with no re-emit,
  no re-charge (`equipslot.go:63-70`). Test `"replay charges once"` calls the
  method twice with the same `transactionId` and asserts `Credit() == 1000`
  (`equipslot_test.go:171-179`) — a real, non-vacuous assertion.
- **Insufficient funds**: `balance < ci.Price()` → `reject("NOT_ENOUGH_CASH")`,
  no debit call is reached (`equipslot.go:96-99`). Test seeds credit 100,
  asserts credit unchanged and an `ERROR` event with
  `Operation == "ENABLE_EQUIP_SLOT"` and `Error == "NOT_ENOUGH_CASH"`
  (`equipslot_test.go:139-152`).
- **Unknown commodity**: `p.comP.GetById(serialNumber)` failing rejects with
  `"UNKNOWN_ERROR"` before any wallet read (`equipslot.go:76-80`). Test uses
  serial `99999`, asserts credit unchanged and `Operation ==
  "ENABLE_EQUIP_SLOT"` on the error event (`equipslot_test.go:161-168`).
- **Success**: asserts `Credit() == 1000` (5000−4000), `Days == 30`, and
  `SlotIndex == -59` "must be the pendant2 canonical position, never the wire
  value" (`equipslot_test.go:118-121`) — the R1 assertion lives exactly where
  it should.
- All four subtests use real HTTP fixtures and a real in-memory tenant DB
  (`databasetest.NewInMemoryTenantDB`), not mocks that assert nothing. No
  vacuous test found in this file.

One design note, non-blocking: `ExtendEquipSlot` (the HTTP call to
atlas-character) happens *inside* the same DB transaction as the wallet debit
and ledger claim (`equipslot.go:109-116`, before `purchaserecord.Record` and
the outbox `buf.Put`). If that HTTP call succeeds but a later step in the same
closure fails (e.g. `purchaserecord.Record`), the whole `ExecuteTransaction`
rolls back — including the ledger claim — while the external side effect on
atlas-character (the slot was already extended) is not undone. A retry of the
same purchase (a fresh transactionId, or the ledger having been rolled back so
the same one is claimable again) could then extend the slot a second time
without an additional charge having landed. This mirrors an existing pattern
in `ring.go` (the asset-creation call happens in-transaction too, per the
report's own comparison), so it is not new to this task and not something
Task 23 introduced — flagged for awareness only, not attributed to this diff.

## Not evaluable

None. Every priority-check item in scope was traced to a concrete file:line
and either confirmed or found wanting within this unit's diff surface.

## Verdict rationale

The functional behavior is correct: R1's hazard is avoided at the one place
it matters, R4's conversion is genuinely derived and arithmetically verified,
the REST client follows convention and is tenant-scoped, and the purchase
transaction's four required behaviors are all real and asserted. The blocking
finding is a test-coverage gap, not a functional defect: the byte-level test
the brief specifically asked to "make ... the thing that would catch a
regression of Step 1's hazard" does not, in fact, exercise the call site where
that hazard lives, and the report's self-review claims coverage it does not
have. That gap is exactly the kind of seam a green build cannot see and this
review exists to catch.
