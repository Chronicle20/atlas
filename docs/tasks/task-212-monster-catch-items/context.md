# Monster Catch Items — Implementation Context

Companion to [`plan.md`](plan.md). Everything here was read out of the tree at
planning time with a `file:line` citation; nothing is recalled from memory.

---

## 1. Where the feature already exists

The three clientbound codecs and their writers are already present and
registered. What is missing is the trigger and all of the behaviour.

| Piece | Location | State |
|---|---|---|
| `CatchMonster` codec | `libs/atlas-packet/monster/clientbound/catch_monster.go` | present; `legacyMobPoolPrefix` gate is **wrong** for v83+ |
| `CatchMonsterWithItem` codec | `libs/atlas-packet/monster/clientbound/catch_monster_with_item.go` | present; **no `uniqueId` field at all** |
| `BridleMobCatchFail` codec | `libs/atlas-packet/character/clientbound/` | present |
| The three writers | `services/atlas-channel/atlas.com/channel/socket/writer/{catch_monster,catch_monster_with_item,bridle_mob_catch_fail}.go` | present, each documented as "no emitter wires this writer yet" |
| Writer registration | `services/atlas-channel/atlas.com/channel/main.go:643-644` (catch), `:811` (bridle fail) | present |
| `USE_CATCH_ITEM` codec / handler / route | — | **absent everywhere** |

The WZ side is complete: every bridle field parses in
`services/atlas-data/atlas.com/data/consumable/reader.go` and survives into
`services/atlas-consumables/atlas.com/consumables/data/consumable/rest.go:39-52`
(`BridleProp`, `BridlePropChg`, `UseDelay`, `DelayMsg`, `MonsterId`,
`MonsterHP`). Only the **getters** are missing from `Model` — the struct fields
exist (`data/consumable/model.go:59-84`) and `rest.go:132-145` populates them.

---

## 2. Template routing reality (read, not assumed)

`services/atlas-configurations/seed-data/templates/`, catch-family writers:

| Template | `BridleMobCatchFail` | `CatchMonster` | `CatchMonsterWithItem` |
|---|---|---|---|
| `template_gms_48_1.json` | — | — | — |
| `template_gms_61_1.json` | 0x4C | 0xBE | 0xBF |
| `template_gms_72_1.json` | 0x4C | 0xDF | 0xE0 |
| `template_gms_79_1.json` | 0x4C | 0xE5 | 0xE6 |
| `template_gms_83_1.json` | 0x4F | 0xFB | 0xFC |
| `template_gms_84_1.json` | 0x51 | 0x101 | 0x102 |
| `template_gms_87_1.json` | 0x51 | 0x10B | 0x10C |
| `template_gms_92_1.json` | — | — | — |
| `template_gms_95_1.json` | 0x52 | 0x12B | 0x12C |
| `template_jms_185_1.json` | 0x49 | 0x10C | 0x10D |

`design.md` §8 says only "v48, and v92 for `CatchMonsterWithItem`" need new
writer routes. **v92 is missing all three** — it is a much thinner template
(81.5K vs 120–195K for its siblings). Plan Task 6 adds all three on v92 and two
on v48 (never `BridleMobCatchFail` on v48 — design §2 F-3 proves the handler
does not exist).

No template routes any `USE_CATCH_ITEM` handler today.

Handler entry shape (`template_gms_83_1.json`, `socket.handlers`):

```json
{"opCode": "0x06", "validator": "LoggedInValidator", "handler": "ServerStatusHandle", "fname": "CLogin::SendCheckUserLimitPacket", "services": ["login"]}
```

Writer entry shape (`socket.writers`):

```json
{"opCode": "0xFB", "writer": "CatchMonster", "fname": "CMob::OnCatchEffect", "services": ["channel"]}
```

---

## 3. Key patterns to copy

**Reward-box flow** — the model for `RequestCatchMonster`.
`services/atlas-consumables/atlas.com/consumables/consumable/processor.go:1079`
(`RequestItemReward`) → pre-reserve validation → `ip.CanAccommodate` →
`consumer.GetManager().RegisterHandler(t, message.AdaptHandler(message.OneTimeConfig(validator, handler)))`
→ `cpp.RequestReserve(...)`. The commit/cancel pair is
`grantRewardOnConfirmed` (`:1216`) / `grantRewardOnFailed` (`:1241`), and
`grantRewardOnFailed`'s comment explains why it emits **no** consumable error:
the channel already renders the failure and two unlock packets would be sent.
The catch flow inherits that reasoning verbatim.

**Fail-closed processor command** — the model for `Processor.Catch`.
`services/atlas-monsters/atlas.com/monsters/monster/processor.go:1711` (`Kill`):
registry lookup → `!m.Alive()` guard → data lookup whose *error* drops the
command rather than reporting failure. `testInformationLookup` (`:74`) is the
test seam; `catch.go` adds `testConsumableLookup` and `testCatchRoll` in the
same shape.

