# Storage

## Tables

### documents

Stores all game data documents as JSON blobs with tenant isolation.

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PRIMARY KEY, DEFAULT uuid_generate_v4() | Unique document identifier |
| tenant_id | UUID | NOT NULL | Tenant identifier for multi-tenancy |
| type | VARCHAR | NOT NULL | Document type discriminator |
| document_id | INTEGER (uint32) | NOT NULL | Domain-specific document identifier |
| content | JSON | NOT NULL | JSON representation of the document |

## Relationships

The documents table stores all data types in a single table with the `type` column discriminating between document types.

Document types:
- CASH
- CHARACTER_TEMPLATE
- COMMODITY
- CONSUMABLE
- EQUIPMENT
- ETC
- FACE
- HAIR
- ITEM_STRING
- MAP
- MONSTER
- NPC
- PET
- QUEST
- REACTOR
- SETUP
- SKILL

## Indexes

Indexes are managed by GORM AutoMigrate based on the entity definition.

## Migration Rules

- Migrations are executed via GORM AutoMigrate
- The `documents` table is created automatically on service startup
- Schema changes are applied incrementally by GORM
