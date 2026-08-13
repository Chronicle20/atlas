# Packet completeness critic — task-206-cash-shop-coupon-codes

**Verdict: CLEAN on the primary claim (COUPON_CODE), 1 informational finding
on the secondary claim (CASHSHOP_OPERATION), 0 hard CHANGED-BUT-UNCLAIMED
holes, 5 pre-existing (non-regressed) CLAIMED-BUT-UNVERIFIED cells surfaced
by the manifest's coarse op×version cross-product.**

`docs/tasks/task-206-cash-shop-coupon-codes/coverage-manifest.yaml` did not
exist before this pass and was constructed here from the branch's actual git
delta (per dispatch instructions), then critiqued. Base: `1e0a321b8`.

## Step 1 — resolved scope

- **COUPON_CODE** (serverbound, `status.json` op, `packet: null` — no
  dedicated packet path recorded, resolved by op name only): the primary
  feature.
- **CASHSHOP_OPERATION** (both directions, packets
  `cash/serverbound/CashShopOperationGetPurchaseRecord` and
  `cash/clientbound/CashBuyFailed`): declared by me after the diff showed the
  branch deliberately touched this op's dispatcher templates (BUY_NORMAL fix,
  new `options.operations`/`options.errors` tables) even though it is not the
  packet the PRD is about. Backed by the branch's own ledger
  (`.superpowers/sdd/plan/progress.md` Task 28: "in scope because PRD FR-2.2
  and design §5.1 asked for them").

## CHANGED-BUT-UNCLAIMED

None found.

| kind | file/packet | evidence | recommendation |
|---|---|---|---|
| — | — | `git diff --name-only $BASE...HEAD -- 'libs/atlas-packet' \| grep '\.go$' \| grep -v _test` returns exactly two files: `cash/clientbound/shop_operation_body.go` and `cash/serverbound/coupon_code.go`. Both dirs are in `claimedPackets` (`CASHSHOP_OPERATION` clientbound, `COUPON_CODE` serverbound). | none |
| — | — | Version-gate `git diff` (`MajorVersion\|MajorAtLeast\|IsRegion\|Region()`) shows 5 hunks, all inside `coupon_code.go` (`IsRegion("JMS")`, `IsRegion("GMS") && !MajorAtLeast(92)`, `IsRegion("JMS")`, two `CreateContext` calls in the test). All attributable to the claimed COUPON_CODE codec. | none |
| — | — | Matrix delta (`status.json` cell-state diff, computed by keying rows on `(op, packet, direction)` and diffing `cells[v].state` base vs. HEAD) shows exactly 10 transitions, all `op=COUPON_CODE`: gms_v48/61/72/79 `n-a→verified`, gms_v83/84/87/92/95/jms_v185 `incomplete→verified`. Zero `CASHSHOP_OPERATION` (or any other op) cell moved. | none |

The `shop_operation_body.go` constant-value correction
(`INVALID_COUPON_COUPON` → `INVALID_COUPON_CODE`, single line) is inside the
`CASHSHOP_OPERATION` clientbound codec dir and is covered by the manifest's
`fields` note; it is label-only (no wire-byte position/width change) per
progress.md Task 5.

The 103-line hex→int template normalization (Task 7's documented side
effect) and the `cashShop.coupons.rateLimit` addition (all 11 templates
incl. `gms_12`) are non-packet / representation-only and are listed in
`out_of_scope` — correctly, since neither is a coverage claim.

## CLAIMED-BUT-UNVERIFIED

| op | version | actual state | recommendation |
|---|---|---|---|
| COUPON_CODE | all 10 | verified | none — matches claim exactly |
| CASHSHOP_OPERATION (serverbound) | gms_v48 | incomplete | pre-existing, untouched by this branch's diff (no cell transition in status.json). Drop from manifest scope, or explicitly annotate as "template-only touch, matrix claim not made" — the current manifest's flat `versions:` cross-product over-claims here. |
| CASHSHOP_OPERATION (serverbound) | gms_v61 | incomplete | same as above — pre-existing, not regressed, not claimed to be fixed by this branch. |
| CASHSHOP_OPERATION (serverbound) | gms_v92 | incomplete | same. |
| CASHSHOP_OPERATION (clientbound) | gms_v48 | n-a | correctly n-a (legitimately inapplicable), not a gap. |
| CASHSHOP_OPERATION (clientbound) | gms_v92 | incomplete | pre-existing, not regressed. |

These five are an artifact of the coverage-manifest schema's flat
`ops × versions` cross-product: the branch genuinely and deliberately edited
`CASHSHOP_OPERATION`'s **templates** (dispatcher mode/error tables) on
specific versions, but never touched its **codec** or **matrix cells** —
`git diff` on `status.json` confirms zero `CASHSHOP_OPERATION` transitions.
Declaring the op at all (to avoid a CHANGED-BUT-UNCLAIMED hole on the
template edits) necessarily also claims full-matrix coverage under this
schema, which the branch never intended or delivered for these five
pre-existing cells. This is a **schema-shape finding**, not a branch defect:
none of these five states regressed — all five were already exactly this
state before the branch (confirmed via the base `status.json`, no diff
touches them). Recommend either (a) the task author adds a manifest comment
disclaiming full CASHSHOP_OPERATION coverage explicitly, or (b) a future
PROCESS.md revision lets `ops` entries carry per-op version subsets instead
of one global list, so a "template-only, partial-op" claim like this one
doesn't have to over-claim or go undeclared.

## Known/accepted limits — confirmed recorded, not re-litigated

- jms_v185 `errors` PARTIAL (30/53) — recorded in `derivation.md` L1036-1053
  and in Task 8's ledger entry; all ten coupon-specific keys are inside the
  proven [178,211] range. Reflected in the manifest `fields`.
- gms_v84 mode 72 / gms_v87 mode 74 / gms_v92 mode 75 / gms_v95 mode 76:
  real, unnamed client sends, deliberately omitted rather than guessed
  (progress.md Task 9, `cash_shop_operation_handle.yaml` header comment).
  Reflected in the manifest `fields`.
- jms_v185's extra unconditional `Encode1(nType)` byte: position/width/origin
  proven, semantics unverified, modelled but never forwarded — reflected in
  the manifest `fields`.
- Most `errors` KEY NAMES are ordinal alignment inference, not decompiled
  text (progress.md Task 8 "OPEN QUESTION ONLY THE LIVE TEST CAN CLOSE") —
  this is a live-verification gap for Task 30 Step 7, not a completeness-
  manifest gap; not re-litigated here.

## Files touched by this pass

- Created: `docs/tasks/task-206-cash-shop-coupon-codes/coverage-manifest.yaml`
- Created: `docs/tasks/task-206-cash-shop-coupon-codes/completeness-critic.md`
- No codec, registry, template, `gates.yaml`, evidence record, or
  `status.json` was modified.
