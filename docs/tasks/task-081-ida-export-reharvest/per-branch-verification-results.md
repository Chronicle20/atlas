# Per-Branch Verification — Live Run Results (Task 8)

**Date:** 2026-06-09. Live run of the per-branch verification pipeline against all four
IDBs (ports 13337/13338/13339/13340). Tasks 1–7 implemented + committed; this is the
end-to-end measurement.

## Procedure

Per version: `resolve-dispatch` (infer joint dispatch selectors, auto-accept ≥0.60
confidence, write to baseline, emit confirmation worklist) → `validate` (per-branch
extract + verify, case↔mode bijection). Selector writes are deterministic.

## resolve-dispatch (auto-accept pass)

| Version | #Mode entries | auto-accepted (≥0.60) | to confirm | undecompilable |
|---|---|---|---|---|
| gms_v83 | 118 | 14 | 96 | 8 |
| gms_v87 | ~110 | 18 | 84 | 8 |
| gms_v95 | ~141 | 31 | 108 | 2 |
| gms_jms_185 | ~97 | 14 | 77 | 6 |
| **total** | | **77** | **365** | **24** |

## validate — before → after

Before = 2026-06-09 baseline (no selectors). After = with the 77 auto-accepted selectors +
if/else parser + bijection.

| Version | verified | divergent | missing-mode | extra-mode | unverifiable |
|---|---|---|---|---|---|
| v83 before | 65 | 64 | — | — | 127 |
| v83 after  | **76** | 66 | 33 | 0 | **114** |
| v87 before | 66 | 73 | — | — | 117 |
| v87 after  | **79** | 78 | 67 | 0 | **99** |
| v95 before | 97 | 102 | — | — | 155 |
| v95 after  | **122** | 106 | 267 | 0 | **126** |
| jms before | 65 | 57 | — | — | 109 |
| jms after  | **75** | 61 | 36 | 0 | **95** |
| **Σ before** | 293 | 296 | — | — | 508 |
| **Σ after**  | **352** | 311 | 403 | **0** | **434** |

**Verified +59 (293→352). Unverifiable −74 (508→434).** The −74 is the slice of the ~450
"per-mode shape not extractable" entries the auto-accepted selectors + if/else parser
cleared. Divergent +15 — of which only ~2/version are selector-driven (e.g. v83:
`OnViewAllCharResult#CharacterViewAllCharacters` hand 46 vs live 49 — a loop/opaque-block
**representation** diff, not a wrong case). The other ~64/version divergent are the
pre-existing flat representation issues (the separate deferred gap). So the auto-accepts are
high precision (v83: 14 accepted, ~12 clean verified, 2 representation-divergent).

> **Superseded:** the 403 missing-mode below was inflated by a per-address bijection
> bug; corrected to **251 distinct** and bulk-allowlisted. See
> `missing-mode-triage.md` for the corrected analysis and the final roll-up.

## Bijection: 403 missing-mode, 0 extra-mode

**0 extra-mode everywhere** — every Atlas `#Mode` writer maps to a real client case (no dead
writers). Clean.

**403 missing-mode** (client dispatch case with no Atlas writer) — highly concentrated:

| Version | missing-mode | distinct base handlers |
|---|---|---|
| v83 | 33 | 5 |
| v87 | 67 | 7 |
| v95 | 267 | 12 |
| jms | 36 | 4 |

v95's 267 sit in just 12 base handlers (~22 cases each) — big switch dispatchers (e.g. the
`CWvsContext` family) where Atlas implements only a few sub-opcodes. These are mostly
**intentionally-unimplemented** features (Atlas is a partial reimplementation) → allowlist
material, not bugs. The bijection output is the precise "what's not built" integration signal
for onboarding a version.

## What remains (the path to zero unverifiable)

The 365-entry to-confirm worklist is the recall ceiling, not a simple confirmation task. On
v83 (96 to confirm): **77 have NO inference signal at all** (empty dispatch, confidence 0 —
handlers like `CClientSocket::OnConnect#Hello`, `OnAliveReq#PingReceive` that are not
equality-dispatch switches) and only **19 are ambiguous-with-proposal** (multiple candidate
cases, IDA-disambiguable). Reaching ~0 unverifiable needs the explicitly-deferred capability
gaps, not grinding:

1. **Demangled `Class::Method` helper-name resolution** — Unresolved spans suppress match
   scores, capping auto-accept at ~12–22%. The dominant recall lever.
2. **Non-equality dispatch modeling** — the 77/version no-signal handlers dispatch on
   if/else-non-equality, flags, or indirect calls the switch/if-`==` model can't represent.
3. **Manual IDA confirmation of the ~19/version ambiguous-with-proposal** — bounded, ~1
   verified each.
4. **Loop/opaque-block/mask modeling (the 296 divergent)** — the other deferred gap.

## Selector persistence note — RESOLVED

The 77 selectors are committed (`b7fe9607`). The typed-marshal `WriteDispatch` turned out to
be **lossy** — it silently dropped hand-authored fields the export structs don't model
(`region` ×156, `note`/`_note` ×173, `size` ×1) — and churned formatting (mixed indent,
Python escaping). Replaced with a **lossless surgical writer** (`774e33e9`) that inserts the
`dispatch` field into each target function's raw object bytes, leaving every other byte
verbatim. Result: the four baselines changed by **+509/-0 lines**, purely additive, 0
non-dispatch content altered — so **no normalization commit was needed**.
