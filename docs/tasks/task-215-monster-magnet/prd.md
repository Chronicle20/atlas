# Monster Magnet — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-12
---

## 1. Overview

Monster Magnet is the 3rd-job warrior crowd-control skill shared by Hero
(`1121001`), Paladin (`1221001`) and Dark Knight (`1321001`). The player holds
the key down, the client picks up to *N* nearby monsters, rolls a per-monster
grab result, drags the successful ones toward the caster, and takes over
control of them. On Atlas today the skill does nothing.

The failure is at the wire boundary. Monster Magnet arrives on the `SPECIAL_MOVE`
serverbound opcode and is decoded by the shared
`libs/atlas-packet/model/skill_usage_info.go` `SkillUsageInfo.Decode`. That
decoder branches on skill id for anti-repeat cast X/Y, Night Lord Shadow Stars,
the party-buff bitmap, and the mob-affecting buff list — but it has **no magnet
branch**. The magnet payload (a per-monster `(objectId, grabResult)` table and a
trailing direction byte) is therefore never read. The reader is left short of the
end of the body, and no handler is registered for the three identities, so the
cast produces zero server-side effect.

This task implements Monster Magnet end to end: the decode branch across all ten
provisioned client versions, a validated server-side application step, the
per-monster grab effect broadcast, aggro wipe, and forced controller handover to
the caster. Two of the four pieces are already built and merely lack a caller —
the `CATCH_MONSTER` codec/writer/template routes were deliberately retained with
no sender by task-212 precisely because that packet is the Monster Magnet grab
effect, and the keydown prepare/keyup relay already covers these three skill ids.

## 2. Goals

Primary goals:

- Decode the Monster Magnet payload correctly on all ten provisioned versions
  (gms v48, v61, v72, v79, v83, v84, v87, v92, v95, and jms_185), with the read
  order derived per version from the client binary rather than assumed.
- Make a Monster Magnet cast produce its four observable server effects:
  the blue grab animation on each grabbed monster, a wiped damage-aggro table on
  each grabbed monster, controller handover of each grabbed monster to the
  caster, and the remote skill-effect animation on the caster.
- Validate the client's claimed target set server-side rather than trusting it.
- Give atlas-monsters two reusable, orthogonal command surfaces — clear-aggro and
  force-control — rather than one magnet-shaped command.
- Pin byte-fixture verification evidence for the Monster Magnet arm of
  `SPECIAL_MOVE` on every version.

Non-goals:

- Server-side repositioning of grabbed monsters. The client that receives
  control performs the drag; the server does not move the monsters. (Matches the
  reference implementation, which also does not move them.)
- Promoting the `SPECIAL_MOVE` row in the coverage matrix to `✅`. That opcode
  carries sixteen distinct fnames (`CGrenade::SendTimeBombInfo`,
  `CUserLocal::DoActiveSkill_*`, `CUserLocal::TryDoingSwallowAbsorb`, …); only
  the `CUserLocal::TryDoingMonsterMagnet` arm is in scope here.
- `CUserLocal::TryDoingSwallowAbsorb` and the other unimplemented
  `DoActiveSkill_*` arms of the same opcode.
- Any change to the keydown prepare/keyup relay (already working — see §7.4).
- Damage. Monster Magnet deals none.

## 3. User Stories

- As a Hero/Paladin/Dark Knight, I want Monster Magnet to visibly grab nearby
  monsters so the skill is usable for crowd control instead of being a dead
  keybind.
- As a party member of the caster, I want to see the caster's magnet animation
  and the grabbed monsters' grab effect so the fight reads correctly on my
  screen.
- As the caster, I want to end up controlling the monsters I grabbed so they
  respond to my client immediately rather than lagging behind their previous
  controller.