**Command-body collision hazard** — `killCommandBody`'s own comment
(`kafka/consumer/monster/kafka.go`) documents FR-3.7 in the codebase's words:
every handler on `COMMAND_TOPIC_MONSTER` unmarshals every message, so a field
name whose type disagrees with a sibling logs one spurious error per message.
`catchCommandBody` reuses `characterId`/`itemId`, both already `uint32` in
sibling bodies.

**Handler shape** — `socket/handler/character_item_use.go`: decode, debug-log,
forward. Registered in `main.go` as
`handlerMap[monstersb.MobDropPickupRequestHandle] = handler.MobDropPickupRequestHandleFunc`
(`main.go:843`).

**The unlock** — `session.Announce(l)(ctx)(wp)(statpkt.StatChangedWriter)(statpkt.NewStatChanged(make([]statpkt.Update, 0), true).Encode)`.
Used in `kafka/consumer/consumable/consumer.go:112,129,137` and
`kafka/consumer/compartment/consumer.go:79,136`. `socket/handler/enable_actions.go`
wraps it but is unexported, so consumers inline it — follow the consumers.

**Byte-fixture tests** — `libs/atlas-packet/monster/clientbound/catch_monster_test.go`
is the reference: `// packet-audit:verify packet=… version=… ida=0x…` markers,
per-version golden slices with a decompile-line comment per byte, then a
`pt.Variants` round-trip loop. `pt.Variants` (`libs/atlas-packet/test/context.go:18-41`)
carries all twelve variants including v48 [7], v61 [8], v72 [9], v79 [10], v92 [11].

**Registry singletons** — atlas-monsters uses `atlasredis.NewRegistry` (not
`TenantRegistry`) with the tenant baked into the key suffix
(`monster/registry.go:284,297`). atlas-consumables' `map/character/registry.go`
is the `TenantRegistry` example the new `catchdelay` registry follows.

**Test harness** — `monster/registry_test.go:26-45` boots miniredis in `TestMain`
and calls `InitMonsterRegistry(rc)`; `processor_test.go:236` supplies
`newRecordingProcessorWithBodies`, which intercepts `emit` and returns a
`*[]emittedBody` of `{Topic, Type, Body}`.

---

## 4. Contracts introduced by this task

**`COMMAND_TOPIC_MONSTER`** (new type)
```
type: "CATCH"
key:  monster uniqueId
body: { characterId: uint32, itemId: uint32 }
envelope: the existing command[E] header (worldId, channelId, mapId, instance, monsterId=uniqueId)
```

**`EVENT_TOPIC_MONSTER_CATCH`** (new topic, produced by atlas-monsters, consumed by atlas-consumables)
```
type: "CATCH_RESOLVED"
key:  characterId
body: { characterId: uint32, itemId: uint32, success: bool, cause: string }
```

**`EVENT_TOPIC_MONSTER_STATUS`** (new types on the existing topic, keyed by MapId)
```
type: "CAUGHT"        body: { characterId: uint32, itemId: uint32 }
type: "CATCH_FAILED"  body: { characterId: uint32, itemId: uint32, cause: string }
```

**`COMMAND_TOPIC_CONSUMABLE`** (new type)
```
type: "REQUEST_CATCH_MONSTER"
body: { source: slot.Position, itemId: item.Id, monsterUniqueId: uint32 }
```

**`EVENT_TOPIC_CONSUMABLE_STATUS`** (new type)
```
type: "CATCH_FAILED"  body: { itemId: uint32, cause: string }
```

**Cause vocabulary** — semantic in the domain services, mapped to the client's
wire byte only in atlas-channel (DOM-25):

| Cause | Emitted by | Wire reason | Client shows |
|---|---|---|---|
| `USE_DELAY` | atlas-consumables | `1` | the item's `delayMsg`, else string 0x110F |
| `INVENTORY_FULL` | atlas-consumables | `0` | string 0x110E |
| `INVALID_ITEM` | atlas-consumables | `0` | string 0x110E |
| `SPECIES_MISMATCH` | atlas-monsters | `0` | string 0x110E |
| `HP_TOO_HIGH` | atlas-monsters | `0` | string 0x110E |
| `ROLL_FAILED` | atlas-monsters | `0` | string 0x110E |
| `UNRESOLVED` | atlas-monsters | *(no packet)* | — (unlock only) |

The client branches on exactly two values (`CWvsContext::OnBridleMobCatchFail`
@0x9d9a80); anything else renders nothing at all, so the server emits only 0 or 1.

---

## 5. Dependency order

```
1  UseCatchItem codec
2  registry + packet-audit fname   ← 1
3  template handler routes         ← 1
4  OnMobPacket prefix fix
5  CatchMonsterWithItem uniqueId   ← 4 (shares the package)
6  v48/v92 writer routes           ← 4, 5
7  matrix promotion                ← 1,2,3,4,5,6
8  RemoveExisting + ClaimMonster
9  monsters consumable client
10 CATCH command + ladder          ← 8, 9
11a data getters + classification
11b request path                   ← 10, 11a   ┐ one compile unit
11c resolution                     ← 10, 11a   ┘
12a channel handler                ← 1, 11b
12b channel renderers              ← 5, 10, 11b
13 topics, docs, sweep             ← all
```

