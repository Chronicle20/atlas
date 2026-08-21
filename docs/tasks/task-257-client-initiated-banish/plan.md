# Client-Initiated Mob Banish Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the client's `MOB_BANISH_PLAYER` request actually banish the character — validated against live field state in `atlas-monsters` — and give both banish paths the WZ portal name and banish message.

**Architecture:** `atlas-channel` decodes the packet and forwards a `BANISH` command on `COMMAND_TOPIC_MONSTER`. `atlas-monsters` — the only service holding live monster state — validates the client-supplied template id against monsters alive in the requesting character's field (the trust boundary), then runs a shared `banishCharacter` executor also used by the existing skill-129 path. That executor emits a `WARP` carrying the new optional `targetPortalName` and, when the WZ `banMsg` is non-empty, a `SEND_MESSAGE`/`PINK_TEXT`. `atlas-portals` resolves the portal name, falling back to the random spawn on a miss.

**Tech Stack:** Go microservices, Kafka (`segmentio/kafka-go` via `atlas-kafka`), JSON:API REST via `atlas-rest`, `logrus`, standard Go `testing`.

**Spec:** [design.md](design.md) (PRD: [prd.md](prd.md))

## Global Constraints

- The mob template id is client-controlled and untrusted. Validation MUST fail closed — no banish executes on the strength of the packet alone (PRD §8, design §4.1-A).
- No existing `WARP` producer's emitted body may change. `TargetPortalName` on the **producer** side (`atlas-monsters`) carries `json:"targetPortalName,omitempty"`; on the **consumer** side (`atlas-portals`) it is `json:"targetPortalName"` with no `omitempty` (never marshalled).
- `COMMAND_TOPIC_MONSTER` fans every message to every handler, which each JSON-unmarshal it. New body field names must be either type-identical to an existing sibling or entirely new. `characterId uint32` is type-identical everywhere; `monsterTemplateId` is new.
- The `BANISH` envelope's `monsterId` is `0`. It means *unique* id everywhere else on that topic; overloading it would look like a well-formed command for a nonexistent monster.
- Ordering is warp first, then message (design §7.3 resolves the PRD's internal contradiction in favour of PRD §4.3's normative text). A warp emit failure returns the error and skips the message. A message emit failure after a successful warp is logged at Warn and swallowed.
- Nothing is ever written back to the socket on any banish path; the client does not await a response.
- No `*_testhelpers.go` files. Test setup uses the existing per-package harnesses and the project Builder pattern.
- Do not touch: `atlas-data`, `libs/atlas-packet`, `atlas-channel`'s `data/monster` projection, `atlas-channel`'s local `WarpBody` copy, deploy manifests.

---

### Task 1: `atlas-portals` — resolve a warp by portal name

**Interfaces:**
- Produces: `portal.Processor.WarpByName(f field.Model, characterId uint32, targetMapId _map.Id, name string)` and the `warpBody.TargetPortalName` JSON field `targetPortalName`, which Task 4 produces from `atlas-monsters`.

### Files

- `services/atlas-portals/atlas.com/portals/portal/kafka.go` — add `TargetPortalName` to `warpBody` (line 38-47)
- `services/atlas-portals/atlas.com/portals/portal/processor.go` — add `WarpByName` to the `Processor` interface (line 20-31) and the impl (after `WarpById`, line 137-139)
- `services/atlas-portals/atlas.com/portals/portal/consumer.go` — insert the name branch in `handleWarpCommand` (line 55-72)
- `services/atlas-portals/atlas.com/portals/portal/warp_by_name_test.go` — new file
- `services/atlas-portals/docs/kafka.md` — add the `body.targetPortalName` row to the `warpEvent` table (line ~45)

Module root for `go build ./... && go test ./...`: `services/atlas-portals/atlas.com/portals`

Patterns to copy:
- `services/atlas-portals/atlas.com/portals/portal/processor.go:116-132` — the resolve-name / warn / fall-back shape inside `Enter`
- `services/atlas-portals/atlas.com/portals/portal/processor_test.go:55-92` — `setupMockDataServer`; `:257-292` — `GetInMapByName` hit and miss fixtures
- `services/atlas-portals/atlas.com/portals/portal/consumer_test.go:31-70` — `createTestContext` and `setupMockDataServerForConsumer`

The new test file is `package portal_test` (like `processor_test.go`), so it can reuse `setupMockDataServer`, `createTestField`, `createPortalResource`, `jsonAPIResponse` and `jsonAPIResource` from that file without redeclaring them.

`WarpToPortal` publishes to Kafka, which is absent in tests. Assertions are therefore on **which data-service paths the mock was asked for** and on the logged warning — the same style the existing `Enter` tests use. The new test file adds its own recording mock (`setupRecordingDataServer`) because `setupMockDataServer` does not record paths.

- [ ] **Step 1: Write the failing tests**

New file `portal/warp_by_name_test.go`, `package portal_test`.

`setupRecordingDataServer(t *testing.T, responses map[string]interface{}) (*[]string, func())` — same body as `setupMockDataServer` (`processor_test.go:56-92`) except it appends every `fullPath` to a captured `[]string` and returns a pointer to it instead of the server.

| test func | setup | assert |
|---|---|---|
| `TestWarpByName_Hit` | recording mock serves `/api/data/maps/200000000/portals?name=st00` → `jsonAPIResponse{Data: []jsonAPIResource{createPortalResource("7", "st00", "", 999999999, "")}}`; call `portal.NewProcessor(logger, ctx).WarpByName(createTestField(100000000), 12345, 200000000, "st00")` | recorded paths contain `/api/data/maps/200000000/portals?name=st00`; recorded paths do NOT contain `/api/data/maps/200000000/portals` (the random-spawn drain), i.e. no fallback happened; no `logrus.WarnLevel` entry |
| `TestWarpByName_MissFallsBackToRandomSpawn` | recording mock serves `/api/data/maps/200000000/portals?name=nope` → `jsonAPIResponse{Data: []jsonAPIResource{}}` and `/api/data/maps/200000000/portals` → `jsonAPIResponse{Data: []jsonAPIResource{createPortalResource("0", "sp", "", 999999999, "")}}`; call `WarpByName(..., 12345, 200000000, "nope")` | exactly one `logrus.WarnLevel` entry whose message contains `nope`, `200000000` and `12345`; recorded paths contain BOTH the `?name=nope` lookup and the bare `/api/data/maps/200000000/portals` drain — the warp was not dropped |

