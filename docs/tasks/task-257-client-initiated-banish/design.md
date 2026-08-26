# Client-Initiated Mob Banish — Design

Task: task-257-client-initiated-banish
PRD: [prd.md](prd.md)
Status: Draft
Created: 2026-08-21

---

## 1. Scope of this document

The PRD fixes *what* is built and *where* the trust boundary sits. This document
fixes *how*: which service owns each decision, the exact function-level seams, the
alternatives that were rejected and why, and the test seams that make each
acceptance criterion checkable. It records one PRD internal contradiction and
resolves it (§7.3).

Three services change: `atlas-channel` (emit), `atlas-monsters` (validate +
execute), `atlas-portals` (portal-name resolution). `atlas-data` is unchanged.

---

## 2. Current state (verified against the branch)

| Concern | Where it lives today | State |
|---|---|---|
| `MOB_BANISH_PLAYER` codec | `libs/atlas-packet/character/serverbound/mob_banish_player.go` | Complete, IDA-verified, single `Encode4(dwMobTemplateID)`, accessor `MobTemplateId()` |
| Channel handler | `services/atlas-channel/atlas.com/channel/socket/handler/mob_banish_player.go:19` | Decode-and-log stub, comment `// behavior: deferred` |
| Skill-129 banish | `services/atlas-monsters/.../monster/processor.go:1248` `executeBanish` | Fetches info, checks `Banish().MapId != 0`, emits `WARP` per target. Drops `PortalName` and `Message` |
| Warp emit (monsters) | `monster/disease.go:96` `warpBody` / `warpCommandProvider` | `{characterId, targetMapId}` only |
| Warp consume (portals) | `portal/kafka.go:38` `warpBody`, `portal/consumer.go:55` `handleWarpCommand` | Precedence `UseTargetPosition` → `TargetPortalId != 0` → random `Warp` |
| Banish data | `monster/information/model.go:26` `Banish{Message, MapId, PortalName}` | Already populated from `atlas-data`'s `banish{message, map_id, portal_name}` |
| Field-scoped liveness | `monster/processor.go:166` `GetInField(f)` → `ByFieldProvider` | Instance-scoped, exactly the lookup §4.2 needs |

Two existing test seams in `atlas-monsters` matter and are reused rather than
reinvented (§8):

- `ProcessorImpl.emit` (`processor.go:98`, type `emitter`) — the injectable Kafka
  emit used by every recently-tested processor path. `executeBanish` currently
  bypasses it and calls `producer.ProviderImpl(...)` inline, which is precisely
  why the skill path has no test today.
- `testInformationLookup` (`processor.go:81`) — the test-only override for
  `information.GetById`, already consulted at three call sites.

---

## 3. Architecture

```
client ──MOB_BANISH_PLAYER(mobTemplateId)──▶ atlas-channel
                                              │  handler → monster.Processor.Banish
                                              ▼
                            COMMAND_TOPIC_MONSTER  type=BANISH
                            body {characterId, monsterTemplateId}
                                              │
                                              ▼
                                        atlas-monsters
                       ┌── validate ──────────┴───────────────┐
                       │ 1. GetInField(f)                     │  ← trust boundary
                       │ 2. ∃ m: m.MonsterId() == template    │
                       │ 3. information.GetById(template)     │
                       │ 4. Banish().MapId != 0               │
                       └──────────────┬───────────────────────┘
                                      ▼
                          banishCharacter(f, characterId, Banish)   ◀── also called by
                                      │                                 executeBanish (skill 129)
                    ┌─────────────────┴──────────────────┐
                    ▼                                    ▼
       COMMAND_TOPIC_PORTAL  WARP            COMMAND_TOPIC_SYSTEM_MESSAGE
       {…, targetPortalName}                 SEND_MESSAGE / PINK_TEXT
                    │                                    │
                    ▼                                    ▼
              atlas-portals                        atlas-channel
     name → GetInMapByName → WarpById        session announce (pink text)
        miss → warn + random Warp
```

The shape is deliberately the one every other client-originated monster action in
this repo already uses (`MonsterDamageFriendly`, Mortal Blow `KILL`, `CATCH`,
`FORCE_CONTROL`): the channel decodes and forwards, `atlas-monsters` — the only
service holding live monster state — validates and acts.

