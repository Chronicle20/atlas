# Backend Audit — task-126 (AP/SP reset items)

Prior per-phase audits live in `audit-backend-guidelines.md` and `audit-plan-adherence.md` (both PASS for the original feature). This file collects post-ship review passes.

## Post-ship review — 2026-07-12

**Scope:** `git diff 208cdee4de..HEAD` restricted to the magician MP-loss fix and the two hand-resolved saga merge reconstructions. gms_87/gms_95/jms_185 templates and the bulk MTS code (from main) were excluded per instruction.

**Reviewer mindset:** adversarial, FAIL-until-proven.

### Build & test gate

| Module | `go build ./...` | `go vet ./...` (scoped) | `go test` (scoped) |
|--------|------------------|--------------------------|--------------------|
| atlas-character `character/` | PASS | PASS | PASS (`ok atlas-character/character 1.809s`) |
| atlas-saga-orchestrator `saga/` | PASS | PASS | PASS (`ok .../saga`, `.../saga/mock`) |

### Verdict

**No blocking or correctness-breaking defects found.** The INT-scaled formula, its integer truncation, the uint16 bounds, the min-pool/decrement `takeMp` consistency, and both hand-resolved saga functions are all verified correct against source. Findings below are all Minor/Moderate (test-coverage + observability + style), none blocking.

### Verified-correct (evidence)

- **Formula & truncation** — `pointResetMagicianTakeMp = 3*int(effectiveInt)/40 + 30` (point_reset.go:71). Integer division matches the client's truncating calc; unit test `TestPointResetMagicianTakeMp` (point_reset_test.go:101) pins {0→30, 14→31, 40→33, 200→45, 999→104}, all correct.
- **uint16 over/underflow** — input is clamped to `math.MaxUint16` (processor.go:1959) or sourced from `c.Intelligence()` (uint16); worst case `3*65535/40+30 = 4945`, well within uint16. No intermediate overflow (`3*int(...)` widens to int). The magician decrement arm is underflow-safe because magician `pointResetMinMp` is always positive, so the guard forces `newMaxMp ≥ takeMp + min > takeMp` before `newMaxMp -= takeMp` (processor.go:2036–2044).
- **`takeMp` applied consistently** — same `takeMp` local drives the min-pool check (processor.go:2036), the MaxMp decrement (2040), and the Mp decrement (2041/2044). No split between check and apply.
- **Merge split — compensator.go** — `compensatePointReset`/`DispatchPointResetRollbacks` (1180/1246) invert only `DestroyAsset → RequestCreateItem`; `compensateMtsOperation`/`DispatchMtsOperationRollbacks` (1281/1345) handle only `AwardCurrency`/`ReleaseFromCharacter`/`ReleaseFromMtsHolding`. No action/payload type is shared or crossed; routing at CompensateFailedStep (216/225) dispatches by `SagaType()` cleanly.
- **Merge split — event_acceptance.go outcomeTable** — `EventKindCharacterApTransferError → OutcomeFailure` (262), `EventKindSkillSpTransferred → OutcomeSuccess` (281), `EventKindSkillSpTransferError → OutcomeFailure` (282). All three classifications are correct (`*Error`→Failure, `*Transferred`→Success); `TestOutcomeTableCompleteness` passes.

### Findings (ranked)

**F1 — [Moderate] The effective-stats *success* path is never exercised by any test.**
`transfer_ap_test.go:547` (Case 13) and its own comment rely on effective-stats being *unreachable* so the handler takes the base-INT fallback (INT 200 → takeMp 45). Consequently the actual feature behavior — effective INT (base+equipment) flowing into `takeMp`, the `es.Intelligence > math.MaxUint16` clamp (processor.go:1959), the `es.Intelligence > 0` gate, and the uint32→uint16 conversion (processor.go:1958–1963) — is untested end-to-end. Only the pure formula (point_reset_test.go) and the fallback branch are covered. A regression that mis-wired the effective-stats read (wrong field, dropped clamp, inverted gate) would pass the entire suite. *Failure scenario:* a future edit swaps `es.Intelligence` for `es.Strength`; every mage silently loses the wrong MaxMP and no test flags it. *Fix:* add an httptest-backed case where effective INT ≠ base INT (equipment bonus) and assert `takeMp` reflects the *effective* value.

**F2 — [Minor] Effective-stats fetch error is swallowed silently (processor.go:1958).**
`if es, err := ...; err == nil && es.Intelligence > 0` discards any error with no log. The sibling callsites in the same file route their error through a logging helper (e.g. processor.go:1135 `resolveEffectiveMax(p.l, ..., err, ...)`); this new call is the outlier. *Failure scenario:* an effective-stats outage silently degrades every magician reset to the base-INT fallback, which under-debits MaxMP relative to the client — a smaller re-appearance of the exact desync this fix targets — with zero operational signal. *Fix:* emit a debug/warn log on the error branch.

