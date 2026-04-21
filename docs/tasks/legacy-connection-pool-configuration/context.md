# Connection Pool Configuration — Context

Last Updated: 2026-02-19

## Key Files

### Services with `database/connection.go` (26 standard services)

| Service | File |
|---------|------|
| atlas-account | `services/atlas-account/atlas.com/account/database/connection.go` |
| atlas-ban | `services/atlas-ban/atlas.com/ban/database/connection.go` |
| atlas-buddies | `services/atlas-buddies/atlas.com/buddies/database/connection.go` |
| atlas-cashshop | `services/atlas-cashshop/atlas.com/cashshop/database/connection.go` |
| atlas-character | `services/atlas-character/atlas.com/character/database/connection.go` |
| atlas-configurations | `services/atlas-configurations/atlas.com/configurations/database/connection.go` |
| atlas-data | `services/atlas-data/atlas.com/data/database/connection.go` |
| atlas-fame | `services/atlas-fame/atlas.com/fame/database/connection.go` |
| atlas-families | `services/atlas-families/atlas.com/family/database/connection.go` |
| atlas-guilds | `services/atlas-guilds/atlas.com/guilds/database/connection.go` |
| atlas-inventory | `services/atlas-inventory/atlas.com/inventory/database/connection.go` |
| atlas-keys | `services/atlas-keys/atlas.com/keys/database/connection.go` |
| atlas-map-actions | `services/atlas-map-actions/atlas.com/map-actions/database/connection.go` |
| atlas-maps | `services/atlas-maps/atlas.com/maps/database/connection.go` |
| atlas-marriages | `services/atlas-marriages/atlas.com/marriages/database/connection.go` |
| atlas-notes | `services/atlas-notes/atlas.com/notes/database/connection.go` |
| atlas-npc-conversations | `services/atlas-npc-conversations/atlas.com/npc/database/connection.go` |
| atlas-npc-shops | `services/atlas-npc-shops/atlas.com/npc/database/connection.go` |
| atlas-party-quests | `services/atlas-party-quests/atlas.com/party-quests/database/connection.go` |
| atlas-pets | `services/atlas-pets/atlas.com/pets/database/connection.go` |
| atlas-portal-actions | `services/atlas-portal-actions/atlas.com/portal/database/connection.go` |
| atlas-quest | `services/atlas-quest/atlas.com/quest/database/connection.go` |
| atlas-reactor-actions | `services/atlas-reactor-actions/atlas.com/reactor/database/connection.go` |
| atlas-saga-orchestrator | `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/database/connection.go` |
| atlas-skills | `services/atlas-skills/atlas.com/skills/database/connection.go` |
| atlas-storage | `services/atlas-storage/atlas.com/storage/database/connection.go` |
| atlas-tenants | `services/atlas-tenants/atlas.com/tenants/database/connection.go` |

### Special files (different naming)

| Service | File |
|---------|------|
| atlas-drop-information | `services/atlas-drop-information/atlas.com/dis/database/database.go` |
| atlas-gachapons | `services/atlas-gachapons/atlas.com/gachapons/database/database.go` |

### Documentation

- `docs/architectural-improvements.md` — Issue tracking (update to RESOLVED when done)

### Kubernetes Config

- `services/atlas-env.yaml` — Shared ConfigMap (add env vars here only if overriding for all services)
- Per-service deployment YAML — Add env var overrides for specific services (e.g., atlas-data)

## Key Decisions

1. **Inline modification, not shared library** — Each service gets its own pool config. Shared library extraction is a separate, larger initiative.
2. **Compiled-in defaults with env var overrides** — Matches the HTTP client timeout pattern from `libs/atlas-rest`.
3. **atlas-data retains higher limits** — Default 30/10 (vs 10/5 for standard services) to preserve current behavior.
4. **ConnMaxLifetime added everywhere** — Including atlas-data which currently lacks it.

## Dependencies

- No blocking dependencies. All changes are internal to each service's `database/` package.
- PostgreSQL `max_connections` should be verified but is not a blocker for code changes.

## Related Work

- "Low: Duplicated Database/REST Boilerplate" — Future shared library would centralize this. Current work is intentionally minimal.
- HTTP Client Timeouts (RESOLVED) — Established the env var override pattern we follow here.
