# Monster Catch Items — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-10
---

## 1. Overview

MapleStory ships a family of "catch" (bridle) consumables — 13 items in the
`02270000`–`02270012` range — that convert a weakened monster into an item
instead of killing it. The player targets a specific monster with a specific
item (Fishing Net → Squid, ghost jars → their ghosts, and so on); if the
monster is the right species, is weak enough, and the roll succeeds, the
monster disappears from the field and the player receives a different item in
its place. This is the mechanism behind several quest chains, the Fishing Net,
and the taming-mob acquisition path.

Atlas already has most of the plumbing for this feature and none of the
behaviour. The three clientbound packets are implemented and registered
(`services/atlas-channel/atlas.com/channel/socket/writer/catch_monster.go`,
`catch_monster_with_item.go`, `bridle_mob_catch_fail.go`, wired in
`main.go:643-644,811`). Every WZ field the feature needs already parses in
`services/atlas-data/atlas.com/data/consumable/reader.go` and survives into
`services/atlas-consumables/atlas.com/consumables/data/consumable/rest.go`.
What is missing is the serverbound trigger: `USE_CATCH_ITEM`
(`CWvsContext::SendBridleItemUseRequest`) has no codec in `libs/atlas-packet`,
no handler in `atlas-channel`, and no route in any of the nine seed templates.
Grep for `CatchMonster` outside `socket/writer/` finds only the writer
registrations in `main.go`. The result is that using a catch item does
nothing — the client sends a packet that is dropped on the floor.

This task implements the trigger end to end and corrects the coverage matrix,
which currently records the packet as not-applicable on four versions where it
demonstrably exists.

## 2. Goals

Primary goals:

- Implement the `USE_CATCH_ITEM` serverbound codec in `libs/atlas-packet` and
  promote its coverage-matrix row from ❌/⬜ to ✅ on all ten matrix versions.
- Route the handler in every applicable tenant seed template.
- Implement catch resolution: species match, HP-threshold gate, probability
  roll, monster removal without kill rewards, and reward-item grant.
- Send the correct client response on success (`CATCH_MONSTER` /
  `CATCH_MONSTER_WITH_ITEM`) and on failure (`BRIDLE_MOB_CATCH_FAIL`), and
  release the client's ExclRequest lock in every terminal path.
- Correct the matrix cells this work proves wrong (see FR-1.2, FR-7).

Non-goals:

- New WZ parsing. Every needed field (`mob`, `create`, `mobHP`,
  `bridleMsgType`, `bridleProp`, `bridlePropChg`, `useDelay`, `delayMsg`,
  `left`/`right`/`top`/`bottom`) already parses and is already exposed over
  REST.
- Pet or mount taming flows beyond what a catch item's `create` item grants.
- Any UI work in `atlas-ui`.
- The GMS v12 client. Its IDB exists but is not a coverage-matrix column, and
  its worker was unreachable during specification, so it is out of scope and
  explicitly unverified.
- Changing the clientbound catch codecs' wire format on any version where the
  matrix already records ✅.

## 3. User Stories

- As a player, I want to use a Fishing Net on a weakened Squid so that I
  receive a Fishing Net (Fish) item and the Squid leaves the field.
- As a player, I want to use a ghost jar on the wrong monster and be told it
  did not work, rather than silently losing the item.
- As a player, I want a failed catch to leave my item in my inventory, so a
  bad roll costs me only time.
- As a player, I want the client to unlock after a catch attempt so I can act
  again immediately, rather than being wedged until the next ExclRequest
  timeout.
- As an operator, I want catch items to behave identically across every client
  version the server supports, rather than only on the versions someone
  remembered to route.

## 4. Functional Requirements

### FR-1 — Serverbound codec (`libs/atlas-packet`)

**FR-1.1 — Body layout.** The request body is identical on every version
inspected. Read from `CWvsContext::SendBridleItemUseRequest`:

| Field | Encoding | Notes |
|---|---|---|
| `updateTime` | 4 bytes | client `get_update_time()` |
| `slot` | 2 bytes | inventory position (`nPOS`, unsigned) |
| `itemId` | 4 bytes | the catch item (`nItemID`, always `/10000 == 227`) |
| `monsterUniqueId` | 4 bytes | the hit monster's field object id |

