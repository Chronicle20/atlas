# Leaf Flat-Validation + Verbatim-Guard Dispatch — Live Run Results

**Date:** 2026-06-10. Live run of recall lever #2 against all four IDBs
(ports 13337–13340). Tasks 1–5 implemented offline; Task 6 (this) run live.

## Headline (corrected post-code-review)

| | verified | divergent | missing-mode | extra-mode | unverifiable | allowlisted |
|---|---|---|---|---|---|---|
| **Before** (post per-branch) | 352 | 311 | 0 | 0 | 434 | 251 |
| **After** (A + B) | **407** | 338 | 0 | 0 | **352** | 254 |
| **Δ** | **+55** | +27 | — | — | **−82** | +3 |

Per version (after): v83 91/70/0/0/95/34 · v87 94/83/0/0/79/64 · v95 132/121/0/0/101/123 ·
jms 90/64/0/0/77/33.

> **Correction:** the first run reported verified 410 / unverifiable 348. Code review
> (FAIL-1) found that a 2-arm dispatcher whose 2nd arm is a compound predicate
> (`else if (a && b)`) was misclassified as a leaf and could flat-validate to a false
> `verified`. The fix (`ac5a030d`, multi-way detection on any `else`-led header) reclassified
> **3 false-verified entries** to honest `unverifiable` — hence 410→407 / 348→352. The
> numbers below this line are the honest post-fix figures.

**+55 verified, −82 unverifiable.** The "per-mode shape not extractable" sub-bucket (373 at
the start of this lever) shrank substantially. `missing-mode 0 / extra-mode 0` held.

## Sub-lever A — leaf flat-validation

Re-validating with the `HasMultiwayDispatch` leaf check (no new selectors) alone moved **77
entries out of unverifiable**: ~14 to verified, ~63 to divergent. The divergences are almost
all **length mismatches** (`hand 46 vs live 48`) — honest loop/opaque-block representation
diffs (the separate deferred gap), now correctly surfaced as divergent rather than hidden as
unverifiable. The +28 net divergent in the final table is this effect.

E2E caught one false-divergence class: leaf handlers whose decompile yields **zero** reads
(e.g. `CClientSocket::OnConnect#Hello`, `hand 6 vs live 0`) were flagged divergent. Fixed
(`3dde2526`): an empty leaf extraction is `unverifiable` (extraction failed), never a
hand-N-vs-live-0 divergence — consistent with the selector branch.

## Sub-lever B — verbatim non-equality selectors

`resolve-dispatch` with `enumerateArms` now proposes verbatim `{Guard}` selectors for
non-equality dispatch arms (`x < 5`, `x & 0x10`, …). Auto-accepts nearly doubled:

| Version | auto-accept (equality-only → +verbatim) |
|---|---|
| v83 | 14 → 25 |
| v87 | 18 → 36 |
| v95 | 31 → 41 |
| jms | 14 → 26 |
| **total** | **77 → 128** |

145 selectors are now persisted (v83:30 v87:39 v95:49 jms:27), written additively by the
surgical writer (**+789 / −0 lines, 0 non-dispatch content changed**).

E2E caught a second integration bug: verbatim `{Guard}` selectors have `Case==0`, so the
equality-based case↔mode bijection counted them as a false `case<0>` **extra-mode** (48 across
the four versions). Fixed (`76ef6db9`): the binding collection skips `Default` AND `Guard`
selectors, keeping the completeness check equality-only as the design specified — extra-mode
returned to 0.

## What remains in unverifiable (348)

- **Indirect/vtable dispatch** — no readable condition (out of scope by design).
- **Genuine multi-way dispatchers** still without a confident selector (the to-confirm
  worklist tail: v83 85 / v87 66 / v95 98 / jms 65 — many residual after auto-accept).
- **Decompile failures / Unresolved spans / ABSENT** — the misc long tail.

## Divergent (339)

Dominated by loop / opaque-block / stat-mask **representation** diffs (length-close mismatches)
— the remaining roadmap gap (the "divergent modeling" lever), now including the leaf-flat
entries that surfaced here. 0 are confirmed real wire bugs from this pass.

## Two bugs caught by running it live

Both were integration bugs invisible to the offline unit tests, fixed with regression tests:
1. leaf-with-zero-live-reads → false divergence (`3dde2526`).
2. verbatim selector → false `case<0>` extra-mode in the bijection (`76ef6db9`).
