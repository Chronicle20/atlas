# Review — fix commit `3c77762b7`

Unit: `bug-character-damage-attack-index-truncation.md` fix
Range reviewed: `903c631ac..3c77762b7` (single commit; `50844522a` docs-only, out of scope)
Reviewer: `task-reviewer` (sonnet)

## Scope

`git diff --stat` — 5 files, +179/-2, all in `libs/atlas-packet/character/clientbound/`:
`damage.go` (+2/-2), `damage_test.go` (+54), `v61_test.go`/`v72_test.go`/`v79_test.go`
(+41 each). Traced into the one consumer that matters for a seam defect:
`services/atlas-channel/atlas.com/channel/socket/handler/character_damage.go:64`
(`CharacterDamageHandleFunc`). No other file in the diff calls a contract whose
correctness this fix depends on.

## 1. Does `>= model.DamageTypePhysical` reproduce `nAttackIdx > -2` exactly?

**PASS.** `model.DamageType` is declared `type DamageType int8`
(`libs/atlas-packet/model/damage_taken_info.go:15`), and `m.attackIdx` in
`CharacterDamage` is typed `model.DamageType`
(`libs/atlas-packet/character/clientbound/damage.go:19`). The comparison
`m.attackIdx >= model.DamageTypePhysical` (`damage.go:52`, `:75`) is therefore a
**signed** int8 comparison against `-1`. For every integer in the int8 domain,
`x >= -1` is definitionally identical to `x > -2` — there is no boundary value
where the two diverge, including all negatives below `DamageTypeCounter` (-2,
-3/`DamageTypeObstacle`, -4/`DamageTypeStat`, down to -128). Verified this holds
by reverting to the old predicate and confirming exactly the byte-exact tests
fail (see §3). The `byte(m.attackIdx)` conversion at `damage.go:50` happens
before the comparison and only affects the wire representation, not the guard
logic — not a source of sign-handling risk here.

## 2. Consumer trace — does the fix introduce a new mismatch anywhere?

**PASS.** `character_damage.go:52-53` decodes the client-sent `DamageTakenInfo`
via `p.Decode(l, ctx)(r, readerOptions)`, which uses the *already-correct*
`m.nAttackIdx >= DamageTypePhysical` at `damage_taken_info.go:122` (decode) —
untouched by this diff, confirmed via `git diff` scoped to that file (no
changes). `character_damage.go:64` then forwards `p.AttackIdx()` verbatim into
`NewCharacterDamage(...).Encode`, broadcast via `ForOtherSessionsInMap`.

Before the fix: serverbound decode already used the correct `>= -1` guard, but
clientbound re-encode used the equality pair (`-1`/`0` only) — an asymmetry
that under-read for `attackIdx >= 1`. This is the actual bug. After the fix,
both sides use the same predicate, so encode and decode are symmetric for the
full int8 domain a client can send: nothing routes to the equality pair
anymore anywhere in the module (confirmed by the sweep in §4). A
hostile/garbage `attackIdx` cannot produce a new over-read: any value
`>= -1` now gets the extra fields on write, matching what every receiving
client's own `CUserRemote::OnHit` guard (`> -2`, confirmed identical across
v48/v61/v72/v79/v83/v84/v87/v92/v95/jms185 per the bug file) expects to read.
No new direction-mismatch is introduced.

The `>= 95` bGuard gate at `damage.go:56` is untouched by this diff (confirmed:
diff only touches lines 52 and 75) and is orthogonal to the attackIdx
predicate — it gates one extra byte after the block this fix controls the
presence of, not the block's presence itself. Correctly left alone per bug
file item #1.

## 3. Do the new tests assert the NEW contract (byte-exact), and is the `-2` boundary covered?

**PASS**, verified by direct RED/GREEN run, not taken on the report's word.

Reverted `damage.go:52`/`:75` to the old equality-pair predicate and re-ran
`go test ./character/clientbound/... -run TestCharacterDamage -v`:

- `TestCharacterDamageMobAttackIndexIncludesBlock` — **FAILED** at the top
  level (the v83 byte-exact assertion), while every one of its 11
  `test.Variants` round-trip subtests (`GMS_v28`, `GMS_v83`, ... `JMS_v185`)
  still **PASSED**. This is the load-bearing evidence: it demonstrates
  concretely, not just by assertion, that a round-trip-only test would have
  stayed green with the bug present — exactly the failure mode the bug file
  describes for why the original suite never caught it.
