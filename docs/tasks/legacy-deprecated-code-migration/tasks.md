# Deprecated Code Migration — Task Checklist

Last Updated: 2026-02-19

## Phase 1: Quick Wins (No Cross-Service Impact) — COMPLETE

### 1.1 Replace `io/ioutil.ReadFile` in atlas-account [S] — DONE
- [x] Replace `ioutil.ReadFile` with `os.ReadFile` in `configuration/loader.go`
- [x] Remove `"io/ioutil"` import
- [x] `go test ./... -count=1` passes
- [x] `go build` succeeds

### 1.2 Remove `rand.Seed` from test files [S] — DONE
- [x] Remove `rand.Seed(0)` from `atlas-channel/tool/uint128_test.go:48`
- [x] Remove `rand.Seed(42)` from `atlas-monster-death/monster/processor_test.go:34`
- [x] `math/rand` import kept (still used for `rand.Uint32()` in channel, removed in monster-death)
- [x] Tests still pass in both services

### 1.3 Migrate `xml.Read` → `xml.FromPathProvider` in atlas-data [S] — DONE
- [x] Update `npc/string_registry.go:43`
- [x] Update `map/string_registry.go:40`
- [x] Update `item/string_registry.go:33,70`
- [x] Update `monster/string_registry.go:35`
- [x] Update `monster/gauge_registry.go:35`
- [x] Update `skill/string_registry.go:35`
- [x] Delete `xml.Read` function from `xml/reader.go`
- [x] `go build` succeeds

### 1.4 Remove `ItemId` deprecation shim from query-aggregator [S] — DONE
- [x] Remove `ItemId` field from `ConditionInput` struct
- [x] Remove `SetItemId` method from `ConditionBuilder`
- [x] Remove `else if input.ItemId != 0` fallback branch
- [x] Remove `ItemId` references from `rest.go` validation
- [x] Update tests referencing `ItemId` → `ReferenceId`
- [x] `go test ./... -count=1` passes
- [x] `go build` succeeds

---

## Phase 2: Single-Service Refactors — COMPLETE

### 2.1 Migrate `lib/pq` → GORM/pgx in atlas-gachapons [M] — DONE
- [x] Created custom `int64Array` type with `Value()`/`Scan()` in `entity.go`
- [x] Replaced `pq.Int64Array` with `int64Array` in `entity.go` and `administrator.go`
- [x] Removed `github.com/lib/pq` from `go.mod`
- [x] `go mod tidy` succeeded
- [x] `go test ./... -count=1` passes
- [x] `go build` succeeds

### 2.2 Remove `PetModelDecorator` from atlas-channel [M] — DONE
- [x] Merged pet-loading behavior into `PetAssetEnrichmentDecorator` (fetches pets, sorts by slot, sets on model, then enriches cash assets if present)
- [x] Removed `PetModelDecorator` from `character/processor.go` interface
- [x] Removed `PetModelDecorator` implementation from `character/processor.go`
- [x] Removed `PetModelDecorator` from `character/mock/processor.go`
- [x] Updated `character/processor_test.go`
- [x] Updated `socket/handler/character_info_request.go:31` → `PetAssetEnrichmentDecorator`
- [x] Updated `socket/handler/cash_shop_entry.go:41` — removed `PetModelDecorator`
- [x] Updated `kafka/consumer/map/consumer.go:89` — removed `PetModelDecorator`
- [x] Updated `kafka/consumer/asset/consumer.go:295` — removed `PetModelDecorator`
- [x] Updated `kafka/consumer/session/consumer.go:136` — removed `PetModelDecorator`
- [x] Updated `kafka/consumer/messenger/consumer.go:117,148` — removed `PetModelDecorator`
- [x] `go test ./... -count=1` passes
- [x] `go build` succeeds
- [x] Eliminates duplicate HTTP calls to atlas-pets service

---

## Phase 3: Cross-Service `server.Marshal` → `MarshalResponse` — COMPLETE

### 3.1 Migrate atlas-buddies [S] — DONE
- [x] Update `list/resource.go` (2 call sites)
- [x] `go test ./... -count=1` passes

### 3.2 Migrate atlas-parties [S] — DONE
- [x] Update `party/resource.go` (4 call sites)
- [x] `go test ./... -count=1` passes

### 3.3 Migrate atlas-maps [S] — DONE
- [x] Update `map/resource.go` (1 call site)
- [x] Update `map/weather/resource.go` (1 call site)
- [x] `go build` succeeds

