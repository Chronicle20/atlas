# Move Character Change-Map Write to atlas-maps + Retire the Location Shim — Design

Status: Approved
Created: 2026-06-12
Source PRD: `prd.md` (this folder)

---

## 1. Problem & Approach (recap)

The UI "Change Map" action has been a silent no-op since task-055: the UI `PATCH /api/characters/{id}` with `{ mapId }`, atlas-character's `Update` reads it, logs `Debug`, discards it, and returns `204`. No `MAP_CHANGED` event, no warp.

The location **owner** is atlas-maps (since task-055). It already serves `GET /characters/{id}/location` and the ingress already routes `/api/characters/{id}/location` to it. This task adds the **write** side to that resource, reusing the existing `CHANGE_MAP` warp logic factored into one shared processor method, then **retires the atlas-character location shim** — but only after every real consumer of the echoed `mapId` is migrated to read location from atlas-maps directly.

The design below is grounded in a full code audit (not the PRD's guesses). Two PRD assumptions were corrected by that audit; see §4.

---

## 2. Key decisions (resolved)

| # | Question | Decision |
|---|---|---|
| OQ-1 | Write response shape | **`204 No Content`** on success. Matches the GET-less ergonomics the UI already handles for `charactersService.update`. |
| OQ-2 | Character has no location row | **Reject `404`.** A character that has ever logged in has a row (logout writes it), so the offline-warp requirement (FR-1.5) is still met. Avoids fabricating a channel and an extra atlas-character call for a never-placed character. |
| OQ-3 | Map-existence check | atlas-maps' existing `data/map/info.Processor.GetById(mapId)` (→ `GET /api/data/maps/{id}`). Error ⇒ target does not exist ⇒ `400`. |
| OQ-4 | query-aggregator public contract | **No public-contract concern.** query-aggregator only serves `POST /validations`; it does not re-expose character `mapId`. Its `mapId` use is an *internal* read for `MapCondition` validation. It is migrated as an active internal consumer, not a contract change. |
| OQ-5 | Destination channel source | **Stored `location.channelId`.** It is the only source atlas-maps has and is kept current by LOGIN / CHANNEL_CHANGED / CHANNEL_CHANGE_REQUEST consumers. |
| OQ-6 | Characters-table map column | **Per-row React Query location lookup.** The table loads from atlas-character directly (not the aggregator); once `mapId` is gone the column reads `GET .../location` per row (deduped/cached, bounded by pagination). No new bulk endpoint. |
| — | Passive field-strip scope | **Strict.** Strip vestigial `MapId` from all five passive mirrors (channel, login, npc-shops, cashshop, messengers) per FR-6.2/6.4. A declared-but-dead `MapId` is a footgun. |

---

## 3. Architecture — atlas-maps location write

### 3.1 Shared warp processor (FR-1.4)

The `CHANGE_MAP` consumer body (`kafka/consumer/character/change_map.go:25-58`) is the canonical warp. It currently does, inline:

1. `lp := location.NewProcessor(l, ctx, db)`; `old := lp.GetById(charId)` → `oldField`.
2. `lp.Set(charId, newField)` — persist destination.
3. Emit `MAP_CHANGED` via `producer.MapChangedStatusProvider(txn, charId, worldId, oldField, newField, portalId)` on `EVENT_TOPIC_CHARACTER_STATUS`.
4. `mp := _map.NewProcessor(l, ctx, pp, db)`; `mp.TransitionMapAndEmit(txn, newField, charId, oldField)`.

Factor steps 1–4 into a single method so the Kafka consumer and the new REST handler share one implementation. New package/processor:

```
services/atlas-maps/atlas.com/maps/character/warp/processor.go

type Processor interface {
    // ChangeMap persists dest as the character's location, emits MAP_CHANGED,
    // and transitions the per-map registries. dest must be a fully-formed field
    // (world, channel, map, instance). Reads the current row internally for the
    // MAP_CHANGED "old" side; if absent, oldField defaults to dest (parity with
    // today's consumer).
    ChangeMap(transactionId uuid.UUID, characterId uint32, worldId world.Id, dest field.Model, portalId uint32) error
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor
```

`NewProcessor` builds `location.NewProcessor`, the producer (`producer.ProviderImpl(l)(ctx)`), and `_map.NewProcessor` internally — exactly what the consumer constructs today. The method body is a verbatim move of steps 1–4.

- **`change_map.go` consumer** becomes: build `newField` from the command body, then `warp.NewProcessor(l, ctx, db).ChangeMap(c.TransactionId, c.CharacterId, c.WorldId, newField, c.Body.PortalId)`.
- **REST handler** (below) calls the same method.

Behavior parity is mechanical because the method *is* the old body, not a re-implementation (satisfies FR-7.2 / the "single implementation" acceptance criterion).

> Note: the REST handler also calls `location.GetById` once (to resolve world+channel and to 404 on a missing row); `ChangeMap` reads the row again internally for `oldField`. This is one extra cheap tenant-scoped DB read on the admin path — acceptable. The plan may pass the already-read current field in to avoid the double read; not required.

### 3.2 New write route & handler (FR-1.1–1.6)

Register on the existing location resource (`character/location/rest.go`'s `InitResource`):

```
PATCH /characters/{characterId}/location
  (ingress: ^/api/characters/[^/]+/location(/.*)?$ → atlas-maps:8080, already exists — no routes.conf change)
```

Body is a JSON:API `character-locations` resource (`GetName()` already returns `"character-locations"`). Only `attributes.mapId` is read; `channelId`/`instance` in the body are **ignored** (map-only warp) and documented as such. Use the project's `rest.RegisterInputHandler[RestModel]` envelope path so a bare body is rejected with the standard 400.

Handler sequence:

1. Parse `characterId` (path) and `targetMapId` (`attributes.mapId`).
2. `cur, err := location.NewProcessor(l, ctx, db).GetById(characterId)`.
   - On not-found ⇒ **`404`** (OQ-2). On other error ⇒ `500`.
3. **Validate target map exists:** `info.NewProcessor(l, ctx).GetById(targetMapId)`. Error ⇒ **`400`** (do not persist, do not emit). Distinguish from infra `500` by the not-found/transport nature of the error (mirror how `Resolve` already treats `info.GetById` failures).
4. Build destination: `dest := field.NewBuilder(cur.WorldId(), cur.ChannelId(), targetMapId).SetInstance(uuid.Nil).Build()` — same world, **same stored channel** (OQ-5), `instance = uuid.Nil` (non-instanced), spawn portal `portalId = 0`.
5. `warp.NewProcessor(l, ctx, db).ChangeMap(uuid.New(), characterId, cur.WorldId(), dest, 0)`.
6. On success ⇒ **`204 No Content`** (OQ-1).

**Online vs offline (FR-1.5):** identical handler path. For an online character, the emitted `MAP_CHANGED` is consumed by atlas-channel and warps the live client; for an offline character the durable row is simply updated and the event is a harmless no-op downstream. Success = durable write succeeded.

**Observability (NFR):** info on warp (`character_id`, `map_id`, `tenant`, `transaction_id`); warn on `400` validation reject; error on infra `500`. No silent swallow — the original bug class.

---

## 4. Consumer migration — corrected scope

Audit verdict (whether the **character-GET** `mapId` actually reaches logic, vs. a same-named live/session/Kafka field):

| Service | Verdict | Evidence | Action |
|---|---|---|---|
| **atlas-parties** | ACTIVE | `character/processor.go:268` builds member field from `fm.MapId()` (foreign GET) with `channelId=0` | Add an atlas-maps location client; build the full field (world, channel, map, instance) from location. Drop `ForeignRestModel.MapId` (`character/rest.go:38,103`). |
| **atlas-consumables** | **ACTIVE** (PRD said passive) | `consumable/processor.go:312,427,436` — `c.MapId()` from `GetById` drives map/position/monster lookups | Add/reuse an atlas-maps location client; source the character's field from location. Drop `MapId` from the character mirror. |
| **atlas-query-aggregator** | ACTIVE (internal only) | `validation/model.go:397` `MapCondition` uses `character.MapId()` from the internal GET client | Repoint `MapCondition` to an atlas-maps location lookup. **Required** — after the shim is gone, `character.MapId()` would read `0` and break the condition. No public-contract change (serves only `POST /validations`). |
| **atlas-channel** | PASSIVE | `f` in `portal/processor.go:35` is a live session field, not the GET; channel reads location/socket itself | Field-strip the vestigial `MapId` from the character mirror. |
| **atlas-login** | PASSIVE | `socket/writer/character_list.go` already fetches map via `location.GetField` | Field-strip. |
| **atlas-npc-shops** | PASSIVE | declared, never read | Field-strip. |
| **atlas-cashshop** | PASSIVE | declared, never read | Field-strip. |
| **atlas-messengers** | PASSIVE | declared, never read | Field-strip. |
| **atlas-fame** | N/A | no `MapId` field | none |

"Field-strip" = remove the `MapId` field from the service's character `RestModel`/`ForeignRestModel`, its `Extract`, and the model getter if present. Decoding tolerates the now-absent JSON key (zero value), so the strip is safe and independent of rollout ordering — but per §6 it still lands before the atlas-character shim removal.

The three **active** services each get a small atlas-maps location client (or reuse one) returning a `field.Model` for a character id, mirroring atlas-character's existing `location/requests.go` (`GET /characters/{id}/location` → `ErrNotFound` on 404). On `ErrNotFound`, preserve each service's current zero-value/skip behavior.

---

## 5. atlas-character — retire the shim (LAST, §6)

- **FR-3.1** Delete the dead `mapId` branch in `Update` (`character/processor.go:1752-1762`).
- **FR-4.1** Remove the `location.GetField` call from `Transform` (`rest.go:82`) and stop populating `MapId`/`Instance` in `transformWithTemporal` (`rest.go:122-123`).
- **FR-4.2** Remove `Instance` from `RestModel` entirely (`rest.go:45`) — GET-output-only, no create use.
- **FR-4.3** Keep `MapId` on `RestModel` (`rest.go:44`) **as create-input only**: `handleCreateCharacter` still passes `input.MapId` to `CreateAndEmit` (`resource.go:161`). Document the input-only asymmetry; it is absent from GET responses after this change.
- **FR-4.5 (corrected):** the `location` client package **stays** — `location.GetField` has four other callers (`processor.go:391` Login, `:425` Logout, `:1144` ChangeHP, `:1194` SetHP). Only the `Transform` call site is removed.

---

## 6. atlas-ui

- **Location service + type (new):** `services/api/locations.service.ts` (or extend an existing service) with `getByCharacterId(id) → GET /api/characters/{id}/location` and `changeMap(id, mapId) → PATCH /api/characters/{id}/location`. Both use the JSON:API envelope `{ data: { type: "character-locations", id, attributes } }` (matching atlas-maps' `GetName()` and the repo's `charactersService.update` envelope shape). A new `CharacterLocation` type holds `{ worldId, channelId, mapId, instance }`.
- **ChangeMapDialog** (`components/features/characters/ChangeMapDialog.tsx`): read current map from the **location** endpoint (replaces `character.attributes.mapId` at lines 19, 51) for the initial value, "differs from current" validation, cancel-reset, and the description; write via `locationsService.changeMap(id, mapId)` (replaces `charactersService.update` at line 90).
- **Characters table map column** (`pages/characters-columns.tsx:151-160`): source `mapId` from a per-row location query (OQ-6), keep the `/maps/{mapId}` link, and render blank/"—" when location is unknown (graceful degrade).
- **Type cleanup** (`types/models/character.ts`): remove `mapId` (line 34) from `CharacterAttributes` and `mapId` from `UpdateCharacterData` (line 44) once no UI code reads them off the character resource. (There is no `instance` on the UI type today; `spawnPoint`/`x`/`y`/`stance` are separate and untouched.)

`npm run build` type-checks `*.test.ts`, so test call sites change in the same commit as the signatures they touch.

---

## 7. Execution ordering (shim removal is last)

1. **atlas-maps** — `warp.Processor` + `ChangeMap`; rewire `change_map.go` to call it; add `PATCH .../location` handler + map-existence validation + `404` on no-row; tests FR-7.1–7.3.
2. **atlas-ui** — location service/type; repoint ChangeMapDialog (read+write) and the table column; tests FR-7.4. (Independent of the Go consumer order; can land any time after step 1 exists, but the visible warp needs step 1 deployed.)
3. **atlas-parties** — location client; full-field member construction; drop `ForeignRestModel.MapId`.
4. **atlas-consumables** — location client; source field from location; drop mirror `MapId`.
5. **atlas-query-aggregator** — repoint `MapCondition` to atlas-maps location.
6. **Passive strips** — atlas-channel, atlas-login, atlas-npc-shops, atlas-cashshop, atlas-messengers: remove vestigial `MapId`.
7. **atlas-character** (LAST) — delete the `Update` branch, the `Transform` shim, the GET `MapId`/`Instance` projection, and the `Instance` struct field; keep `MapId` create-input.

Rationale: steps 3–6 ensure no live reader depends on the echoed `mapId` before step 7 removes it. Steps are individually compilable; nothing in 3–6 *requires* 7 to build.

---

## 8. Testing

- **FR-7.1** atlas-maps: `PATCH .../location` persists the destination field and emits `MAP_CHANGED` (the assertion the old atlas-character path lost). Byte/field-level assertion on the emitted `StatusEventMapChangedBody`.
- **FR-7.2** Parity: the `CHANGE_MAP` consumer and the REST handler both invoke `warp.Processor.ChangeMap` — assert single shared method (e.g. table-driven test that the destination field + emitted event are identical for a command-driven and a REST-driven warp of the same target).
- **FR-7.3** Invalid `mapId` ⇒ `400`, no `location.Set`, no emission. Plus: no-row character ⇒ `404`.
- **FR-7.4** UI: ChangeMapDialog reads current map from the location endpoint and writes via it; table column renders from location and degrades when unknown.
- Migration regressions: atlas-parties full-field construction (channel no longer hardcoded 0), atlas-consumables map/position lookups, query-aggregator `MapCondition` all source from atlas-maps.

---

## 9. Risks & mitigations

- **Stale stored channel (OQ-5):** if a channel change hasn't been consumed by atlas-maps yet, an admin warp could target the previous channel. Acceptable for a rare admin action; the warp still lands the character correctly on the resolved channel. No live-session lookup added.
- **Map-existence false negative:** if atlas-data is briefly unreachable, validation must return `500` (infra), not `400` (bad map) — distinguish by error type so a transient outage doesn't tell the admin "no such map." Mirror `Resolve`'s existing error handling.
- **Blast radius:** 8 Go modules + atlas-ui change. Each Go module touched must pass the full gate; `go.mod`-touched services must `docker buildx bake`. The strict passive-strip choice adds 5 bake targets — intentional (FR-6.4 cleanliness).
- **Ordering violation:** removing the shim (step 7) before steps 3–6 land would break atlas-parties/consumables/query-aggregator and the UI column. Enforced by ordering; reviewers verify step 7 is last.

---

## 10. Verification gate (per changed module)

`go test -race ./...`, `go vet ./...`, `go build ./...` clean; `docker buildx bake atlas-<svc>` for every service whose `go.mod` was touched (maps, parties, consumables, query-aggregator, channel, login, npc-shops, cashshop, messengers, character); `tools/redis-key-guard.sh` clean from repo root; atlas-ui `npm run build` (type-checks tests).

---

## 11. Out of scope (unchanged from PRD)

Channel changes (`CHANGE_CHANNEL_REQUEST`), character-creation spawn-map contract, instanced/PQ/portal-script warps, any new UI surface beyond repointing the existing dialog and column, and map-access/level-gating authorization. Validation is map *existence* only.