Verified byte-for-byte on gms_v48 (`0x70e0c5`), gms_v61 (`0x832005`),
gms_v72 (`0x90457d`), gms_v79 (`0x9558e5`), and gms_v95 (`0x9e08c0`). No
version-gated field divergence was observed; the implementation MUST NOT
introduce a `MajorAtLeast` gate unless the remaining IDBs prove one.

**FR-1.2 — Opcodes.** The codec must be registered at these opcodes. The four
legacy values were read directly off the `COutPacket` constructor argument and
supersede the matrix's current ⬜ n-a:

| Version | Opcode | Evidence |
|---|---|---|
| gms_v48 | `0x03F` | `push 3Fh` @ `0x70e2cc` |
| gms_v61 | `0x04A` | `push 4Ah` @ `0x832526` |
| gms_v72 | `0x050` | `push 50h` @ `0x904aa9` |
| gms_v79 | `0x04F` | `push 4Fh` @ `0x955e11` |
| gms_v83 | `0x051` | registry `docs/packets/registry/gms_v83.yaml:2264` |
| gms_v84 | `0x051` | registry |
| gms_v87 | `0x054` | registry |
| gms_v92 | `0x058` | registry |
| gms_v95 | `0x057` | registry; confirmed `COutPacket(87)` @ `0x9e08c0` |
| jms_v185 | `0x049` | registry |

**FR-1.3 — Both directions.** The struct must implement `Encode` and `Decode`,
per `docs/packets/IMPLEMENTING_A_PACKET.md`.

**FR-1.4 — Registry correction.** `docs/packets/registry/gms_v48.yaml`,
`gms_v61.yaml`, `gms_v72.yaml`, and `gms_v79.yaml` must gain the
`USE_CATCH_ITEM` entry with `fname: CWvsContext::SendBridleItemUseRequest` and
the opcode from FR-1.2. The n-a consistency gate must pass afterward.

### FR-2 — Channel handler and template routing

**FR-2.1** — A new handler in
`services/atlas-channel/atlas.com/channel/socket/handler/` decodes the request
and forwards to `atlas-consumables`, following the shape of
`character_item_use.go`.

**FR-2.2** — The handler must be routed in all ten applicable tenant socket
templates under `services/atlas-configurations/seed-data/templates/`, at the
FR-1.2 opcode for each version, with a non-empty validator. A handler with a
missing validator is silently dropped at load.

**FR-2.3** — The opcode MUST come from tenant configuration, never a hard-coded
constant in Go (DOM-25).

**FR-2.4 — ExclRequest release.** The client sets ExclRequest before sending on
every version inspected (`CWvsContext::SetExclRequestSent` immediately after
the `COutPacket` ctor on v48/v61/v72/v79; the equivalent field store on v95).
Every terminal outcome — success, species mismatch, HP gate, failed roll,
monster already gone, inventory full — MUST release the client lock. A path
that neither warps nor sends an unlocking packet leaves the client wedged.

### FR-3 — Catch resolution (`atlas-consumables` → `atlas-monsters`)

**FR-3.1 — Ownership split.** `atlas-consumables` validates the item and emits
a catch command; `atlas-monsters` owns the monster-state checks and the roll,
fail-closed. This mirrors how `CommandTypeKill` re-checks alive+boss rather
than trusting the caller.

**FR-3.2 — Item validation (`atlas-consumables`).** Before emitting:

- the item id is in the `227xxxx` range;
- the character actually holds the item at the claimed slot;
- the consumable's `create` item id is non-zero;
- the character has room for the `create` item in its destination inventory.

**FR-3.3 — Monster validation (`atlas-monsters`).** On receiving the command:

- the monster exists in the field and is alive;
- the monster's template id equals the consumable's `mob`;
- the HP gate passes (see FR-3.4);
- the probability roll passes (see FR-3.5).

Any failure is a catch failure, not an error.

**FR-3.4 — HP gate.** `mobHP` gates the attempt and is interpreted as a
**percentage of the monster's maximum HP**. The attempt passes when
`hp <= maxHp * mobHP / 100`, evaluated so that integer truncation cannot make a
full-HP monster catchable at `mobHP` values below 100. An absent or zero
`mobHP` means no HP gate.

