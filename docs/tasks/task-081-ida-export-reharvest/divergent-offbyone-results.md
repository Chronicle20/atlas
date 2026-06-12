# Systematic Off-By-One Divergent Remediation — Results

**Date:** 2026-06-10. Live run against all four IDBs (ports 13337–13340).

## Headline

| | verified | divergent | missing-mode | extra-mode | unverifiable | allowlisted |
|---|---|---|---|---|---|---|
| **Before** (post non-eq lever) | 407 | 338 | 0 | 0 | 352 | 254 |
| **After** | **461** | **284** | 0 | 0 | 352 | 254 |
| **Δ** | **+54** | **−54** | — | — | — | — |

Per version (after): v83 101/60/95 · v87 102/75/79 · v95 159/94/101 · jms 99/55/77.

**Exactly +54 verified / −54 divergent** — every remediated entry flipped clean to verified, no
collateral movement (missing/extra stayed 0, unverifiable unchanged).

## What was done

1. **`diff-shape` diagnostic** (new subcommand) surfaced the hand-vs-live read lists for divergent
   entries with the divergence position classified (leading/trailing/interior).
2. **Characterization** (`divergent-characterization.md`): the off-by-one ±1 deltas split
   109 leading / 59 interior / 46 trailing. The clean systematic sub-cluster was **54 entries**
   (serverbound dialog handlers — cash-shop/shop/trunk/trade/minroom/personal-shop) with the
   identical `hand == live[1:]` signature: a leading single byte the hand baselines omit.
3. **IDA spot-check** of 3 samples confirmed each is a genuine leading `COutPacket::Encode1`
   sub-action byte (0x1D / 0x06 / 0x1E), written to the same packet, distinct from the constructor
   opcode — the baselines were each short one real leading field.
4. **Surgical remediation** via the new `PrependCall`: prepended the leading `Encode1` to all 54
   (v83:10 v87:8 v95:27 jms:9), additive `+216 / −0`, 0 other content changed.

## Findings — no genuine encoder bugs isolated this pass

The 54 were all **hand-baseline omissions** (the trace started after the leading byte), not Atlas
encoder bugs. `divergent-findings.md` records no genuine Atlas-vs-client field differences from this
conservative pass; if any surface in future triage they go there as encoder work.

## What remains divergent (284) — by design (option 3)

- **~55 non-single-byte leading omissions** (leading `Decode4`/`DecodeStr` rather than a byte) — a
  different pattern, not spot-checked; left honest divergent for a future pass.
- **59 interior + 46 trailing ±1** — width regrouping and trailing optional fields.
- **The large-delta tail** — loop/movement-path and opaque/mask packets (`OnMobEnterField`,
  `OnCharacterInfo`, `OnAvatarModified`, …).

These stay **honest divergent** per the option-3 decision (no `ValidateShape` byte-equivalence
absorption). `ValidateShape` was not modified.

## Reusable deliverable

`diff-shape` is the durable output: a read-only diagnostic that surfaces hand-vs-live read lists
for any divergent entry, the engine for all future representation triage.
