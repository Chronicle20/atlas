# FR-1.5 — COMMAND_TOPIC_CHARACTER_BUFF producer audit

Every service producing onto `COMMAND_TOPIC_CHARACTER_BUFF`, audited for the
seconds-into-a-milliseconds-field defect. Contract owner:
`services/atlas-buffs/atlas.com/buffs/kafka/message/character/kafka.go`
(`ApplyCommandBody.Duration` — milliseconds).

| Service | Verdict | Evidence (file:line) | Notes |
|---|---|---|---|
| atlas-monsters | **fixed** | `monster/processor.go:1091` (`buildMistCreateBody`, `Duration: durMs`), `:1109` (`executeStatBuff`, `duration := time.Duration(sd.Duration()) * time.Millisecond`), `:1246` (`executeDebuff`, `duration := int32(sd.Duration())`) | `sd.Duration()` (`mobskill.Model.Duration()`) is milliseconds since task-190 FR-1.1. Two double-conversions in `buildMistCreateBody`/`executeStatBuff` were removed; `executeDebuff` forwards `sd.Duration()` verbatim and needed no edit (FR-1.3) — confirmed unchanged in this task. |
| atlas-maps | **fixed** | `tasks/mist_tick.go:91` | `Duration: int32(m.DiseaseDuration().Milliseconds())` — reverses commit `11e07dfa7`; call-site comment (lines 80-90) already documents the contract, and this task additionally added the one-line struct-field pointer at line 58 above `applyDiseaseBody.Duration`. |
| atlas-channel | correct | `skill/handler/common.go:162` (`buff.NewProcessor(l, ctx).Apply(f, characterId, ..., e.Duration(), statupsToApply)`), `skill/handler/mysticdoor/mysticdoor.go:73,127` (`applyDoorBuff(..., e.Duration(), ...)` forwarding into `Apply(f, characterId, int32(skillId), level, duration, statups)`) | `data/skill/effect/model.go:81` documents `Duration()` as milliseconds (task-054). `skill/handler/mount.go:23` `MountBuffDuration` and `skill/handler/hide/hide.go:35` `HideBuffDuration` are `int32(math.MaxInt32)` sentinels — unit-agnostic, not derived from any WZ/seconds value. Correction from the brief's lead list: the file is `skill/handler/mysticdoor/mysticdoor.go` (nested `mysticdoor/` package directory), not a flat `skill/handler/mysticdoor.go` — no flat file of that name exists. |
| atlas-consumables | correct | `consumable/processor.go:212-217` (`plan.duration = val`, comment: "The consumable `time` spec is already in milliseconds"), `character/buff/producer.go:35` (`Duration: duration`) | Matches task-140 fix (`88d270bf1`); `plan.duration` flows unchanged into the `ApplyCommandBody.Duration` field. |
| atlas-summons | correct | `data/skill/effect/model.go:43` (`func (m Model) Duration() int32 { return m.duration }`, doc comment: "Duration returns the effect duration in milliseconds"), `summon/processor.go:188` (`SetBuffDuration(hex.Duration())`), `summon/beholder_task.go:134` (`buffmsg.ApplyProvider(..., m.BuffDuration(), ...)`), `buff/producer.go:83` | `hex.Duration()` (ms) is stored as `summon.Model.buffDuration` and forwarded unchanged through `beholder_task.go` into `ApplyProvider`'s `Duration` field. |
| atlas-messages | correct | `buff/processor.go:33-44` (`Apply` — `duration := effect.Duration()`; `if durationOverride > 0 { duration = durationOverride * 1000 }`), `command/buff/commands.go:45-51` (`durationOverride = int32(dur)` parsed from the `@buff <target> <skill> [duration]` GM chat command regex) | Two paths, both correctly ms on the wire: the default path forwards `effect.Duration()` (already ms) unchanged; the GM-override path parses a human-typed **seconds** value from chat input and explicitly multiplies by 1000 before assigning — this is a deliberate seconds→ms conversion for a human input surface, not a defect. Verified by reading both the multiply site and its only caller. |
| atlas-saga-orchestrator | n-a | `kafka/message/buff/kafka.go:26` (`CancelAllBody struct{}`), `buff/processor.go:15-21` (`Processor` interface: `CancelAllAndEmit`/`CancelAll` only) | Produces only `CANCEL_ALL` onto `COMMAND_TOPIC_CHARACTER_BUFF`; confirmed by reading the full command body set in this service's kafka.go — there is no `ApplyCommandBody` and no `Duration` field anywhere in the package. Pointer comment correctly not added (per brief Step 2).

## Range check

Maximum authored WZ mob-skill `time` observed: **6000** seconds, read from a
live atlas-data query (`GET /api/data/mob-skills`, paged at `page[size]=250`,
509 rows total — see `.superpowers/sdd/plan/task-1-report.md`). Not
re-derived in this task per the controller's instruction; cited verbatim.

