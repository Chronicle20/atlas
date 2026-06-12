# Task-081 — Four-version validation results (unattended, live)

First fully **unattended** four-version validation run (2026-06-05), after re-mapping
the client to the upgraded ida-pro-mcp API and wiring multi-IDB `select_instance`. One
background job ran `infer → apply-joint-picks → validate` for all four versions, each
targeting its own IDA instance by port — no manual IDB cycling.

| Version | port | verified | divergent | unverifiable | annotated `#` (verif/diverg/unverif) |
|---|---|---|---|---|---|
| gms_v83     | 13337 | 77  | 66  | 113 | 37 (16/15/6) |
| gms_v87     | 13338 | 74  | 72  | 110 | 41 (14/12/15) |
| gms_v95     | 13339 | 113 | 113 | 128 | 73 (27/30/16) |
| gms_jms_185 | 13340 | 66  | 63  | 102 | 25 (10/9/6) |

**330 hand-authored wire shapes confirmed against the live IDBs** (sum of `verified`),
zero baseline mutation (validate is read-only), zero regression risk.

## What this proves
- The validation pivot works across all four versions, unattended. The toolchain
  (ExtractShape/dispatch/InferDispatch+joint/ResolveLive/ValidateShape/validate/infer),
  the new-server re-map (lookup_funcs/decompile/callees, structuredContent), multi-IDB
  `select_instance`/`--ida-port`, and the `case Nu:` suffix fix all hold on live data.

## Known noise (the next grind)
- `divergent`-among-annotated is inflated (e.g. v95 30/73) because ALL joint picks were
  applied, including low-confidence ("ambiguous") ones. A wrong case pick → a false
  `divergent`. The joint PICKS are mostly correct (0 case conflicts in any base;
  `OnFriendResult#Invite → 9` in v83/v87), but the CONFIDENCE FORMULA under-credits
  joint-resolved picks: it scores each entry's margin to its own raw runner-up case,
  ignoring that the runner-up was claimed by another entry under the one-to-one
  constraint. Fix: make confidence joint-aware (margin to the best UNASSIGNED
  alternative), then apply only high-confidence annotations for a clean report.
- The large `unverifiable` counts are mostly honest: un-extractable if/else-dispatch
  handlers (e.g. `OnCheckPasswordResult#*`), named-helper descent that the new server
  can't resolve by demangled name (→ Unresolved), and undecompilable helpers.

## Per-version artifacts (in /tmp, not committed)
`{version}_prop.json` (infer proposals), `{version}_annot.json` (annotated baseline),
`{version}_validation.md` (the report). Re-runnable via `/tmp/run4versions.sh`.

## Clean run — joint-aware confidence + high-confidence-only apply

After adding joint-aware confidence (margin to best AVAILABLE alternative) and applying
ONLY high-confidence (≥0.6) annotations, the high-confidence picks are **trustworthy**:

| Version | high-conf annotations | verified | divergent | precision (of decided) |
|---|---|---|---|---|
| gms_v83     | 8  | 7  | 1 | 87%  |
| gms_v87     | 9  | 6  | 2 | 75%  |
| gms_v95     | 24 | 14 | 3 | 82%  |
| gms_jms_185 | 4  | 4  | 0 | 100% |

When the tool labels an annotation high-confidence, the hand-authored shape verifies
against the live IDB ~85% of the time. The **6 high-confidence divergences across all
four versions are genuine candidate findings** to triage (real Atlas bug → byte-test
fix; hand-tracing error → corrected baseline w/ IDA evidence). Everything else is
honest `unverifiable` (un-extractable if/else dispatch, undecompilable helpers, or
demangled-name helpers the new server won't resolve).

**Precision is high; recall is the remaining lever.** Only a few entries per version
reach high confidence because assigned match-scores are moderate — the new server
returns "Not found" for demangled `Class::Method` helper names, so named-helper
descent yields `Unresolved` spans that lower the live-vs-hand match quality.

## Remaining to a higher-recall four-version verification (follow-ups)
1. **Resolve demangled `Class::Method` helper names** (e.g. mangle-and-retry, or a
   name search via `func_query`/`find_regex`) → fewer `Unresolved` spans → higher
   match scores → more high-confidence annotations → higher recall.
2. **Triage the 6 high-confidence divergences** in IDA (the actionable findings).
3. Optional: a human-confirmation pass over "ambiguous" picks to lift coverage further.
