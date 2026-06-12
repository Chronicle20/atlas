# Systematic Off-By-One Divergent Remediation — Design

**Task:** task-081-ida-export-reharvest (extension — representation lever, conservative scope)
**Date:** 2026-06-10
**Status:** Approved (brainstorming), pending implementation plan

## Goal

Reduce the **divergent** bucket (338) by resolving the **systematic off-by-one cluster
(~175)** — the largest sub-cluster, where the live read count differs from the hand read count
by exactly one. **No byte-equivalence absorption** (the conservative choice): width-regrouping,
loop, and opaque/mask divergences stay **honest divergent**. Genuine Atlas-vs-client field
differences are **flagged as real findings**, never papered over.

## Scope decision (from brainstorming)

The brainstorming weighed three classifications for byte-equivalent-but-differently-decomposed
shapes:
1. a distinct `representation` verdict,
2. fold into `verified`,
3. **keep them divergent; only fix the systematic off-by-one** (chosen).

Option 3 means: **no change to `ValidateShape`'s comparison** (no count-mismatch absorption — it
would risk masking a real wire bug as verified). Instead, find and fix the *shared root cause* of
the off-by-one cluster, and leave width/opaque/loop diffs as honest divergent.

## The lever is investigation-led — value is contingent

The off-by-one cluster is ~175 entries (~96 at live=hand+1, ~79 at live=hand−1). The entire
`CCashShop` family (`OnBuy`, `OnBuyCouple`, `OnBuyFriendship`, `OnBuyNormal`, `OnBuyPackage`,
`OnBuySlotInc`, `SendGiftsPacket`, `OnRebateLockerItem`) is consistently `live = hand + 1`. That
regularity strongly suggests a **shared cause** (e.g. a leading cash-shop action/wrapper byte the
client reads before the leaf, which the hand baselines omit) — a cheap family-wide fix. **But if
characterization instead shows 175 independent regroupings, option 3 leaves them divergent** (no
absorption). The design surfaces that outcome honestly rather than guessing; the payoff depends on
how systematic the cluster actually is.

## Background

`ValidateShape` (`internal/idasrc/shapediff.go`) compares hand vs live by **read count and
position** (with per-position width tolerance via `fieldEquivalent`). When read *counts* differ it
reports a length divergence. The `seqScore`/`alignMatches` DP in `infer.go` has byte-equivalent
absorb logic, but it is used for *inference*, not validation — and per option 3 we are **not**
porting it into `ValidateShape`.

The existing `dispatcher` mechanism (`internal/idasrc/export.go` `dispatcherPrefix`) already
auto-prepends prefix bytes a dispatcher reads before a leaf handler — kinds `per-mob`, `per-pet`,
`per-pet-remote`, `per-user-remote`. A `#Mode`/leaf entry annotated `"dispatcher": "<kind>"` gets
those prefix reads prepended to its hand shape during resolve. This is the natural mechanism for a
shared leading-wrapper-byte off-by-one.

## Components

### 1. Divergent-shape diagnostic — `packet-audit diff-shape` (new subcommand)

**The reusable centerpiece.** Today `validate` reports `length: hand N vs live M` but not the read
*lists*, so the extra read can't be seen without hand-decompiling. `diff-shape` emits, per divergent
entry, the hand read-sequence and the live read-sequence **side by side**, with the differing
position classified as **leading** / **trailing** / **interior**. Output is a deterministic
markdown + JSON report.

- Inputs mirror `validate`: `--version`, `--baseline` (default), `--ida-port`, `--report`.
- Core is offline-testable with the existing `validateFakeMCP` fake; only the live run needs IDA.
- It reuses `ResolveLive` + `ExtractShape` + the baseline loader; the side-by-side alignment is a
  simple longest-common-prefix / longest-common-suffix split to locate where the lists diverge
  (this is *diagnostic only* — it does NOT change any verdict).

### 2. Live characterization (IDA-gated)

Run `diff-shape` over the off-by-one entries; cluster by base-handler family and by delta position.
Produce `divergent-characterization.md`: per cluster, the shared extra read (op + position) and its
category (shared-prefix / one-off-omission / genuine-difference).

### 3. Remediation by category (the approved blend)

- **Shared prefix/wrapper read** → the `dispatcher` mechanism. If an existing kind fits, annotate
  the family's baseline entries `"dispatcher": "<kind>"`. If a new shared prefix is found (e.g. a
  cash-shop action byte), **add a new kind** to `dispatcherPrefix` (code) + annotate the family
  (data). One change fixes the whole family.
- **One-off trace omission** (hand genuinely missed a real field) → correct that entry's baseline
  `calls` via the existing lossless surgical writer path (no reformat).
- **Genuine Atlas-vs-client difference** (client reads a field Atlas does not write, or vice versa)
  → record in `divergent-findings.md` as encoder work. **Do NOT auto-resolve.** These are the real
  bugs the lever surfaces.

### 4. Re-validate (IDA-gated)

Measure the divergent reduction; confirm shared-prefix/omission cases flip to verified and genuine
findings remain flagged. Width/opaque/loop divergences are expected to remain.

## Testing

- **diff-shape:** unit tests with `validateFakeMCP` — a divergent pair where the extra read is
  leading (classified `leading`), trailing (`trailing`), interior (`interior`); a verified pair
  produces no diff row. Determinism (byte-stable report).
- **New dispatcher kind (if added):** an `export_test.go` case mirroring the existing per-mob/
  per-pet tests, asserting the prefix reads are prepended to a leaf's resolved shape.
- **Baseline corrections:** validated by re-validate flipping the entry to verified.
- **Gates (CLAUDE.md):** `go test -race ./...`, `go vet ./...`, `go build ./...` on
  `tools/packet-audit`. Not a service → no docker bake; no redis.

## Out of scope (per option 3)

- **Byte-equivalence absorption in `ValidateShape`** (width regrouping `N×Decode1 ≡ Decode<N>`,
  opaque-buffer-absorbs-N-concrete) — rejected; those stay divergent.
- **Loop/movement-path and mask/opaque-block modeling** (the large-delta tail: `OnMobEnterField`,
  `OnCharacterInfo`, `OnAvatarModified`, …) — stay divergent.
- **The B2 diamond-descent bug; the residual unverifiable.**