**F3 — [Minor] REST call fires for all MP-source transfers, not just magicians (processor.go:1957).**
The fetch is gated only on `from == CommandDistributeApAbilityMp`; the magician gate is applied later, inside the transaction, once `c.JobId()` is known. A warrior/thief/pirate doing an MP→X reset triggers a wasted effective-stats round-trip whose result is discarded. Correct, just wasteful; AP transfer is user-initiated (not hot-path), so low impact. Reordering to load jobId first would move the call inside the tx boundary, which the code deliberately avoids — acceptable as-is, noted for completeness.

**F4 — [Minor / DOM-21 observation] `isPointResetMagician` uses a raw numeric branch classifier instead of the file's `job.IsA` idiom (point_reset.go:53).**
`int(jobId)%1000/100 == 2` reimplements branch classification with magic numbers, whereas `pointResetPolicyRows` one screen up (point_reset.go:29) classifies the same magician branch via `job.IsA(..., job.Id(200), job.BlazeWizardStage1Id)`. DOM-21 nonetheless **passes**: no shared helper in `libs/atlas-constants/job` replicates this exact classifier (`GetType` = job/1000 gives class, not branch), and the numeric form is *intentionally broader* — it also matches Evan (2200s) to mirror the client classifier `sub_A0EC6B (job%1000/100)`, which `job.IsA(200, 1200)` would miss (Evan 2200/100=22). Recommend (optional) promoting a shared `job.IsMagicianBranch` to atlas-constants rather than a service-local magic-number expression.
  - *Sub-note:* because the formula overrides `takeMp` for Evan (2200s) but `pointResetPolicyFor`/`pointResetMinMp` have no Evan row and fall to the DEFAULT policy, an Evan would receive magician-scaled MP loss with default HP/gain/min values. Unverified vs client; Evan is v84+ content and out of this task's v83 scope, so low risk — flagged only for the asymmetry.

**F5 — [Not introduced here] Pre-existing uint16 underflow risk for *non-magician* low-level MP resets.**
For non-magicians where `pointResetMinMp` is negative at very low level (e.g. Bowman base `14*L-15 < 0` at L=1), the guard `int(newMaxMp)-int(takeMp) < min` can pass with `newMaxMp < takeMp`, underflowing `newMaxMp -= takeMp` to ~65535 (processor.go:2036–2040). This path predates this diff (was `policy.takeMp`) and the magician arm this task touches is safe. Recorded only so it is not mistaken as task-126-introduced; not a finding against this change.

## Post-ship review — resolutions

All five findings addressed (commit follows this note):

- **F1 (effective-INT success path untested)** — added `TransferAP_Case14_Magician_MPtoSTR_EffectiveIntUsed` (transfer_ap_test.go): httptest atlas-effective-stats returning INT 300 while base INT is 100; asserts `MaxMp 2000 → 1948` (takeMp 52), which base INT (37 → 1963) would fail. Exercises the fetch, `>0` gate, uint32→uint16 clamp, and field-read wiring.
- **F2 (silent fetch error)** — the effective-stats failure now logs `WithError(...).Warnf(...)` naming the base-INT fallback (processor.go, magician MP-source arm).
- **F3 (fetch fired for all MP transfers)** — the fetch moved inside the `isPointResetMagician` branch (inside the tx, mirroring ChangeHP/ChangeMP), so only magicians incur the round-trip.
- **F4 (raw numeric classifier)** — `isPointResetMagician` now uses `job.Is(jobId, job.Id(200)) || job.Is(jobId, job.BlazeWizardStage1Id)`, identical to the magician `pointResetPolicy` row, so a character gets INT-scaled MP loss iff it also gets the magician policy (no Evan hybrid). Rationale documented on the function.
- **F5 (MaxMp underflow at negative min pools)** — the MaxMp decrement now clamps via `if int(newMaxMp)-int(takeMp) < 0 { newMaxMp = 0 }`, matching the existing `newMp` guard. Regression test `TransferAP_Case15_MPtoSTR_MaxMpUnderflowClampedToZero` (level-1 pirate, minMp −37, takeMp 16 > MaxMp 10) asserts clamp-to-0 rather than ~65530.

Verification: `go build ./...`, `go vet ./character/`, and `go test ./... -count=1` all clean; `-race` clean on the changed package.
