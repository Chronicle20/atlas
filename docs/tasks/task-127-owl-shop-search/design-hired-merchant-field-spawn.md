# Design — Hired Merchant as Persistent Field NPC

Status: **DRAFT for review** — cross-service feature; do not implement §5 wiring until approved.

## 1. Goal

A hired merchant (permit class `503xxxx`, `ShopType = HiredMerchant`) must appear
in the map as a standalone NPC ("employee") that:

- is visible to **every** player in the map, not just the owner;
- **persists after the owner logs off** (it already does at the data layer — see §3);
- spawns to players who **enter the map** afterward;
- despawns when the shop closes (manual / sold-out / expired / empty);
- shows the shop's balloon (title + owner name) above the sprite.

Today none of this renders: an open shop's field box is drawn **attached to the
owner's character avatar**, so when the owner leaves there is nothing to hang it
on. This design adds the missing field entity and its lifecycle.

## 2. Wire protocol (IDA-derived, all 9 versions)

Three clientbound packets read by `CEmployeePool` — **byte-identical across every
version that has the feature** (v61/72/79/83/84/87/95/jms185); **v48 has no
hired-merchant feature, codec N/A**. No version gate on field order is required.

```
SPAWN   (OnEmployeeEnterField):
  u32 employeeId          // object id / map key
  u32 templateId          // employee sprite template
  i16 x, i16 y, i16 foothold
  str ownerName           // nametag
  <balloon block>
DESTROY (OnEmployeeLeaveField):
  u32 employeeId          // only field
UPDATE  (OnEmployeeMiniRoomBalloon):
  u32 employeeId
  <balloon block>

<balloon block> (CEmployee::SetBalloon):
  u8 miniRoomType         // 0 => no balloon, stop
  if miniRoomType != 0:
    u32 miniRoomSN        // the shop's mini-room serial
    str balloonTitle      // shop title
    u8 a, u8 b, u8 c      // curVisitors, maxVisitors, gameKind/spec (semantics TBD, non-crashing)
```

The three `a/b/c` bytes: exact display semantics are unconfirmed (likely
current-visitor count / capacity / open-flag). They do not gate any read, so any
plausible value is non-crashing; we populate from visitor count + capacity and
flag the unconfirmed field.

New codecs live in `libs/atlas-packet/interaction/clientbound/` (or a new
`employee/` package — decide during impl). Encode + Decode + byte fixtures with
`packet-audit:verify` markers per version; matrix cells promoted for the 8
feature-bearing versions.

## 3. Current state (grounded)

- **atlas-merchant owns the shop.** Row in `shops` (`shop/entity.go:12-34`) carries
  `CharacterId, ShopType, State, WorldId, ChannelId, MapId, InstanceId, X, Y,
  PermitItemId, ExpiresAt, MesoBalance`. States `Draft/Open/Maintenance/Closed`
  (`shop/state.go`).
- **Hired merchants already persist across logoff.** `character/consumer.go:36-59`
  `handleLogout` closes a shop on disconnect **only when `ShopType == CharacterShop`** —
  `HiredMerchant` is deliberately excluded. Its end-of-life is the 24h
  `ExpirationTask` (`shop/task.go:27-58`) or explicit close.
- **Per-map shop index already exists.** Redis `mapPlacement` (mapId → shopIds) in
  `shop/registry.go`, maintained by `addToMapIndex/removeFromMapIndex`
  (`processor.go:971-985`). REST `getByField` (`shop/provider.go:54`, `state != Closed`)
  is the authoritative per-field shop set, exposed via `handleGetFieldMerchants`.
- **Spawn-to-enterer fan-out is built.** atlas-maps emits `CHARACTER_ENTER` on
  `EVENT_TOPIC_MAP_STATUS`; atlas-channel `SpawnForSelf`
  (`consumer/map/consumer.go:160-339`) spawns everything already in the field to
  the enterer, including merchants at `:270-274 → spawnMerchantsForSession:667-689`.
- **Live broadcast pattern = monsters.** `consumer/monster/consumer.go:121`
  `handleStatusEventCreated` → `_map.ForSessionsInMap(field, spawn...)` announces a
  new entity to everyone already in the map; `DESTROYED` mirrors it. This is the
  template to copy.
- **Merchant box today** = `MiniRoom.Spawn` writes `WriteInt(ownerCharacterId)`
  first (`libs/atlas-packet/interaction/mini_room.go:69`) → attached to the owner
  avatar. `handleShopOpenedEvent` (`consumer/merchant/consumer.go:116-157`)
  broadcasts it map-wide; `handleShopClosedEvent:184-211` despawns.

## 4. The gap (net-new)

1. **No field entity / employee pool anywhere** (confirmed net-new by exhaustive
   grep). Need an entity type + a per-map registry of employee presences.
2. **Rendering anchor.** The box is anchored to `ownerCharacterId`; a persistent
   employee needs its own `employeeId` sprite the box hangs on when the owner is
   absent → the new SPAWN packet must precede the mini-room box.
