# atlas-trades

Owns 1:1 player-to-player trading: the trade room lifecycle, the durable
escrow custody of staged items and meso, the completed-trade ledger, and the
saga-driven settlement between two characters.

Live trade room state (`trade.Registry`) is a process-local in-memory
registry, not the database, so the service pins `replicas: 1` and carries no
HPA — that is a correctness constraint, not a capacity choice: a second
replica would split rooms across pods and break in-flight trades.

The database backs the escrow custody tables, the in-flight settlement
record, the completed-trade ledger, and the transactional outbox
(`libs/atlas-outbox`), which is drained to Kafka by the outbox drainer booted
in `main.go`. Every command handler batches its status events and saga
commands into one `message.Buffer` and flushes them inside the same
transaction as its durable writes and registry mutation, so a command's
Kafka output and its database state cannot diverge.

atlas-trades never writes a game packet — atlas-channel owns the wire and
translates this service's semantic status events into client opcodes.
atlas-trades never mutates inventory or meso directly — it drives every
movement of an asset or meso balance through composite sagas submitted to
atlas-saga-orchestrator.

## External Dependencies

- PostgreSQL: escrow custody (`trade_escrow_items`, `trade_escrow_mesos`,
  `trade_escrow_meso_stakes`, `trade_escrow_meso_refunds`), the in-flight
  settlement record (`trade_settlements`, `trade_settlement_sides`,
  `trade_settlement_items`), the completed-trade ledger
  (`trade_ledger_entries`, `trade_ledger_sides`, `trade_ledger_items`), and
  the transactional outbox
- Kafka: consumes trade commands, trade custody commands, invite status,
  saga status, character status and session status; produces trade status
  events, trade custody status (ack) events, invite commands and saga
  commands
- atlas-tenants: REST API for the per-tenant trade configuration
  (`trade-configs`) — meso tax tiers, max staged items, minimum trade level,
  attestation timeout
- atlas-characters: REST API for character level, HP and meso
- atlas-maps: REST API for a character's current field
- atlas-inventory: REST API for reading an asset in a compartment slot and
  for reading a whole compartment
- atlas-data: REST API for item trade-block flags and slot-max, and for map
  field limits
- atlas-saga-orchestrator: REST API for reading a saga's outcome at startup
  reconciliation; Kafka for submitting and receiving the outcome of every
  saga this service runs
- OpenTelemetry: distributed tracing via OTLP/gRPC (standard Atlas service
  bootstrap)

## Runtime Configuration

| Variable | Purpose |
|---|---|
| `REST_PORT` | HTTP listen port (`8080` in-cluster). |
| `DB_NAME`, `DB_USER`, `DB_PASSWORD` | PostgreSQL connection (`atlas-trades`, env-suffixed by the kustomize overlays). |
| `BOOTSTRAP_SERVERS` | Kafka brokers. |
| `LOG_LEVEL` | Logrus level. |
| `COMMAND_TOPIC_TRADE` | Inbound trade room commands (from atlas-channel). |
| `EVENT_TOPIC_TRADE_STATUS` | Outbound trade status events (to atlas-channel). |
| `COMMAND_TOPIC_TRADE_CUSTODY` | Inbound escrow custody commands (from atlas-saga-orchestrator). |
| `EVENT_TOPIC_TRADE_CUSTODY_STATUS` | Outbound escrow custody ack events (to atlas-saga-orchestrator). |
| `COMMAND_TOPIC_INVITE` | Outbound trade-invite commands (to atlas-invites). |
| `EVENT_TOPIC_INVITE_STATUS` | Inbound trade-invite answers (from atlas-invites). |
| `COMMAND_TOPIC_SAGA` | Outbound saga commands (to atlas-saga-orchestrator). |
| `EVENT_TOPIC_SAGA_STATUS` | Inbound saga terminal outcomes (from atlas-saga-orchestrator). |
| `EVENT_TOPIC_CHARACTER_STATUS` | Inbound character status (LOGOUT, MAP_CHANGED, CHANNEL_CHANGED). |
| `EVENT_TOPIC_SESSION_STATUS` | Inbound session status (DESTROYED). |
| `TENANTS` | atlas-tenants REST API base URL. |
| `CHARACTERS` | atlas-characters REST API base URL. |
| `MAPS` | atlas-maps REST API base URL. |
| `INVENTORY` | atlas-inventory REST API base URL. |
| `DATA` | atlas-data REST API base URL. |
| `SAGAS` | atlas-saga-orchestrator REST API base URL. |

Ingress routes `/api/trades/...` to this service.

## Documentation

- [Domain](docs/domain.md)
- [Kafka](docs/kafka.md)
- [REST](docs/rest.md)
- [Storage](docs/storage.md)
