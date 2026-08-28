# Trade Storage

This service uses PostgreSQL for escrow custody, the in-flight settlement
record, and the completed-trade ledger, plus the shared transactional outbox
(`libs/atlas-outbox`). The live trade room registry is process-local
in-memory state and is not persisted.

## Tables

### trade_escrow_items

One staged asset held in trade custody.

| Column | Type | Description |
|--------|------|-------------|
| id | uuid, PK | Escrow row id; also the staging saga's transaction id. |
| tenant_id, tenant_region, tenant_major, tenant_minor | | Tenant, denormalised for startup reconciliation with no tenant in context. |
| room_id | uuid | Owning trade room. Not a foreign key — rooms are in-memory and do not survive a restart. |
| owner_id | | Staging character. |
| trade_slot | byte | Client dialog slot. |
| source_inventory_type | | Provenance only; not replayed on return. |
| source_slot | | Provenance only. |
| asset_id | | Source asset identity. |
| template_id, quantity | | Staged item and staged amount. |
| expiration, cash_id, rechargeable | | Cash/timed-item snapshot fields. |
| strength, dexterity, intelligence, luck, hp, mp, weapon_attack, magic_attack, weapon_defense, magic_defense, accuracy, avoidability, hands, speed, jump, slots | | Equip stat snapshot. |
| level_type, level, experience, hammers_applied, flags, owner | | Additional item-state snapshot fields. |
| pet_id, pet_name, pet_level, closeness, fullness | | Pet snapshot fields. |
| returning_at | *timestamp | Single-claimant latch: non-null while a `trade_unwind` has claimed the exclusive right to return this row. |
| returning_tx_id | *uuid, indexed | The `trade_unwind` transaction that holds the claim; lets a failed unwind release exactly the rows it took. |
| created_at | timestamp | |
| deleted_at | timestamp, soft-delete, indexed | Set on a normal release (restorable); a spurious accept is hard-deleted instead. |

Unique index on `(tenant_id, id)`. Composite indexes on `(tenant_id, room_id)`
and `(tenant_id, owner_id)`.

### trade_escrow_mesos

One participant's escrowed meso for one room.

| Column | Type | Description |
|--------|------|-------------|
| id | uuid, PK | |
| tenant_id, tenant_region, tenant_major, tenant_minor | | Tenant, denormalised. |
| room_id | uuid | |
| owner_id | | |
| amount | int64, signed | The CONFIRMED escrowed total: the sum of stake deltas that have actually landed. Transiently negative depending on stake resolution order; never assigned directly. |
| created_at, updated_at | timestamp | |

Unique index on `(tenant_id, id)` and on `(tenant_id, room_id, owner_id)`.
The row is REPLACED on each stage rather than accumulated.

### trade_escrow_meso_stakes

One in-flight `award_mesos` debit/credit against a participant's escrow row.
More than one row can exist per `(room_id, owner_id)` at once.

| Column | Type | Description |
|--------|------|-------------|
| id | uuid, PK | The stakeId the saga was submitted with (not a surrogate — resolution looks it up by this value). |
| tenant_id, tenant_region, tenant_major, tenant_minor | | Tenant, denormalised. |
| room_id | | |
| owner_id | | |
| amount | uint32 | Absolute total the player typed for this stake. |
| delta | int32, signed | Signed movement this stake submitted; the only safe basis for refunding an orphaned stake. |
| created_at | timestamp | |

Composite index on `(tenant_id, room_id, owner_id)`.

### trade_escrow_meso_refunds

What one `trade_unwind` took from a participant's escrowed meso, so a failed
unwind can put it back. Rows are removed on the unwind's terminal state
(restored on failure, discarded on success).

| Column | Type | Description |
|--------|------|-------------|
| id | uuid, PK | |
| transaction_id | uuid, indexed | The `trade_unwind` transaction this refund record belongs to. |
| tenant_id, tenant_region, tenant_major, tenant_minor | | Tenant, denormalised. |
| room_id | | |
| owner_id | | |
| amount | int64, signed | Exactly what the unwind took. |
| created_at | timestamp | |

### trade_settlements

One in-flight settlement, submitted alongside the settlement saga and
deleted once its terminal status has been handled.

| Column | Type | Description |
|--------|------|-------------|
| id | uuid, PK | |
| tenant_id, tenant_region, tenant_major, tenant_minor | | Tenant, denormalised for startup reconciliation with no tenant in context. |
| transaction_id | uuid | The settlement saga's transaction id. |
| room_id, handle, room_type | | Denormalised room identity — the room itself is gone by the time a restart requires this record. |
| world_id, channel_id, map_id, instance | | Denormalised field. |
| owner_id, visitor_id | | |
| submitted_at | timestamp, indexed | |

Unique index on `(tenant_id, transaction_id)`.

### trade_settlement_sides

One participant's contribution, with the tax split already resolved (frozen
against a later tenant configuration change).