Logger for both: `logger, hook := logtest.NewNullLogger(); logger.SetLevel(logrus.DebugLevel)`. Context: `test.CreateTestContext()` (import `atlas-portals/test`, as `processor_test.go:5` does).

`handleWarpCommand` precedence — `package portal` (internal), so put these in the existing `portal/consumer_test.go` rather than the new external-test file. Table-driven `TestHandleWarpCommand_Precedence`, setup copied from `consumer_test.go:72-129` (`setupMockDataServerForConsumer` + `createTestContext`), each case building a `warpEvent{WorldId: 1, ChannelId: 1, MapId: 100000000, Type: CommandTypeWarp, Body: <row>}` and calling `handleWarpCommand(logger, ctx, cmd)`:

| subtest name | body | expected debug log substrings | data-service path expected |
|---|---|---|---|
| `position wins over name` | `warpBody{CharacterId: 1, TargetMapId: 200000000, TargetPortalName: "st00", UseTargetPosition: true, TargetX: 10, TargetY: 20}` | `position` | none (no lookup at all) |
| `portal id wins over name` | `warpBody{CharacterId: 2, TargetMapId: 200000000, TargetPortalId: 5, TargetPortalName: "st00"}` | `portal [5]` | none |
| `name used when id and position unset` | `warpBody{CharacterId: 3, TargetMapId: 200000000, TargetPortalName: "st00"}` | `portal [st00]` | `/api/data/maps/200000000/portals?name=st00` |
| `random spawn when all unset` | `warpBody{CharacterId: 4, TargetMapId: 200000000}` | neither `portal [` nor `position` | `/api/data/maps/200000000/portals` |

Assert the log substrings with the existing `containsAll` helper (`consumer_test.go:232`). Serve `/api/data/maps/200000000/portals?name=st00` → one portal resource and `/api/data/maps/200000000/portals` → one portal resource so no case errors out.

- [ ] **Step 2: Run the tests to verify they fail**

```bash
cd services/atlas-portals/atlas.com/portals && go test ./portal/... -run 'WarpByName|HandleWarpCommand_Precedence' -v
```

Expected: compile failure — `WarpByName` undefined, `warpBody` has no field `TargetPortalName`.

- [ ] **Step 3: Add `TargetPortalName` to `warpBody`**

In `portal/kafka.go`, inside `warpBody` after `TargetPortalId`:

```go
	// TargetPortalName, when non-empty and TargetPortalId is zero, lands the
	// character on the portal of that name in the target map. Resolution
	// failure falls back to the random-spawn Warp rather than dropping the
	// warp — failing to banish is worse than banishing to a default spawn.
	TargetPortalName string `json:"targetPortalName"`
```

- [ ] **Step 4: Add `WarpByName` to the interface and impl**

In `portal/processor.go`, add to the `Processor` interface after `WarpById`:

```go
	WarpByName(f field.Model, characterId uint32, targetMapId _map.Id, name string)
```

and after the `WarpById` impl:

```go
// WarpByName lands the character on the named portal of the target map. A name
// that does not resolve falls back to the random-spawn Warp with a warning
// rather than dropping the warp; the same resolve-warn-default shape Enter
// uses for a portal's declared target.
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

- [ ] **Step 5: Insert the precedence branch in `handleWarpCommand`**

In `portal/consumer.go`, between the `TargetPortalId != 0` branch and the trailing random-spawn call:

```go
	if command.Body.TargetPortalName != "" {
		l.Debugf("Received command for Character [%d] to warp to map [%d] portal [%s] from map [%d].", command.Body.CharacterId, command.Body.TargetMapId, command.Body.TargetPortalName, command.MapId)
		NewProcessor(l, ctx).WarpByName(f, command.Body.CharacterId, command.Body.TargetMapId, command.Body.TargetPortalName)
		return
	}
```

- [ ] **Step 6: Run the tests to verify they pass**

```bash
cd services/atlas-portals/atlas.com/portals && go test ./portal/... -run 'WarpByName|HandleWarpCommand_Precedence' -v
```

Expected: PASS.

- [ ] **Step 7: Run the module build and full test suite**

```bash
cd services/atlas-portals/atlas.com/portals && go build ./... && go test ./...
```

Expected: all packages ok.

- [ ] **Step 8: Update `services/atlas-portals/docs/kafka.md`**

In the `warpEvent` field table, insert a row directly after the `body.targetPortalId` row:

```
| body.targetPortalName | string | Target portal name; used when targetPortalId is 0. Unresolvable names fall back to a random spawn point |
```

- [ ] **Step 9: Commit**

```bash
git add services/atlas-portals/atlas.com/portals/portal/kafka.go \
        services/atlas-portals/atlas.com/portals/portal/processor.go \
        services/atlas-portals/atlas.com/portals/portal/consumer.go \
        services/atlas-portals/atlas.com/portals/portal/consumer_test.go \
        services/atlas-portals/atlas.com/portals/portal/warp_by_name_test.go \
        services/atlas-portals/docs/kafka.md
