# task-275 — Planning Context

Phase 3 companion to `plan.md`. What the implementer needs that the task
bodies do not repeat.

## Key files, as they stand before Task 1

| Path | Lines | Role |
|---|---|---|
| `libs/atlas-constants/skill/model.go` | 155 | `NeedsMasterLevel` at `:115`, its 37-line comment at `:78-114`, and `skill.Is` at `:148`. Deleted in Task 6 (the function and comment; `Is` stays). |
| `libs/atlas-constants/skill/model_test.go` | 114 | `TestNeedsMasterLevelMatchesClientRule` `:57-102` and `TestNeedsMasterLevelNotSkillBookIndexed` `:103-113` migrate to `job/master_level_test.go`. `TestIsKeyDownSkill` `:10-56` stays. |
| `libs/atlas-constants/job/advancement.go` | 23 | `Advancement` — the function `ClientJobLevel` must NOT be confused with. |
| `libs/atlas-constants/constants/for.go` | 53 | `For(region string, major, minor uint16)` — the decomposed-scalar idiom D1 matches, and the `strings.ToUpper` normalisation at `:40`. |
| `libs/atlas-packet/character/data.go` | 862 | `isEvanJob` `:268-272`; extended-SP gate `:316`/`:397`; master-level call `:670`/`:696`; `SkillEntry.NeedsMasterLevel` field comment `:69-74`. |
| `libs/atlas-packet/character/data_evan_test.go` | 122 | `TestIsEvanJob` `:13-34` (deleted, Task 5); `TestEvanExtendedSPv84` `:40-73` and `TestEvanSkillMasterLevelLength` `:91-122` (kept, unchanged). |
| `libs/atlas-packet/character/data_test.go` | ~450 | `TestCharacterDataWithSkillsRoundTrip` `:242-283`; two `skill.NeedsMasterLevel(..., true)` calls at `:275-276`. |
| `libs/atlas-packet/test/context.go` | — | `pt.Variants` `:18-41`, `pt.CreateContext` `:43`, `pt.Encode` `:129`, `pt.RoundTrip` `:135`. |
| `services/atlas-channel/.../socket/writer/character_data.go` | 199 | `:72-80` — comment references only; no code change. |

## Decisions carried in from design.md

- **D1 signature is `(region string, major uint16)`, not `tenant.Model`.** This
  is a deliberate departure from PRD FR-1's "takes the tenant". The binding
  reason is `libs/atlas-constants/go.mod`: it has exactly one non-indirect
  dependency (`github.com/google/uuid`) and must not gain an
  `atlas-tenant` edge. `constants.For` already takes decomposed scalars for
  the same reason. If the "takes the tenant" line turns out to have been
  load-bearing, this is the decision to revisit — not something to "fix" mid-
  execution.
- **Both predicates live in `libs/atlas-constants/job`, not `skill`.**
  Confirmed in the tree: `job/model.go:6`, `job/identity.go:6` already import
  `skill`, so `job` → `skill` is the existing direction and `NeedsMasterLevel`
  taking a `skill.Id` from inside `job` is cycle-free. The reverse (`skill`
  importing `job`) would be a cycle, which is why the function has to move.
- **`ClientJobLevel` is a fresh port, not a reuse of `job.Advancement`.**
  `Advancement` returns -1 for the whole Evan stage line (`advancement.go:9`)
  and has no 43x branch; both of those are exactly the cases this task needs.
- **The v95 `is_ignore_master_level_for_common` list is in scope** (design
  §2.3), which grows the task past the PRD's §2 non-goal. Fourteen of its
  sixteen ids belong to jobs an Atlas tenant can create, so the pre-task
  encoder writes a 4-byte master-level int on GMS v95 that the client does not
  read. This is the one place FR-9's "no verified GMS cell moves a byte" is
  knowingly violated — because the currently-verified bytes are wrong.

## Sequencing constraints

1. **Task 1 must land first and must be captured from the unmodified
   encoder.** Its goldens are worthless if generated after a code change. The
   character is job 312 with skills `3110000`/`3121002`, chosen to be outside
   every arm this task adds — including the v95 ignore list, whose job-312
   members are `3120010` and `3120011`.
2. **Task 5 before Task 6.** `skill.NeedsMasterLevel` stays in the tree until
   its last caller is gone, so every commit builds.
3. Tasks 2 → 3 → 4 are ordered by symbol dependency (`ClientJobLevel` and
   `isEvanJob` are consumed by `NeedsMasterLevel`).

## Task sizing notes

No task was deliberately left oversized. Task 5 is the widest at four files
across one module (`libs/atlas-packet`); it is not split because encode and
decode must move to the same authority in a single commit (FR-8) and because
deleting `isEvanJob` breaks compilation until both gates are retargeted.

Task 8 spans `services/atlas-channel` and the repo-root gates, so
`plan-lint.sh` F4 may flag it as multi-service. It is a comment edit plus
read-only verification, not implementation, and splitting it would separate
the final gate from the last change it gates.

## Evidence re-pin

**Finding (Task 8 Step 2) — to be confirmed by the implementer and recorded
here.** The planning read of the artifacts says no re-pin is owed:

- `docs/packets/evidence/gms_v95/field.clientbound.FieldSetField.yaml` pins
  only `ida.function` (`CStage::OnSetField`), `ida.address` (`0x71a0a0`) and
  `decompile_sha256` — a client-side hash no server change can move. There is
  no byte fixture in the record.
- `libs/atlas-packet/field/clientbound/set_field_test.go` carries the
  `packet-audit:verify` markers for all seven SET_FIELD cells (`:13-19`,
  `:151`), and each golden asserts the CharacterData middle as an **opaque
  span** re-derived by calling `cd.Encode` (`:76`) rather than as a byte
  literal. Its fixtures are job 312 with **no skills**, so neither the ignore
  list nor any 43x arm can reach them.

If the implementer finds otherwise — a hard-coded CharacterData byte literal,
or a fixture holding one of the sixteen ignore-list ids — that is a
`packet-verifier` re-derivation, not a comment edit. Stop and report.

## Out of scope, recorded (design §9)

1. `libs/atlas-packet/stat/clientbound/changed.go` carries the same
   extended-SP divergence on the `OnStatChanged` path, but its model holds
   only `(statType, value)` pairs with **no job id in scope** (`changed.go:30-40`),
   so it cannot read the fixed helper. Fixing it needs the character's job
   threaded into the stat-change model — a data-model change the PRD excludes.
   Separate task.
2. Modelling real extended-SP contents. Atlas writes a count of 0 because it
   has no per-master-level SP allocation model; the decode-side discard loop
   is retained so a client-authored nonzero count still parses.
3. Jobs 3212 / 3312 in the v95 ignore list — modelled for fidelity, bound by
   no Atlas identity.
4. Dual Blade gameplay. `job/model.go:113` still carries
   `// jobId = job.BladeRecruit TODO`; creating a Dual Blade is a separate
   task. This one only makes the wire correct if one exists.