| Column | Type | Description |
|--------|------|-------------|
| id | uuid, PK | |
| tenant_id | indexed | |
| entry_id | uuid, indexed, FK -> trade_settlements.id | |
| position, character_id, character_name | | |
| meso_staged, meso_tax, meso_delivered | uint32 | |

### trade_settlement_items

One escrowed asset carried into the settlement payload.

| Column | Type | Description |
|--------|------|-------------|
| id | uuid, PK | |
| tenant_id | indexed | |
| side_id | uuid, indexed, FK -> trade_settlement_sides.id | |
| escrow_id | uuid | The custody row this settlement releases from. |
| inventory_type, source_slot | | Provenance, recorded not replayed. |
| asset_id, template_id, quantity | | |

### trade_ledger_entries

One settled trade. Immutable, never updated or deleted.

| Column | Type | Description |
|--------|------|-------------|
| id | uuid, PK | |
| tenant_id | indexed | |
| transaction_id | | The settlement saga's transaction id. |
| world_id, channel_id, map_id | | |
| room_type | byte | |
| settled_at | timestamp, indexed | |

Unique index on `(tenant_id, transaction_id)` — the write-side idempotency
guard: a duplicate settle for the same settlement saga cannot produce a
second row.

### trade_ledger_sides

Exactly two rows per entry. `character_name` is denormalised because names
change and the ledger is a point-in-time record.

| Column | Type | Description |
|--------|------|-------------|
| id | uuid, PK | |
| tenant_id | | Composite index with character_id. |
| entry_id | uuid, indexed, FK -> trade_ledger_entries.id | |
| character_id | | Composite index with tenant_id. |
| character_name | | |
| meso_staged, meso_tax, meso_delivered | uint32 | |

Composite index `(tenant_id, character_id)`.

### trade_ledger_items

One asset a side gave. `asset_id`/`reference_id` are nullable — present only
for identity-bearing assets (equips, pets, cash).

| Column | Type | Description |
|--------|------|-------------|
| id | uuid, PK | |
| tenant_id | indexed | |
| side_id | uuid, indexed, FK -> trade_ledger_sides.id | |
| item_id, quantity | | |
| asset_id | *asset.Id | Nullable. |
| reference_id | *uint32 | Nullable. |

## Relationships

- `trade_escrow_items` and `trade_escrow_mesos`/`trade_escrow_meso_stakes`
  are related only through `(room_id, owner_id)`; there is no foreign key,
  since a room does not persist as a database entity.
- `trade_settlement_sides` -> `trade_settlements` and
  `trade_settlement_items` -> `trade_settlement_sides` are `foreignKey`
  relations (GORM `has many`).
- `trade_ledger_sides` -> `trade_ledger_entries` and `trade_ledger_items` ->
  `trade_ledger_sides` are `foreignKey` relations (GORM `has many`).
- `trade_escrow_meso_refunds.transaction_id` names the `trade_unwind` saga
  that produced it; there is no foreign key to a saga table (sagas are owned
  by atlas-saga-orchestrator).

## Indexes

| Table | Index |
|-------|-------|
| trade_escrow_items | unique `(tenant_id, id)`; `(tenant_id, room_id)`; `(tenant_id, owner_id)`; `returning_tx_id`; `deleted_at` |
| trade_escrow_mesos | unique `(tenant_id, id)`; unique `(tenant_id, room_id, owner_id)` |
| trade_escrow_meso_stakes | `(tenant_id, room_id, owner_id)` |
| trade_escrow_meso_refunds | `transaction_id` |
| trade_settlements | unique `(tenant_id, transaction_id)`; `tenant_id`; `submitted_at` |
| trade_settlement_sides | `tenant_id`; `entry_id`; `character_id` |
| trade_settlement_items | `tenant_id`; `side_id` |
| trade_ledger_entries | unique `(tenant_id, transaction_id)`; `tenant_id`; `settled_at` |
| trade_ledger_sides | `entry_id`; composite `(tenant_id, character_id)` |
| trade_ledger_items | `tenant_id`; `side_id` |

## Migration Rules

- Every table is created via `db.AutoMigrate`, run at boot for the ledger,
  settlement, and escrow tables plus the shared outbox table.
- The escrow migration additionally backfills and drops two retired column
  groups: a legacy single-slot in-flight meso stake (`pending_stake_id`,
  `pending_amount`, `pending_delta` on `trade_escrow_mesos`) is copied into
  `trade_escrow_meso_stakes` before the columns are dropped, and a set of
  never-populated or renamed columns on `trade_escrow_items` (`ring_id`,
  `item_level`, `item_exp`, `vicious_count`) is dropped outright. Both drops
  are conditional on the column still existing, so a re-run on an
  already-migrated database is a no-op.
- Ledger and settlement rows are immutable once written; the ledger tables
  have no update or delete path at all. Settlement rows are deleted (never
  updated) once resolved. Escrow rows are soft-deleted on release and
  hard-deleted only for a spurious accept.
