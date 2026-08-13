# Monster Magnet — Implementation Context

Task: task-215-monster-magnet
PRD: [`prd.md`](prd.md) · Design: [`design.md`](design.md) · Plan: [`plan.md`](plan.md)
Created: 2026-08-12

This document is the orientation pack for anyone (human or subagent) executing
[`plan.md`](plan.md). It records what was verified from source during planning,
what the plan deliberately changes relative to the PRD, and the one open
decision that execution must not silently resolve.

---

## 1. Verified starting state

Everything below was read from source in this worktree during planning. Line
numbers are as of branch `task-215-monster-magnet` at plan time.

### 1.1 The decoder

`libs/atlas-packet/model/skill_usage_info.go` (330 lines). `SkillUsageInfo`
has nine private fields, getters, a `SkillUsageInfoBuilder`, and three
membership lists (`isMobAffectingBuff`, `isPartyBuff`,
`isAntiRepeatBuffSkill`) built on raw `skill.Is(...)` wire-id compares.

`Decode` (`:25-63`) reads `updateTime/skillId/skillLevel`, then falls through
four additive `if` blocks. There is **no** magnet branch — confirmed by
inspection; none of `HeroMonsterMagnetId` / `PaladinMonsterMagnetId` /
`DarkKnightMonsterMagnetId` appears anywhere in the file.

The existing version gate at `:40-41` is the idiom the magnet gate must copy:

```go
if isAntiRepeatBuffSkill(skill.Id(m.skillId)) &&
    ((t.IsRegion("GMS") && t.MajorAtLeast(72)) || t.IsRegion("JMS")) {
```

with a 9-line comment above it naming the decompiled address per version.

The package already imports `atlas-constants/skill` but **not**
`atlas-constants/constants`. Adding that import introduces no cycle:
`constants` imports only `skill` and `job`.

Resolver API (`libs/atlas-constants/skill/identity.go`):
- `constants.For(region string, major, minor uint16) SkillJobSet` (`for.go:39`)
- `(skill.Set).Resolve(wireId Id) (Identity, bool)` (`identity.go:38`)
- `skill.IsIdentity(id Identity, refs ...Identity) bool` (`identity.go:99`)

The three magnet Identities exist and are already members of
`IsKeyDownSkillIdentity` (`identity.go:139-141`). Wire ids:
`HeroMonsterMagnetId = 1121001` (`constants.go:2950`),
`PaladinMonsterMagnetId = 1221001` (`:2978`),
`DarkKnightMonsterMagnetId = 1321001` (`:3007`).

### 1.2 Existing decode-test harness

`libs/atlas-packet/model/skill_usage_info_test.go` establishes the exact
pattern the new fixtures use — no new harness is needed:

```go
req := request.Request(buf)
reader := request.NewRequestReader(&req, 0)
ctx := pt.CreateContext("GMS", 61, 1)
m := &SkillUsageInfo{}
m.Decode(nil, ctx)(&reader, nil)
...
if reader.Available() > 0 { t.Fatalf("reader has %d unconsumed bytes", reader.Available()) }
```

`pt` is `github.com/Chronicle20/atlas/libs/atlas-packet/test`. The
`reader.Available() == 0` assertion is the load-bearing one: it is what proves
the layout consumed the whole body.

### 1.3 The channel-side skill pipeline

- `socket/handler/character_skill_use.go:44-185` — `CharacterUseSkillHandleFunc`.
  Decodes, loads the caster (skill-decorated), validates the caster owns the
  skill at the claimed level, fetches the WZ effect, calls
  `handler.UseSkill(...)` at `:161`, then broadcasts self (`:176`) and foreign
  (`:178`) SKILL_USE, then `enableActions` (`:180`).
- `skill/handler/common.go:101-211` — `UseSkill`. Charges `HPConsume`/`MPConsume`
  (`:137-142`), item consume, cooldown, mount branch, generic buff apply,
  `applyToMobs` (`:196`), then the per-skill dispatcher `Lookup(castId)`
  (`:200-206`).
- `skill/handler/common.go:213-381` — `applyToMobs`. Cap check at `:226-237`,
  `hasEffectBbox` fork at `:248`, `calculateBoundingBox` + `rectQueryFunc` +
  `intersectMobIds` at `:271-290`, anomaly log at `:292-304`, then the
  reflect/prop/apply-or-cancel loop.
