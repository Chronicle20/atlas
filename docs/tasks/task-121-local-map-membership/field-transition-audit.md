# task-121 — Session-Field Write-Path Audit (FR-1.3)

Every code path that changes a session's world/channel/map/instance, and where
it updates the session registry. Grep basis: all call sites of the registry
field mutators (`setWorldId`, `setChannelId`, `setMapId`, `setInstance`,
`SetField`) in `services/atlas-channel/atlas.com/channel`.

Verification commands (run from `services/atlas-channel/atlas.com/channel`):

```bash
grep -n "setWorldId\|setChannelId" session/processor.go
grep -rn "SetField(" --include='*.go' kafka/ session/ | grep -v _test
grep -rn "SetMapId" --include='*.go' . | grep -v setMapId
```

| # | Transition | Site | Registry update | Verdict |
|---|-----------|------|-----------------|---------|
| 1 | Socket connect | `session/processor.go:311-312` `Create` — `setWorldId` + `setChannelId` before `Add` (registry insert at `session/processor.go:313`) | world/channel fixed at creation for the session's lifetime | ✅ |
| 2 | Login spawn-in | `kafka/consumer/session/consumer.go:190` — `SetField(s.SessionId(), f)` with `f` from `location.GetField` (includes instance) | full field set before `SessionCreated`/SetField packet/`SpawnForSelf` | ✅ fixed in Tasks 1-5 (was `SetMapId`, dropping the instance) |
| 3 | Every map/instance change (portal warp, GM warp, revive/forced return, transport arrival, instance enter/exit) | `kafka/consumer/character/consumer.go:249` — `SetField(sessionId, targetField)` from the `MAP_CHANGED` status event, before the warp packet and `SpawnForSelf` | full field set before dependent broadcasts | ✅ |
| — | Channel change | `kafka/consumer/session/consumer.go:126-138` (`handleChannelChange` / `processChannelChangeReturn`) — the handler only announces `ChannelChangeWriter`; no field mutator is called here. The client physically disconnects and opens a new socket to the target channel pod, which runs paths 1+2 fresh | no in-place field mutation exists | ✅ by construction |

Additional grep result requiring explanation: `session/model.go:154` matches
`grep -rn "SetMapId" . | grep -v setMapId` (the filter is case-sensitive and
only excludes literal `setMapId`). This is not a stray caller bypassing the
registry — it is the `field.Model` builder call (`ns.Field().Clone().SetMapId(id).Build()`)
*inside* the private `session.Model.setMapId` method's own implementation
(`session/model.go:152-156`), i.e. the one place `setMapId` is defined, not a
second, independent write path. No other `SetMapId` call sites exist outside
this definition.

All transition kinds enumerated in PRD FR-1.1 are server-driven through the
single `MAP_CHANGED` status event (path 3); atlas-channel has no other
field-writing entry point.

## Caller inventory (FR-3.1)

Command run from `services/atlas-channel/atlas.com/channel`:

```bash
grep -rln "ForSessionsInMap\|ForOtherSessionsInMap\|CharacterIdsInMap\|GetCharacterIdsInMap\|ForSessionsInSessionsMap" --include='*.go' . | grep -v _test | sort
```

32 non-test files match, all confirmed to use the result only to address
local sessions or reason about this pod's map population (design §5
categories 1-3). This includes `map/processor.go` itself, which *defines*
the recipient providers rather than calling another definer — excluding it
leaves **31 actual caller files**.

```
services/atlas-channel/atlas.com/channel/kafka/consumer/asset/consumer.go
services/atlas-channel/atlas.com/channel/kafka/consumer/buff/consumer.go
services/atlas-channel/atlas.com/channel/kafka/consumer/chair/consumer.go
services/atlas-channel/atlas.com/channel/kafka/consumer/chalkboard/consumer.go
services/atlas-channel/atlas.com/channel/kafka/consumer/character/consumer.go
services/atlas-channel/atlas.com/channel/kafka/consumer/consumable/consumer.go
services/atlas-channel/atlas.com/channel/kafka/consumer/door/consumer.go
services/atlas-channel/atlas.com/channel/kafka/consumer/drop/consumer.go
services/atlas-channel/atlas.com/channel/kafka/consumer/expression/consumer.go
services/atlas-channel/atlas.com/channel/kafka/consumer/guild/consumer.go
services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go
services/atlas-channel/atlas.com/channel/kafka/consumer/merchant/consumer.go
services/atlas-channel/atlas.com/channel/kafka/consumer/message/consumer.go
services/atlas-channel/atlas.com/channel/kafka/consumer/mist/consumer.go
services/atlas-channel/atlas.com/channel/kafka/consumer/monsterbook/consumer.go
services/atlas-channel/atlas.com/channel/kafka/consumer/monster/consumer.go
services/atlas-channel/atlas.com/channel/kafka/consumer/mount/consumer.go
services/atlas-channel/atlas.com/channel/kafka/consumer/party_quest/consumer.go
services/atlas-channel/atlas.com/channel/kafka/consumer/pet/consumer.go
services/atlas-channel/atlas.com/channel/kafka/consumer/quest/consumer.go
services/atlas-channel/atlas.com/channel/kafka/consumer/reactor/consumer.go
services/atlas-channel/atlas.com/channel/kafka/consumer/route/consumer.go
services/atlas-channel/atlas.com/channel/kafka/consumer/summon/consumer.go
services/atlas-channel/atlas.com/channel/map/processor.go  (defines the providers; not itself a caller)
services/atlas-channel/atlas.com/channel/movement/processor.go
services/atlas-channel/atlas.com/channel/skill/handler/heal/heal.go
services/atlas-channel/atlas.com/channel/skill/handler/recipients.go
services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go
services/atlas-channel/atlas.com/channel/socket/handler/character_buff_cancel.go
services/atlas-channel/atlas.com/channel/socket/handler/character_damage.go
services/atlas-channel/atlas.com/channel/socket/handler/character_skill_prepare.go
services/atlas-channel/atlas.com/channel/socket/handler/character_skill_use.go
```

Category breakdown (design §5):

1. **Packet fan-out** — the `kafka/consumer/*` consumers (asset, buff, chair,
   chalkboard, character, consumable, door, drop, expression, guild,
   merchant, message, mist, monsterbook, monster, mount, party_quest, pet,
   quest, reactor, route, summon), `movement/processor.go`,
   `skill/handler/heal/heal.go`, `skill/handler/recipients.go`, and the
   `socket/handler/character_*` handlers. Result feeds `ForEachByCharacterId`
   → `session.Announce`; delivery is only possible to local sessions by
   definition.
2. **Membership enumeration for spawn logic** — `kafka/consumer/map/consumer.go`
   (`fetchOtherCharactersInMap`, `enterMap`): ids are used to fetch character
   models and then, again, to address local sessions.
3. **Side-effecting iteration** — `kafka/consumer/map/consumer.go` (weather /
   consumable-effect saga applied per character in map). Same sufficiency
   argument as category 2.

No caller requires atlas-maps' authoritative view; no REST escape hatch is
needed (the PRD expected none).

## Verification gate (CLAUDE.md)

Run from `services/atlas-channel/atlas.com/channel`:

```bash
go test -race ./... && go vet ./... && go build ./...
```

Run from the worktree root:

```bash
docker buildx bake atlas-channel
tools/redis-key-guard.sh
```

All four commands were run for this task and passed clean (see the Task 6
report for verbatim output).
