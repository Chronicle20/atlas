# Storage

This service uses Redis for state storage. There are no database tables or migrations.

## Tables

None.

## Redis Key Structures

| Namespace | Key Type | Value Type | Description |
|-----------|----------|------------|-------------|
| buffs | uint32 (characterId) | character.Model (JSON) | Active buffs per character, keyed by characterId within tenant |
| buffs-poison | uint32 (characterId) | time.Time (JSON) | Last poison tick timestamp per character, keyed by characterId within tenant |
| atlas:buffs:_tenants | Set | tenant.Model (JSON) | Set of tenants with active buff data |

## Relationships

None.

## Indexes

None.

## Migration Rules

None.
