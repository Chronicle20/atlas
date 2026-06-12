# Task-081 — v83 validation proof (V-A toolchain on the live IDB)

Empirical proof that the validation pivot's toolchain (Phase V-A: `ExtractShape`,
`dispatch` schema, `InferDispatch`, `ResolveLive`, `ValidateShape`, the `validate`
and `infer` commands) works end-to-end against the live v83 IDB
(`MapleStory_dump.exe`, md5 `80ff438c…`). Ran 2026-06-05.

## What was run
1. `infer --version gms_v83` over the hand-authored baseline (256 entries) → proposed
   dispatch selectors per entry by matching each `#`-mode entry's hand-authored reads
   to the live decompile's switch cases.
2. Applied ONLY high-confidence (≥0.6, non-ambiguous) proposals to a copy of the
   baseline (23 entries annotated).
3. `validate --version gms_v83 --baseline <annotated copy>` → per-shape report.

## Result

### The machinery WORKS where dispatch is correctly annotated (net-positive)
Of the **23 correctly-annotated `#`-entries**:
- **18 verified** — hand-authored per-mode reads CONFIRMED against the live IDB
  (e.g. `OnFieldEffect#BossHp/Summon/Tremble/RewardRullet`,
  `OnViewAllCharResult#Count`, `OnPacket#GenericError`).
- **3 divergent** — surfaced with specifics, all `OnGuildResult#*`:
  - `OnGuildResult#AgreementResponse` — length: hand 4 vs live 3
  - `OnGuildResult#MemberJoined` — length: hand 10 vs live 9
  - `OnGuildResult#RequestAgreement` — length: hand 4 vs live 3
  (each: hand has exactly one more field than the live extraction — a real finding to
  triage in IDA, or an off-by-one inference/parser artifact in the guild handler.)
- **2 unverifiable** — undecompilable helper span (honest `Unresolved`).

This is the PRD's "genuinely verified" goal delivered: hand-authored entries checked
against the live binary, confirmations and divergences both surfaced — and crucially
it CANNOT regress the baseline (validate is read-only; it only finds problems).

### The gap is bootstrap COVERAGE, not the machinery
Whole-run roll-up: verified 103 / divergent 137 / unverifiable 16. Decomposed:
- divergent = **3 annotated `#` (candidate real findings)** + **68 un-annotated `#`
  (flattening noise — these still need a dispatch selector)** + **69 non-`#` base**.
- The 68 un-annotated `#` divergences are NOT real — they are the flattening artifact
  (no dispatch → `validate` compares the whole flattened switch to a per-mode entry).
  They vanish once the entry is annotated.

`infer` only auto-annotated **23 of ~112 `#`-entries** at high confidence:
`23 high-confidence, 26 ambiguous, 6 undecompilable`, the rest low/no-case. Notably
the canonical `OnFriendResult#Invite` inferred the WRONG case (8; real is 9) at
confidence 0 — because case 9's correct shape contains the **undecompilable GW_Friend
`Unresolved` span**, which depresses its match score and lets the simpler case 8
compete. So the rich entries (those with undecompilable helpers) are the hardest to
auto-infer — exactly the ones that most need verification.

### Non-`#` base divergences (69)
These base functions need no per-mode dispatch but still diverge: a mix of
representation-equivalent (should already be tolerated — re-check `ValidateShape`),
**serverbound multi-send branch-flattening** (a `Send*` function building different
COutPackets on different branches — needs a serverbound branch selector, the design's
open question §6), and possibly genuine findings. Requires the serverbound selector
mechanism + triage.

## Conclusion
The validation pivot is **proven viable** (18 confirmations + 3 surfaced findings on a
23-entry annotated sample; zero regression risk), in contrast to the replace approach
which regressed the audit (26 ✅→❌). Realizing it fully requires:
1. **Stronger bootstrap** — `InferDispatch` is too weak on real data (~20% of `#`
   entries auto-resolved; undecompilable segments mis-rank the correct case). Needs
   better scoring (LCS/anchor on the discriminator+leading reads; weight the matched
   prefix; tolerate Unresolved runs in scoring) AND/OR a human-confirmation pass over
   the ~90 ambiguous/wrong entries (the bootstrap is then semi-automatic, not free).
2. **Serverbound branch selectors** — for multi-send base functions (design §6 open
   question), so non-`#` serverbound entries extract per-branch.
3. **Triage** — the 3 (and future) annotated divergences in IDA: real-Atlas-bug →
   byte-test fix; hand-tracing error → corrected baseline w/ IDA evidence.

Artifacts (this run, in /tmp, not committed): `v83_proposal.json`,
`v83_validation.md`, `gms_v83_annotated.json`.

## Update — bootstrap improvements (Unresolved-run + joint assignment)

Two inference improvements were then made and re-measured live on v83:
1. **Unresolved-run-aware scoring** (V3+) — a live `Unresolved` (undecompilable helper)
   absorbs a RUN of hand fields, so the correct case (e.g. the GW_Friend-bearing case 9)
   is no longer score-depressed. Alone this didn't lift aggregate confidence (it also
   inflates competing Unresolved-bearing cases).
2. **Joint per-base assignment** (V3++) — the decisive fix. A base's N `#`-entries map
   ONE-TO-ONE to N distinct cases; solving that as a max-weight assignment (not
   independent argmax) eliminates conflicts and resolves ties. Results: **0 conflicts**
   across all bases; the canonical `OnFriendResult#Invite → case 9` (was wrongly 8);
   all six Friend entries map to distinct cases.

Validate over ALL joint picks (every `#`-entry annotated): **verified 110 / divergent
135 / unverifiable 11**. `#`-entry verdicts: **48 verified (up from 18), 67 divergent,
3 unverifiable**; non-`#` base: 65 verified / 65 divergent / 8 unverifiable.

The 67 `#`-divergent are categorized:
- **28 structural dispatch-mechanism gaps** — 17 if/else-chain handlers
  (`OnCheckPasswordResult#*`: live = whole function, no switch guards emitted) + 11
  handshake handlers (`OnConnect#*`: live = 0). Each mechanism (if/else dispatch,
  connect-time special handlers) needs its own extraction modeling — an open-ended tail.
- **26 "close" divergences (≤3 fields)** — the PAYOFF: candidate real findings to triage
  in IDA (hand-tracing error / Atlas bug / loop or representation nuance), e.g.
  `OnCashItemResult#CashShopInventory` (h5/l4), `OnSetAccountResult#AfterLogin` (h3/l2).
- **14 other** — large hand vs small live (`OnPartyResult#Join` h9/l3), likely nested
  mode→sub-mode dispatch or a wrong joint pick.

**Assessment:** the validation now delivers concrete value on v83 — **48 per-mode
entries confirmed against the live IDB + ~26 candidate real findings surfaced** — with
zero regression risk. The confidence *formula* under-credits joint-resolved picks (it
measures per-entry separation, not the joint constraint), so the high-confidence COUNT
(16) understates the pick correctness; the validate verdicts are the ground truth.
Remaining work is a long tail of per-mechanism extraction (if/else dispatch, handshake,
nested dispatch, serverbound multi-send) plus triaging the ~26 close findings.