- `skill/handler/mob_select.go` — `calculateBoundingBox`, `hasEffectBbox`,
  `intersectMobIds` (`:49-65`, reusable as-is).
- `skill/handler/registry.go` — `Register(skill2.Identity, Handler)` /
  `Lookup`. The doc comment explains why `Handler` (not `AttackCastHandler`)
  is correct for a non-damage use-skill: the `Handler` registry doubles as the
  "this handler owns the HP/MP cost" signal.
- `skill/handler/registrations/registrations.go` — ten blank imports; new
  subpackages are appended here.
- `skill/handler/dispel/dispel.go` — the exemplar subpackage: `init()` calling
  `channelhandler.Register`, `var`-function seams, `Apply` returning `nil`
  unconditionally, a single `Debug` summary log at the tail.

### 1.4 The effect model

`data/skill/effect/rest.go` / `model.go`: `MobCount` (`rest.go:43` →
`model.go:163`) and `Prop` exist. **`Range` does not exist** — verified by
grep. Upstream atlas-data does serve it:
`services/atlas-data/atlas.com/data/skill/effect/rest.go:78`
(`Range int32 \`json:"range"\``), populated from
`skill/reader.go:282` (`GetIntegerWithDefault("range", 0)`), with `"range"`
declared as a `commonExpr` node in `skill/common.go:115`. So the field is on
the wire today and only the channel-side decode is missing.

### 1.5 The CATCH_MONSTER seam

`libs/atlas-packet/monster/clientbound/catch_monster.go:14` —
`const CatchMonsterWriter = "CatchMonster"`; `NewCatchMonster(uniqueId, result, success)`.
`socket/writer/catch_monster.go` — `CatchMonsterBody(uniqueId uint32, result byte, success byte)`,
whose doc comment says verbatim: *"No emitter wires this writer … The codec +
template routes stay as an intentional seam so another mechanic … needs no
packet-plumbing pass."* This task is that mechanic.

Broadcast helper: `_map.Processor.ForOtherSessionsInMap(f field.Model,
referenceCharacterId uint32, o model.Operator[session.Model]) error`
(`map/processor.go:103`).

### 1.6 The effect-direction seam

`libs/atlas-packet/character/effect_body.go:63-85` — both
`CharacterSkillUseEffectBody` and `CharacterSkillUseEffectForeignBody` already
take a trailing `left bool` and already derive
`isMonsterMagnet := skill.Is(skill.Id(skillId), Hero/Paladin/DarkKnight MonsterMagnetId)`,
handing both to `NewEffectSkillUse` / `NewEffectSkillUseForeign`. The codec is
done. The only gap is that `socket/handler/effects.go:24` and `:36` hard-code
`false` for that argument, and no caller can pass anything else.

`AnnounceBerserkEffect` / `AnnounceForeignBerserkEffect`
(`effects.go:47-67`) are the precedent: same problem (a per-skill bool the
codec gates on skill id), same solution (a parallel announce function that
threads the bool), leaving the plain variants untouched for their four other
call sites.

### 1.7 The monster command contract (two copies, two modules)

| Side | File | Style |
|---|---|---|
| atlas-channel (producer) | `kafka/message/monster/kafka.go` | **Exported** types: `Command[E]`, `DamageCommandBody`, … 8 command-type consts |
| atlas-monsters (consumer) | `kafka/consumer/monster/kafka.go` | **Unexported** types: `command[E]`, `damageCommandBody`, … 17 command-type consts |

They are **not** byte-identical mirrors (unlike the trade contract, which has a
guard script) — the consumer carries field-level commands the producer never
sends, and the naming conventions differ by case. What must agree is the
**type-name string** and the **json tags**. There is no guard; the plan pairs
both edits into one task and one commit.

Shared-topic hazard, documented in both files' `KillCommandBody` /
`killCommandBody` comments: every handler registered on
`COMMAND_TOPIC_MONSTER` json-unmarshals **every** message into its own body
type. A new field name whose Go type disagrees with a sibling body logs one
spurious unmarshal error per message. This is why the new bodies are an empty
struct and a single `CharacterId uint32` (which already appears with that name
and type in `damageCommandBody`, `killCommandBody`, `catchCommandBody`).

