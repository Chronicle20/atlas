# Kafka Retry Logic: Context

**Last Updated: 2026-02-19**

---

## Key Files

### Shared Libraries (modify)

| File | Purpose |
|------|---------|
| `libs/atlas-kafka/retry/retry.go` | Current shared retry logic (fixed 1s sleep) |
| `libs/atlas-kafka/retry/retry_test.go` | Tests for shared retry |
| `libs/atlas-kafka/consumer/manager.go:174-194` | Consumer fetch retry (10 attempts, 1s fixed) |
| `libs/atlas-kafka/producer/producer.go:80-86,111-122` | Producer write retry (10 attempts, 1s fixed) |
| `libs/atlas-rest/retry/retry.go` | REST client retry (identical pattern) |
| `libs/atlas-rest/requests/config.go` | REST default retry count (1 = no retry) |

### Reference Implementation

| File | Purpose |
|------|---------|
| `services/atlas-marriages/atlas.com/marriages/retry/retry.go` | Full exponential backoff with config builder, context awareness, error classification |
| `services/atlas-marriages/atlas.com/marriages/retry/retry_test.go` | Comprehensive test suite |
| `services/atlas-marriages/atlas.com/marriages/scheduler/proposal_expiry.go:110-136` | Usage examples with different configs |

### Services with Local Retry Packages (delete)

All contain identical `retry/retry.go` with fixed 1s sleep, used for DB connection retry:

- `services/atlas-account/atlas.com/account/retry/`
- `services/atlas-ban/atlas.com/ban/retry/`
- `services/atlas-buddies/atlas.com/buddies/retry/`
- `services/atlas-cashshop/atlas.com/cashshop/retry/`
- `services/atlas-character/atlas.com/character/retry/`
- `services/atlas-configurations/atlas.com/configurations/retry/`
- `services/atlas-data/atlas.com/data/retry/`
- `services/atlas-drop-information/atlas.com/drop-information/retry/`
- `services/atlas-fame/atlas.com/fame/retry/`
- `services/atlas-families/atlas.com/families/retry/`
- `services/atlas-gachapons/atlas.com/gachapons/retry/`
- `services/atlas-guilds/atlas.com/guilds/retry/`
- `services/atlas-inventory/atlas.com/inventory/retry/`
- `services/atlas-keys/atlas.com/keys/retry/`
- `services/atlas-map-actions/atlas.com/map-actions/retry/`
- `services/atlas-maps/atlas.com/maps/retry/` (variant: linear backoff)
- `services/atlas-notes/atlas.com/notes/retry/`
- `services/atlas-npc-conversations/atlas.com/npc-conversations/retry/`
- `services/atlas-npc-shops/atlas.com/npc-shops/retry/`
- `services/atlas-party-quests/atlas.com/party-quests/retry/`
- `services/atlas-pets/atlas.com/pets/retry/`
- `services/atlas-portal-actions/atlas.com/portal-actions/retry/`
- `services/atlas-quest/atlas.com/quest/retry/`
- `services/atlas-reactor-actions/atlas.com/reactor-actions/retry/`
- `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/retry/`
- `services/atlas-skills/atlas.com/skills/retry/`
- `services/atlas-storage/atlas.com/storage/retry/`
- `services/atlas-tenants/atlas.com/tenants/retry/`

### Special Cases

| File | Notes |
|------|-------|
| `services/atlas-maps/atlas.com/maps/retry/retry.go` | Linear backoff variant (`attempt * 1s`) |
| `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/processor.go:66,312-329` | Optimistic lock retry (3 attempts, 10ms fixed) — leave as-is, domain-specific |
| `services/atlas-inventory/atlas.com/inventory/compartment/lock_registry.go:12-16` | Redis distributed lock spin-loop — leave as-is, domain-specific |

---

## Key Decisions

### Decision 1: Library Extraction Strategy

**Options:**
- A: Make `libs/atlas-rest` depend on `libs/atlas-kafka/retry` (coupling)
- B: Extract to new `libs/atlas-retry` module (clean separation)
- C: Copy upgraded logic into both (duplication)

**Recommended: Option B** — creates `libs/atlas-retry` as a standalone module. Both `atlas-kafka` and `atlas-rest` import it. Requires adding to `go.work`.

### Decision 2: Backward Compatibility of `Try()`

Keep `Try(fn, retries)` as a wrapper that uses fixed 1s delay (no backoff) to avoid breaking any caller that depends on the exact timing. New code should use `ExecuteWithRetry(ctx, config, fn)`.

### Decision 3: Jitter Strategy

Use **full jitter**: `delay = random(0, min(maxDelay, initialDelay * factor^attempt))`. This provides the best decorrelation per AWS architecture blog research. The atlas-marriages implementation does NOT include jitter — the shared library should add it.

### Decision 4: Domain-Specific Retries Left Alone

The saga optimistic lock retry (10ms fixed) and inventory distributed lock (50ms spin) are domain-specific patterns that should NOT be replaced by the generic retry library. They have different semantics (contention-based, not failure-based).

---

## Dependencies

- `go.work` must include new `libs/atlas-retry` if Option B is chosen
- No external Go module dependencies needed (`math/rand` and `context` are stdlib)
- atlas-marriages tests provide proven patterns for testing backoff timing

---

## Related Documentation

- `docs/architectural-improvements.md` — tracks cross-cutting improvements
- `dev/active/redis-registry-migration/` — prior bulk migration across services (process reference)
