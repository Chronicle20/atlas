# Review — fix-jms185-attack-decode

Range reviewed: `7542fc66b..5623a3a22` (7 commits), as given by the task brief.
`git diff --stat` against `origin/main` also shows ~55 unrelated files (job
carousel / character-factory / atlas-ui rework); that is because the local
`origin/main` remote-tracking ref is stale (tip `67c9279d9`) relative to the
actual merge-base `1de2e6c83` used to cut this branch — confirmed by
`git merge-base origin/main HEAD` == `7542fc66b^`. None of that churn is part
of this unit; scope for this review is the 16 files touched by the 7 commits
(`git diff --stat 7542fc66b^..5623a3a22`).

## Summary

The attack-head fix (dr blocks, single skill-data CRC, melee trailing word,
per-mob CRC), the MOVE_PLAYER 29-byte anti-cheat header, and the CMovePath
keypad+rect tail for character/npc/monster/pet are all correctly implemented:
Encode/Decode gates are textually identical everywhere, nibble packing is
correct at every boundary (0/1/2/3/4/5 entries checked by hand and in a
dedicated odd-length fixture), the tail correctly stays out of the shared
`model.Movement` and lives only in serverbound wrappers, the monster tail sits
between the movement blob and the trailing chase block as required, and the
new JMS byte fixtures are genuinely captured-wire and genuinely fail without
the fix (verified: `TestAttackMeleeRequestBytesJMS185`,
`TestCharacterMoveBytesJMS185`, `TestNPCActionBytesJMS185` all decode to
specific asserted field values and re-encode byte-identically).

One finding blocks: the MOVE_PLAYER keypad tail was widened from JMS-only to
**unconditional across all ten client versions**, but only 4 of the 9 GMS
versions (v48/v61/v72/v79) got an updated byte fixture. The 5 that didn't
(v83/v84/v87/v92/v95) still carry `packet-audit:verify` markers backed only by
a symmetric `RoundTrip` (`TestCharacterMove`) — exactly the false-positive
class this branch exists to eliminate, now reproduced inside the branch for a
production behaviour change on GMS.

## Findings

### BLOCKING

1. `libs/atlas-packet/character/serverbound/move.go:130` (`moveKeyPadTail`)
   and `move_test.go:207-209` — `moveKeyPadTail` was changed from
   `t.IsRegion("JMS")` to unconditional `true` (commit `8677d2240`), which
   changes production decode/encode behaviour for GMS v83/v84/v87/v92/v95 —
   these clients will now be expected to send/receive an extra
   `1 + ceil(count/2) + 8` bytes on every serverbound MOVE_PLAYER. The commit
   message for `8677d2240` explicitly states it "Updated the GMS Move byte
   fixtures (v48/v61/v72/v79)" — four versions only. `TestCharacterMove`
   (`move_test.go:214`) still carries
   `// packet-audit:verify packet=character/serverbound/Move version=gms_v87 ida=0xa5c937`,
   `version=gms_v95 ida=0x9a0d20`, and `version=gms_v84 ida=0xa1334e`
   (`move_test.go:207-209`), and `version_bounds_test.go:74` still carries the
   pre-existing `version=gms_v83` marker — all four backed only by
   `test.RoundTrip`, which is symmetric and passes regardless of whether the
   18-byte tail assumption for that specific version is correct. v92 has no
   marker at all but is exercised by the same unconditional gate.
   `movepath-tail-brief.md:67-70` and `movepath-tail-findings.md:60-64`
   (written by this same branch) both state the rule this violates: "Every
   widened version needs a byte fixture before its cell can be re-claimed —
   round-trips cannot catch this class of bug, which is how the jms cell held
   a false ✅ in the first place." The evidence backing the widening (9
   independent read-only decompile passes) is solid, but it was never turned
   into Go-level proof for 5 of the 9 GMS versions, and the risk is asymmetric
   in the wrong direction here: `movepath-tail-findings.md:79-82` itself notes
   that reading a tail that doesn't actually exist over-reads and desyncs the
   *next* packet — a real regression risk for any of those five versions if
   any one of the nine decompile reads is wrong, with nothing in the test
   suite that would catch it.

   Fix: add a byte-pinned fixture (following the `TestCharacterMoveByteV72`
   pattern already in this file, including a non-empty keyPadStates run) for
   each of v83/v84/v87/v92/v95 before merging, or explicitly document in the
   PR/commit why those five are being deferred and drop their
   `packet-audit:verify` markers until fixtures land so the audit trail
   doesn't claim more than is proven.

## Non-blocking

