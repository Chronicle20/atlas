# Kafka Retry Logic: Task Checklist

**Last Updated: 2026-02-19**
**Status: COMPLETE**

---

## Phase 1: Upgrade Shared Library

- [x] **1.1** Add `RetryConfig` struct and `ExecuteWithRetry` to shared retry library [M]
  - [x] `RetryConfig` with `MaxRetries`, `InitialDelay`, `MaxDelay`, `BackoffFactor`
  - [x] Builder methods: `WithMaxRetries()`, `WithInitialDelay()`, `WithMaxDelay()`, `WithBackoffFactor()`
  - [x] `DefaultConfig()` returns sensible defaults
  - [x] Exponential backoff: `delay = initialDelay * factor^(attempt-1)`
  - [x] Full jitter: `delay = rand(0, calculatedDelay)`
  - [x] Max delay cap
  - [x] Context-aware sleep (`select` on `ctx.Done()` and `time.After`)
  - [x] Error wrapping with `%w` preserves original error
- [x] **1.2** Make legacy `Try()` a backward-compatible wrapper [S]
- [x] **1.3** Add comprehensive tests [M]
  - [x] Exponential delay bounds
  - [x] Jitter within range
  - [x] Max delay cap
  - [x] Context cancellation
  - [x] Error wrapping
  - [x] Zero-retry config

## Phase 2: Upgrade Kafka Consumer & Producer

- [x] **2.1** Update consumer fetch retry in `consumer/manager.go` [S]
  - [x] Use `ExecuteWithRetry` with context
  - [x] Config: 10 retries, 100ms initial, 10s max, 2.0 factor
- [x] **2.2** Update producer write retry in `producer/producer.go` [S]
  - [x] Use `ExecuteWithRetry` with context
  - [x] Config: 10 retries, 100ms initial, 10s max, 2.0 factor
- [x] **2.3** Update/verify all `libs/atlas-kafka` tests [S]

## Phase 3: Upgrade REST Client

- [x] **3.1** Replace `libs/atlas-rest/retry` with shared library [S]
  - [x] Extract to `libs/atlas-retry`
  - [x] Add to `go.work`
  - [x] Update `libs/atlas-kafka` to import `libs/atlas-retry`
  - [x] Update `libs/atlas-rest` to import `libs/atlas-retry`
- [x] **3.2** Configure REST default retry count [S]

## Phase 4: Consolidate Service Retry Packages

- [x] **4.1** Verify complete list of service-local retry packages [S]
- [x] **4.2** Replace DB connection retry in each service [L]
  - [x] atlas-account
  - [x] atlas-ban
  - [x] atlas-buddies
  - [x] atlas-cashshop
  - [x] atlas-character
  - [x] atlas-configurations
  - [x] atlas-data
  - [x] atlas-drop-information
  - [x] atlas-fame
  - [x] atlas-families
  - [x] atlas-gachapons
  - [x] atlas-guilds
  - [x] atlas-inventory
  - [x] atlas-keys
  - [x] atlas-map-actions
  - [x] atlas-notes
  - [x] atlas-npc-conversations
  - [x] atlas-npc-shops
  - [x] atlas-party-quests
  - [x] atlas-pets
  - [x] atlas-portal-actions
  - [x] atlas-quest
  - [x] atlas-reactor-actions
  - [x] atlas-saga-orchestrator
  - [x] atlas-skills
  - [x] atlas-storage
  - [x] atlas-tenants
- [x] **4.3** Delete all local retry packages and dead code [S]

## Phase 5: Clean Up

- [x] **5.1** Remove atlas-marriages local retry, use shared library [S]
- [x] **5.2** Remove atlas-maps local retry, use shared library [S]
- [x] **5.3** Final verification: no service-local retry packages remain [S]
- [x] **5.4** Build and test all services [M]