---

## 4. Alternatives considered

### 4.1 Where validation lives

**A. `atlas-monsters` validates (chosen).**
It is the sole owner of the live monster registry. `GetInField(f)` is an in-process
registry read; no network hop is added to reach the authority. The four checks and
the banish execution sit in the same function, so there is no window between
"validated" and "acted", and the same code path serves the skill-129 caller.

**B. `atlas-channel` validates, then emits `WARP` directly to `atlas-portals`.**
Rejected. The channel's `data/monster` projection is *template* data from
`atlas-data`, not live field state — it cannot answer "is a mob of this template
alive in this field". Answering it would require a new REST call from channel into
`atlas-monsters` on a packet path, and would fork banish into two implementations
(the PRD's explicit convergence goal, §2). It also puts the trust boundary in the
service that received the untrusted bytes, which is the weaker place for it.

**C. `atlas-maps` validates.**
Rejected. `atlas-maps` knows characters-in-field, not monsters-in-field; it would
have to call `atlas-monsters` anyway.

### 4.2 What the `BANISH` envelope carries

**A. Envelope `monsterId: 0`, body `{characterId, monsterTemplateId}` (chosen).**
The client supplies a *template* id. The envelope's `monsterId` means *unique* id
everywhere else on `COMMAND_TOPIC_MONSTER`, and every sibling handler unmarshals
every message on this topic. Overloading it would make a `BANISH` message look
like a well-formed command addressed to a nonexistent unique monster — a real
hazard given `handleDamageCommand` and friends key off `c.MonsterId`. Leaving it
zero and naming the template explicitly in the body is unambiguous.

**B. Channel resolves the template to a unique id and sends that.**
Rejected for the same reason as §4.1-B: the channel does not hold live monster
state, and resolving would require a REST round trip on the packet path.

#### Field-name collision audit (shared fan-to-every-handler topic)

`services/atlas-monsters/.../kafka/consumer/monster/kafka.go` carries an explicit
caution (see `killCommandBody`'s comment): every handler on this topic
JSON-unmarshals every message, so a field name whose Go type disagrees with a
sibling body's produces one spurious unmarshal error per message. Audit of the new
body against every sibling in that file:

| New field | Type | Sibling occurrences | Verdict |
|---|---|---|---|
| `characterId` | `uint32` | `damageCommandBody`, `useSkillCommandBody`, `drainMpCommandBody`, `killCommandBody`, `catchCommandBody`, `forceControlCommandBody` — all `uint32` | Safe, type-identical |
| `monsterTemplateId` | `uint32` | none | Safe, no collision |

Note `spawnFieldCommandBody.MonsterId` is `uint32` and holds a *template* id, but
the JSON key differs (`monsterId` vs `monsterTemplateId`), so there is no overlap.
`uint32` is wide enough for every template id in both data sets.

### 4.3 Portal-name resolution in `atlas-portals`

**A. New `Processor.WarpByName(f, characterId, targetMapId, name)` (chosen).**
`handleWarpCommand` stays a pure router — check `Type`, build the field, pick a
branch — matching what it already does for the three existing branches. The
resolve-or-fall-back policy lives next to `Warp`/`WarpById`/`WarpToPosition`, where
the same "resolve name, warn, default" shape already exists inside `Enter`
(`processor.go:120`). The method is directly unit-testable against the existing
`setupMockDataServer` harness (`portal/processor_test.go`), whereas testing it
through the consumer would require standing up a Kafka handler.

**B. Resolve inline in `handleWarpCommand`.**
Rejected: puts policy in the router, duplicates `Enter`'s fallback shape, and is
harder to test.

`WarpByName` falls back to `p.Warp` (random spawn) on a miss, not to portal 0.
`Warp` itself already degrades to portal 0 when a map has no portals, so the
fallback inherits the more forgiving behavior. Warn-and-warp rather than drop:
per PRD §4.4, failing to banish is worse than banishing to a default spawn.

**Precedence** in `handleWarpCommand` becomes:

1. `UseTargetPosition` → `WarpToPosition` *(unchanged, highest)*
2. `TargetPortalId != 0` → `WarpById` *(unchanged)*
3. `TargetPortalName != ""` → `WarpByName` *(new)*
4. otherwise → `Warp` *(unchanged)*

`"sp"` is resolved by name like any other portal — it is a real portal name in map
data, not a sentinel. No special case.

### 4.4 Shared banish execution

**A. Private `banishCharacter(f, characterId, b information.Banish) error` on
`ProcessorImpl` (chosen).** Takes the already-resolved `Banish` value rather than a
monster id, so it makes no lookups of its own and both callers keep ownership of
their own resolution: the skill path already has the `information.Model` from its
own `GetById`; the command path has it from validation step 3. One fetch per
banish, not two.

**B. Helper takes `monsterTemplateId` and fetches internally.**
Rejected: doubles the information fetch on the skill path and hides an I/O call
inside what should be a pure emit step.

---

## 5. Detailed design

### 5.1 `atlas-channel`

**`kafka/message/monster/kafka.go`** — add:

```go
CommandTypeBanish = "BANISH"

// BanishCommandBody asks atlas-monsters to banish a character on the strength
// of a client MOB_BANISH_PLAYER request. The monster template id is
// client-supplied and untrusted; atlas-monsters validates it against live
// field state. Mirrors atlas-monsters' banishCommandBody — edit both together.
type BanishCommandBody struct {
    CharacterId       uint32 `json:"characterId"`
    MonsterTemplateId uint32 `json:"monsterTemplateId"`
}
```

**`monster/producer.go`** — add `BanishCommandProvider(f field.Model, characterId
uint32, monsterTemplateId uint32)`, envelope `MonsterId: 0`, `Type:
monster2.CommandTypeBanish`, key `producer.CreateKey(int(characterId))`.

The key deviates from the monster-id keying used by `ClearAggroCommandProvider` /
`ForceControlCommandProvider` and matches the character-id keying used by the
portal `WARP` producers. This is intentional and PRD-specified (§5.1): the command
is about a character's map transition, and the ordering that matters is *this
character's* banish requests against each other, not requests against one monster.

**`monster/processor.go`** — add to the interface and impl:

```go
// Banish forwards a client MOB_BANISH_PLAYER request to atlas-monsters, which
// owns live monster state and is the only service that can validate the
// client-supplied template id against the field.
func (p *ProcessorImpl) Banish(f field.Model, characterId uint32, monsterTemplateId uint32) error
```

**`socket/handler/mob_banish_player.go`** — replace the deferred comment with
`_ = monster.NewProcessor(l, ctx).Banish(s.Field(), s.CharacterId(), p.MobTemplateId())`,
keeping the existing `l.Debugf` line. Shape matches
`monster_damage_friendly.go` exactly. The handler returns nothing to the client;
the client does not await a response.

`data/monster` (`Model`, `RestModel`) is **not** touched — the channel makes no
banish decision.

### 5.2 `atlas-monsters`

**`kafka/consumer/monster/kafka.go`** — add `CommandTypeBanish = "BANISH"` and:

```go
// banishCommandBody asks the processor to banish a character out of a field on
// the strength of a client MOB_BANISH_PLAYER request. monsterTemplateId is
// client-supplied and untrusted — Banish revalidates it against live field
// state before acting. Both fields are uint32; characterId already appears at
// that type in sibling bodies and monsterTemplateId appears in none, so this
// cannot collide on the shared, fan-to-every-handler command topic. The
// envelope's monsterId is deliberately left 0: it means *unique* id everywhere
// else on this topic. Mirrors atlas-channel's monster2.BanishCommandBody —
// edit both together.
type banishCommandBody struct {
    CharacterId       uint32 `json:"characterId"`
    MonsterTemplateId uint32 `json:"monsterTemplateId"`
}
```

**`kafka/consumer/monster/consumer.go`** — register `handleBanishCommand` in
`InitHandlers` alongside the other `EnvCommandTopic` handlers:

```go
func handleBanishCommand(l logrus.FieldLogger, ctx context.Context, c command[banishCommandBody]) {
    if c.Type != CommandTypeBanish {
        return
    }
    f := field.NewBuilder(c.WorldId, c.ChannelId, c.MapId).SetInstance(c.Instance).Build()
    if err := monster.NewProcessor(l, ctx).Banish(f, c.Body.CharacterId, c.Body.MonsterTemplateId); err != nil {
        l.WithError(err).Debugf("BANISH rejected for character [%d] template [%d] field [%s].", c.Body.CharacterId, c.Body.MonsterTemplateId, f.Id())
    }
}
```

Field construction matches `handleAddPuppetCommand`/`handleRemovePuppetCommand`.

**`monster/processor.go`** — add to the `Processor` interface and impl:

```go
// Banish honors a client-initiated MOB_BANISH_PLAYER request. The template id
// arrives from the client and is untrusted: the banish executes only when a
// monster of that template is actually alive in the requesting character's
// field, which is the trust boundary for this path (PRD §4.2). Every failure
// returns an error and takes no action; the caller logs.
func (p *ProcessorImpl) Banish(f field.Model, characterId uint32, monsterTemplateId uint32) error
```

Order of checks, aborting at the first failure, each returning a distinct
`errors.New`/`fmt.Errorf` naming character id, template id and field so the
consumer's single log line satisfies PRD §8's observability requirement:

1. `ms, err := p.GetInField(f)` — error → return it wrapped.
2. `slices.ContainsFunc(ms, func(m Model) bool { return m.MonsterId() == monsterTemplateId })`
   — false → return "no live monster of template … in field …". **Trust boundary.**
3. information lookup for `monsterTemplateId` — error → return wrapped.
4. `b := info.Banish(); b.MapId == 0` — return "template … has no banish map".

Then `return p.banishCharacter(f, characterId, b)`.

The information lookup goes through the existing `testInformationLookup` hook
(the `if testInformationLookup != nil { … } else { information.NewProcessor(...).GetById(...) }`
shape already used at `processor.go:980`, `:1384`, `:1692`), so step 3 and step 4
are unit-testable without a live `atlas-data`. Extracting that repeated shape into
a small private `p.monsterInformation(id)` helper and using it at the new call
site is in scope; converting the three existing call sites to it is **not** — that
is unrelated churn.

**`monster/disease.go`** — extend the warp producer:

```go
type warpBody struct {
    CharacterId uint32  `json:"characterId"`
    TargetMapId _map.Id `json:"targetMapId"`
    // TargetPortalName, when non-empty, lands the character on the portal of
    // that name in the target map. omitempty keeps an omitting producer's
    // bytes byte-identical to today (PRD acceptance: no existing WARP
    // producer's body changes).
    TargetPortalName string `json:"targetPortalName,omitempty"`
}

func warpCommandProvider(f field.Model, characterId uint32, targetMapId _map.Id, portalName string) model.Provider[[]kafka.Message]
```

`warpCommandProvider` has exactly one existing caller (`executeBanish`), which is
being rewritten anyway, so widening the signature costs nothing and avoids a
second near-identical provider.

`omitempty` matters: PRD acceptance requires that no existing `WARP` producer's
emitted body changes. It is also consistent with the precedent set by
`spawnFieldCommandBody`'s optional provenance fields in the same service.

**New `kafka/message/system_message/kafka.go`** — a byte-for-byte copy of
`services/atlas-party-quests/.../kafka/message/system_message/kafka.go`
(`EnvCommandTopic`, `CommandSendMessage`, `Command[E]`, `SendMessageBody`). This
local-copy-per-service pattern is the established convention here — the same file
already exists in three services. No shared library is introduced; that would be a
cross-cutting refactor outside this task.

**`monster/disease.go`** (or a sibling file in the same package) — add
`sendMessageProvider(f field.Model, characterId uint32, messageType string, msg string)`,
modeled on `atlas-party-quests/.../instance/producer.go:186`: `TransactionId:
uuid.Nil`, `WorldId`/`ChannelId` from the field, `CharacterId`, `Type:
system_message.CommandSendMessage`, body `{MessageType, Message}`, key
`producer.CreateKey(int(characterId))`.

**`monster/processor.go`** — the shared executor:

```go
// banishCharacter emits the two commands a banish is made of: the map change,
// then the WZ banish message. Shared by the skill-129 path (executeBanish) and
// the client-initiated path (Banish) so portal and message handling can never
// diverge between them.
func (p *ProcessorImpl) banishCharacter(f field.Model, characterId uint32, b information.Banish) error {
    if err := p.emit(EnvCommandTopicPortal, warpCommandProvider(f, characterId, _map.Id(b.MapId), b.PortalName)); err != nil {
        return err
    }
    if b.Message != "" {
        if err := p.emit(system_message.EnvCommandTopic, sendMessageProvider(f, characterId, "PINK_TEXT", b.Message)); err != nil {
            p.l.WithError(err).Warnf("Banished character [%d] but unable to send banish message.", characterId)
        }
    }
    return nil
}
```

Emitting via `p.emit` rather than `producer.ProviderImpl(...)` inline is what makes
both callers testable (§8) and is the pattern every other tested path in this file
already uses.

**`executeBanish`** (`processor.go:1248`) is rewritten to keep its existing
`getDiseaseTargets` selection and its existing two guards (info fetch error, zero
map), and to call `p.banishCharacter(m.Field(), characterId, ma.Banish())` per
target, logging per-target failures as it does now. Its behavior change is
additive: it now carries the portal name and sends the message.

`EnvCommandTopicPortal` already exists in `disease.go:38`;
`system_message.EnvCommandTopic` comes from the new local package.

### 5.3 `atlas-portals`

**`portal/kafka.go`** — add to `warpBody`:

```go
// TargetPortalName, when non-empty and TargetPortalId is zero, lands the
// character on the portal of that name in the target map. Resolution failure
// falls back to the random-spawn Warp rather than dropping the warp.
TargetPortalName string `json:"targetPortalName"`
```

**`portal/processor.go`** — add `WarpByName` to the `Processor` interface and impl:

```go
func (p *ProcessorImpl) WarpByName(f field.Model, characterId uint32, targetMapId _map.Id, name string) {
    tp, err := p.GetInMapByName(targetMapId, name)
    if err != nil {
        p.l.WithError(err).Warnf("Unable to locate portal [%s] in map [%d] for character [%d]. Falling back to a random spawn point.", name, targetMapId, characterId)
        p.Warp(f, characterId, targetMapId)
        return
    }
    p.WarpById(f, characterId, targetMapId, tp.Id())
}
```

**`portal/consumer.go`** — insert branch 3 of the precedence in §4.3 into
`handleWarpCommand`, with a `Debugf` matching the surrounding branches' phrasing.

### 5.4 Deployment / configuration

No change. `deploy/k8s/base/env-configmap.yaml:92` already defines
`COMMAND_TOPIC_SYSTEM_MESSAGE`, and `deploy/k8s/base/atlas-monsters.yaml` consumes
the configmap wholesale via `envFrom.configMapRef`; the compose stack passes the
same `.env` to every service through the `atlas-defaults` anchor. `atlas-monsters`
therefore already resolves the new topic. No new topic is created — the command
type is new, the topic is not.

---

## 6. Data flow, end to end

1. Client sends `MOB_BANISH_PLAYER{mobTemplateId}`.
2. Channel decodes, logs, emits `BANISH` on `COMMAND_TOPIC_MONSTER` keyed by
   character id, envelope from `s.Field()`, `monsterId: 0`.
3. `atlas-monsters`' `handleBanishCommand` rebuilds the field (world, channel,
   map, instance) and calls `Processor.Banish`.
4. `Banish` runs the four checks. Any failure returns; the consumer logs at Debug
   with character id, template id and field. Nothing is emitted.
5. On success, `banishCharacter` emits `WARP{characterId, targetMapId,
   targetPortalName?}` on `COMMAND_TOPIC_PORTAL`, then, when `banMsg` is
   non-empty, `SEND_MESSAGE{PINK_TEXT, banMsg}` on
   `COMMAND_TOPIC_SYSTEM_MESSAGE`.
6. `atlas-portals` resolves the portal by name (or falls back) and emits the
   character `CHANGE_MAP` command through the existing `WarpToPortal` path.
7. `atlas-channel`'s system-message consumer matches tenant + world + channel and
   announces the pink text to the character's session.

Tenancy rides the existing header parsers on both hops; the channel's
system-message handler already enforces `t.Is(sc.Tenant())` and
`sc.Is(t, cmd.WorldId, cmd.ChannelId)` before announcing
(`kafka/consumer/system_message/consumer.go:113-118`), and the new producer
inherits that gate unchanged.

---

## 7. Decisions the PRD left open or under-specified

### 7.1 `Banish` return type

The processor returns `error` for every abort, including the validation
rejections, and does **not** log them itself; the consumer logs once. This avoids
the double-log that would result from logging in both places, and keeps every
rejection reason in one grep-able line that carries character id, template id and
field (PRD §8). A successful banish logs the resolved map and portal at Debug from
inside `Banish`.

### 7.2 Rejections are not errors *to the client*

Nothing is written back to the socket on any path. The client does not await a
response to `SendBanMapByMobRequest`, and PRD §4.1 forbids an error path.

### 7.3 Message-vs-warp ordering — PRD contradiction, resolved

PRD §4.3 says: emit the message **after** the warp emit returns without error, so
a player is never told they were banished when they were not. PRD §9 says the fix
for a lost message would be to sequence the message **before** the warp, "which
§4.3 already does" — which is the opposite of what §4.3 says.

**Resolved in favor of §4.3's normative text: warp first, then message.** The
guard §4.3 states — never claim a banish that did not happen — requires warp-first
and is a correctness property; §9's note is a parenthetical about a speculative
symptom. The loss risk §9 worries about is also low: the map change is
same-channel, the session stays connected across it, and `SEND_MESSAGE` is
delivered by character id to whatever session that character holds on the channel.
If live testing shows the message is dropped across the transition, flipping the
order is a one-line change — but it would then be trading a real correctness
guard for a cosmetic one, and should be a deliberate follow-up decision, not a
silent reordering.

A warp emit failure returns the error and skips the message. A *message* emit
failure after a successful warp is logged at Warn and swallowed — the banish
already happened, and there is nothing to roll back.

### 7.4 `ban/banType`

Out of scope per PRD §9; `atlas-data` still does not read it and this task does
not add it.

### 7.5 The `potal` WZ typo

Not handled, per PRD §6. Those three nodes resolve to `atlas-data`'s `"sp"`
default and land the character on a spawn portal — which, with `WarpByName`, now
resolves `"sp"` by name and behaves correctly for the maps that have an `sp`
portal, and falls back to the random spawn for the ones that do not.

### 7.6 `atlas-channel`'s local `WarpBody` copy

Left unchanged (PRD §9). No channel producer needs a portal-name warp in this
task; adding the field speculatively would be dead code.

---

## 8. Testing strategy

All tests are Go unit tests in the existing per-service harnesses. No new test
infrastructure and no `*_testhelpers.go` file.

### `atlas-monsters` — `monster` package

Harness: `newRecordingProcessorWithBodies(t, ten)` (`processor_test.go:236`)
supplies a `ProcessorImpl` whose `emit` records `{Topic, Type, Body}` per message;
`testInformationLookup` (`processor.go:81`) stubs the information fetch;
`GetMonsterRegistry().CreateMonster(...)` populates live field state, exactly as
`kill_test.go` does.

One prerequisite: `information.ModelBuilder` (`information/builder.go`) has no
`SetBanish`. Add `SetBanish(Banish) *ModelBuilder` following the existing setters
— it is required to stub steps 3–4 and to assert the portal/message branches.

Cases (new `monster/banish_test.go`):

| Case | Setup | Assert |
|---|---|---|
| Reject — no live monster of template | registry empty for the field | zero emitted messages, non-nil error |
| Reject — wrong template alive | live monster with a different `MonsterId()` | zero emitted, non-nil error |
| Reject — information fetch fails | `testInformationLookup` returns error | zero emitted, non-nil error |
| Reject — zero banish map | `SetBanish(Banish{MapId: 0})` | zero emitted, non-nil error |
| Accept — portal name present | `Banish{MapId: 926120410, PortalName: "st00"}` | one `WARP` with `targetPortalName == "st00"` |
| Accept — portal name absent | `Banish{MapId: …, PortalName: ""}` | one `WARP`; `targetPortalName` key absent from the marshalled body (`omitempty`) |
| Accept — message present | `Banish{…, Message: "…"}` | two messages, `WARP` first, then `SEND_MESSAGE` with `messageType == "PINK_TEXT"` and the exact text |
| Accept — message absent | `Banish{…, Message: ""}` | exactly one message, no `SEND_MESSAGE` |
| Skill path converges | `executeBanish` via a mob-skill-129 model with a banish carrying portal + message | same `WARP` + `SEND_MESSAGE` shape as the command path |

The `GetInField` lookup-error abort is covered structurally by the empty-registry
case; the registry read does not fail independently in-process, so no separate
case is written for it. This is stated here so its absence is a decision rather
than an omission.

Also add a producer-shape test in the existing `producer_test.go` style pinning
the `BANISH` envelope (`monsterId: 0`, `type: "BANISH"`) — mirroring
`producer_magnet_test.go`'s pinning of the `CLEAR_AGGRO` envelope.

### `atlas-channel` — `monster` package

`monster/producer_*_test.go` pattern: decode `BanishCommandProvider`'s single
message and assert `Type == "BANISH"`, `MonsterId == 0`, body fields, and that the
key is the character id. This is the mirror-file guard for the two hand-copied
body structs.

### `atlas-portals` — `portal` package

Harness: `setupMockDataServer(t, …)` (`portal/processor_test.go`).

| Case | Assert |
|---|---|
| `WarpByName` hit | resolves `/api/data/maps/{id}/portals?name=st00` and warps to that portal id |
| `WarpByName` miss | falls back to the random-spawn `Warp` path; the warp is not dropped |
| `handleWarpCommand` precedence | `UseTargetPosition` wins over a set name; `TargetPortalId != 0` wins over a set name; name used when both are unset; random spawn when all three unset |

### Gate

`tools/verify.sh` flagless must exit 0 before the branch is claimed done.

---

## 9. Risks

| Risk | Mitigation |
|---|---|
| The two hand-copied `BanishCommandBody`/`banishCommandBody` structs drift | Mirror comments on both (the convention already used for `ClearAggro`/`ForceControl`), plus the channel producer-shape test |
| A new field on a shared fan-to-every-handler topic causes spurious unmarshal errors in sibling handlers | Collision audit in §4.2; both field names are either type-identical to existing siblings or entirely new |
| `targetPortalName` changes an existing `WARP` producer's bytes | `omitempty` on the monsters-side producer; explicit test asserting the key is absent when unset. `atlas-portals`' consumer-side field needs no `omitempty` (it is never marshalled) |
| Message renders on the wrong side of the map transition | Accepted (PRD §9); §7.3 records the ordering decision and the one-line reversal if live testing disagrees |
| Refactoring `executeBanish` onto `p.emit` changes skill-129 behavior | The emit target and topic are unchanged; only the injection seam and the two added fields differ. The skill-path convergence test pins the resulting shape |

---

## 10. Files touched

**`atlas-channel`**
- `kafka/message/monster/kafka.go` — `CommandTypeBanish`, `BanishCommandBody`
- `monster/producer.go` — `BanishCommandProvider`
- `monster/processor.go` — `Banish` on interface + impl
- `socket/handler/mob_banish_player.go` — emit instead of the deferred comment
- `monster/producer_*_test.go` — envelope pin

**`atlas-monsters`**
- `kafka/consumer/monster/kafka.go` — `CommandTypeBanish`, `banishCommandBody`
- `kafka/consumer/monster/consumer.go` — `handleBanishCommand` + registration
- `monster/processor.go` — `Banish`, `banishCharacter`, `executeBanish` rewrite
- `monster/disease.go` — `warpBody.TargetPortalName`, widened
  `warpCommandProvider`, `sendMessageProvider`
- `monster/information/builder.go` — `SetBanish`
- `kafka/message/system_message/kafka.go` — new local package
- `monster/banish_test.go` — new

**`atlas-portals`**
- `portal/kafka.go` — `warpBody.TargetPortalName`
- `portal/processor.go` — `WarpByName` on interface + impl
- `portal/consumer.go` — precedence branch
- `portal/processor_test.go` / `portal/consumer_test.go` — new cases

**Docs** — `services/atlas-channel/docs/kafka.md`,
`services/atlas-monsters/docs/kafka.md`, `services/atlas-portals/docs/kafka.md`
record the new command type and the new `WARP` field.

**Unchanged** — `atlas-data`, `atlas-channel`'s `data/monster` projection and its
local `WarpBody` copy, `libs/atlas-packet`, all deploy manifests.
