# atlas-mts

## Responsibility

atlas-mts is the marketplace (MTS — Maple Trade Station) service. It owns
marketplace listings (fixed-price sales, auctions, and want-ad offers),
auction bids and their NX escrow, take-home holdings, character wish-lists
(cart entries and wanted want-ads), settled transaction history, and a
read-through view of a character's cash-shop wallet balances for the buy
pre-check. It runs a periodic DB-driven sweep that settles or expires
auctions and fixed-price listings whose sale term has passed, and
participates in cross-service saga flows (listing creation, buy/settle, bid
escrow, take-home) coordinated by the atlas-saga orchestrator over Kafka.

atlas-mts does not own currency balances: all NX/mesos/points mutation is
performed by the saga orchestrator's AwardMesos/AwardCurrency/MtsBidEscrow
steps against the owning services (atlas-cashshop, character mesos); this
service only reads a cash-shop balance for a best-effort pre-check.

## External Dependencies

- PostgreSQL (via GORM, `gorm.io/driver/postgres`) / SQLite
  (`gorm.io/driver/sqlite`, used by the test harness) — primary datastore for
  listings, bids, holdings, wish entries, transaction history, and the shared
  ITC serial counter.
- Kafka — command/status topics for the high-level MTS domain
  (`COMMAND_TOPIC_MTS` / `EVENT_TOPIC_MTS_STATUS`), the custody sub-protocol
  (`COMMAND_TOPIC_MTS_CUSTODY` / `EVENT_TOPIC_MTS_CUSTODY_STATUS`), and the
  shared saga orchestrator (`COMMAND_TOPIC_SAGA`). See
  [docs/kafka.md](docs/kafka.md).
- atlas-cashshop (REST) — the authoritative wallet: read-only balance checks
  for the buy pre-check and the wallet passthrough endpoint. All balance
  mutation happens via saga steps, never a direct atlas-mts write.
- atlas-tenants (REST) — per-tenant MTS configuration (listing fee,
  commission rate/base, active-listing cap, sell-level gate, auction
  duration bounds, fixed-sale term, price floor, page size, minimum bid
  increment), fetched and cached by the configuration registry.
- atlas-saga orchestrator (via Kafka) — drives the multi-step
  list/buy/take-home/bid-escrow flows and their compensation.
- atlas-outbox — transactional-outbox library backing the atomic
  local-write-plus-Kafka-emit pattern used by the command handlers; a
  background drainer (leader-gated by a Postgres advisory lock) publishes
  queued rows after commit.

## Runtime Configuration

- `REST_PORT` — HTTP listen port.
- `EXPIRATION_CHECK_INTERVAL_SECONDS` — cadence of the auction/listing
  expiration sweep (falls back to 60s if unset or invalid).
- `MTS_TEST_ROUTES_ENABLED` — when set to `"true"`, mounts an additional
  `/api/test/*` route set (seed/expire/sweep/simulated purchase and bid)
  used for end-to-end testing. These routes are never routed through
  ingress and must never be enabled in production.
- Kafka topic-name environment variables corresponding to the tokens
  documented in [docs/kafka.md](docs/kafka.md) (`COMMAND_TOPIC_MTS`,
  `EVENT_TOPIC_MTS_STATUS`, `COMMAND_TOPIC_MTS_CUSTODY`,
  `EVENT_TOPIC_MTS_CUSTODY_STATUS`, `COMMAND_TOPIC_SAGA`), plus the
  consumer group id (resolved via `consumergroup.Resolve("MTS Service")`).
- Standard `atlas-service` / `atlas-database` / `atlas-tracing` bootstrap
  environment variables apply, as in other Atlas services.

## Documentation

- [docs/domain.md](docs/domain.md) — domain models, invariants, and
  processors.
- [docs/kafka.md](docs/kafka.md) — Kafka topics, message types, and
  transaction semantics.
- [docs/rest.md](docs/rest.md) — public HTTP endpoints.
- [docs/storage.md](docs/storage.md) — persistent storage schema.
