# Atlas Effective Stats Service

Computes and maintains effective (temporary) character statistics in real-time by aggregating base stats with bonuses from equipment, buffs, and passive skills.

## Responsibility

This service provides computed effective stats for logged-in characters. It maintains a Redis-backed registry of character stat models that are lazily initialized on first query or session creation and kept up to date via Kafka events for equipment changes, buff application/expiry, and base stat modifications. State is rebuilt from external services on demand.

## External Dependencies

- **Redis**: Tenant-scoped cache for character effective stats models
- **atlas-character**: Source of base character stats (STR, DEX, INT, LUK, MaxHP, MaxMP)
- **atlas-inventory**: Source of equipped item stats (flat asset fields in the equip compartment)
- **atlas-buffs**: Source of active buff stat changes
- **atlas-skills**: Source of character skill levels
- **atlas-data**: Source of skill effect data (passive skill bonuses)
- **Kafka cluster**: Event streaming for real-time updates and command publishing

## Runtime Configuration

| Variable | Description |
|----------|-------------|
| `REST_PORT` | HTTP server port |
| `BOOTSTRAP_SERVERS` | Kafka broker addresses |
| `TRACE_ENDPOINT` | OpenTelemetry trace exporter endpoint |
| `LOG_LEVEL` | Log level |

## Documentation

- [Domain](docs/domain.md)
- [REST API](docs/rest.md)
- [Kafka Integration](docs/kafka.md)
- [Storage](docs/storage.md)