Producer shape (`monster/producer.go`): every provider is
`key := producer.CreateKey(int(monsterId))` + a `&monster2.Command[Body]{...}`
+ `producer.SingleMessageProvider(key, value)`. Consumer registration
(`kafka/consumer/monster/consumer.go:29-87`): one
`rf(t, message.AdaptHandler(message.PersistentConfig(handleXCommand)))` line
per handler, each handler opening with `if c.Type != CommandTypeX { return }`.

### 1.8 atlas-monsters state

- `monster/processor.go:387-417` — `StartControl(uniqueId, controllerId)`.
  Stop-then-start, `ControlMonster`, emit `START_CONTROL`, then the
  `ControllerHasAggro` gate at `:410` deciding whether to
  `RepickAndEmit(RepickReasonControlChange)`.
- `monster/registry.go:412-416` — `ControlMonster` → `atomicUpdate` →
  `m.Control(characterId)`, which (`model.go:172-176`) sets only
  `controlCharacterId`. **Nothing in the existing control path sets
  `controllerHasAggro`** — this is why FR-5.2 needs new plumbing.
- `monster/registry.go:707-752` — `DecayDamageEntries`. The template for the
  new wipe. Note it does **not** use `atomicUpdate`; it calls
  `r.reg.Update(ctx, monsterSuffix(t, uniqueId), func(cur storedMonster) storedMonster {...})`
  directly, mutating `cur.DamageEntries` and `cur.ControllerHasAggro` on the
  **stored** struct, and captures `aggroFlippedOff` / `controllerCharacterId`
  in closure variables. It maps `atlasredis.ErrNotFound` → `errMonsterNotFound`.
  The new `ClearDamageEntries` follows this shape exactly, because `Model` has
  no builder setter for clearing damage entries or for `ControllerHasAggro`.
- `monster/registry.go:689-699` — `DecaySummary{Monster, ControllerCharacterId, AggroFlippedOff}`.
  `ClearSummary` mirrors it field-for-field.
- `monster/aggro_task.go:98-104` — the only `DecayDamageEntries` caller; emits
  `aggroChangedStatusEventProvider(summary.Monster, summary.ControllerCharacterId, false)`
  when `summary.AggroFlippedOff`. The clear-aggro processor method emits the
  same provider on the same condition.
- `monster/producer.go:53` — `aggroChangedStatusEventProvider(m Model, controllerCharacterId uint32, hasAggro bool)`.
- `ProcessorImpl` seams (`processor.go:88-120`): `emit`, `inFieldFn`,
  `hiddenFn`, `locationFn` — all injectable, which is how the atlas-monsters
  tests run without kafka or redis-backed peers.

---

## 2. The one open decision execution must not resolve silently

**PRD FR-8 (byte fixtures with `packet-audit:verify` markers + pinned evidence
for the Monster Magnet arm, without promoting the `SPECIAL_MOVE` row) is not
achievable as written.** This was discovered during planning by reading the
tooling, and it is the single place the plan departs from the PRD on something
the user may care about.

Evidence:

1. `docs/packets/audits/status.json`, row `op: SPECIAL_MOVE`, carries **16
   fnames** — `CUserLocal::TryDoingMonsterMagnet` is one `fname_alt` among
   fifteen others. The matrix's unit of promotion is the whole op × version
   cell; there is no per-fname cell.
2. `tools/packet-audit/cmd/matrix.go:232-249` — every
   `packet-audit:verify` marker found under `libs/atlas-packet` is matched
   against `(packet, version)` in the evidence records **and** the audit
   reports, at the same IDA address. A marker matching neither is reported as
   `orphan marker …` and, per `matrix_markers_test.go:100-109`, **hard-fails
   `matrix --check`** — a blocking CI gate
   (`.github/workflows/packet-matrix.yml`).

So the two available marker outcomes are: orphan (CI red), or matched (which
promotes the whole `SPECIAL_MOVE` cell to ✅ while fifteen fnames remain
unverified — precisely the false-✅ FR-8.3 forbids).

Two further facts that bear on any future resolution:

- `VERIFYING_A_PACKET.md §9` requires a serverbound cell to have a codec in
  `<pkg>/serverbound/` linked by the op's **primary** fname
  (`CUserLocal::DoActiveSkill_Heal` for `SPECIAL_MOVE`) via the
  `candidatesFromFName` switch in `cmd/run.go`. `SkillUsageInfo` lives in
  `libs/atlas-packet/model/`, is not linked, and has no wrapper.
