# task-152 Mortal Blow — Whole-Branch Code Review

Range: `c09233e3f..7b36f9aa3` (8 commits). Reviewed via the full diff plus direct reads of the
current source (`character_attack_common.go`, `processor.go` in both services, `registry.go`,
`kafka.go`/`consumer.go` in both services, `information` package, PRD).

### Strengths

- **Wire contract is exact.** Channel's `KillCommandBody{CharacterId, SkillId}` (`kafka/message/monster/kafka.go:94-97`)
  and atlas-monsters' `killCommandBody{CharacterId, SkillId}` (`kafka/consumer/monster/kafka.go:110-113`)
  have identical JSON tags (`characterId`, `skillId`); `CommandTypeKill = "KILL"` matches on both
  sides; the enclosing `Command[E]`/`command[E]` envelopes are already structurally identical
  (`monsterId`, `type`, `body`). `handleKillCommand` (`kafka/consumer/monster/consumer.go:177-184`)
  decodes `c.MonsterId`/`c.Body.CharacterId`/`c.Body.SkillId` correctly. No decode mismatch.
- **Partition ordering claim verified.** Both `DamageCommandProvider` and `KillCommandProvider`
  key with `producer.CreateKey(int(monsterId))` (`monster/producer.go:130,176`), and in
  `processAttack` the `deps.applyDamage` call (which emits DAMAGE) happens synchronously *before*
  `deps.onDamageApplied` fires (which triggers `mortalBlowTryProc` → KILL) — so DAMAGE always
  precedes KILL on the same partition, matching the "if the attack itself already killed the
  monster, KILL finds it gone" design.
- **Boss guard is genuinely fail-closed and cannot be bypassed by the channel.** `ProcessorImpl.Kill`
  (`monster/processor.go:1723-1737`) returns immediately on `infoErr != nil` — it never falls
  through to read `info.Boss()` on a zero-value `Model`. Confirmed against `DrainMp`
  (`processor.go:1649-1699`), which is genuinely fail-*open* (`infoErr == nil && infoModel.Boss()`
  — an error simply skips the boss check) — the documented divergence is accurate, not just
  asserted.
- **`damageCore` split is behavior-preserving.** Diffed byte-for-byte against the plan's
  Main-Merge Reconciliation block: `Damage` now does alive+reflect then calls `damageCore`; the
  moved tail (info fetch → aggro emit) is unchanged including the GM-hidden controller-switch
  guard and the `information.NewProcessor(...).GetById(...)` non-curried call. The only
  substitution is `id`→`m.UniqueId()` in the three kill-path cleanup calls, which is a no-op
  (`m` was fetched by `id`).
- **`Kill` reuses `damageCore` correctly**: no `checkReflect` call (a kill "attack" has no attack
  type — correct), and the `math.MaxUint32` line is safe because `Registry.ApplyDamage`
  (`registry.go:436-443`) clamps `actual = min(damage, hp)` before recording the damage entry —
  independently verified by reading `registry.go`, not just cited from the plan. `kill_test.go`'s
  `TestKill_NonBoss_KilledAndRemoved` asserts the clamped value (5000, not MaxUint32) lands in the
  KILLED event body — a real behavioral check, not a tautology.
- **`onDamageApplied` closure correctly preserves MP Eater / drain / Pick Pocket and adds Mortal
  Blow as a fourth independent `if` branch** (`character_attack_common.go:758-777`); reflected and
  status-only `DamageInfo` entries never reach the callback (verified both by reading
  `processDamageInfoEntry`, `character_attack_common.go:117-210`, and by the three
  `TestProcessDamageInfoEntry_*` tests exercising that gate through the real function, not a
  reimplementation).
- **Ownership/effect-level resolution verified.** `se` is resolved via
  `skill2.NewProcessor(...).GetEffect(ai.SkillId(), sk.Level())` only after the ownership check at
  `character_attack_common.go:658-661` destroys the session for unowned skill ids — the
  `isMortalBlowAttack` doc comment's claim ("upstream ownership guard... sufficient") is accurate,
  not just asserted.
- **No numeric literals in production code.** `3110001`/`3210001` appear only in comments;
  `skill3.RangerMortalBlowId`/`SniperMortalBlowId` are used everywhere else and both constants
  exist in `libs/atlas-constants/skill/constants.go:3102` and neighboring lines.
- **Test quality is good, not tautological.** Pure-function tests (`mortalBlowEligible`,
  `mortalBlowKillRoll`, `isMortalBlowAttack`) hit boundary cases including a genuine `uint64`
  overflow probe at `MaxUint32`. Flow tests via `mortalBlowDeps` pin every branch (inert effect
  skips the snapshot fetch entirely, snapshot error swallowed, above-threshold no-roll, roll-fail
  no-emit, proc-emits-with-correct-args, emit-error swallowed). Monsters-side `kill_test.go` covers
  the full guard matrix: non-boss kill+removal+clamped-credit, boss drop (zero events, HP
  untouched), fail-closed info-lookup error (zero events), missing monster no-op, dead-monster
  no-op. This is exactly the failure-isolation surface the PRD's FR-5/FR-4 require.