- As a server operator, I want a client that reports impossible magnet targets
  (too many, or out of the skill's rect) to be rejected and logged, not obeyed.
- As a packet maintainer, I want the Monster Magnet arm of `SPECIAL_MOVE` backed
  by a pinned byte fixture per version so a future decoder edit that breaks it
  fails CI rather than the game.

## 4. Functional Requirements

### FR-1 — Wire decode (`libs/atlas-packet`)

- **FR-1.1** `SkillUsageInfo.Decode` gains a Monster Magnet branch, entered when
  the decoded skill id resolves to `skill.HeroMonsterMagnet`,
  `skill.PaladinMonsterMagnet`, or `skill.DarkKnightMonsterMagnet`.
- **FR-1.2** The branch reads the magnet grab table: a count, then that many
  `(monsterObjectId, grabResult)` entries, followed by the trailing direction
  byte. The exact widths, order, and whether the shared trailing `delay` short is
  also present **MUST be derived from the client binary per version** (see
  FR-1.4) — this PRD deliberately does not assert them.
- **FR-1.3** The magnet branch is mutually exclusive with the existing
  `isMobAffectingBuff` branch: none of the three magnet ids appear in
  `isMobAffectingBuff`, `isPartyBuff`, or `isAntiRepeatBuffSkill` today, and none
  may be added. A magnet cast must consume the magnet table, never the plain
  `affectedMobIds` array.
- **FR-1.4** Version gating follows the established idiom in this file: region
  test plus `MajorAtLeast`, never a raw `> N` comparison, with an inline comment
  naming the decompiled address per version that justifies the gate. The existing
  `isAntiRepeatBuffSkill` gate in this same function is the reference for both
  the idiom and the comment style. If the payload shape proves identical across
  all ten versions, that fact must itself be stated in a comment with the
  per-version addresses that establish it.
- **FR-1.5** New model state is exposed through getters consistent with the rest
  of the type (`MagnetGrabs()`, `Direction()`), and through
  `SkillUsageInfoBuilder` setters so tests can construct values without going
  through the wire.
- **FR-1.6** A grab entry is an immutable value type carrying the monster object
  id and the grab result, with getters — not a bare tuple or a map.

### FR-2 — Target validation (`atlas-channel`)

- **FR-2.1** The server does **not** trust the client's claimed target set. A
  claimed monster is accepted only if it passes the same class of checks the
  existing `skill/handler/common.go` `applyToMobs` performs for mob-affecting
  buffs: the WZ `mobCount` cap, and membership in the skill's effect bounding box
  projected from the caster's position and facing.
- **FR-2.2** Exceeding the WZ `mobCount` cap rejects the **entire** cast (zero
  monsters grabbed), matching the existing FR-4.3 policy in `applyToMobs`.
- **FR-2.3** A claimed monster outside the server-computed rect is dropped
  individually and logged as an anomaly; the rest of the cast proceeds. Matches
  the existing out-of-rect policy.
- **FR-2.4** If the skill's WZ effect has no bounding box (`lt`/`rb` absent), the
  rect check is skipped and the client's list is accepted subject to the cap
  only, with a debug log — matching the existing FR-4.2 fallback. Whether Monster
  Magnet's WZ effect actually carries `lt`/`rb` must be verified against local WZ
  data during design, not assumed.
- **FR-2.5** A monster the client reports as a **failed** grab is not acted on:
  no effect broadcast, no aggro clear, no control handover.
- **FR-2.6** The validation logic shared with `applyToMobs` is factored into a
  reusable helper rather than copy-pasted. Monster Magnet must not take the
  status-apply or status-cancel branch of `applyToMobs` — it applies no monster
  status, and the reflect-skip and prop-roll steps do not apply to it.
- **FR-2.7** Failure to load the caster or to run the rect query drops the whole
  cast and logs an error, matching the existing bail-on-error policy.

### FR-3 — Grab effect broadcast (`atlas-channel`)

- **FR-3.1** For each validated, successfully-grabbed monster, the server
  broadcasts `CATCH_MONSTER` to every session in the field.
- **FR-3.2** This reuses the existing `writer.CatchMonsterBody` /
  `monsterpkt.CatchMonster` codec and its existing template routes in all nine
  seed templates. **No new codec, writer, or template entry is required for the
  grab effect** — task-212 established that `CATCH_MONSTER` (`CMob::OnCatchEffect`)
  is the blue Monster Magnet grab render and left it senderless for this task.
- **FR-3.3** `CATCH_MONSTER_WITH_ITEM` must **not** be sent. That is the
  item-keyed capture render owned by task-212's catch-item flow; sending both
  stacks two animations.
- **FR-3.4** The `result`/`success` arguments to the writer are populated from
  the client-reported grab result for that monster, after validation.

### FR-4 — Aggro clear (`atlas-monsters`)

- **FR-4.1** A new `COMMAND_TOPIC_MONSTER` command type clears a monster's
  accumulated damage-aggro table.
- **FR-4.2** The clear is a **full wipe** of all damage entries for that monster
  — every character's entry, not just the caster's, and not a decay toward the
  aggro floor.
- **FR-4.3** The command is orthogonal and generally reusable: its body carries
  no magnet-specific fields.
- **FR-4.4** The wipe interacts correctly with the existing aggro decay sweep
  (`AggroSweepInterval`, `AggroDecayMultiplier`, `AggroDecayFloor`) and with the
  controller picker — a wiped table must not leave the picker holding a stale
  reference or repicking on empty state.
- **FR-4.5** The command is idempotent: wiping an already-empty table is a
  no-op, not an error.
- **FR-4.6** A command naming a monster that no longer exists (killed between
  cast and consumption) is logged and dropped, not retried into an error loop.

### FR-5 — Forced controller handover (`atlas-monsters`)

- **FR-5.1** A second new `COMMAND_TOPIC_MONSTER` command type forces a
  monster's controller to a named character, bypassing the normal picker
  selection.
- **FR-5.2** The handover sets the controller-has-aggro flag, so the resulting
  `START_CONTROL` event drives `writer.StartControlMonsterBody(m, true)` on the
  channel side.
- **FR-5.3** The handover routes through the existing
  `monster.ProcessorImpl.StartControl` path so the existing stop-then-start
  sequencing, `START_CONTROL` emission, and `RepickReasonControlChange`
  semantics are preserved. It must not write controller state directly.
- **FR-5.4** Forcing control to the character who is already the controller is a
  no-op that does not emit a redundant control packet.
- **FR-5.5** A command naming a nonexistent monster, or a character not present
  in that field, is logged and dropped.
- **FR-5.6** This command is orthogonal to FR-4: either may be issued without the
  other.

### FR-6 — Skill effect broadcast and direction (`atlas-channel`)

- **FR-6.1** On a successful cast the caster's skill effect is broadcast to the
  other sessions in the field, so remote players see the magnet animation.
- **FR-6.2** The decoded direction byte is **carried into that broadcast**, not
  decoded and discarded — the client uses it to orient the animation.
- **FR-6.3** The broadcast fires once per cast, not once per grabbed monster.

### FR-7 — Handler registration (`atlas-channel`)

- **FR-7.1** A handler is registered in the identity-keyed `skill/handler`
  registry for all three magnet identities. Registration is version-blind by
  construction (task-187) — one registration per identity covers every version.
- **FR-7.2** It registers in the `Handler` registry (the `USE_SKILL`/`SPECIAL_MOVE`
  path), **not** the `AttackCastHandler` registry. Monster Magnet deals no damage
  and the client does not deliver it on an attack packet. Registering in
  `Handler` also correctly signals that the handler owns the HP/MP cost.
- **FR-7.3** The handler lives in its own subpackage under
  `skill/handler/`, following the layout of `hide/`, `dispel/`, `timeleap/`.

### FR-8 — Packet verification

- **FR-8.1** A byte-fixture test with a `packet-audit:verify` marker covers the
  Monster Magnet arm of `SPECIAL_MOVE` for each of the ten versions.
- **FR-8.2** Each fixture's read order is derived from that version's client
  binary via the documented decompile procedure, with the evidence record pinned
  alongside.
- **FR-8.3** These fixtures do not promote the `SPECIAL_MOVE` matrix cells to
  `✅` — the row's other fifteen fnames remain unverified. The evidence is pinned
  for the magnet arm specifically, and any matrix/consistency gate that this
  implies must be left green.

## 5. API Surface

No REST endpoints are added or modified.

Two new `COMMAND_TOPIC_MONSTER` command types are added to the monster command
contract. The contract is mirrored across producer and consumer modules
(`services/atlas-channel/.../kafka/message/monster/kafka.go` and the
atlas-monsters side); both copies must be updated together — they are separate Go
modules, so a field-name divergence fails no build and silently decodes to a
zero-valued body.

Existing types today: `CommandTypeDamage`, `CommandTypeDamageFriendly`,
`CommandTypeApplyStatus`, `CommandTypeCancelStatus`, `CommandTypeUseSkill`,
`CommandTypeUseBasicAttack`, `CommandTypeDrainMp`, `CommandTypeKill`.

Added:

| Command type | Body | Semantics |
|---|---|---|
| clear-aggro (FR-4) | empty or minimal; no magnet-specific fields | Full wipe of the monster's damage-aggro table |
| force-control (FR-5) | target character id | Force controller handover with aggro flag set |

Both reuse the existing `Command[E]` envelope (`worldId`, `channelId`, `mapId`,
`instance`, `monsterId`, `type`, `body`). Exact type-name strings and body field
names are a design-phase decision.

## 6. Data Model

No new persisted entities, no database migrations, no new tables or columns.

State touched is all in-memory in atlas-monsters:

- the per-monster damage-entry table used for aggro (`monster/registry.go`,
  `monster/aggro.go`) — wiped by FR-4;
- the per-monster controller assignment (`monster/processor.go`,
  `monster/picker.go`) — reassigned by FR-5.

New in-memory model state in `libs/atlas-packet`: the magnet grab table and
direction byte on `SkillUsageInfo` (FR-1.5, FR-1.6), following the existing
private-fields-plus-getters-plus-builder convention of that type.

## 7. Service Impact

### 7.1 `libs/atlas-packet`

`model/skill_usage_info.go`: magnet decode branch, new fields, getters, builder
setters, grab-entry value type. Byte-fixture tests per version.

### 7.2 `services/atlas-channel`

- New `skill/handler/<magnet>/` subpackage registering all three identities.
- Validation helper extracted from `skill/handler/common.go` `applyToMobs`
  (FR-2.6).
- New producer providers for the two monster commands.
- Caller added for the existing, currently senderless `writer.CatchMonsterBody`.
- Skill-effect broadcast carrying the direction byte.

### 7.3 `services/atlas-monsters`

- Two new command types on the monster command consumer.
- Aggro full-wipe operation on the monster processor/registry.
- Forced-control path routed through the existing `StartControl`.

### 7.4 Explicitly unaffected

- **Keydown prepare/keyup relay.** Already generic and already covers these
  skills. `socket/handler/character_skill_prepare.go:33-68` broadcasts a foreign
  prepare for any skill whose resolved Identity satisfies
  `skill.IsKeyDownSkillIdentity`, and `socket/handler/character_buff_cancel.go:55-60`
  relays the keyup through the same predicate. All three magnet identities are
  already listed in `IsKeyDownSkillIdentity`
  (`libs/atlas-constants/skill/identity.go:139-141`). No change needed — this
  becomes an acceptance check (§10), not a work item.
- **`libs/atlas-constants`.** The three identities, their per-version wire-id
  maps, and their keydown classification all already exist. No new constants are
  anticipated; if design finds one missing, the shared library is still the place
  for it (DOM-21).
- **Seed templates.** `CATCH_MONSTER` is already routed as a writer in all nine
  templates, and `SPECIAL_MOVE` is already routed as a handler. No template edits
  are anticipated. If design finds a gap, the config-table rule applies: the entry
  goes into **every** version template, at its sorted `opCode` position.

## 8. Non-Functional Requirements

- **Multi-tenancy.** Version gating resolves through
  `tenant.MustFromContext(ctx)` and the version-aware resolver
  (`constants.For(region, major, minor).Skill.Resolve`), never by comparing raw
  wire ids against constants — the skill-job-id guard enforces this.
- **Performance.** Monster Magnet is a keydown skill and can be re-cast rapidly
  against up to the WZ `mobCount` cap of monsters. The validation path must
  perform a single rect query per cast (as `applyToMobs` does), not one lookup per
  claimed monster.
- **Concurrency.** The aggro wipe and the control handover mutate shared
  atlas-monsters registry state; both must take the existing locks. No bare `go`
  statements — goroutines go through `routine.Go` (goroutine guard).
- **Observability.** Anomalies (over-cap cast, out-of-rect target) log at warn
  with the caster id, skill id and level, claimed vs server-side monster ids, and
  the computed rect — matching the existing `monster_buff_anomaly_*` event
  vocabulary so existing dashboards pick them up. Normal operation logs a
  per-cast summary at debug.
- **Security posture.** The client is an untrusted input source: FR-2 is the
  enforcement point. A client that fabricates targets gains nothing beyond a
  logged anomaly.
- **Backward compatibility.** No wire change may alter the decode of an
  already-verified `SPECIAL_MOVE` cast for any other skill on any version. The
  magnet branch is additive and reachable only for the three magnet ids.

## 9. Open Questions

These are design-phase items, all resolvable from source or WZ data — none are
blocking:

1. Exact magnet payload layout per version (grab-count width, object-id width,
   grab-result width, whether the trailing `delay` short is present alongside the
   direction byte, and byte order relative to it). Resolve by decompiling
   `CUserLocal::TryDoingMonsterMagnet` and `CUserLocal::SendSkillUseRequest` per
   version. Whether the layout diverges across the ten versions is itself
   unknown — the same function already diverges structurally at the v72 boundary
   for the anti-repeat gate.
2. Whether Monster Magnet's WZ effect carries `lt`/`rb`, which decides whether
   FR-2.1's rect check or FR-2.4's fallback is the live path. Verify against local
   WZ data.
3. The semantics of the grab-result value. The reference implementation passes
   `result == 3` as the client-side image selector; whether the wire value is a
   boolean, a small enum, or a raw roll needs confirmation before FR-3.4 maps it
   onto the writer's `result`/`success` arguments.
4. Whether the aggro wipe should also clear derived controller-picker state, or
   whether an explicit repick is the correct follow-on (FR-4.4).
5. Command type-name strings and body field names for the two new monster
   commands.

## 10. Acceptance Criteria

Behavioral:

- [ ] Casting Monster Magnet as Hero (1121001), Paladin (1221001), and Dark
      Knight (1321001) grabs nearby monsters — verified live on at least the
      primary test version.
- [ ] Each grabbed monster plays the blue grab animation on all clients in the
      field.
- [ ] Each grabbed monster's damage-aggro table is fully wiped.
- [ ] The caster becomes the controller of each grabbed monster, with the aggro
      flag set.
- [ ] Remote players see the caster's magnet skill effect, oriented by the
      decoded direction byte.
- [ ] Monsters the client reports as failed grabs are untouched.
- [ ] No `CATCH_MONSTER_WITH_ITEM` is sent by the magnet path; the task-212
      catch-item flow is unchanged and still plays exactly one animation.
- [ ] The keydown prepare and keyup still relay to remote players for all three
      magnet skills (regression check on the already-working path, §7.4).
- [ ] Casts of other `SPECIAL_MOVE` skills — a mob-affecting buff, a party buff,
      Shadow Stars, Resurrection — decode unchanged.

Validation:

- [ ] A cast claiming more monsters than the skill's WZ `mobCount` is rejected
      entirely and logged.
- [ ] A cast claiming a monster outside the server-computed rect drops that
      monster, keeps the rest, and logs the anomaly.
- [ ] A force-control command for a character already controlling the monster
      emits no redundant control packet.
- [ ] A clear-aggro command against an already-empty table is a no-op.
- [ ] Commands naming a nonexistent monster are logged and dropped.

Verification gates (per CLAUDE.md §Build & Verification):

- [ ] `go test -race ./...` clean in `libs/atlas-packet`, `services/atlas-channel`,
      `services/atlas-monsters`.
- [ ] `go vet ./...` clean in the same modules.
- [ ] `go build ./...` clean in each changed service.
- [ ] `docker buildx bake atlas-channel` and `atlas-monsters` succeed if either
      `go.mod` was touched.
- [ ] `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh`,
      `tools/skill-job-id-guard.sh`, `tools/lint.sh --check` all clean.
- [ ] If any seed template changed: `tools/template-opcode-order-guard.sh`,
      `tools/template-duplicate-binding-guard.sh`,
      `tools/template-movement-types-guard.sh` clean.
- [ ] Byte fixtures with `packet-audit:verify` markers exist for the Monster
      Magnet arm on all ten versions, with evidence records pinned, and the matrix
      consistency gates are green.
- [ ] Code review run (three modular reviewer agents) before the PR is opened.