git commit -m "feat(atlas-portals): resolve WARP commands by target portal name"
```

---

### Task 2: `atlas-channel` — emit the `BANISH` command

**Interfaces:**
- Consumes: nothing from earlier tasks.
- Produces: the wire contract Task 3 consumes — envelope `Command[E]` with `MonsterId: 0`, `Type: "BANISH"`, body `{"characterId": uint32, "monsterTemplateId": uint32}`, partition key = character id.

### Files

- `services/atlas-channel/atlas.com/channel/kafka/message/monster/kafka.go` — add `CommandTypeBanish` to the const block (line 13-22) and `BanishCommandBody` (after `ForceControlCommandBody`, line 122-124)
- `services/atlas-channel/atlas.com/channel/monster/producer.go` — add `BanishCommandProvider` (after `ForceControlCommandProvider`, line 210-224)
- `services/atlas-channel/atlas.com/channel/monster/processor.go` — add `Banish` to the `Processor` interface (line 17-35) and the impl (after `ForceControl`, line 169-172)
- `services/atlas-channel/atlas.com/channel/socket/handler/mob_banish_player.go` — replace the `// behavior: deferred` comment on line 19 with the emit
- `services/atlas-channel/atlas.com/channel/monster/producer_banish_test.go` — new file
- `services/atlas-channel/docs/kafka.md` — the `COMMAND_TOPIC_MONSTER` entry (line 490-494)

Module root for `go build ./... && go test ./...`: `services/atlas-channel/atlas.com/channel`

Patterns to copy:
- `services/atlas-channel/atlas.com/channel/socket/handler/monster_damage_friendly.go:16-23` — the exact handler shape (decode, debug log, `_ = monster.NewProcessor(l, ctx).X(...)`)
- `services/atlas-channel/atlas.com/channel/monster/producer.go:208-224` — `ForceControlCommandProvider`
- `services/atlas-channel/atlas.com/channel/monster/producer_magnet_test.go:13-72` — provider-shape test, including `magnetTestField()`

- [ ] **Step 1: Write the failing test**

New file `monster/producer_banish_test.go`, `package monster` (internal, like `producer_magnet_test.go`). It reuses `magnetTestField()` from `producer_magnet_test.go` — do not redeclare it.

`TestBanishCommandProviderShape` — call `BanishCommandProvider(magnetTestField(), 4242, 9500324)`:

| assertion | expected |
|---|---|
| `err` from the provider | nil |
| `len(msgs)` | 1 |
| `c.Type` (unmarshalled into `monster2.Command[monster2.BanishCommandBody]`) | `monster2.CommandTypeBanish` |
| `c.MonsterId` | `0` — the client supplies a template id, not a unique id |
| `c.Body.CharacterId` | `4242` |
| `c.Body.MonsterTemplateId` | `9500324` |

`TestBanishCommandKeysOnCharacter` — the `BANISH` key must be the character id, not the monster id, so a character's banish requests stay ordered against each other:

```go
banish, err := BanishCommandProvider(magnetTestField(), 4242, 9500324)()
// ...
force, err := ForceControlCommandProvider(magnetTestField(), 4242, 777)()
// ...
```
Assert `string(banish[0].Key) == string(force[0].Key)` — `ForceControlCommandProvider(f, 4242, 777)` keys on monster id `4242` and `BanishCommandProvider(f, 4242, ...)` keys on character id `4242`, so identical keys prove `BanishCommandProvider` keyed on its *first* uint32 argument (the character id) rather than on the template id `9500324`. Add a comment saying exactly that, so the equality is not misread as "both key on monster id".

- [ ] **Step 2: Run the test to verify it fails**

```bash
cd services/atlas-channel/atlas.com/channel && go test ./monster/... -run Banish -v
```

Expected: compile failure — `BanishCommandProvider` undefined, `monster2.CommandTypeBanish` undefined.

- [ ] **Step 3: Add the command type and body**

In `kafka/message/monster/kafka.go`, add to the const block after `CommandTypeForceControl`:

```go
	CommandTypeBanish         = "BANISH"
```

(align with the existing gofmt column) and after `ForceControlCommandBody`:

```go
// BanishCommandBody asks atlas-monsters to banish a character out of a field on
// the strength of a client MOB_BANISH_PLAYER request. MonsterTemplateId is
// client-supplied and untrusted; atlas-monsters revalidates it against live
// field state before acting. Both fields are uint32: characterId already
// appears at that name and type in DamageCommandBody / KillCommandBody /
// ForceControlCommandBody, and monsterTemplateId appears in no sibling body, so
// neither can collide on the shared, fan-to-every-handler command topic. The
// envelope's monsterId is deliberately left 0 — it means *unique* id everywhere
// else on this topic. Mirrors atlas-monsters' banishCommandBody — edit both
// together.
type BanishCommandBody struct {
	CharacterId       uint32 `json:"characterId"`
	MonsterTemplateId uint32 `json:"monsterTemplateId"`
}
```

- [ ] **Step 4: Add the provider**

Append to `monster/producer.go`:

```go
// BanishCommandProvider asks atlas-monsters to honor a client MOB_BANISH_PLAYER
// request. Keyed on the character id rather than the monster id — unlike every
// other monster command here — because the command is about a character's map
// transition, and the ordering that matters is this character's banish requests
// against each other. MonsterId is 0: the client supplies a template id, and
// the envelope field means *unique* id everywhere else on this topic.
func BanishCommandProvider(f field.Model, characterId uint32, monsterTemplateId uint32) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &monster2.Command[monster2.BanishCommandBody]{
		WorldId:   f.WorldId(),
		ChannelId: f.ChannelId(),
		MapId:     f.MapId(),
		Instance:  f.Instance(),
		MonsterId: 0,
		Type:      monster2.CommandTypeBanish,
		Body: monster2.BanishCommandBody{
			CharacterId:       characterId,
			MonsterTemplateId: monsterTemplateId,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
```

- [ ] **Step 5: Add `Banish` to the processor**

In `monster/processor.go`, add to the `Processor` interface after `ForceControl`:

```go
	Banish(f field.Model, characterId uint32, monsterTemplateId uint32) error
```

and append the impl:

```go
// Banish forwards a client MOB_BANISH_PLAYER request to atlas-monsters, which
// owns live monster state and is the only service that can validate the
// client-supplied template id against the field. The channel makes no banish
// decision and resolves no monster data.
func (p *ProcessorImpl) Banish(f field.Model, characterId uint32, monsterTemplateId uint32) error {
	p.l.Debugf("Character [%d] requesting banish by monster template [%d].", characterId, monsterTemplateId)
	return producer.ProviderImpl(p.l)(p.ctx)(monster2.EnvCommandTopic)(BanishCommandProvider(f, characterId, monsterTemplateId))
}
```