### 3.4 Migrate atlas-monsters [S] — DONE
- [x] Update `world/resource.go` (2 call sites)
- [x] Update `monster/resource.go` (1 call site)
- [x] `go build` succeeds

### 3.5 Migrate atlas-messengers [S] — DONE
- [x] Update `messenger/resource.go` (4 call sites)
- [x] `go test ./... -count=1` passes

### 3.6 Migrate atlas-quest [S] — DONE
- [x] Update `quest/resource.go` (7 call sites)
- [x] `go test ./... -count=1` passes

### 3.7 Migrate atlas-reactors [S] — DONE
- [x] Update `reactor/resource.go` (3 call sites)
- [x] `go build` succeeds

### 3.8 Migrate atlas-character [S] — DONE
- [x] Update `character/rest_test.go` (1 call site, uses `make(map[string][]string)`)
- [x] `go test ./... -count=1` passes

### 3.9 Migrate atlas-keys [S] — DONE
- [x] Update `character/resource.go` (1 call site)
- [x] `go build` succeeds

### 3.10 Delete `server.Marshal` from libs/atlas-rest [S] — DONE
- [x] Delete `Marshal` function from `server/response.go`
- [x] Verify: `grep -r 'server.Marshal[' services/ libs/` returns no results

---

## Phase 4: Cross-Service `AwardInventory` → `AwardAsset`

### 4.1 Phase 4a — Migrate producers [M] — DONE
- [x] Add `AwardAsset` constant to `libs/atlas-script-core/saga/model.go`
- [x] Add `AwardAsset` constant to `atlas-quest/kafka/message/saga/kafka.go`
- [x] Add `AwardAsset` constant to `atlas-messages/saga/model.go`
- [x] Add `AwardAsset` alias to `atlas-npc-conversations/saga/model.go`
- [x] Update unmarshal case in `atlas-messages/saga/model.go` to handle both
- [x] Update `atlas-quest/kafka/producer/saga/producer.go:57` — `AwardInventory` → `AwardAsset`
- [x] `go test ./... -count=1` passes in atlas-quest
- [x] Update `atlas-messages/command/character/inventory/commands.go:93` — `AwardInventory` → `AwardAsset`
- [x] `go test ./... -count=1` passes in atlas-messages
- [x] Update `atlas-npc-conversations/conversation/operation_executor.go:853` — `AwardInventory` → `AwardAsset`
- [x] Update `atlas-npc-conversations/conversation/processor.go:720` — `AwardInventory` → `AwardAsset`
- [x] `atlas-npc-conversations/conversation` package builds (pre-existing issue in kafka/consumer/saga unrelated)

### 4.2 Phase 4b — Remove backward compatibility [M] — DONE

**Orchestrator cleanup:**
- [x] Removed `AwardInventory` constant from `saga/model.go`
- [x] Removed `handleAwardInventory` wrapper from `handler.go`
- [x] Removed `AwardInventory` case from `GetHandler` switch
- [x] Removed `AwardInventory` from unmarshal switch in `model.go`
- [x] Renamed `unmarshalAwardInventoryPayload` → `unmarshalAwardAssetPayload` in `rest.go`
- [x] Updated REST payload map: `AwardInventory` → `AwardAsset`
- [x] Removed `TestHandleAwardInventory` test function
- [x] Removed deprecated test cases from `TestHandleAwardAsset`
- [x] Updated all test files to use `AwardAsset` exclusively
- [x] Updated Bruno API spec and docs
- [x] `go test ./... -count=1` passes
- [x] `go build` succeeds

**Other service cleanup:**
- [x] Removed `AwardInventory` from `atlas-character-factory/saga/model.go`
- [x] Removed `AwardInventory` from `atlas-messages/saga/model.go` + updated unmarshal + all tests
- [x] Removed `AwardInventory` from `libs/atlas-script-core/saga/model.go` + updated unmarshal
- [x] Removed `AwardInventory` from `atlas-quest/kafka/message/saga/kafka.go`
- [x] Removed `AwardInventory` from `atlas-npc-conversations/saga/model.go`
- [x] `go test ./... -count=1` passes in all services
- [x] `go build` succeeds in all services

---

## Final Verification — DONE
- [x] `grep -r 'AwardInventory' services/ libs/` returns no results
- [x] `grep -r 'award_inventory' services/ libs/` returns no results
- [x] All affected services build and pass tests