This is a decision, not a derivation: the client never performs the check, so
it cannot be read from the IDB. It rests on the observed value set across the
13 items — 30, 40, 100, or absent — where 100 is only meaningful as "any HP"
under a percentage reading and would be an implausible absolute threshold for
the monsters involved. Record it as an assumption in `design.md`; if live
testing shows catches succeeding or failing at the wrong HP, this is the first
thing to revisit.

**FR-3.5 — Probability.** `bridleProp` is the base success percentage and
`bridlePropChg` its multiplier; only `02270002` carries them
(`bridleProp 50`, `bridlePropChg 1.2`). Items without `bridleProp` are
deterministic once the species and HP gates pass. The design phase must settle
what `bridlePropChg` multiplies.

**FR-3.6 — Removal semantics.** A successful catch removes the monster via a
**new** `COMMAND_TOPIC_MONSTER` command type (e.g. `CATCH`), distinct from
`KILL`. The catch must not award experience, must not roll drops, and must not
emit monster-death events. Reusing `KILL` with a flag is explicitly rejected —
it would pollute the kill/death saga.

**FR-3.7 — New command body.** The command body must not reuse a field name
whose type collides with a sibling body on the shared topic. Every handler on
`COMMAND_TOPIC_MONSTER` unmarshals every message, so a field like `skillId`
typed `byte` in one body and `uint32` in another produces a spurious unmarshal
error per message. Keep the body minimal (character id, item id).

**FR-3.8 — Reward grant.** On success the character receives one unit of the
consumable's `create` item, and the catch item is consumed.

**FR-3.9 — No consumption on failure.** A failed catch — for any reason — does
NOT consume the catch item.

### FR-4 — Client responses

**FR-4.1 — Success.** Send `CATCH_MONSTER` (the on-mob capture effect, keyed by
the monster's unique id) and `CATCH_MONSTER_WITH_ITEM` (the item-grant
presentation, keyed by item id). The design phase must settle whether
`bridleMsgType` selects between them or whether both always fire.

**FR-4.2 — Failure.** Send `BRIDLE_MOB_CATCH_FAIL`. Its body is
`Decode1 reason` + `Decode4 itemId` + `Decode4` (read and discarded), per
`CWvsContext::OnBridleMobCatchFail` @ `0x9d9a80`.

**FR-4.3 — Reason codes are binary.** The client branches on exactly two
values:

- `reason == 0` → generic string 0x110E;
- `reason == 1` → the item's `delayMsg`, falling back to string 0x110F when the
  item has none;
- any other value → the client renders nothing at all.

Therefore the server MUST emit only 0 or 1. Enumerating richer reasons
(wrong mob / HP too high / roll failed) is impossible on the wire; the design
phase must map internal failure causes onto these two values.

### FR-5 — Client-side preconditions (context, not server requirements)

The client refuses to send at all when: the item is not in the `227xxxx` range;
ExclRequest is already outstanding; character HP is 0; fewer than 200ms have
elapsed since the last ExclRequest; the destination inventory has no free slot;
or `CMobPool::FindHitMobInRect` finds no monster of the target species inside
the item's `left/right/top/bottom` rectangle offset by the player position. In
the no-monster case the client prints a local message selected by
`bridleMsgType` (0–3) and sends nothing. The server must not assume these
checks happened — it revalidates independently — but they explain why some
failure modes are unreachable from the server's perspective.

### FR-6 — Multi-tenancy

All monster, consumable, and inventory access is tenant-scoped via
`tenant.MustFromContext(ctx)`. Item data is resolved per tenant version — the
`0227.img` contents differ across regions and versions, and the reward ids must
not be cached across tenants.

### FR-7 — Coverage-matrix corrections

Beyond promoting `USE_CATCH_ITEM`, this task must re-derive and fix the
clientbound cells its evidence contradicts:

- `CATCH_MONSTER` is ❌ on gms_v92 (`STATUS.md:320`) and
  `CATCH_MONSTER_WITH_ITEM` is ⬜ n-a on gms_v92 (`:322`), yet v92 carries the
  serverbound trigger at `0x058`. A version that can request a catch can
  receive its result.
- All three clientbound rows (`BRIDLE_MOB_CATCH_FAIL` `:127`, `CATCH_MONSTER`
  `:320`, `CATCH_MONSTER_WITH_ITEM` `:322`) are ⬜ n-a on gms_v48, yet v48
  carries the request at `0x03F`. The v48 IDB has no *named*
  `OnBridleMobCatchFail` or `OnCatchEffect`, but unnamed is not absent — the
  handlers must be located before any n-a is affirmed.
