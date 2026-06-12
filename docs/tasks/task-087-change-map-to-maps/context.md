# Task-087 — Context & Key Facts

Companion to `plan.md`. Captures the code audit that grounds the plan so an
implementer needs no prior context. All paths are relative to the worktree root
`.worktrees/task-087-change-map-to-maps/`.

## Goal (one line)

Move the character change-map **write** into atlas-maps (the location owner),
share one warp implementation between the Kafka consumer and a new REST handler,
migrate every consumer off the atlas-character `mapId` echo, then delete the
atlas-character location shim — shim removal **last**.

## Execution ordering (hard constraint)

The atlas-character shim removal (Task 11) MUST land after every consumer
migration (Tasks 7–10). A partial rollout that strips the echo early breaks
atlas-parties, atlas-consumables, atlas-query-aggregator, and the UI map column.
Tasks 1–2 (atlas-maps) come first because the write endpoint must exist before
the UI can call it. Within Go, Tasks 3–10 are individually compilable and do not
require Task 11 to build.

## atlas-maps — the write side (Tasks 1–2)

### Canonical warp logic (to be factored into one method)
`services/atlas-maps/atlas.com/maps/kafka/consumer/character/change_map.go:25-58`
does, inline:
1. `lp := location.NewProcessor(l, ctx, db)`; `old := lp.GetById(charId)` →
   `oldField` (defaults to the new field if no row).
2. `lp.Set(charId, newField)` — persist destination (returns error on failure).
3. Emit `MAP_CHANGED` via
   `producer.MapChangedStatusProvider(txn, charId, worldId, oldField, newField, portalId)`
   on `characterKafka.EnvEventTopicCharacterStatus` (logged-and-continue on error).
4. `mp := _map.NewProcessor(l, ctx, pp, db)`;
   `mp.TransitionMapAndEmit(txn, newField, charId, oldField)` (warn-and-continue).

### Key signatures
- `location.Processor` (`character/location/processor.go:17-22`):
  `GetById(uint32) (Model, error)`, `Set(uint32, field.Model) (Model, error)`,
  `Delete`, `Resolve`. `NewProcessor(l, ctx, db)`. Test seam:
  `newProcessorWithInfo(l, ctx, db, ip)`.
- `location.Model.Field() field.Model` (`character/location/model.go:27`).
- `info.Processor.GetById(mapId _map.Id) (info.Model, error)`
  (`data/map/info/processor.go:14-16`) → `GET /api/data/maps/{id}`. On a
  nonexistent map atlas-data returns 404 ⇒ the requests layer returns
  `requests.ErrNotFound` (`errors.Is(err, requests.ErrNotFound)`); other errors
  are infra. Test seam: `info` tests inject a `stubInfoProcessor`
  (`character/location/processor_test.go:19-26`).
- `_map.Processor.TransitionMapAndEmit(txn, newField, charId, oldField) error`
  (`map/processor.go:120`). `NewProcessor(l, ctx, producer.Provider, db)`.
- `producer.MapChangedStatusProvider(...)` (`kafka/producer/character.go:39`)
  builds the `StatusEventMapChangedBody` (`TargetMapId`, `OldMapId`, `ChannelId`,
  `TargetPortalId`, …).
- `producer.Provider = func(token string) producer.MessageProducer`
  (`kafka/producer/producer.go:10`);
  `MessageProducer = func(provider model.Provider[[]kafka.Message]) error`
  (`libs/atlas-kafka/producer/message.go:54`).
- `message.Emit(pp)(func(buf *message.Buffer) error)` /
  `buf.GetAll() map[string][]kafka.Message` (`kafka/message/message.go`).

### REST input-handler pattern in atlas-maps
- `rest.RegisterInputHandler[M](l)(si)(name, handler)` →
  handler is `rest.InputHandler[M] = func(d *HandlerDependency, c *HandlerContext, model M) http.HandlerFunc`
  (`rest/handler.go:21,29`; `libs/atlas-rest/server/context.go:43`).
- `rest.ParseCharacterId(l, next func(uint32) http.HandlerFunc)`
  (`rest/handler.go:49`).
- Existing GET handler closes over `db`:
  `character/location/resource.go:27` `handleGetCharacterLocation(db)`.
- Location resource `GetName()` already returns `"character-locations"`
  (`character/location/rest.go:22`).

### Test harness facts
- `location` tests use sqlite in-memory + `newCtxTenant(t)` for a tenant context
  (`character/location/processor_test.go`).
- `map` tests use a capturing `mockProducerProvider` exposing
  `.Provider() producer.Provider` and `.GetMessages(topic) []kafka.Message`
  (`map/processor_test.go:116-152`).
- Ingress already routes `^/api/characters/[^/]+/location(/.*)?$ → atlas-maps:8080`
  (`deploy/shared/routes.conf`). **No routes.conf change.**

## Consumer migration audit (Tasks 7–10)