Tasks 1–7 (codec/matrix) and 8–12 (behaviour) are independent of each other and
can proceed in parallel, except that Task 5 changes
`NewCatchMonsterWithItem`'s signature, which Task 12b calls.

---

## 6. Decisions and their revisit triggers

Four assumptions carried from `design.md` §11. None can be derived from any
client, because the client never computes them.

| # | Assumption | Lives in | Revisit when |
|---|---|---|---|
| A-1 | `mobHP` is a **percentage** of max HP | `monster/catch.go:catchHpGatePasses` | catches succeed/fail at the wrong HP |
| A-2 | `bridlePropChg` is a **one-shot** multiplier on `bridleProp`, clamped to 100 | `monster/catch.go:effectiveCatchChance` | item `02270002`'s catch rate feels wrong |
| A-3 | `result = 1` is the "captured" animation | `kafka/consumer/monster/consumer.go:handleStatusEventCaught` | the wrong animation plays |
| A-4 | `useDelay` is **server-enforced** | `catchdelay/registry.go` | spam-catching is possible, or the delay message never appears |

A-1 is written as a cross-multiplication (`hp*100 <= maxHp*mobHP`) specifically
so integer truncation cannot let a full-HP monster through at `mobHP` below 100
— five of the thirteen catch items use `mobHP: 100`, which is meaningful only as
"any HP" under the percentage reading.

A-4 is load-bearing: without it, wire reason 1 is unreachable and `delayMsg` is
dead data.

---

## 7. Deliberate design choices worth not re-litigating

- **The outcome is published twice.** The channel needs the capture effect to
  arrive before `DESTROYED` (the client's `CMobPool::OnMobPacket` resolves the
  mob via `GetMob` and silently drops the packet once it is gone); the status
  topic is keyed by `MapId` (`monster/producer.go:36`), so emitting `CAUGHT`
  immediately before `DESTROYED` makes that a partition guarantee. Consumables
  must **not** join that topic — it carries a `DAMAGED` event per hit and every
  handler unmarshals every message. Hence a dedicated low-volume topic.
- **`CATCH` is a new command type, never `KILL` with a flag.** Reusing `KILL`
  would pollute the kill/death saga; a catch awards no experience, rolls no
  drops, and emits no death events.
- **atlas-monsters does not grant the reward.** That would put inventory
  semantics inside the monster service and duplicate the reservation machinery.
- **No cache on the new monsters-side consumable client.** `0227.img` differs by
  region and version and the `create` reward ids differ with it; catch attempts
  are rare. Design §7 chooses no cache over a tenant-keyed one.
- **The v48 fail packet is simply not routed.** The channel attempts the
  announce, the writer registry reports it unconfigured, and the unlock is
  emitted from its own statement — so a missing route cannot wedge the client.

---

## 8. Traps this task walks past

- **A handler with an empty `validator` is silently dropped at load.** All ten
  `USE_CATCH_ITEM` entries carry `LoggedInValidator`.
- **`RemoveMonster` is `Get`-then-`Del` with the reply discarded**
  (`monster/registry.go:547`), so two concurrent catches can both "succeed".
  `ClaimMonster` exists because of this; `RemoveMonster` keeps its signature and
  all its existing callers.
- **`go build` cannot catch a missing `COPY libs/...`** in the shared root
  `Dockerfile` — only `docker buildx bake` can. Three services changed, so the
  bake is mandatory (Task 13 Step 4).
- **An unsuffixed topic name in a kustomize overlay is a silent cross-env
  bleed.** Both overlays get the suffixed `EVENT_TOPIC_MONSTER_CATCH` form.
- **A serverbound matrix cell needs three artifacts** (marker, pinned evidence,
  audit report) **and** a template route. Evidence without a report is a
  `matrix --check` "dangling evidence" failure.
- **`candidatesFromFName` is what links a registry op to a codec.** A new
  serverbound op that is not added there produces no report and cannot promote.
- **`tools/lint.sh --check` false-fails without nvm on PATH** — that is an
  environment problem, not a pass.

---

## 9. Known open edge

`design.md` §4.2 accepts one residual risk unchanged: a partial publish where
the monster is removed but `CATCH_RESOLVED` never lands leaves a dangling
reservation. This is the same exposure the existing reward-box flow carries
(`ConsumeReward`'s create request can fail to publish and dangle its
once-handler), so the design matches the codebase's accepted level rather than
inventing a new compensation mechanism. Emission order is
`CATCH_RESOLVED` → `CAUGHT` → `DESTROYED`, so the player-visible economic
outcome is the first thing attempted after the claim.

---

## 10. Out of scope, explicitly

- **gms_v12.** `template_gms_12_1.json` is not a coverage-matrix column and is
  not routed. Its IDB worker was unreachable during specification.
- **Pet and mount taming flows** beyond what a catch item's `create` grants.
- **atlas-ui.** No frontend files change, so
  `superpowers:requesting-code-review` dispatches only the plan-adherence and
  backend reviewers.
- **New WZ parsing.** Every field the feature needs already parses and is
  already exposed over REST.