- The `gms_v48` cell for `SPECIAL_MOVE` is currently `"state": "n-a",
  "opcode": -1`. That is inconsistent with the v48 seed template, which **does**
  bind `CharacterUseSkillHandle` (`template_gms_48_1.json:577`), and with
  design §2's v48 derivation at `0x6AD842` / opcode `0x46`. Correcting that
  cell is a matrix change with its own evidentiary bar (`VERIFYING_A_PACKET.md`
  "Is this cell `n-a`?"), unrelated to the magnet behaviour.

**Plan's resolution (Task 5):** write the ten per-version byte fixtures with
the full derived read order and per-field IDA address citations in comments,
using the existing `pt.CreateContext` harness and the `reader.Available() == 0`
assertion — but **without** a `packet-audit:verify` marker and **without** a
pinned evidence record. This satisfies FR-8.1's regression intent and FR-8.2's
"derived from that version's client binary" requirement, and keeps FR-8.3's
"matrix gates stay green" literally true. It does not produce a matrix-visible
artifact.

The alternative — a full `SPECIAL_MOVE` verification campaign across all
sixteen fnames plus a v48 `n-a` correction — is a separate task of its own
size. **Do not start it inside this branch without asking.**

---

## 3. Deltas from the PRD, consolidated

Design §9 already lists five; Task 5's marker decision is the sixth. All six,
in one place:

| # | PRD text | Actual | Source |
|---|---|---|---|
| 1 | FR-1.2: one payload shape | Two. gms_61+/jms: `uint32` count, per-entry bool, trailing direction byte, **no** delay. gms_48: `byte` count, no per-entry result, trailing delay short, **no** direction, **leading caster-id entry**. | design §2 |
| 2 | FR-1.4: gate at the v72 boundary | Gate is `IsRegion("GMS") && !MajorAtLeast(61)` — v48 only. | design §2.3 |
| 3 | FR-2.1/2.4/2.6: rect from `lt`/`rb` | Monster Magnet has neither. Region derives from WZ `range` as the AABB of the client's trapezoid; a new `Range` field on the channel effect model. | design §3, §4.2 |
| 4 | FR-3.1: broadcast to every session | **Other** sessions only — the caster renders locally via `CMob::OnHit` → `ShowCatchEffect`. Wire result is a boolean: `result=1, success=1`. | design §4.3 |
| 5 | FR-6: new broadcast needed | Already fires per cast; only the `left` argument is new. **Inert on gms_48**, which binds no `CharacterEffect` writer at all. | design §4.6 |
| 6 | FR-8.1/8.2: `packet-audit:verify` marker + pinned evidence | Not achievable without promoting a 16-fname row or failing CI. Fixtures ship without markers. | §2 above |

---

## 4. Verification commands

Run from the worktree root (`.worktrees/task-215-monster-magnet`). Per
CLAUDE.md §Build & Verification, these are the gates for this task's blast
radius (three Go modules; **no** `go.mod` change and **no** template change is
anticipated, so `docker buildx bake` and the three template guards are not in
scope unless something changes that assumption).

```bash
# per-module (run in each of the three module roots)
go test -race ./...
go vet ./...
go build ./...

# repo-root guards
tools/redis-key-guard.sh
tools/goroutine-guard.sh
tools/skill-job-id-guard.sh
tools/buff-duration-guard.sh
tools/lint.sh --check          # tools/lint.sh with no flags to auto-fix first

# packet matrix must be green and UNCHANGED
go run ./tools/packet-audit matrix --check
```

Module roots:
- `libs/atlas-packet`
- `services/atlas-channel/atlas.com/channel`
- `services/atlas-monsters/atlas.com/monsters`

`tools/lint.sh --check` false-fails without nvm on PATH; run `nvm use 22`
first if the atlas-ui half of the guard errors out.

---

## 5. Dependency order

```
Task 1 (decode)  ─┐
                  ├─► Task 6 (handler) ─► Task 7 (registration + wiring)
Task 2 (Range)   ─┤
Task 3 (contract+producers) ─┤
Task 4 (monsters commands)  ─┘
Task 5 (fixtures)  depends on Task 1 only
Task 8 (direction) depends on Task 1 only
Task 9 (final verification) depends on everything
```

Tasks 1–4 are mutually independent and can be executed in parallel by separate
agents. Task 5 and Task 8 unblock as soon as Task 1 lands. Tasks 6, 7, 9 are
strictly sequential at the tail.