2. Evidence-tracking asymmetry: `docs/packets/evidence/jms_v185/character.serverbound.CharacterAttackMeleeRequest.yaml`
   and `.../character.serverbound.Move.yaml` were updated with a `verifies:`
   line pointing at the new byte-pinned tests (per
   `IMPLEMENTING_A_PACKET.md:356`'s documented process), but the equivalent
   evidence YAMLs for the three sibling ops that also gained brand-new
   byte-pinned tests in this same branch —
   `docs/packets/evidence/jms_v185/monster.serverbound.MonsterMovementRequest.yaml`,
   `.../pet.serverbound.PetMovementRequest.yaml`,
   `.../npc.serverbound.NpcActionRequest.yaml` — were not touched and have no
   `verifies:` field at all. `npc.serverbound.NpcActionRequest.yaml` is
   particularly notable: it's the one sibling with genuinely captured wire
   (`TestNPCActionBytesJMS185`), yet its evidence file still looks identical
   to the untouched, decompile-only monster/pet ones. `docs/packets/audits/status.json`
   / `STATUS.md` were also not regenerated, though `IMPLEMENTING_A_PACKET.md:363-365`
   says to "commit the test, the evidence YAMLs, and the regenerated
   STATUS.md/status.json together." This doesn't affect wire correctness, only
   audit-trail completeness/automation.

3. `docs/tasks/fix-jms185-attack-decode/diagnosis.md:123-128` and
   `docs/packets/audits/jms_v185/CharacterAttackMeleeRequest.md` (added note)
   correctly and explicitly scope out JMS magic/ranged attacks as still
   unverified — no finding here, called out only because it was checked and
   confirmed honest (the gates for `gmsMagicSecondaryDrBlock` /
   `gmsMagicTrailingWord` were correctly left GMS-only, per
   `libs/atlas-packet/model/attack_info.go`'s diff).

## Verified PASS (with evidence)

- **Encode/Decode symmetry.** Every gate function
  (`attackDrBlocks`, `singleSkillDataCrc`, `meleeTrailingWord`, `perMobCrc`,
  `moveDrBlocks`, `moveCrc`, `moveKeyPadTail`, `monsterMoveKeyPadTail`,
  `npcMoveKeyPadTail`, `petMoveKeyPadTail`) is called identically in both
  Encode and Decode of its owning file — checked by direct diff read of
  `libs/atlas-packet/model/attack_info.go`, `model/damage_info.go`,
  `character/serverbound/move.go`, `monster/serverbound/movement.go`,
  `npc/serverbound/action.go`, `pet/serverbound/movement.go`. No asymmetry
  found.
- **Shared-model hazard.** `git diff 7542fc66b^..5623a3a22 -- libs/atlas-packet/model/movement.go`
  is empty — `model.Movement` was not touched. All four tail
  implementations (`character`, `monster`, `npc`, `pet` serverbound wrappers)
  live outside it, each behind its own version-gate function, matching
  `movepath-tail-brief.md`'s stated constraint.
- **Monster field ordering.** `libs/atlas-packet/monster/serverbound/movement.go`
  writes/reads the tail immediately after `m.movement.*(...)` and before the
  `bChasing`/`hasTarget`/`bChasing2`/`bChasingHack`/`tChaseDuration` block in
  both Encode and Decode; `TestMonsterMovementBytesJMS185` pins this exact
  placement byte-for-byte and additionally asserts the trailing chase fields
  decode correctly after the tail (`movement_test.go`, new test).
- **Nibble packing.** Verified by hand-simulation for counts 0–5 (encode then
  decode round-trips to the original states, byte count = `1+ceil(n/2)+8` in
  every case) and confirmed against `TestCharacterMoveByteV72` (odd count = 3,
  `0x21, 0x03`) and `TestMonsterMovementBytesJMS185` (odd count = 3) /
  `TestPetMovementBytesJMS185` (even count = 4). Both encode and decode use
  the same low-nibble-first, final-byte-single-nibble convention.
- **Evidence honesty.** Every new fixture comment states its evidentiary tier
  correctly: `TestAttackMeleeRequestBytesJMS185` and
  `TestCharacterMoveBytesJMS185` and `TestNPCActionBytesJMS185` say "pinned to
  WIRE OBSERVED FROM THE LIVE CLIENT" / "live-captured jms_v185 ... wire" and
  are genuinely decode+reencode tests against real hex frames (not
  round-trips); `TestMonsterMovementBytesJMS185` and
  `TestPetMovementBytesJMS185` explicitly say "hand-built from the decompiled
  field order — this is NOT captured wire." No comment overclaims.
- **Gate scope — attack/damage.** `attackDrBlocks`, `singleSkillDataCrc`,
  `meleeTrailingWord`, `perMobCrc` are GMS-version-gated plus a JMS clause,
  each with a doc comment naming exactly which capture forces the JMS clause
  and which decompiled address corroborates it. `gmsMagicSecondaryDrBlock`
  and `gmsMagicTrailingWord` correctly stayed GMS-only (JMS magic unverified,
  explicitly noted).
- **Gate scope — movement tails.** `npcMoveKeyPadTail` and `petMoveKeyPadTail`
  and `monsterMoveKeyPadTail` are all `t.IsRegion("JMS")` only, each with a
  doc comment stating "the only client this sender was read on... do not
  extend to GMS without reading each GMS version['s]... directly" — matches
  "one binary read" scope. `moveKeyPadTail` (character) is the sole gate
  widened to all ten versions; see BLOCKING finding 1 for the fixture gap
  this creates.
- **Round-trip false positives.** `TestAttackMeleeRequest`,
  `TestNPCActionWithoutMovement`, `TestPetMovement`,
  `TestMonsterMovementVersionBoundary` all had their pre-existing
  `packet-audit:verify version=jms_v185` markers *removed* in this branch,
  each replaced with a comment explaining a RoundTrip cannot catch this class
  of bug and pointing at the new byte-pinned test that now carries the
  marker. This is the correct fix for the failure mode described in
  `diagnosis.md`. (The one place this same discipline was not fully carried
  through is `TestCharacterMove`/`gms_v83/84/87/95` — see finding 1.)
- **Build/tests.** `go build ./...` clean in `libs/atlas-packet` and in
  `services/atlas-channel/atlas.com/channel` (consumer of
  `character/serverbound.Move`); `go test ./character/... ./monster/...
  ./npc/... ./pet/... ./model/...` in `libs/atlas-packet` all pass. No
  `legacyGmsSingleCrc` / `gmsAttackDrBlocks` (renamed functions) references
  remain anywhere in the repo.
- **MovementTypesJMS185 fixture fidelity.** The 33-entry table in
  `libs/atlas-packet/test/movement_types.go` matches
  `services/atlas-configurations/seed-data/templates/template_jms_185_1.json`'s
  `opCode: 0x20` / `CharacterMoveHandle` `types` array (spot-checked the first
  10 entries verbatim).

## Not evaluable

- Whether the 9 "independent read-only IDA investigations" behind
  `movepath-tail-findings.md` were actually independent/correct — this
  review cannot re-run IDA against the referenced IDBs; it can only observe
  that the findings doc is internally consistent and that its own stated
  bar (byte fixture per widened version) was not met for 5 of 9 versions
  (finding 1).
- JMS magic/ranged attack correctness — explicitly out of scope per
  `diagnosis.md`, not touched by this branch, not evaluated here.
- Whether `docs/packets/audits/status.json` / `STATUS.md` regeneration would
  surface new orphan/drift lines — not run (`go run ./tools/packet-audit
  matrix --check` not executed as part of this review); flagged as
  non-blocking finding 2 instead.

---

```text
verdict: CHANGES_REQUIRED
artifact: docs/tasks/fix-jms185-attack-decode/review.md
scope_confirmed: 7 commits (7542fc66b..5623a3a22), 16 files under libs/atlas-packet + docs/tasks/fix-jms185-attack-decode; origin/main..HEAD noise (job-carousel/character-factory/atlas-ui) is from a stale local origin/main ref and is not part of this unit
blocking: 1
  - libs/atlas-packet/character/serverbound/move.go:130 (moveKeyPadTail widened to unconditional) and move_test.go:207-209 / version_bounds_test.go:74 — GMS v83/v84/v87/v92/v95 now expect the 18+ byte CMovePath tail but keep packet-audit:verify markers backed only by TestCharacterMove's symmetric RoundTrip; no byte fixture proves the tail assumption for those five versions, reproducing inside this branch the exact false-positive class it exists to close
non_blocking: 2
  - docs/packets/evidence/jms_v185/{monster.serverbound.MonsterMovementRequest,pet.serverbound.PetMovementRequest,npc.serverbound.NpcActionRequest}.yaml — no verifies: field added despite new byte-pinned tests in this branch, and status.json/STATUS.md not regenerated per IMPLEMENTING_A_PACKET.md's process
  - (informational, not a defect) JMS magic/ranged attacks correctly and explicitly left unverified — confirms diagnosis.md's stated scope, no action needed
not_evaluable: 2
  - correctness of the 9 external IDA decompile passes behind movepath-tail-findings.md (cannot be re-run from this review)
  - whether packet-audit matrix --check would surface new drift from the un-regenerated status.json
```
