# Atlas Effective Stats Service

Computes and maintains effective (temporary) character statistics in real-time by aggregating base stats with bonuses from equipment, buffs, and passive skills.

## Responsibility

This service provides computed effective stats for logged-in characters. It maintains an in-memory registry of character stat models that are lazily initialized on first query and updated via Kafka events. The service does not persist data; state is rebuilt from external services on demand.

## External Dependencies

- **atlas-character**: Source of base character stats (STR, DEX, INT, LUK, MaxHP, MaxMP)
- **atlas-inventory**: Source of equipped item stats
- **atlas-buffs**: Source of active buff stat changes
- **atlas-skills**: Source of character skill levels
- **atlas-data**: Source of skill effect data (passive skill bonuses)
- **Kafka cluster**: Event streaming for real-time updates

## Runtime Configuration

| Variable | Description |
|----------|-------------|
| `REST_PORT` | HTTP server port (default: 8080) |
| `BOOTSTRAP_SERVERS` | Kafka broker addresses |
| `CHARACTERS_BASE_URL` | atlas-character service URL |
| `INVENTORY_BASE_URL` | atlas-inventory service URL |
| `BUFFS_BASE_URL` | atlas-buffs service URL |
| `SKILLS_BASE_URL` | atlas-skills service URL |
| `DATA_BASE_URL` | atlas-data service URL |
| `JAEGER_HOST` | Jaeger tracing endpoint |

## Documentation

- [Domain](docs/domain.md)
- [REST API](docs/rest.md)
- [Kafka Integration](docs/kafka.md)