- `BRIDLE_MOB_CATCH_FAIL` is 🟡ᶠ on gms_v61/v72/v79/v87; those cells should be
  promoted to ✅ with byte fixtures while this family is open.

## 5. API Surface

No new public REST endpoints. Existing consumers:

- `GET /api/data/consumables/{itemId}` (atlas-data) — already returns `create`,
  `monsterId`, `monsterHP`, `bridleMsgType`, `bridleProp`, `bridlePropChg`.
- `GET` monster by field/unique id (atlas-monsters) — already returns `hp` and
  `maxHp` (`monster/rest.go:27-28`) for the FR-3.4 gate.

New Kafka contract on `COMMAND_TOPIC_MONSTER`:

```
type: "CATCH"
body: { characterId: uint32, itemId: uint32 }
```

Envelope fields (`worldId`, `channelId`, `mapId`, `instance`, `monsterId`) are
the existing shared `command[E]` header.

The result is reported back on the existing monster status/event topic so
`atlas-consumables` can grant the reward and `atlas-channel` can send the
response packets. The exact event shape is a design-phase decision.

## 6. Data Model

No new persisted entities and no migrations. All inputs are WZ-derived and
already modeled:

| WZ node | Model field | Location |
|---|---|---|
| `info/mob` | `MonsterId` | `data/consumable/reader.go:78` |
| `info/create` | `Create` | `reader.go:59` |
| `info/mobHP` | `MonsterHP` | `reader.go:81` |
| `info/bridleMsgType` | `BridleMsgType` | `reader.go:68` |
| `info/bridleProp` | `BridleProp` | `reader.go:69` |
| `info/bridlePropChg` | `BridlePropChg` | `reader.go:70` |
| `info/useDelay`, `info/delayMsg` | `UseDelay`, `DelayMsg` | `reader.go:72-73` |

The 13 catch items and their bindings, from
`Item.wz/Consume/0227.img.xml` (GMS 83.1):

| Item | `mob` | `create` | `mobHP` | `bridleMsgType` | `bridleProp` / `Chg` |
|---|---|---|---|---|---|
| 02270000 | 9300101 | 1902000 | — | — | — |
| 02270001 | 9500197 | 4031830 | 40 | 1 | — |
| 02270002 | 9300157 | 4031868 | 40 | 2 | 50 / 1.2 |
| 02270003 | 9500320 | 4031887 | 40 | — | — |
| 02270004 | 9300175 | 4001169 | 40 | — | — |
| 02270005 | 9300187 | 2109001 | 30 | — | — |
| 02270006 | 9300189 | 2109002 | 30 | — | — |
| 02270007 | 9300191 | 2109003 | 30 | — | — |
| 02270008 | 9500336 | 2022323 | — | 4 | — |
| 02270009 | 9300302 | 2109000 | 100 | — | — |
| 02270010 | 9300302 | 2109001 | 100 | — | — |
| 02270011 | 9300302 | 2109002 | 100 | — | — |
| 02270012 | 9300302 | 2109003 | 100 | — | — |

Note `02270002` carries `useDelay 800` and `02270008` carries `useDelay 3000`
with `delayMsg "You cannot use the Fishing Net yet."` — the delay message is
what `BRIDLE_MOB_CATCH_FAIL` reason 1 renders.

These values are the GMS 83.1 tenant dump and are illustrative of the binding
shape; the implementation must read them per-tenant at runtime, never hard-code
them.

## 7. Service Impact

| Service | Change |
|---|---|
| `libs/atlas-packet` | New `USE_CATCH_ITEM` serverbound struct with Encode+Decode; byte fixtures per version |
| `atlas-channel` | New handler; forwards to atlas-consumables; sends the three response packets; releases ExclRequest |
| `atlas-consumables` | Item validation, catch command emission, reward grant, item consumption on success only |
| `atlas-monsters` | New `CATCH` command type; species/HP/roll checks; reward-free monster removal |
| `atlas-configurations` | `USE_CATCH_ITEM` handler entry in ten seed templates, each with a validator, inserted at its sorted `opCode` position |
| `docs/packets` | Registry entries for v48/v61/v72/v79; matrix promotions; FR-7 clientbound corrections |
| `atlas-data` | None expected — all fields already parse |