- [ ] **Step 6: Run the test to verify it passes**

```bash
cd services/atlas-channel/atlas.com/channel && go test ./monster/... -run Banish -v
```

Expected: PASS.

- [ ] **Step 7: Wire the handler**

In `socket/handler/mob_banish_player.go`, replace line 19 (`// behavior: deferred (decode-and-log only)`) with:

```go
		_ = monster.NewProcessor(l, ctx).Banish(s.Field(), s.CharacterId(), p.MobTemplateId())
```

and add `"atlas-channel/monster"` to the import block as the first entry (matching `monster_damage_friendly.go:4`). Keep the existing `l.Debugf` line above it. No stub, no TODO, and nothing is written back to the socket.

- [ ] **Step 8: Run the module build and full test suite**

```bash
cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./...
```

Expected: all packages ok.

- [ ] **Step 9: Update `services/atlas-channel/docs/kafka.md`**

In the `### COMMAND_TOPIC_MONSTER` entry, extend the `Message Type:` line with `, Command[BanishCommandBody]` and append to the `Purpose:` sentence:

```
BANISH forwards a client MOB_BANISH_PLAYER request (CharacterId, MonsterTemplateId) with MonsterId 0; atlas-monsters validates the client-supplied template id against live field state before acting.
```

- [ ] **Step 10: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/kafka/message/monster/kafka.go \
        services/atlas-channel/atlas.com/channel/monster/producer.go \
        services/atlas-channel/atlas.com/channel/monster/processor.go \
        services/atlas-channel/atlas.com/channel/monster/producer_banish_test.go \
        services/atlas-channel/atlas.com/channel/socket/handler/mob_banish_player.go \
        services/atlas-channel/docs/kafka.md
