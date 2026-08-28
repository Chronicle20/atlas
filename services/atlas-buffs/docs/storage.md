# Storage

This service uses Redis for state storage. There are no database tables or migrations.

## Tables

None.

## Redis Key Structures

| Namespace | Key Type | Value Type | Description |
|-----------|----------|------------|-------------|
| buffs | uint32 (characterId) | character.Model (JSON) | Active buffs per character, keyed by characterId within tenant |
| buffs-tick | TickKey (characterId:statType) | time.Time (JSON), TTL 5 minutes | Last periodic-effect tick timestamp per (character, statType) within tenant |
| buffs-berserk | uint32 (characterId) | berserk.Model (JSON) | Tracked Dark Knight Berserk state per character, keyed by characterId within tenant |
| atlas:buffs:_tenants | Set | tenant.Model (JSON) | Set of tenants with active buff or berserk-tracking data; shared by the buff and berserk registries |

## Relationships

None.

## Indexes

None.

## Migration Rules

None.
