# Context — task-099 Keydown Skill Prepare/Cancel Broadcast

Companion to `plan.md` / `design.md`. Orientation for an engineer with zero prior context.

## What this task is

Keydown/hold skills (Hurricane, Monster Magnet, Rapid Fire, BigBang, Piercing Arrow, Poison Bomb, Evan Breaths, WindArcher Hurricane, ThunderBreaker Corkscrew) show a looping cast aura around the caster. Today **observers** in the same map see the projectiles but not the aura. Cause (IDA-verified): the aura is driven by a dedicated **prepare** packet (keydown) and torn down by a **cancel** packet (keyup) — atlas-channel implements neither. This task makes atlas-channel relay both to other map sessions. Pure socket relay — no DB, Kafka, or cross-service.

## Why it is NOT the attack packet (do not "fix" the attack writer)

IDA (v95) proved the remote attack packet reads `tKeyDown` only for the BigBang trio + Evan magic skills, never for shoot skills like Hurricane. The narrow `isKeydownSkill` in `services/atlas-channel/atlas.com/channel/socket/writer/character_attack_common.go` is therefore CORRECT — broadening it would write a field the observer never reads and corrupt the shoot-attack packet. Leave it untouched (design D9; Task 7 Step 3 guards it).

## Key codebase anchors (verified current)

| Anchor | Location |
|---|---|
| Handler template | `services/atlas-channel/atlas.com/channel/socket/handler/character_skill_use.go` (const L35, `HandleFunc` L37-117, ownership validation L51-70, broadcast L108-110) |
| Broadcast helpers | `services/atlas-channel/atlas.com/channel/socket/handler/effects.go` (`AnnounceForeignSkillUse`, L1-39) |
| Handler registration | `main.go` `produceHandlers()` ~L788 (`handlerMap[handler.CharacterUseSkillHandle] = handler.CharacterUseSkillHandleFunc`) |
| Writer list | `main.go` `produceWriters()` ~L683 (`charcb.CharacterEffectForeignWriter,`) |
| Inbound codec pattern | `libs/atlas-packet/model/attack_info.go` (struct + `Decode(l,ctx)`, `tenant.MustFromContext(ctx)` version branches, keydown read L232-236) |
| Clientbound writer pattern | `libs/atlas-packet/character/clientbound/effect_skill_use.go` (`EffectSkillUseForeign` L106-201; `Operation()` L157 returns writer name) |
| Writer-name const decl | `libs/atlas-packet/character/clientbound/effect.go` (L12-13) |
| Body constructor pattern | `libs/atlas-packet/character/` (`CharacterSkillUseEffectForeignBody`, used by effects.go) |
| Reader API | `libs/atlas-socket/request/reader.go` (`ReadUint32/ReadByte/ReadUint16/...`) |
| Keydown classifier | `libs/atlas-constants/skill/model.go` `IsKeyDownSkill` (14 ids incl. `BowmasterHurricaneId`) |
| Test convention | `libs/atlas-packet/model/attack_info_test.go` (`pt.Variants`, `pt.CreateContext`, `pt.RoundTrip`) |
| Config opcode resolution | `libs/atlas-opcodes/producer.go` `BuildHandlerMap` (missing validator → silently dropped) / `BuildWriterProducer` |
| Seed templates | `services/atlas-configurations/seed-data/templates/template_{gms_83_1,gms_84_1,gms_87_1,gms_95_1,jms_185_1}.json` (`socket.handlers[]` {opCode,validator,handler}, `socket.writers[]` {opCode,writer}) |
| Packet docs | `docs/packets/audits/STATUS.md`, `docs/packets/audits/VERIFYING_A_PACKET.md`, `docs/packets/registry/<ver>.yaml` (SKILL_EFFECT rows) |

## Verified facts (don't re-derive)

- **Broadcast pattern:** handler → `_map.NewProcessor(l,ctx).ForOtherSessionsInMap(field, casterId, op)` → `session.Announce(l)(ctx)(wp)(writerName)(bodyEncoder)`. No `socket/writer/` wrapper is needed for effect-style packets (the handler calls the `character` body constructor directly). Same shape as `AnnounceForeignSkillUse`.
- **v95 working model** (pin the rest in Task 1): serverbound prepare `0x069` = skillId u32, level u8, action u16 (bit15=move, low15=action), actionSpeed u8; serverbound cancel `0x068` = skillId u32; clientbound remote-prepare nType215 (`0x0D7`) = charId + the prepare fields; clientbound remote-cancel nType217 = charId + skillId. **v83 registry hints:** serverbound SKILL_EFFECT `0x5D` (`DoActiveSkill_Prepare`), clientbound SKILL_EFFECT `0xBE` (`OnSkillPrepare`); cancel opcodes for v83 are UNKNOWN — pin via IDB (Cosmic's CANCEL_BUFF `0x5C` is NOT trustworthy).
- **`EffectSkillUseForeign.Operation()` returns the non-foreign writer const** in current code — a latent bug. Do NOT copy it; the new foreign writers must return their own foreign const, or registration won't resolve.
- A `socket.handlers` row without a `validator` is silently dropped by `BuildHandlerMap` — every new handler row needs `LoggedInValidator`.
- Handlers/writers do NOT hot-reload from the config projection — existing tenants need a live config patch + channel restart (operational, Task 7 Step 5).

## Decisions locked (design.md)

- **D1** foreign-only writers (caster renders its own aura; no self echo).
- **D2** dedicated writers (distinct opcodes), not a mode on `CharacterEffectForeign`.
- **D3** on ownership/keydown miss → log + drop; do NOT `session.Destroy` (visual packet, low stakes).
- **D4** gate on `skill.IsKeyDownSkill`.
- **D5** one version-conditional codec per packet (`AttackInfo` idiom), read order pinned per IDB.
- **D6** termination = relay the keyup cancel (necessary; observer won't auto-clear while caster stays in map); disconnect/map-leave handled by avatar removal (no code, no server keydown state); death/stun verified empirically in Task 7.
- **D7** v92 DEFERRED (no IDB; no ported assumptions). Ship v83/v84/v87/v95/jms185.
- **D8** MovingShootAttackPrepare (nType 216) included only if Task 1 shows an in-scope skill uses it.
- **D9** attack writer `isKeydownSkill` untouched.

## Dependencies & ordering

- **Task 1 (IDB wire-spec) is a hard gate** — Tasks 2/3/5/6 consume `wire-spec.md`. Don't hardcode opcodes/read-orders before it's committed.
- Tasks 2 & 3 (codecs) are independent. Task 4 (channel) depends on both. Task 5 (seed) depends on Task 1 + the writer/handler names from 3/4. Task 6 (matrix) depends on the byte fixtures from 2/3.
- IDA work is per-IDB (`select_instance` per call; one active instance at a time) — dispatch per-version, ideally via subagents to keep context lean.

## Out of scope

- v92 (deferred), attack/tKeyDown changes, buff/stat/damage/cooldown logic, summon/monster keydown, the broader unimplemented `EffectSkillUse` conditional branches, and the live-config patch + manual in-map validation (operational follow-ups documented in Task 7, executed outside this code branch).

## Verification

Per changed module (`libs/atlas-packet`, `services/atlas-channel/atlas.com/channel`): `go test -race ./...`, `go vet ./...`, `go build ./...`. Repo: `tools/redis-key-guard.sh` (GOWORK=off), `jq` parse of the 5 templates. No `go.mod` change → docker-bake N/A. Matrix: STATUS.md cells promoted only with a `packet-audit:verify` byte-fixture test.