| Service | Verdict | Where `mapId` is read | Action |
|---|---|---|---|
| atlas-parties | ACTIVE | `character/processor.go:268` builds member field from `fm.MapId()` with channel hardcoded `0` | new location client; full field; drop `ForeignRestModel.MapId` (`character/rest.go:38,103`) |
| atlas-consumables | ACTIVE | `consumable/processor.go:427,436` `c.MapId()` (the `cp.GetById` mirror) in `ConsumeSummoningSack` | new location client; source map from location; drop mirror `MapId` (`character/rest.go:38`, `character/model.go:207`) |
| atlas-query-aggregator | ACTIVE (internal) | `validation/model.go:397` `MapCondition` uses `character.MapId()` | new single-char location client; repoint `MapCondition`; drop mirror `MapId` (`character/rest.go:39`, `character/model.go:218`). Does NOT re-expose mapId in its own responses (verified). |
| atlas-channel | PASSIVE | mirror `MapId` declared, never read (`character/rest.go:38`, `model.go:217`). The `f.MapId()` in `portal/processor.go:35` is a live-session field, not the GET. | field-strip |
| atlas-login | PASSIVE | mirror `MapId` declared, never read; `character_list.go` already uses atlas-maps `location.GetField` | field-strip (`character/rest.go:38`, `model.go:205`) |
| atlas-npc-shops (module `atlas-npc`) | PASSIVE | declared, never read | field-strip (`character/rest.go:38`, `model.go:202`) |
| atlas-cashshop | PASSIVE | declared, never read | field-strip (`character/rest.go:38`, `model.go:205`) |
| atlas-messengers | PASSIVE | declared on `ForeignRestModel`, never read | field-strip (`character/rest.go:37`, `model.go:128`) |
| atlas-fame | N/A | no `MapId` field | none |

### Location-client reference (to mirror in parties/consumables/query-aggregator)
atlas-character already has the exact client to copy:
`services/atlas-character/atlas.com/character/location/requests.go`
— `RootUrl("MAPS")`, `Resource = "characters/%d/location"`, `GetField(l, ctx, id) (field.Model, error)` returning
`location.ErrNotFound` on HTTP 404. Each active service gets its own copy in a
new `location` package (services don't share internal packages).

- atlas-consumables already has an in-memory live-field registry
  (`map/character` `GetMap`) — that is **not** the atlas-character echo and is
  left untouched. Only the `cp.GetById` mirror `c.MapId()` reads migrate.
- atlas-query-aggregator already has a `map` client using `RootUrl("MAPS")` for
  `characters-in-map` (`map/requests.go`) but **no** single-character location
  endpoint — add a new `location` package.

## atlas-character — the shim (Task 11, LAST)
- Dead write branch: `character/processor.go` the `if input.MapId != 0 { … Debug … }`
  block (the design's "1752-1762").
- `Transform` shim: `character/rest.go:78-92` calls `location.GetField`; projection
  populates `MapId`/`Instance` at `rest.go:122-123`.
- `RestModel.MapId` (`rest.go:44`) stays — create input only (`resource.go:161`
  passes `input.MapId` to `CreateAndEmit`). `RestModel.Instance` (`rest.go:45`)
  is removed entirely.
- `location.GetField` has **4 other callers** that stay (`processor.go:391, 425,
  1144, 1194`); only the `Transform` call site (`rest.go:82`) is removed. The
  `location` client package stays.

## atlas-ui (Tasks 3–6)
- `charactersService.update` builds `{data:{type:"characters",id,attributes}}` →
  `PATCH /api/characters/{id}` (`src/services/api/characters.service.ts:48-61`).
- `ChangeMapDialog` (`src/components/features/characters/ChangeMapDialog.tsx`)
  reads `character.attributes.mapId` (lines 19 initial, 51 validation, 94 reset,
  156 description) and writes via `charactersService.update(id, {mapId})` (line 90).
- Table map column (`src/pages/characters-columns.tsx:150-161`) renders
  `MapCell` from `row.getValue("attributes_mapId")`; columns receive data from
  the parent `Character[]` query — no per-row hook yet.
- UI `Character` type: `CharacterAttributes.mapId` (`types/models/character.ts:34`),
  `UpdateCharacterData.mapId` (line 44). No `instance` on the UI type.
- Pattern service for GET-by-id + PATCH envelope: `src/services/api/maps.service.ts`.
- React Query convention: namespaced key objects + `useQuery`/`useMutation`
  (`src/lib/hooks/api/useCharacters.ts`, `useCharacterEffectiveStats.ts`).
- Tests are vitest; no existing ChangeMapDialog/charactersService tests; service
  tests live under `src/services/api/__tests__/`.
- `npm run build` is `tsc -b` and type-checks `*.test.ts` — test call sites must
  change in the same commit as the signatures they touch.

## Open questions — all resolved in design.md §2
204 on success (OQ-1); 404 when no location row (OQ-2); `info.GetById`
map-existence check (OQ-3); query-aggregator is internal-only (OQ-4); stored
`location.channelId` for the destination channel (OQ-5); per-row React Query for
the table (OQ-6).

## Verification gate (per changed module)
`go test -race ./...`, `go vet ./...`, `go build ./...` clean;
`docker buildx bake atlas-<svc>` from the worktree root for any service whose
`go.mod`/`go.sum` changed; `tools/redis-key-guard.sh` clean from repo root
(`GOWORK=off`); atlas-ui `npm run build`.