3. **No "owner left, keep employee standing" signal.** Logoff merely *skips*
   closing the shop; atlas-channel is never told to swap the box from the (now
   despawning) owner avatar to a standalone employee sprite.

## 5. Proposed design

**Ownership: atlas-merchant projects the field employee; atlas-channel renders it
(monster pattern).** atlas-merchant already owns the authoritative shop state, the
per-map index, expiry, and the state machine — the employee is the *field
projection* of a `HiredMerchant` shop. atlas-channel never owns field entities, it
only renders them. So:

### 5a. New Kafka events — a merchant **field** status stream

Add employee lifecycle events. Preferred: **new event types on a dedicated
merchant-field topic** (distinct from the UI-oriented `EVENT_TOPIC_MERCHANT_STATUS`),
mirroring `EVENT_TOPIC_MONSTER_STATUS`:

- `EMPLOYEE_SPAWNED`  — {employeeId, templateId, worldId, channelId, mapId, instanceId, x, y, foothold, ownerName, shopId, miniRoomType, title, visitorCount, capacity}
- `EMPLOYEE_DESPAWNED` — {employeeId, worldId, channelId, mapId, instanceId}
- `EMPLOYEE_UPDATED`  — {employeeId, …balloon fields}

`employeeId` is allocated by atlas-merchant (new column on `shops`, or a derived
stable id). `templateId` = the hired-merchant NPC sprite (confirm the fixed
template id from data; flag if unknown).

### 5b. Triggers (atlas-merchant)

- **Materialize** on `Draft→Open` (`OpenShop`) for a `HiredMerchant`, and re-emit
  for existing Open hired-merchant shops on channel/instance bring-up (same way
  merchants are re-listed via `getByField`).
- **Keep standing on owner logoff:** `handleLogout` already keeps the shop Open;
  add a positive `EMPLOYEE_SPAWNED` (or a "detach from avatar" signal) so the box
  re-anchors to the employee sprite once the owner avatar despawns.
- **Despawn** on any `SHOP_CLOSED` (all reasons) — reuse the existing close emit.
- **Update** on visitor enter/exit and `LISTING_PURCHASED` (balloon count refresh).

### 5c. Rendering (atlas-channel)

- New consumer on the merchant-field topic → on `EMPLOYEE_SPAWNED`,
  `_map.ForSessionsInMap(field, spawnEmployee...)`; on `SpawnForSelf`, add an
  employee pass (`ForEachInField`) alongside `spawnMerchantsForSession`.
- New writer emits the **SPAWN** employee packet, then the mini-room box anchored
  to `employeeId` (not the owner character). `DESPAWN`/`UPDATE` writers for the
  other two.
- Reconcile with the existing owner-avatar box: while the owner is present, decide
  whether the box rides the avatar (today) or always the employee. **Open decision
  D1** below.

### 5d. Per-map employee registry

Reuse/extend `shop/registry.go` `mapPlacement` as the employee pool rather than a
parallel structure; `getByField` already returns exactly the set that should have
a field presence.

## 6. Cross-version scope

- Codecs + fixtures for **v61/72/79/83/84/87/95/jms185** (8 versions). **v48 is
  feature-absent** — no codec, no template routing, matrix N/A.
- Backfill/confirm the enter-result + merchant-op verification on the legacy quad
  where the feature exists (v61/72/79 display merchants but have **no management
  dialog** — enter-result of a *hired* merchant via `CEntrustedShopDlg` does not
  exist pre-v83; scope enter-result verification to v83+).

## 7. Open decisions (need answers before/while implementing)

- **D1 — box anchor while owner present:** always anchor the box to the employee
  sprite (uniform, simplest), or keep it on the owner avatar until they leave then
  swap? Uniform-employee is cleaner and matches "it's an NPC," but changes how a
  present owner sees their own shop.
- **D2 — templateId source:** the fixed hired-merchant employee sprite id — pull
  from data or is it a known constant? (Flag: unverified.)
- **D3 — employeeId allocation:** new `shops` column vs. derived id; must be stable
  across channel restarts for DESTROY/UPDATE to match.
- **D4 — balloon a/b/c bytes:** confirm semantics against a live capture or accept
  visitor-count/capacity/flag mapping as non-crashing best-effort.

## 8. Implementation phases

1. **Codecs** — 3 employee packets + fixtures + matrix promotion (8 versions).
   Independent, safe, lands first.
2. **atlas-merchant** — employeeId, field-status topic + events, triggers
   (open / logoff-keep / close / visitor-update), re-materialize on bring-up.
3. **atlas-channel** — consumer + writers, `SpawnForSelf` employee pass, box
   re-anchor to employeeId.
4. **Verify** — race/vet/build all touched modules, `docker buildx bake` for
   atlas-merchant + atlas-channel (go.mod untouched? templates/new files may add
   deps — bake regardless), matrix `--check`, guards; then code review.