git commit -m "feat(atlas-channel): emit BANISH on MOB_BANISH_PLAYER"
```

---

### Task 3: `atlas-monsters` — banish plumbing (system-message package, portal name, builder setter)

This task lands the three mechanical prerequisites Task 4's logic needs. It is deliberately kept separate so Task 4's review surface is the validation and execution logic alone.

**Interfaces:**
- Consumes: the `targetPortalName` JSON key defined by Task 1.
- Produces, for Task 4:
  - `system_message.EnvCommandTopic`, `system_message.CommandSendMessage`, `system_message.Command[E]`, `system_message.SendMessageBody` in package `atlas-monsters/kafka/message/system_message`
  - `warpCommandProvider(f field.Model, characterId uint32, targetMapId _map.Id, portalName string) model.Provider[[]kafka.Message]` (widened by one argument)
  - `sendMessageProvider(f field.Model, characterId uint32, messageType string, msg string) model.Provider[[]kafka.Message]`
  - `information.(*ModelBuilder).SetBanish(b Banish) *ModelBuilder`

### Files

- `services/atlas-monsters/atlas.com/monsters/kafka/message/system_message/kafka.go` — **new file**
- `services/atlas-monsters/atlas.com/monsters/monster/disease.go` — `warpBody.TargetPortalName`, widen `warpCommandProvider` (line 97-116), add `sendMessageProvider`
- `services/atlas-monsters/atlas.com/monsters/monster/processor.go` — update the single existing `warpCommandProvider` call site inside `executeBanish` (line 1263) to pass `ma.Banish().PortalName`
- `services/atlas-monsters/atlas.com/monsters/monster/information/builder.go` — add the `banish` field and `SetBanish`
- `services/atlas-monsters/atlas.com/monsters/monster/banish_producer_test.go` — new file
- `services/atlas-monsters/atlas.com/monsters/monster/information/model.go` — read-only; `Banish{Message, MapId, PortalName}` at line 26-30, accessor `Banish()` at line 74

Module root for `go build ./... && go test ./...`: `services/atlas-monsters/atlas.com/monsters`

Patterns to copy:
- `services/atlas-party-quests/atlas.com/party-quests/kafka/message/system_message/kafka.go` — read-only; the whole file is the template for the new package (copy it verbatim, change only the package doc comment)
- `services/atlas-party-quests/atlas.com/party-quests/instance/producer.go:186-200` — `sendMessageProvider`
- `services/atlas-monsters/atlas.com/monsters/kafka/message/mist/kafka.go:1-6` — the "mirrors another service's file" package doc comment convention
- `services/atlas-monsters/atlas.com/monsters/monster/information/builder.go:41-56` — existing setter shape
- `services/atlas-monsters/atlas.com/monsters/monster/producer_test.go:15-44` — provider-shape test shape (`package monster`, decode `msgs[0].Value` into an anonymous envelope struct)

- [ ] **Step 1: Write the failing tests**

New file `monster/banish_producer_test.go`, `package monster`. Field for both: `field.NewBuilder(world.Id(1), channel.Id(2), _map.Id(100000000)).SetInstance(uuid.Nil).Build()`.

`TestWarpCommandProviderCarriesPortalName` — call `warpCommandProvider(f, 4242, _map.Id(926120410), "st00")`:

| assertion | expected |
|---|---|
| `len(msgs)` | 1 |
| envelope `type` | `"WARP"` |
| `body.characterId` | `4242` |
| `body.targetMapId` | `926120410` |
| `body.targetPortalName` | `"st00"` |

Decode into an anonymous struct `struct{ Type string \`json:"type"\`; Body warpBody \`json:"body"\` }`.

`TestWarpCommandProviderOmitsEmptyPortalName` — call `warpCommandProvider(f, 4242, _map.Id(926120410), "")`, then unmarshal `msgs[0].Value` into `map[string]json.RawMessage`, take `raw["body"]`, unmarshal *that* into `map[string]json.RawMessage`, and assert the key `targetPortalName` is **absent**. This is the acceptance criterion that no existing `WARP` producer's body changes.

`TestSendMessageProviderShape` — call `sendMessageProvider(f, 4242, "PINK_TEXT", "You have been banished.")`:

| assertion | expected |
|---|---|
| `len(msgs)` | 1 |
| envelope `type` | `system_message.CommandSendMessage` (`"SEND_MESSAGE"`) |
| envelope `worldId` | `1` |
| envelope `channelId` | `2` |
| envelope `characterId` | `4242` |
| `body.messageType` | `"PINK_TEXT"` |
| `body.message` | `"You have been banished."` |

Decode into `system_message.Command[system_message.SendMessageBody]`, importing `system_message "atlas-monsters/kafka/message/system_message"`.

`TestModelBuilderSetBanish` — in the same file:

```go
b := information.Banish{Message: "Get out.", MapId: 926120410, PortalName: "st00"}
m := information.NewModelBuilder().SetBanish(b).Build()
```
assert `m.Banish() == b` (a comparable struct of three scalars). Import `"atlas-monsters/monster/information"`.

- [ ] **Step 2: Run the tests to verify they fail**

```bash
cd services/atlas-monsters/atlas.com/monsters && go test ./monster/... -run 'WarpCommandProvider|SendMessageProvider|SetBanish' -v
```

Expected: compile failure — package `atlas-monsters/kafka/message/system_message` does not exist, `sendMessageProvider` undefined, `SetBanish` undefined, and `warpCommandProvider` takes 3 arguments not 4.

- [ ] **Step 3: Create the local `system_message` package**

New file `kafka/message/system_message/kafka.go` — the same shapes as `services/atlas-party-quests/atlas.com/party-quests/kafka/message/system_message/kafka.go`, with a package doc comment in the `mist` convention:

```go
// Package system_message defines the wire shape for atlas-channel's system
// message commands. The types mirror
// services/atlas-party-quests/atlas.com/party-quests/kafka/message/system_message/kafka.go
// (matching JSON tags) so atlas-monsters can publish SEND_MESSAGE commands
// without importing across a service boundary. This local-copy-per-service
// pattern is the established convention; no shared library is introduced.
package system_message

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

const (
	EnvCommandTopic    = "COMMAND_TOPIC_SYSTEM_MESSAGE"
	CommandSendMessage = "SEND_MESSAGE"
)

type Command[E any] struct {
	TransactionId uuid.UUID  `json:"transactionId"`
	WorldId       world.Id   `json:"worldId"`
	ChannelId     channel.Id `json:"channelId"`
	CharacterId   uint32     `json:"characterId"`
	Type          string     `json:"type"`
	Body          E          `json:"body"`
}

type SendMessageBody struct {
	MessageType string `json:"messageType"`
	Message     string `json:"message"`
}
```

- [ ] **Step 4: Widen `warpBody` and `warpCommandProvider`, add `sendMessageProvider`**

In `monster/disease.go`, replace the `warpBody` struct and `warpCommandProvider`:

```go
type warpBody struct {
	CharacterId uint32  `json:"characterId"`
	TargetMapId _map.Id `json:"targetMapId"`
	// TargetPortalName, when non-empty, lands the character on the portal of
	// that name in the target map (WZ ban/banMap/0/portal). omitempty keeps an
	// omitting producer's bytes byte-identical to today.
	TargetPortalName string `json:"targetPortalName,omitempty"`
}

func warpCommandProvider(f field.Model, characterId uint32, targetMapId _map.Id, portalName string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &warpCommand{
		WorldId:   f.WorldId(),
		ChannelId: f.ChannelId(),
		MapId:     f.MapId(),
		Instance:  f.Instance(),
		Type:      "WARP",
		Body: warpBody{
			CharacterId:      characterId,
			TargetMapId:      targetMapId,
			TargetPortalName: portalName,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
```

and append:

```go
// sendMessageProvider builds a SEND_MESSAGE command for atlas-channel to
// announce text to a character's session. Used for the WZ banish message.
func sendMessageProvider(f field.Model, characterId uint32, messageType string, msg string) model.Provider[[]kafka.Message] {
	key := producer.CreateKey(int(characterId))
	value := &system_message.Command[system_message.SendMessageBody]{
		TransactionId: uuid.Nil,
		WorldId:       f.WorldId(),
		ChannelId:     f.ChannelId(),
		CharacterId:   characterId,
		Type:          system_message.CommandSendMessage,
		Body: system_message.SendMessageBody{
			MessageType: messageType,
			Message:     msg,
		},
	}
	return producer.SingleMessageProvider(key, value)
}
```

Add `system_message "atlas-monsters/kafka/message/system_message"` to the import block. `uuid` is already imported (`disease.go:4`).

- [ ] **Step 5: Update the one existing `warpCommandProvider` call site**

In `monster/processor.go:1263`, inside `executeBanish`, change

```go
		err := producer.ProviderImpl(p.l)(p.ctx)(EnvCommandTopicPortal)(warpCommandProvider(m.Field(), characterId, map2.Id(banishMapId)))
```

to pass the portal name as the new fourth argument:

```go
		err := producer.ProviderImpl(p.l)(p.ctx)(EnvCommandTopicPortal)(warpCommandProvider(m.Field(), characterId, map2.Id(banishMapId), ma.Banish().PortalName))
```

(This call site is rewritten again in Task 4; passing the name here keeps the tree compiling and correct in between.)

- [ ] **Step 6: Add `SetBanish` to `ModelBuilder`**

In `monster/information/builder.go`, add `banish Banish` to the `ModelBuilder` struct, then after `SetResistances`:

```go
// SetBanish sets the banish node on the builder. Used by tests that drive the
// banish paths in Banish and executeBanish.
func (b *ModelBuilder) SetBanish(banish Banish) *ModelBuilder {
	b.banish = banish
	return b
}
```

and add `banish: b.banish,` to the struct literal returned by `Build()`.

- [ ] **Step 7: Run the tests to verify they pass**

```bash
cd services/atlas-monsters/atlas.com/monsters && go test ./monster/... -run 'WarpCommandProvider|SendMessageProvider|SetBanish' -v
```

Expected: PASS.

- [ ] **Step 8: Run the module build and full test suite**

```bash
cd services/atlas-monsters/atlas.com/monsters && go build ./... && go test ./...
```

Expected: all packages ok.

- [ ] **Step 9: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/kafka/message/system_message/kafka.go \
        services/atlas-monsters/atlas.com/monsters/monster/disease.go \
        services/atlas-monsters/atlas.com/monsters/monster/processor.go \
        services/atlas-monsters/atlas.com/monsters/monster/information/builder.go \
        services/atlas-monsters/atlas.com/monsters/monster/banish_producer_test.go
git commit -m "feat(atlas-monsters): carry the banish portal name and add a system-message producer"
```

---

### Task 4: `atlas-monsters` — validate and execute the banish

**Interfaces:**
- Consumes: everything Task 3 produced, plus the wire contract Task 2 produces.
- Produces: `monster.Processor.Banish(f field.Model, characterId uint32, monsterTemplateId uint32) error` and the private `(*ProcessorImpl).banishCharacter(f field.Model, characterId uint32, b information.Banish) error`.

### Files

- `services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/kafka.go` — add `CommandTypeBanish` to the const block (line 11-34) and `banishCommandBody` (after `forceControlCommandBody`, line 163-165)
- `services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/consumer.go` — register and define `handleBanishCommand`
- `services/atlas-monsters/atlas.com/monsters/monster/processor.go` — add `Banish` to the `Processor` interface, add `Banish` + `banishCharacter` impls, rewrite `executeBanish` (line 1247-1268)
- `services/atlas-monsters/atlas.com/monsters/monster/banish_test.go` — new file
- `services/atlas-monsters/docs/kafka.md` — the `COMMAND_TOPIC_PORTAL` / `WARP` section (line 892-914)
- `services/atlas-monsters/atlas.com/monsters/monster/model.go` — read-only; `MonsterId()` accessor at line 135

Module root for `go build ./... && go test ./...`: `services/atlas-monsters/atlas.com/monsters`

Patterns to copy:
- `services/atlas-monsters/atlas.com/monsters/monster/kill_test.go:22-71` — the whole harness: registry `Clear`/`CreateMonster`, `testInformationLookup` save-and-restore, `newRecordingProcessorWithBodies`, per-event body decode
- `services/atlas-monsters/atlas.com/monsters/monster/processor_test.go:236-263` — `newRecordingProcessorWithBodies` and the `emittedBody{Topic, Type, Body}` shape
- `services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/consumer.go:229-238` — `handleAddPuppetCommand`'s field construction

- [ ] **Step 1: Write the failing tests**

New file `monster/banish_test.go`, `package monster`. Every case follows `kill_test.go`'s opening: `r := GetMonsterRegistry()`, `ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)`, `ctx := context.Background()`, `r.Clear(ctx)`, then save/restore `testInformationLookup` with `defer`. Field: `f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()`. Live monsters are created with `r.CreateMonster(ctx, ten, f, <templateId>, 0, 0, 0, 5, 0, 5000, 100, "", "")`. Processor: `p, events := newRecordingProcessorWithBodies(t, ten)`.

Fixture constants at the top of the file:

```go
const (
	banishTemplateId = uint32(9500324)
	banishMapId      = uint32(926120410)
	banishPortal     = "st00"
	banishMessage    = "You do not belong here."
	banishCharacter  = uint32(4242)
)
```

`TestBanish` — table-driven over the four rejections. Each row: seed the registry, install `testInformationLookup`, call `err := p.Banish(f, banishCharacter, banishTemplateId)`, assert `err != nil` and `len(*events) == 0`.

| subtest | registry seeded with | `testInformationLookup` returns | expect |
|---|---|---|---|
| `no live monster in field` | nothing | never called; return `information.Model{}, nil` | non-nil error, 0 emitted |
| `wrong template alive` | one monster, template `1000000` | never called; return `information.Model{}, nil` | non-nil error, 0 emitted |
| `information fetch fails` | one monster, template `9500324` | `information.Model{}, errors.New("boom")` | non-nil error, 0 emitted |
| `zero banish map` | one monster, template `9500324` | `information.NewModelBuilder().SetBanish(information.Banish{MapId: 0}).Build(), nil` | non-nil error, 0 emitted |

`TestBanish_PortalNamePresent` — registry has template `9500324` in `f`; lookup returns `SetBanish(information.Banish{MapId: banishMapId, PortalName: banishPortal})`. Call `p.Banish(f, banishCharacter, banishTemplateId)`. Assert: `err == nil`; `len(*events) == 1`; `(*events)[0].Topic == EnvCommandTopicPortal`; `(*events)[0].Type == "WARP"`; decoding `(*events)[0].Body` into `warpBody` gives `CharacterId == 4242`, `TargetMapId == _map.Id(926120410)`, `TargetPortalName == "st00"`.

`TestBanish_PortalNameAbsent` — same, with `PortalName: ""`. Assert one `WARP`, and that decoding `(*events)[0].Body` into `map[string]json.RawMessage` shows no `targetPortalName` key.

`TestBanish_MessagePresent` — lookup returns `SetBanish(information.Banish{MapId: banishMapId, PortalName: banishPortal, Message: banishMessage})`. Assert: `err == nil`; `len(*events) == 2`; `(*events)[0].Type == "WARP"` on `EnvCommandTopicPortal`; `(*events)[1].Topic == system_message.EnvCommandTopic` and `(*events)[1].Type == system_message.CommandSendMessage`; decoding `(*events)[1].Body` into `system_message.SendMessageBody` gives `MessageType == "PINK_TEXT"` and `Message == banishMessage`. The order assertion is the point: warp first, then message.

`TestBanish_MessageAbsent` — `Message: ""`. Assert `len(*events) == 1` and no `SEND_MESSAGE`.

`TestExecuteBanish_ConvergesOnSharedExecutor` — the skill-129 path. Setup: create the monster with `r.CreateMonster(...)` for template `9500324`, then `r.ControlMonster(ten, uniqueId, banishCharacter)` so `getDiseaseTargets` returns exactly that one character via the single-target branch (`m.ControlCharacterId()`). Fetch the model with `m, err := r.GetMonster(ten, uniqueId)`. Install `testInformationLookup` returning `SetBanish(information.Banish{MapId: banishMapId, PortalName: banishPortal, Message: banishMessage})`. Build the mob-skill model with:

```go
sd := mobskill.NewModelBuilder().
	SetSkillId(uint16(monster2.SkillTypeBanish)).
	SetLevel(1).
	Build()