The binding constraint is `int32` (`executeDebuff` narrows `uint32` →
`int32`), so overflow after the ×1000 needs `time > 2_147_483` seconds ≈
24.8 days. The observed maximum of 6000 seconds is ≈358× below that bound
(2,147,483 / 6000 ≈ 357.9).

## FR-3.2 Step 9 — real-tree guard run and triage

`./tools/buff-duration-guard.sh` run from the repo root against the full
tree: **exit 0, zero diagnostics**, across all 79 `services/` and `libs/`
Go modules. This matches Step 9's expectation ("Tasks 2 and 3 already
removed both historical defects").

### Investigated: `atlas-messages` `buff/processor.go:42-44`

The controller flagged this site ahead of time as one the analyzer might
report. It was re-verified independently in this task (not taken on the
controller's word):

- `services/atlas-messages/atlas.com/messages/command/buff/commands.go:29`
  — the GM chat command regex `^@buff\s+(\w+)\s+"?([^"]+)"?(?:\s+(\d+))?$`
  for `@buff <target> <skill> [duration]`.
- `commands.go:45-51` — `durationOverride` is parsed via
  `strconv.Atoi(match[3])` directly from that regex-captured group, i.e. a
  human-typed value.
- `commands.go:180` — `bp.Apply(f, id, c.Id(), foundSkill.Id(), maxLevel, durationOverride)`
  is the only caller.
- `buff/processor.go:33-45` — `Apply`: `duration := effect.Duration()`
  (already ms); `if durationOverride > 0 { duration = durationOverride * 1000 }`
  converts the human-typed seconds value to ms.
- `kafka/message/buff/kafka.go:44-61` — `ApplyCommandProvider` builds the
  `ApplyCommandBody{... Duration: duration ...}` composite literal from the
  `duration` **function parameter**.

Verdict: **confirmed correct**, not a defect — this is the one legitimate
seconds-authored input surface for `ApplyCommandBody.Duration` (a human
types seconds at the GM console; the wire field is ms).

**Why the guard did not flag it (verified empirically, not assumed):** the
composite literal that trips BD-1 lives in
`kafka/message/buff/kafka.go`, a different Go package from
`buff/processor.go` where `duration = durationOverride * 1000` executes.
`buffdurationguard`'s `collectAssignments` walks `pass.Files`, which is
scoped to a single package per analysis pass; the "one hop into the local
assignment" rule cannot cross a function-call boundary into another
package. Confirmed by building the analyzer and running it directly against
`services/atlas-messages/atlas.com/messages` both before and after adding
the annotation below — 0 diagnostics either way. This is a real scope
limit of the analyzer (same-package, one-hop local-variable tracing only,
per the brief's design), not a bug in this implementation.

**Action taken:** added a `//buffdurationguard:allow` annotation directly
above the `duration = durationOverride * 1000` line in
`services/atlas-messages/atlas.com/messages/buff/processor.go`, plus an
explanatory comment naming the GM chat command as the seconds-authored
source. This is documentation for human readers (and for any future
strengthening of the analyzer to cross-package tracing) — it does not
suppress an active diagnostic, since none fires there today.

No other diagnostics were produced anywhere in the tree; there was nothing
else to triage.

## FR-3.2 guard demonstration

Reintroduced the exact pre-task-190 form in atlas-maps:

```
$ git rev-parse --abbrev-ref HEAD
task-190-disease-duration-cancel-debuff
$ sed -i 's|Duration: int32(m.DiseaseDuration().Milliseconds()),|Duration: int32(m.DiseaseDuration() / time.Second),|' \
  services/atlas-maps/atlas.com/maps/tasks/mist_tick.go
$ ./tools/buff-duration-guard.sh; echo "exit=$?"
...
buffdurationguard: <repo-root>/services/atlas-maps/atlas.com/maps
<repo-root>/services/atlas-maps/atlas.com/maps/tasks/mist_tick.go:92:42: buffdurationguard: duration fields on the character-buff command are MILLISECONDS; drop the seconds-to-ms scaling. Contract owner: atlas-buffs kafka/message/character/kafka.go (or annotate with //buffdurationguard:allow <justification>)
...
buffdurationguard: FAIL — seconds-valued buff duration emitter found
  The COMMAND_TOPIC_CHARACTER_BUFF duration field is MILLISECONDS.
  Contract owner: services/atlas-buffs/atlas.com/buffs/kafka/message/character/kafka.go
exit=1
```

Restored:

```
$ git checkout -- services/atlas-maps/atlas.com/maps/tasks/mist_tick.go
$ ./tools/buff-duration-guard.sh; echo "exit=$?"
...
exit=0
```
