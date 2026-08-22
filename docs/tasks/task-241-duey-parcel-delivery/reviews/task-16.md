# Review: Task 16 — atlas-channel Duey fee table and send validation

**Range:** `f1fd89fb9..66125ac` (single commit `66125ac`)
**Brief:** `.superpowers/sdd/plan/task-16-brief.md`
**Report:** `.superpowers/sdd/plan/task-16-report.md`
**Scope:** `services/atlas-channel/atlas.com/channel/parcel/{fee,validation}.go` and their
`_test.go` files — new package, no I/O, pure functions only.

## Scope confirmation

The diff matches the brief exactly: two new production files (`fee.go`, `validation.go`),
two new test files, nothing else touched (`git diff --stat` confirms `344 insertions(+)`,
0 deletions, 4 files, all under `parcel/`). No handler, no Kafka, no REST wiring — correctly
deferred to Task 17.

## 1. Fee-table values trace to source

Compared every row of `Fee`'s descending tier switch (`fee.go:36-54`) against design.md
§6.3's formula line by line — rates (0.008, 0.018, 0.03, 0.04, 0.05, 0.06) and thresholds
(100,000 / 1,000,000 / 5,000,000 / 10,000,000 / 25,000,000 / 100,000,000) match exactly,
same descending comparison order, same `uint32(float64(m) * rate)` truncation form the
design mandates in place of NFR-8's integer formula (with the float-vs-integer rationale
recorded in a comment citing the same IDA addresses the design cites).

Independently recomputed all 13 `TestFee` table rows in Python against the design formula
(not against the Go code) — all 13 match exactly, including every tier boundary
(99,999→0, 100,000→800, 999,999→7,999, 1,000,000→18,000, etc. through 100,000,000→6,000,000).
**PASS.**

## 2. Tier boundary semantics

Each `case mesoAmount >= N` is a `>=` (inclusive at the lower edge), matching design §6.3's
`m >= N` conditions exactly, and the descending switch means the *first* (highest)
matching tier wins — same semantics as the design's ordered list. `TestFee` pins both the
value just below each boundary (e.g. 99,999, 999,999, 4,999,999 ...) and the value at each
boundary (100,000, 1,000,000, 5,000,000, ...) for all seven tiers. **PASS.**

## 3. `TotalCost` overflow contract

`fee.go:59-65`: `total := uint64(mesoAmount) + uint64(Fee(mesoAmount))` — both operands are
widened to `uint64` *before* the addition, so there is no `uint32`-domain wraparound prior
to the widening (the failure mode the task called out). The surcharge is also added as
`uint64(SendSurcharge)`. The returned bool is `total <= math.MaxUint32`.

Verified the three added boundary rows in `TestTotalCost` (`fee_test.go:46-48`) independently
in Python: 4,051,855,938→4,294,967,294 (`ok=true`, one below `MaxUint32`),
4,051,855,939→4,294,967,295 (`ok=true`, exactly at `MaxUint32`),
4,051,855,940→4,294,967,296 (`ok=false`, one above). All three match; all three would fail
if `total <= math.MaxUint32` were changed to `<` or if either summand were left in `uint32`
before widening (a `uint32+uint32` addition near the `uint32` ceiling would wrap silently and
produce a *smaller*, still-`ok=true`-passing value — the test would then observe `ok=true` with
a garbage `total`, which would fail the `total != tt.expected` assertion). **PASS.**

## 4. Constants reuse

Grepped `libs/atlas-constants/` for `meso|parcel|mailbox|message.*length|surcharge` — no
parcel-domain constants exist there (only unrelated skill/stat names containing "Meso").
Confirmed independently.

