# Connection Pool Configuration — Tasks

Last Updated: 2026-02-19

## Phase 1: Add Pool Configuration

### Standard Services (defaults: MaxOpen=10, MaxIdle=5, Lifetime=5m, IdleTime=3m)

- [ ] atlas-account
- [ ] atlas-ban
- [ ] atlas-buddies
- [ ] atlas-cashshop
- [ ] atlas-character
- [ ] atlas-configurations
- [ ] atlas-drop-information (database.go)
- [ ] atlas-fame
- [ ] atlas-families
- [ ] atlas-gachapons (database.go)
- [ ] atlas-guilds
- [ ] atlas-inventory
- [ ] atlas-keys
- [ ] atlas-map-actions
- [ ] atlas-maps
- [ ] atlas-marriages
- [ ] atlas-notes
- [ ] atlas-npc-conversations
- [ ] atlas-npc-shops
- [ ] atlas-party-quests
- [ ] atlas-pets
- [ ] atlas-portal-actions
- [ ] atlas-quest
- [ ] atlas-reactor-actions
- [ ] atlas-saga-orchestrator
- [ ] atlas-skills
- [ ] atlas-storage
- [ ] atlas-tenants

### Special: atlas-data (defaults: MaxOpen=30, MaxIdle=10, Lifetime=5m, IdleTime=3m)

- [ ] atlas-data — replace hardcoded values with env var pattern, add ConnMaxLifetime/ConnMaxIdleTime

## Phase 2: Verification

- [ ] Build all 27 services
- [ ] Run tests for all 27 services
- [ ] Verify no regressions (pool settings are harmless on SQLite test DBs)

## Phase 3: Documentation

- [ ] Update `docs/architectural-improvements.md` — mark "Medium: No Connection Pool Configuration" as RESOLVED