- **Version-agnostic design is real, not just claimed.** The attack packet writes `skillId`
  ungated across every version (cited and independently plausible given `isMortalBlowAttack`
  gates purely on `ai.SkillId()`/`ai.AttackType()`, both populated identically regardless of the
  legacy CRC/action-byte-width gates the concurrent v48/61/72/79 bring-up touched).
- **Builder pattern / no test-helper files honored.** Both new test files use
  `monster.NewModelBuilder`/`information.NewModelBuilder`/`effect.Extract`/`field.NewBuilder`
  directly; no `*_testhelpers.go` was added.
- **The stale `// TODO Mortal Blow` and the stale task-007 arrow-consumption comment were both
  correctly removed/fixed** (`character_attack_common.go` TODO block; `character_attack_projectile.go:119-123`),
  satisfying "no TODOs in deliverables" for the feature's own scope.

### Issues

#### Critical (Must Fix)

None found.

#### Important (Should Fix)

None found. (One item that looked like it might qualify — a mismatched "Cosmic parity" formula
citation — turned out to be correct on closer reading; see Minor below for the residual
documentation note.)

#### Minor (Nice to Have)

1. **Redundant `information` lookup per Mortal Blow kill.** `ProcessorImpl.Kill`
   (`monster/processor.go:1723-1737`) fetches `information.Model` once, authoritatively, for the
   boss guard. `damageCore` (`monster/processor.go:573-580`) then fetches it *again* — via the same
   `information.NewProcessor(p.l, p.ctx).GetById(m.MonsterId())` call, not honoring
   `testInformationLookup` — purely to populate `isBoss`/`revives` for the emitted event bodies.
   This is a second HTTP round-trip to atlas-data per Mortal Blow kill (harmless functionally,
   since `damageCore`'s own fetch failing just defaults `isBoss=false`/`revives=nil` on an event
   whose boss-guard already passed as non-boss — same value, no behavior change — but it is an
   avoidable extra external call on every proc). Not worth blocking the PR on; a follow-up could
   thread the already-fetched `information.Model` from `Kill` into `damageCore` as an optional
   parameter if this ever shows up in latency/error-budget numbers.
2. **Same-formula "Cosmic parity" comment could read as internally inconsistent without the PRD
   in hand.** `mortalBlowEligible`'s doc (`character_attack_common.go:376-381`) and the test file's
   citation (`character_attack_mortal_blow_test.go:34`, `Cosmic parity: (getStats().getHp() *
   getX()) / 100`) use `getStats().getHp()` to mean the monster's *stat-block* (max) HP, not
   current HP — a MapleStory-server-code idiom that's correct once you know it (confirmed against
   PRD §95, which is itself marked IDA-verified 2026-07-10, and against the implemented formula
   `hp <= maxHp*x/100`), but reads ambiguously in isolation next to a parameter named `maxHp`. Not
   a functional issue — the code and the PRD agree — but a one-clause gloss ("`getStats().getHp()`
   = the monster's max/base HP, not its current HP") would save the next reader a PRD round-trip.
3. **Pre-existing TOCTOU between `Kill`'s alive check and `damageCore`'s `ApplyDamage`, shared with
   `Damage`.** `Kill` checks `m.Alive()` at `monster/processor.go:1730` then calls `damageCore`,
   which calls `GetMonsterRegistry().ApplyDamage` moments later; a concurrent kill of the same
   monster between those two calls is possible in principle. This is the exact same race `Damage`
   already has (unchanged by this refactor — `Damage`'s alive check and its own `ApplyDamage` call
   have the identical gap) and `ApplyDamage`'s clamp-to-zero makes a second concurrent hit a no-op
   HP-wise, so the practical blast radius is a possible duplicate `KILLED` emission under a very
   tight race, not a double-kill or double-credit. Flagging only because Mortal Blow is a new
   caller into this path, not because the branch introduces a new failure mode; not a
   merge-blocker.

### Recommendations

- Optional follow-up (not blocking): thread the boss/`information.Model` fetched in `Kill` into
  `damageCore` to avoid the double atlas-data round-trip noted above.
- Optional follow-up (not blocking): a short one-line gloss on the `getStats().getHp()` comment
  citation would remove the need for a reader to cross-reference the PRD.

### Assessment

**Ready to merge?** Yes

**Reasoning:** All five focus areas (wire contract, fail-closed boss guard, `damageCore` reuse
without reflect/mis-credit, `onDamageApplied` closure preservation, and cross-commit seams) check
out against the live source, not just the diff/plan narrative; no critical or important issues
were found, and the two minor items are pre-existing-pattern or documentation-clarity notes that
don't affect correctness.
