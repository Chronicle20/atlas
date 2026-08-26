# report: bug-zombify-no-visible-effect — encoder fix

Status: DONE
Commit: `2e91e814d` — fix(atlas-packet,atlas-messages): populate UNDEAD rOption so zombify renders
Agent: task-implementer (sonnet)

> Recorded by the controller. The implementing agent's `Write` tool refused the
> report path ("Subagents should return findings as text, not write report
> files"), overriding the brief. Content below is its returned report; the
> controller verified the commit, diff and constant usage before recording it.

## What changed

1. `libs/atlas-packet/model/character_temporary_stat.go` — `getBaseTemporaryStats`,
   `twoStateDynamic` branch. New `UNDEAD` case emitting
   `NewCharacterTemporaryStatBaseWithOptions(true, s.Value(), composite, narrow)`
   where `composite = s.SourceId() | (int32(s.Level()) << 16)`. `DashSpeed` /
   `DashJump` keep the zeroed default; `UNDEAD` stays in `baseStatNames`. The old
   "no evidence was gathered" comment is replaced with the IDA citations
   (`CUser::UpdateAffectedSkillList` @0x93e344, `sub_672293` @0x672293,
   `TemporaryStatBase<long>::DecodeForClient` @0x793ef2,
   `CWvsContext::OnTemporaryStatSet` @0xa202be) and a note that the icon half is
   unconfirmed.
2. `libs/atlas-packet/model/character_temporary_stat_test.go` — `TestCTSDashSpeedStaysZeroed`
   comment no longer claims UNDEAD stays zeroed. New `TestCTSUndeadPopulatedBlock`
   (pins `nOption=1`, `rOption=327813` = `133 | 5<<16`) and `TestCTSUndeadRoundTrip`
   (GMS v83/v95, JMS v185; v61 excluded — no Undead slot in its 6-member group).
3. `services/atlas-messages/atlas.com/messages/command/disease/commands.go` — new
   `diseaseMobSkill(statName) (sourceId int32, level byte)` returning
   `monster.SkillTypeUndead` for UNDEAD and `0` otherwise, wired into
   `DiseaseCommandProducer` in place of the hardcoded `0, 0, 1`.
4. `services/atlas-messages/atlas.com/messages/command/disease/commands_test.go` —
   pins UNDEAD's sourceId/level and guards that other diseases stay at `sourceId=0`.

## Tests

- `libs/atlas-packet`: `go build ./...` clean; `go test ./...` all packages ok.
- `services/atlas-messages/atlas.com/messages`: `go build ./...` clean;
  `go test ./...` all packages ok.

Module-local only, per the brief. `tools/verify.sh` deliberately not run by the
implementer.

## No pre-existing fixture needed re-pinning

Claimed by the implementer and consistent with the change: every pre-existing
UNDEAD-touching test used `sourceId=0, level=0`, so the composite is still `0` and
the emitted bytes are identical to the old zeroed block. The full suite was green
both before and after. **This is the load-bearing claim of the change** — it is
what makes the wire change a no-op for every already-verified cell — and it is the
first thing the reviewer should attack.

## Affected packet-audit coverage cells

Not touched, per the brief; these need a `packet-verifier` pass:

- `BuffGive` / `BuffGiveForeign`, `.md` + `.json`, for gms_v72, gms_v79, gms_v83,
  gms_v84, gms_v87, gms_v92, gms_v95, jms_v185 — every version where UNDEAD is a
  two-state group member.
- `gms_v48` uses the legacy mask path and never reaches this block **by
  inference** — flag for the verifier to confirm rather than assume.
- `docs/packets/audits/OPAQUE_LEDGER.md` cites the test file generally, not a byte
  pin — flag for the verifier.

## Controller verification of this report

- `git show --stat 2e91e814d` — 4 files, +133/-10, matching the list above.
- The `UNDEAD` hunk reads `composite := s.SourceId() | (int32(s.Level()) << 16)`
  and is guarded by `if bs.name == character.TemporaryStatTypeUndead`, with the
  `DashSpeed`/`DashJump` zeroed default preserved below it.
- `libs/atlas-constants/monster/skill.go:44` defines `SkillTypeUndead = 133`; the
  command uses `int32(monsterconstant.SkillTypeUndead)`, not a literal.

## Known-open

The icon half. Populating `rOption` restores the animation path unconditionally.
The self-view disease icon depends on whether `SecondaryStat::DecodeForLocal`
negates the VIEWELEM reason for the Undead slot, which is unread. Live re-test is
the next discriminator, not more static analysis.