```

No bounding box and no count are set, so `HasBoundingBox()` is false and `Count()` is 0 — `getDiseaseTargets` takes the single-target branch and returns exactly `[]uint32{m.ControlCharacterId()}`. Import `mobskill "atlas-monsters/monster/mobskill"` and `monster2 "github.com/Chronicle20/atlas/libs/atlas-constants/monster"` (the same aliases `processor_test.go:891` uses). Call `p.executeBanish(m, sd)`. Assert the same two events, in the same order, with the same bodies as `TestBanish_MessagePresent` — this is the convergence guarantee.

If `executeBanish` still calls `information.NewProcessor(...).GetById` directly rather than the `testInformationLookup` hook, Step 5 routes it through the hook; the test is written against the post-Step-5 behavior.

- [ ] **Step 2: Run the tests to verify they fail**

```bash
cd services/atlas-monsters/atlas.com/monsters && go test ./monster/... -run 'TestBanish|TestExecuteBanish' -v
```

Expected: compile failure — `p.Banish` undefined.

- [ ] **Step 3: Add the shared executor and `Banish`**

In `monster/processor.go`, add to the `Processor` interface under `// Commands`, after `ForceControl`:

```go
	Banish(f field.Model, characterId uint32, monsterTemplateId uint32) error
```

Add a small private information-lookup helper next to the new code (routing through the test hook; do NOT convert the three existing call sites at `:980`, `:1384`, `:1692` — that is unrelated churn):

