# atlas-merchant

The merchant service manages personal (character) shops and hired merchants placed in Free Market rooms. It owns the full shop lifecycle — creation, setup, opening, maintenance, and closing (with a close reason) — along with item listing management, bundle purchases with fee calculation, visitor occupancy, a per-shop blacklist and visit list, shop chat messages, and post-closure item/meso storage via Frederick (the hired merchant NPC).

Character shops close automatically when the owner disconnects. Hired merchants operate independently of the owner's session, expire after 24 hours, and store unsold items and accumulated mesos at Frederick for later retrieval; a tiered notification scheduler reminds owners to collect stored goods. The service also records item-search demand per world and exposes both a listing search and a top-searches hot list.

## External Dependencies

- **PostgreSQL** — shops, listings, messages, per-shop blacklists and visit lists, listing search counts, Frederick items/mesos/notifications, and the transactional outbox
- **Redis** — active-shop owner-occupancy registry, map placement index, and transient visitor tracking
- **Kafka** — command ingestion; merchant status/listing events, and compartment/character integration commands (published through a transactional outbox drainer)
- **OpenTelemetry** — distributed tracing via OTLP/gRPC
- **atlas-data** — portal position data for placement validation (outbound REST)

## Runtime Configuration

| Variable | Purpose |
|---|---|
| `REST_PORT` | HTTP server port |
| `BOOTSTRAP_SERVERS` | Kafka broker address |
| `COMMAND_TOPIC_MERCHANT` | Merchant command topic |
| `EVENT_TOPIC_MERCHANT_STATUS` | Merchant status event topic |
| `EVENT_TOPIC_MERCHANT_LISTING` | Merchant listing event topic |
| `COMMAND_TOPIC_COMPARTMENT` | Compartment (inventory) command topic |
| `EVENT_TOPIC_COMPARTMENT_STATUS` | Compartment status event topic |
| `COMMAND_TOPIC_CHARACTER` | Character command topic |
| `EVENT_TOPIC_CHARACTER_STATUS` | Character status event topic |
| `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME` | PostgreSQL connection |
| `REDIS_URL`, `REDIS_PASSWORD` | Redis connection |
| `ATLAS_ENV` | Redis key prefix |
| `DATA_SERVICE_URL` (falls back to `BASE_SERVICE_URL`) | Base URL for the atlas-data REST service |
| `TRACE_ENDPOINT` | OTLP trace collector endpoint |
| `LOG_LEVEL` | Logging level |

## Documentation

- [Domain](docs/domain.md)
- [Kafka](docs/kafka.md)
- [REST](docs/rest.md)
- [Storage](docs/storage.md)
</content>