- `TestCharacterDamageByteOutputV61MobAttackIndex`,
  `...V72MobAttackIndex`, `...V79MobAttackIndex` — all three **FAILED**.
- `TestCharacterDamageCounterOmitsBlock` and the three
  `...CounterOmitsBlock` per-version tests — all **PASSED** even with the bug
  present, which is correct: `DamageTypeCounter = -2` was never mis-handled by
  the old equality pair either (it isn't `-1` or `0`), so this sub-case
  doesn't distinguish old from new behavior. It's still worth keeping as a
  boundary regression guard for the *new* predicate, since `>= -1` could in
  principle be wrong in the other direction (e.g. an off-by-one to `>= -2`
  would flip this test) — it just isn't diagnostic of the original bug.

Restored `damage.go` via `git checkout`, re-ran the full set: 12/12 PASS.
Working tree is clean after the revert-and-restore (`git status --porcelain`
empty). The report's RED/GREEN claim is corroborated exactly, including the
detail about round-trip subtests staying green.

The `-2` boundary is genuinely covered on both sides: `attackIdx = 1` proves
inclusion, `DamageTypeCounter = -2` proves exclusion, both via byte-exact
`want := []byte{...}` comparisons (`damage_test.go:29-51`, and per-version
equivalents), not `RoundTrip`.

## 4. Per-version fixtures — addresses reused, not invented?

**PASS.** The new `...MobAttackIndex` / `...CounterOmitsBlock` tests in
`v61_test.go`, `v72_test.go`, `v79_test.go` carry no new IDA addresses in their
own bodies — each comment explicitly states "Same instruction addresses as
TestCharacterDamage[Encode]ByteOutputV{61,72,79}." Confirmed those pre-existing
addresses are the ones actually annotated on the physical-damage fixture
immediately above each new test:

- v61: `packet-audit:verify ... ida=0x7cb9ff` (`v61_test.go:466`)
- v72: `packet-audit:verify ... ida=0x88c5ad` (`v72_test.go:477`)
- v79: `packet-audit:verify ... ida=0x8d9489` (`v79_test.go:490`)

Cross-checked each against `docs/packets/ida-exports/`:
`grep -n '"address": "0x7cb9ff"' gms_v61.json` → `CUserRemote::OnHit`;
`0x88c5ad` → `gms_v72.json` `CUserRemote::OnHit`; `0x8d9489` → `gms_v79.json`
`CUserRemote::OnHit`. All three match the version-scope table in the bug file
(`bug-...md:87`) verbatim. No invented addresses.

Also confirmed via grep that `damage_taken_info.go:122`/`:174` (the serverbound
sibling the report cites as "already correct") are genuinely unmodified by
this diff — the `git diff` for that file is empty; only `damage.go` changed.

## Sweep claim (report §"Open item #2 — sweep result")

Re-ran the grep independently: `grep -rn "DamageTypePhysical\|DamageTypeMagic\|DamageTypeCounter" --include="*.go" libs/atlas-packet | grep -v _test.go` returns exactly the 4 const declarations plus the 4 non-test comparison sites (`damage_taken_info.go:122,174`, `damage.go:52,75`) — all `>= DamageTypePhysical`. No equality-pair predicate remains anywhere in the module. Claim confirmed.

## Not evaluable

- Whether the `>= 95` bGuard gate at `damage.go:56` is actually correct for v87
  (open item #1 in the bug file) — explicitly out of scope per task
  instructions, needs the v87 IDB, not touched by this diff.
- Live re-test (mob attack index >= 1, second character in map) — not
  performed by the implementer, not evaluable from a static diff review.
- `tools/verify.sh` full gate — not run as part of this review; report states
  module-local `go build`/`go test` passed and explicitly disclaims gate
  verdict status.

## Verdict rationale

Requirement met exactly: the predicate now matches the client's guard for the
entire `int8` domain, the fix is applied symmetrically to both `Encode` and
`Decode`, the one live consumer forwards attack index without transformation
and is now symmetric end-to-end, tests are byte-exact and independently
verified (not merely trusted) to fail without the fix while round-trip-only
subtests do not, and no IDA address was fabricated. No blocking finding.