```go
// monsterInformation resolves a monster template's information, honoring the
// test-only lookup hook so the banish paths are unit-testable without a live
// atlas-data.
func (p *ProcessorImpl) monsterInformation(monsterId uint32) (information.Model, error) {
	if testInformationLookup != nil {
		return testInformationLookup(monsterId)
	}
	return information.NewProcessor(p.l, p.ctx).GetById(monsterId)
}
```

Then the executor and the entry point:

```go
// banishCharacter emits the two commands a banish is made of: the map change,
// then the WZ banish message. Shared by the skill-129 path (executeBanish) and
// the client-initiated path (Banish) so portal and message handling can never
// diverge between them. Warp first: a message emitted before a failed warp
// would tell a player they were banished when they were not. A message emit
// failure after a successful warp is logged and swallowed — the banish already
// happened and there is nothing to roll back.
func (p *ProcessorImpl) banishCharacter(f field.Model, characterId uint32, b information.Banish) error {
	if err := p.emit(EnvCommandTopicPortal, warpCommandProvider(f, characterId, map2.Id(b.MapId), b.PortalName)); err != nil {
		return err
	}
	if b.Message != "" {
		if err := p.emit(system_message.EnvCommandTopic, sendMessageProvider(f, characterId, "PINK_TEXT", b.Message)); err != nil {
			p.l.WithError(err).Warnf("Banished character [%d] but unable to send banish message.", characterId)
		}
	}
	return nil
}

// Banish honors a client-initiated MOB_BANISH_PLAYER request. The template id
// arrives from the client and is untrusted: the banish executes only when a
// monster of that template is actually alive in the requesting character's
// field, which is the trust boundary for this path. Every failure returns an
// error naming the character, template and field, and takes no action; the
// caller logs once.
func (p *ProcessorImpl) Banish(f field.Model, characterId uint32, monsterTemplateId uint32) error {
	ms, err := p.GetInField(f)
	if err != nil {
		return fmt.Errorf("unable to read monsters in field [%s] for banish of character [%d] template [%d]: %w", f.Id(), characterId, monsterTemplateId, err)
	}
	alive := false
	for _, m := range ms {
		if m.MonsterId() == monsterTemplateId {
			alive = true
			break
		}
	}
	if !alive {
		return fmt.Errorf("no live monster of template [%d] in field [%s] for banish of character [%d]", monsterTemplateId, f.Id(), characterId)
	}
	info, err := p.monsterInformation(monsterTemplateId)
	if err != nil {
		return fmt.Errorf("unable to get information for template [%d] banishing character [%d] in field [%s]: %w", monsterTemplateId, characterId, f.Id(), err)
	}
	b := info.Banish()
	if b.MapId == 0 {
		return fmt.Errorf("template [%d] has no banish map; not banishing character [%d] in field [%s]", monsterTemplateId, characterId, f.Id())
	}
	p.l.Debugf("Banishing character [%d] to map [%d] portal [%s] via template [%d].", characterId, b.MapId, b.PortalName, monsterTemplateId)
	return p.banishCharacter(f, characterId, b)
}
```

Add `system_message "atlas-monsters/kafka/message/system_message"` and `"fmt"` to the import block if not already present.

- [ ] **Step 4: Run the `Banish` tests to verify they pass**

```bash
cd services/atlas-monsters/atlas.com/monsters && go test ./monster/... -run TestBanish -v
```

Expected: PASS.

- [ ] **Step 5: Rewrite `executeBanish` onto the shared executor**

Replace the body of `executeBanish` (`monster/processor.go:1247-1268`), keeping its existing target selection and its two guards, and routing the information fetch through the hook so the convergence test can stub it:

```go
// executeBanish warps target players to the monster's banish map. Shares
// banishCharacter with the client-initiated Banish path so the portal name and
// the WZ banish message are honored identically on both.
func (p *ProcessorImpl) executeBanish(m Model, sd mobskill.Model) {
	ma, err := p.monsterInformation(m.MonsterId())
	if err != nil {
		p.l.WithError(err).Errorf("Unable to get monster info for banish from monster [%d].", m.UniqueId())
		return
	}

	b := ma.Banish()
	if b.MapId == 0 {
		p.l.Debugf("Monster [%d] has no banish map configured.", m.UniqueId())
		return
	}

	targets := p.getDiseaseTargets(m, sd)
	for _, characterId := range targets {
		if err := p.banishCharacter(m.Field(), characterId, b); err != nil {
			p.l.WithError(err).Errorf("Unable to banish character [%d] from monster [%d] to map [%d].", characterId, m.UniqueId(), b.MapId)
		}
	}
}
```

- [ ] **Step 6: Run the convergence test to verify it passes**

```bash
cd services/atlas-monsters/atlas.com/monsters && go test ./monster/... -run 'TestBanish|TestExecuteBanish' -v
```

Expected: PASS.

- [ ] **Step 7: Add the consumer command type, body and handler**

In `kafka/consumer/monster/kafka.go`, add to the const block after `CommandTypeForceControl`:

```go
	CommandTypeBanish            = "BANISH"
```

and after `forceControlCommandBody`:

```go
// banishCommandBody asks the processor to banish a character out of a field on
// the strength of a client MOB_BANISH_PLAYER request. MonsterTemplateId is
// client-supplied and untrusted — Banish revalidates it against live field
// state before acting. Both fields are uint32: characterId already appears at
// that name and type in sibling bodies and monsterTemplateId appears in none,
// so neither can collide on this shared, fan-to-every-handler topic (see
// killCommandBody's note). The envelope's monsterId is deliberately left 0 — it
// means *unique* id everywhere else here. Mirrors atlas-channel's
// monster2.BanishCommandBody — edit both together.
type banishCommandBody struct {
	CharacterId       uint32 `json:"characterId"`
	MonsterTemplateId uint32 `json:"monsterTemplateId"`
}
```

In `kafka/consumer/monster/consumer.go`, register the handler in `InitHandlers` immediately after the `handleForceControlCommand` registration:

```go
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleBanishCommand))); err != nil {
			return err
		}
```

and add the handler next to `handleForceControlCommand`:

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

- [ ] **Step 8: Run the module build and full test suite**

```bash
cd services/atlas-monsters/atlas.com/monsters && go build ./... && go test ./...
```

Expected: all packages ok.

- [ ] **Step 9: Update `services/atlas-monsters/docs/kafka.md`**

In the `#### WARP` block, add `"targetPortalName": "st00"` to the JSON body and change the section blurb from "produced when monster banish skills target players" to "produced when a monster banish (skill 129 or a client MOB_BANISH_PLAYER request) ejects a player". Then add, after the `WARP` block:

````
### COMMAND_TOPIC_SYSTEM_MESSAGE

System message commands produced when a banished character has a WZ banish message.

**Message Type:**

#### SEND_MESSAGE

Announces text to a character's session.

```json
{
  "transactionId": "uuid",
  "worldId": 0,
  "channelId": 0,
  "characterId": 0,
  "type": "SEND_MESSAGE",
  "body": {
    "messageType": "PINK_TEXT",
    "message": ""
  }
}
```
````

Also record the consumed `BANISH` command in the `### COMMAND_TOPIC_MONSTER` section of the same file. Append it after the `#### CATCH` block (line 347-362), copying that block's shape exactly:

````
#### BANISH

Asks the processor to banish a character out of a field. Emitted by atlas-channel on a client MOB_BANISH_PLAYER request. monsterId is 0 — the client supplies a *template* id, carried in the body, and it is untrusted: the processor honors the command only when a monster of that template is alive in the character's field and the template has a non-zero WZ banish map.

```json
{
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "instance": "uuid",
  "monsterId": 0,
  "type": "BANISH",
  "body": {
    "characterId": 0,
    "monsterTemplateId": 0
  }
}
```
````

- [ ] **Step 10: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/kafka.go \
        services/atlas-monsters/atlas.com/monsters/kafka/consumer/monster/consumer.go \
        services/atlas-monsters/atlas.com/monsters/monster/processor.go \
        services/atlas-monsters/atlas.com/monsters/monster/banish_test.go \
        services/atlas-monsters/docs/kafka.md
git commit -m "feat(atlas-monsters): validate and execute client-initiated banish"
```

---

### Task 5: Verification gate

### Files

- No source files. Read-only across the three touched services.

- [ ] **Step 1: Run the flagless verification gate**

```bash
tools/verify.sh
```

Expected: exit 0. Only the flagless invocation counts — `--quick` / `--no-docker` skip the bake and `-race`.

- [ ] **Step 2: Fix any failure and re-run**

If a guard fails, read `docs/verification.md` for that guard's invariant, fix the cause (not the guard), and re-run the flagless script until it exits 0. Do not claim the branch done from a flagged or partial run.

- [ ] **Step 3: Commit any fixes**

```bash
git add -A
git commit -m "fix(task-257): address verification gate findings"
```

(Skip if the gate passed with no changes.)

---

## Acceptance traceability

| PRD acceptance criterion | Task |
|---|---|
| Handler emits `BANISH` with field, character id, template id; no stub | 2 |
| `atlas-monsters` warps only when a live monster of that template is in the field and the map is non-zero | 4 |
| Rejection with no live instance logs character id, template id, field; no warp | 4 |
| Rejection on missing/zero banish map; no warp | 4 |
| `WARP` carries `targetPortalName`; `atlas-portals` lands on it | 3 (producer), 1 (consumer) |
| Unresolvable portal name warns and falls back to random spawn; warp not dropped | 1 |
| Non-empty `banMsg` delivered as `PINK_TEXT`; empty sends nothing | 4 |
| `executeBanish` routes through the shared executor, gaining portal + message | 4 |
| Unit tests for all four aborts, both portal branches, both message branches, portals precedence + fallback | 1, 3, 4 |
| No existing `WARP` producer's body changes when `targetPortalName` is unset | 3 |
| `tools/verify.sh` exits 0 (flagless) | 5 |