## 8. Non-Functional Requirements

- **Idempotency.** A redelivered catch command must not remove a second monster
  or grant a second reward. Kafka is at-least-once.
- **Race safety.** Two players catching the same monster concurrently must
  produce exactly one success. The removal and the success determination happen
  in `atlas-monsters` (FR-3.1) so this is decidable in one place.
- **Observability.** Log species mismatch, HP-gate failure, and roll failure
  distinctly at debug, since the wire reason (FR-4.3) collapses them to one
  byte and cannot be used for diagnosis.
- **Multi-tenancy.** Per FR-6.
- **No wire regressions.** Adding the serverbound codec must not alter the byte
  layout of any clientbound catch packet on a version already recorded ✅.

## 9. Open Questions

1. ~~**`mobHP` semantics**~~ — resolved: percentage of max HP (FR-3.4). Held as
   a stated assumption, not a derivation; first suspect if live catch behaviour
   is wrong at the HP boundary.
2. **`bridlePropChg` semantics** (FR-3.5) — what does the 1.2 multiply? Only
   one item uses it, so the cost of getting it wrong is contained.
3. **Response packet selection** (FR-4.1) — does `bridleMsgType` choose between
   `CATCH_MONSTER` and `CATCH_MONSTER_WITH_ITEM`, or do both always fire?
4. **Internal cause → wire reason mapping** (FR-4.3) — which failures report 0
   and which report 1? Reason 1 renders the item's `delayMsg`, which suggests
   it is the "not yet / try again" case and 0 is the flat refusal.
5. **v48 clientbound handlers** (FR-7) — the v48 IDB has no named
   `OnBridleMobCatchFail`/`OnCatchEffect`. They must be located (or their
   absence proven) before the ⬜ n-a is either kept or replaced.
6. **gms_v12** — unverified. Its IDB worker was unreachable during
   specification and it is not a matrix column.
7. **`useDelay` enforcement** — is the 800ms/3000ms delay server-enforced, or
   purely a client-side gate? The client's own 200ms ExclRequest floor is
   separate from `useDelay`.

## 10. Acceptance Criteria

- [ ] `USE_CATCH_ITEM` struct exists in `libs/atlas-packet` with `Encode` and
      `Decode`, matching the FR-1.1 layout.
- [ ] Byte-fixture tests carry `packet-audit:verify` markers for all ten
      versions.
- [ ] `docs/packets/registry/gms_v{48,61,72,79}.yaml` contain `USE_CATCH_ITEM`
      at the FR-1.2 opcodes with the correct `fname`.
- [ ] `STATUS.md` shows `USE_CATCH_ITEM` ✅ on all ten columns; no ⬜ remains.
- [ ] The FR-7 clientbound cells are re-derived: v92 `CATCH_MONSTER` and
      `CATCH_MONSTER_WITH_ITEM` resolved, v48 clientbound n-a either proven or
      replaced, and the 🟡ᶠ `BRIDLE_MOB_CATCH_FAIL` cells promoted.
- [ ] The handler is routed in all ten seed templates, each with a non-empty
      validator, at its sorted `opCode` position.
- [ ] `tools/template-opcode-order-guard.sh` and
      `tools/template-duplicate-binding-guard.sh` pass.
- [ ] Using a correct catch item on a qualifying monster removes the monster,
      grants the `create` item, consumes the catch item, and plays the capture
      effect.
- [ ] Using a catch item on the wrong species sends `BRIDLE_MOB_CATCH_FAIL`,
      leaves the item in inventory, and unlocks the client.
- [ ] A failed probability roll leaves the item in inventory.
- [ ] A caught monster yields no experience, no drops, and no death events.
- [ ] A redelivered `CATCH` command grants no second reward.
- [ ] Every terminal path releases ExclRequest — verified by acting again
      immediately after each outcome.
- [ ] `go test -race ./...`, `go vet ./...`, and `go build ./...` clean in
      every changed module.
- [ ] `docker buildx bake atlas-channel atlas-consumables atlas-monsters`
      succeeds.
- [ ] `tools/lint.sh --check`, `tools/redis-key-guard.sh`, and
      `tools/goroutine-guard.sh` clean.