Also swept `services/` for the same constant names and nearby magic numbers
(`MesoLimit*`, `MaxMessageLength`, `MailboxCapacity`, `SendSurcharge`, `MaxParcelMeso`,
`100_000_000`). One structurally similar table exists —
`services/atlas-trades/atlas.com/trades/configuration/model.go:107-114` — a per-tenant,
REST-configurable trade-tax tier table using the **same six rates and thresholds**
(0.008/0.018/0.03/0.04/0.05/0.06 at the same breakpoints). This is not a `libs/atlas-constants`
equivalent (it's per-tenant configuration state, not a constant, and belongs to a different
domain — trade tax vs. Duey delivery fee), and the design explicitly specifies the Duey fee as
a fixed client-matching float formula rather than tenant-configurable, so this is not a scope
violation. Noting it as a non-blocking observation: the two services independently encode the
same in-game tax table, so a future consolidation opportunity exists, but nothing in Task 16's
brief or design calls for sharing it, and hard-coding matches the design's stated rationale
(matching the client's fixed double-precision quote). Other `MaxMessageLength` hits in
`atlas-kites`/`atlas-tenants` are unrelated per-service config knobs with different semantics
(chat/kite message caps, not Duey messages) — not a reuse candidate. **PASS** with a
non-blocking note.

## 5. `ValidateSend` completeness

All four `RejectReason` constants are reachable: `RejectIncorrectRequest` (three sites —
`validation.go:33`, `:36`, `:41`, `:51`), `RejectMesoLimit` (`:45`), `RejectNotEnoughMesos`
(`:55`), `RejectNone` (`:58`, the fallthrough). No unreachable reason.

Every design §6.2 check that the brief scopes to this package is present, in the design's
order: nothing-attached → cap/overflow → meso limit → message length → affordability. The
five checks the design defers to remote lookups (name resolution, same-account, mailbox
capacity, ticket check) are correctly absent, and the brief/report both name them as Task 17's
responsibility.

One structural note, non-blocking: the overflow branch (`validation.go:40-42`,
`if !ok { return RejectIncorrectRequest }`) is unreachable in practice given the current
constants. `MaxParcelMeso` (100,000,000) is checked immediately before it, and the maximum
possible `TotalCost` at that cap (100,000,000 + 6,000,000 fee + 5,000 surcharge = 106,005,000)
is nowhere near `math.MaxUint32` (4,294,967,295), so `ok` can never be `false` once the cap
check has passed. This matches design §6.2's table, which lists "overflows, or exceeds the
cap" as a single OR'd row mapping to one result arm — the implementation is not wrong, and
the report accurately discloses this ("the cap check happens to subsume the literal overflow
test case ... but the `ok`-from-`TotalCost` check is still present and exercised directly by
boundary cases in `TestTotalCost`"). Flagging only because the `validation_test.go` "overflow"
subtest (`:76-82`) does not actually exercise `validation.go:40-42` — it is caught by the cap
check at `:35-37` instead — so that specific line in `ValidateSend` currently has no direct
test coverage of its own (only indirect coverage via `TestTotalCost`, a different function).
**PASS** with a non-blocking coverage note.

## 6. Test honesty — reject-reason specificity

All 15 `TestValidateSend` subtests assert `got != <specific RejectReason constant>`, never a
generic "did it reject" check — e.g. `low level over limit` asserts `!= RejectMesoLimit`
specifically (not just `!= RejectNone`), `cannot afford` asserts `!= RejectNotEnoughMesos`
specifically. Two rules could not silently swap reasons and still pass. **PASS.**

Spot-verified the check-order-sensitive cases by hand:
- `at the parcel cap` (100,000,000 meso, quick=false, sender 4,294,967,295): total =
  100,000,000 + 6,000,000 + 5,000 = 106,005,000, affordable → `RejectNone`. Confirmed.
- `can afford exactly` (1,000,000 meso, sender 1,023,000): total = 1,000,000 + 18,000 + 5,000
  = 1,023,000 exactly; `uint64(SenderMeso) < total` is `1,023,000 < 1,023,000` = false →
  `RejectNone`, confirming the affordability boundary is `<`, not `<=`. Confirmed.
- `low level at limit` (level 15, meso 1,000,000): `MesoAmount > MesoLimitAmount` is
  `1,000,000 > 1,000,000` = false → meso-limit check does not fire, falls through to
  affordability with a large sender balance → `RejectNone`. Confirmed the `>` (not `>=`)
  matches design §6.4 ("amount **exceeds** 1,000,000").

All computed and cross-checked against the Go source and design text; no discrepancies found.

## Not evaluable

None — the entire unit is in-scope pure functions with no I/O and no cross-service seam;
nothing outside the reviewed surface was needed to evaluate it.

## Verdict rationale

Every fee value, every tier boundary, the overflow contract, the constants-reuse claim, and
`ValidateSend`'s check order and reject-reason mapping all trace cleanly to design.md §6.2–§6.4
and independently recomputed arithmetic. Tests assert specific reject reasons, not just
pass/fail, and the three `TotalCost` overflow-boundary rows would genuinely fail if the
`uint64` widening were removed. Two non-blocking notes recorded above (a duplicate tax-table
observation, and one branch in `ValidateSend` with only indirect test coverage) — neither is a
defect.
