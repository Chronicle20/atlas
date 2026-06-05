# task-082 — Context

Companion to `plan.md`. Captures the key files, decisions, and dependencies an implementer needs, with the evidence each was grounded against.

## Problem (one line)

The v83 client's Character-Info decoder calls `CMobTemplate::GetMobTemplate(cover)` directly on the monster-book cover field; atlas-channel sends the stored **card item id** (e.g. `2380000`) there, which is not a valid mob id → client crash. The fix sends the cover card's **mob id** in that one packet; all other cover surfaces stay card-id.

Root cause + IDA evidence: memory `bug_monsterbook_cover_charinfo_is_mobid.md`; PRD §1; design §1.

## Chosen architecture

Set-time resolution in **atlas-monster-book** (design §2; PRD OQ-4):
- On cover set, resolve `coverCardId → mobId` via a new outbound atlas-data consumable client; persist `cover_mob_id` next to `cover_card_id`.
- Expose `coverMonsterId` on `GET /characters/{id}/monster-book`.
- atlas-channel's Character-Info handler already fetches that model per view (`MonsterBookDecorator`), so the writer reads the mob id with **zero** new round-trips on the hot path.

Rejected alternatives (design §2.1): encode-time resolution in channel (B2 — needs a brand-new client + cache in channel), and resolve-in-channel-set-handler (B1 — strict loser; REST PATCH bypasses channel).

## Key files (verified)

### atlas-monster-book (`services/atlas-monster-book/atlas.com/monster-book/`)
- `collection/processor.go` — `SetCoverAndEmit` (line ~214) validates ownership then emits `COVER_CHANGED`. `ProcessorImpl{l,ctx,db,t,cp}`; `NewProcessor(l,ctx,db)`; `WithTransaction`. **Add `dp consumable.Processor` + `resolveCoverMobId`.**
- `collection/administrator.go` — `setCover(db,tenantId,characterId,coverCardId,eventId)` (line ~54) uses the `last_cover_event_id` idempotency guard via an `Updates(map[...])`. **Add `coverMobId` param + `"cover_mob_id"` to the map.**
- `collection/entity.go` — GORM `entity`, `Migration` = `AutoMigrate(&entity{})`. **Add `CoverMobId uint32 gorm:"not null;default:0"`.**
- `collection/model.go` / `builder.go` — immutable Model + Builder + `Make` (entity→Model) + `ToEntity`. **Add `coverMobId` everywhere `coverCardId` appears.**
- `collection/rest.go` — `RestModel` + `Transform`. **Add `CoverMonsterId uint32 json:"coverMonsterId"`.** `PatchInput` unchanged.
- `character/resource.go` — `handleGet`/`handlePatch` both `collection.NewProcessor(d.Logger(), d.Context(), db)` then `Transform`. **No signature change** (NewProcessor builds `dp` internally).
- `kafka/consumer/monsterbook/consumer.go:84-85` — `SET_COVER` command → `colp.SetCoverAndEmit(...)`. **No change** (resolution is inside SetCoverAndEmit).
- New package `data/consumable/` — mirror `services/atlas-npc-shops/atlas.com/npc/data/consumable/` (canonical), with a **partial** RestModel + swappable `baseURLProvider`.

### atlas-channel (`services/atlas-channel/atlas.com/channel/`)
- `monsterbook/processor.go` — `Collection` struct (line 18) + getters. **Add `coverMonsterId` + `CoverMonsterId()`.**
- `monsterbook/rest.go` — `CollectionRestModel` (already has the JSON:API ref stubs) + `Extract`. **Add `CoverMonsterId` + map in `Extract`.**
- `monsterbook/model.go` — `Model` delegates to `Collection`. **Add `CoverMonsterId()`.**
- `socket/writer/character_info.go:60` — currently `Cover: uint32(mb.CoverCardId())`. **Change to `Cover: mb.CoverMonsterId()`** (the crash fix).
- `socket/handler/character_info_request.go:31` — appends `cp.MonsterBookDecorator`; `:62` announces `CharacterInfoBody`. (Read-only context.)
- `character/processor.go:177` — `MonsterBookDecorator` fails open on REST error (renders empty book). (Read-only context — fail-safe already exists.)

