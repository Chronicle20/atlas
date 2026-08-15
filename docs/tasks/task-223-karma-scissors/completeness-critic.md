# Packet completeness critic — task-223-karma-scissors

**Verdict: CLEAN — 0 findings.**

Branch: `task-223-karma-scissors`, diff base `c688f2c792b784c0128f1afa8be78c33a6c45915..HEAD` (22 commits).
Manifest: `docs/tasks/task-223-karma-scissors/coverage-manifest.yaml` @ `ff95c3a62`.

## Step 1 — manifest resolution

`ops` resolves to a single status.json row: `op: USE_CASH_ITEM`, `packet:
cash/serverbound/CashItemUseMegaphone`, `direction: serverbound` (both manifest
entries name the same row). `versions` claims all 10 matrix columns.
`out_of_scope`: `model/asset`, `inventory/tradeAvailable`.

## Step 2 — CHANGED-BUT-UNCLAIMED

**Touched codecs.** Only one non-test `.go` file under `libs/atlas-packet`
changed in the whole branch:

```
$ git diff --name-only $BASE...HEAD -- 'libs/atlas-packet' | grep '\.go$' | grep -v '_test\.go$'
libs/atlas-packet/cash/serverbound/item_use_karma_scissors.go
```

This is the new `ItemUseKarmaScissors` struct in `cash/serverbound/`, the
exact directory the manifest's `fields` entry names ("this task adds
ItemUseKarmaScissors ... inside the existing USE_CASH_ITEM opcode"), i.e. it
shares the `cash/serverbound` dir claimed via the `USE_CASH_ITEM` /
`cash/serverbound/CashItemUseMegaphone` op resolution. Claimed, no gap.

**Touched version gates.** The only `MajorVersion`/`MajorAtLeast`/`IsRegion`/
`Region()` hit inside `libs/atlas-packet` in the whole diff is:

```
+				ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
```

— a test-harness call in `item_use_karma_scissors_test.go` (`pt.CreateContext`
building a fixture context for the round-trip loop), not a wire-layout gate.
No codec's version gate moved. `docs/packets/gates.yaml` is untouched
(`git diff --name-only ... -- docs/packets/gates.yaml` empty).

**Matrix delta.** `docs/packets/audits/status.json` is untouched by this
branch (`git diff --name-only $BASE...HEAD -- docs/packets/audits/status.json`
is empty) — no cell state moved, so there is no matrix delta to reconcile
against the manifest.

**Adjacent non-codec changes** (outside the critic's `libs/atlas-packet`
scope, but worth recording since the task description asked for judgment on
them):
- `libs/atlas-constants/item/constants.go` adds `ClassificationKarmaScissors
  = Classification(552)` — a dispatch-key constant for the arm the manifest
  already declares, not a new codec or a wire-layout change; `libs/atlas-constants`
  is not a packet codec package.
- `services/atlas-channel/.../character_cash_item_use.go` adds
  `CashSlotItemTypeKarmaScissors`/`CashSlotItemTypeKarmaScissorsV95` and a
  `karmaScissorsCashSlotItemType(t)` resolver, and *refactors* (does not
  change) the pre-existing seal-timed `t.Region()=="GMS" && t.MajorVersion()>=95`
  branch into `sealTimedCashSlotItemType(t)` — same predicate, extracted to a
  named function so a new disjointness test can assert against the exact code
  path the runtime executes. This file lives in atlas-channel, not
  `libs/atlas-packet`; it selects which struct decodes a given wire opcode,
  it does not alter any struct's field layout. No scope hole.

## Step 3 — CLAIMED-BUT-UNVERIFIED

Manifest `fields` explicitly claims 3 of the 10 declared columns remain
unverified (`incomplete`, not silently dropped): `gms_v83`, `gms_v84`,
`gms_v92`. Cross-checked against the FINAL (HEAD) `status.json` row for
`cash/serverbound/CashItemUseMegaphone`:

| version | status.json state | manifest claim |
|---|---|---|
| gms_v48 | verified | (declared, verified) |
| gms_v61 | verified | (declared, verified) |
| gms_v72 | verified | (declared, verified) |
| gms_v79 | verified | (declared, verified) |
| gms_v83 | **incomplete** | incomplete, disclosed |
| gms_v84 | **incomplete** | incomplete, disclosed |
| gms_v87 | verified | (declared, verified) |
| gms_v92 | **incomplete** | incomplete, disclosed |
| gms_v95 | verified | (declared, verified) |
| jms_v185 | verified | (declared, verified) |

Exact match — no version is claimed verified while actually `partial`/
`incomplete`/`n-a`, and none of the three genuinely-unverified columns is
mis-claimed as covered. No `n-a` state is involved (all cells are
`verified` or `incomplete`), so the `n-a`-vs-manifest mismatch check does not
apply.

## Template-binding check (Task 16 claim)

The branch touches zero files under
`services/atlas-configurations/seed-data/templates/`. All 10 matrix-tracked
templates (`template_gms_{48,61,72,79,83,84,87,92,95}_1.json`,
`template_jms_185_1.json`) already reference the cash-item-use handler:

```
$ grep -rl "CashItemUse" services/atlas-configurations/seed-data/templates/*.json
template_gms_92_1.json  template_gms_61_1.json  template_gms_48_1.json
template_gms_87_1.json  template_gms_72_1.json  template_gms_83_1.json
template_gms_95_1.json  template_jms_185_1.json template_gms_84_1.json
template_gms_79_1.json
```

Task 16's "already bound in every tenant template" claim holds — the karma
arm rides the existing `USE_CASH_ITEM` opcode binding, so no template
mutation was required and none was made.

## Discrete-struct-per-mode check

`ItemUseKarmaScissors` is a standalone struct (not a type alias of
`ItemUseSeal`), matching `docs/packets/DISPATCHER_FAMILY.md`'s
discrete-struct-per-mode rule even though this op is not itself governed by
`docs/packets/evidence/families.yaml` (only `CCashShop::OnCashItemResult`,
the *clientbound* result dispatcher, is tracked there — this is the
*serverbound* item-use opcode, which the matrix has always represented as a
single row/registry entry regardless of how many sub-arm structs the handler
dispatches to; `ItemUseSeal` has no separate row either, so the absence of a
dedicated row for `ItemUseKarmaScissors` is consistent prior practice, not a
gap).

## Summary

No CHANGED-BUT-UNCLAIMED findings. No CLAIMED-BUT-UNVERIFIED findings. The
manifest's "no wire-layout change to any already-verified column" claim holds
(status.json untouched, no gate lines touched in `libs/atlas-packet`), and its
three-column unverified-and-disclosed claim matches `status.json` exactly.