### libs/atlas-packet (`libs/atlas-packet/`)
- `character/clientbound/info.go` — `MonsterBookInfo.Cover uint32` / `WriteInt`; gated `GMS<=87 || JMS`. **No change** (semantic value only).
- `character/data.go:700-720` — `encodeMonsterBook` writes `MonsterBook.CoverCardId` (login-draw, flag `0x20000`). **No change** (FR-10 default).
- Tests: `character/clientbound/info_test.go` (has `pt.RoundTrip`, `pt.Variants`, `pt.CreateContext`), `character/data_test.go`.

### deploy
- `deploy/k8s/base/atlas-monster-book.yaml` — container `monster-book`, env list has `LOG_LEVEL` + `DB_*`, envFrom `atlas-env` configmap. **Add `DATA_SERVICE_URL`.**

## Decisions resolved (design §4)

- **OQ-3 / FR-7:** `COVER_CHANGED` stays card-id only. The only channel consumer (`handleCoverChanged`) sends `0x54` with the card id (window resolves card→mob client-side). **No Kafka change.**
- **OQ-1 / FR-10:** Login-draw `CharacterData` cover stays card id (`data.go` unchanged). Evidence: live crash occurred only on Character Info, never at login despite a cover being set; the window resolves card→mob itself. Execution gate: best-effort IDA confirm; default is no-change. (Task 9.)
- **OQ-2:** Lazy backfill — existing rows keep `cover_mob_id = 0` (safe no-cover render) until next set. The one live affected cover is already cleared. **No backfill job.**
- **OQ-4:** Set-time resolution in monster-book (this design).

## Dependencies & gotchas

- **`atlas-rest` enters atlas-monster-book's `require` block** (currently only a `replace`). → `go mod tidy` and **`docker buildx bake atlas-monster-book`** are mandatory (CLAUDE.md). `atlas-rest` is already `COPY`'d in the repo-root `Dockerfile` (lines 42, 71) — no Dockerfile/`go.work` edit, no new shared lib.
- **JSON:API ref stubs are mandatory** on the new consumable RestModel (`libs/atlas-rest/CLAUDE.md`): `GetReferences`, `GetReferencedIDs`, `SetToOneReferenceID`, `SetToManyReferenceIDs`. Missing stubs surface as a generic "not found" when the upstream response has a `relationships` block. The integration test must serve a fixture **with** a `relationships` block (FakeClient mocks bypass the unmarshal path).
- **404 vs other errors:** `requests.ErrNotFound` only on HTTP 404 (see test). `resolveCoverMobId` treats *any* error as fail-safe → mob id 0 + warning.
- **Kafka in tests:** `SetCoverAndEmit` calls `message.Emit` → `producer.ProviderImpl`, which needs a live broker. Existing tests only exercise the validation-rejection path. The plan therefore unit-tests `resolveCoverMobId` (fake `dp`) and `setCover` (sqlite) **separately**, avoiding the broker.
- **Test fakes:** in-package (`package collection`) tests construct `&ProcessorImpl{...}` literals and a local `fakeConsumable`. Build `consumable.Model` values via the exported `consumable.Extract(consumable.RestModel{...})` (its fields are unexported). No `*_testhelpers.go` (CLAUDE.md).
- **`requests.RootUrl("DATA")`** reads `DATA_SERVICE_URL`, else `BASE_SERVICE_URL` (`libs/atlas-rest/requests/url.go`). The `SetBaseURLForTest` helper swaps a package-level `baseURLProvider` and appends `/api/` (mirrors channel monsterbook `requests.go`).

## atlas-data response shape (consumed, unchanged)

`GET /api/data/consumables/{id}` → JSON:API resource type `consumables` with attributes incl. `monsterBook` (bool) and `monsterId` (uint32, parsed from WZ `info/mob`). Source: `services/atlas-data/atlas.com/data/consumable/rest.go:44-106`, `reader.go`.

## Verification gates (CLAUDE.md)

`go test -race ./...`, `go vet ./...`, `go build ./...` in atlas-monster-book / atlas-channel / libs/atlas-packet; `docker buildx bake atlas-monster-book`; `tools/redis-key-guard.sh` (no new redis usage). Plus the FR-10 IDA confirmation recorded in the task.

## Test data conventions

- Cover card item id: `2380000` (a real monster-book card; used across existing tests).
- Resolved mob id: `100100` (placeholder distinct from the card id, so a test failing on the old behavior reports `2380000`).
- Tenant: `GMS / 83 / 1` (matches existing test contexts and the live affected tenant's version gate `GMS<=87`).
