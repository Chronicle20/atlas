# Storage

## Tables

### templates

Stores configuration templates.

| Column | Type | Constraints |
|--------|------|-------------|
| id | uuid | default uuid_generate_v4() |
| region | varchar | not null |
| major_version | smallint | not null |
| minor_version | smallint | not null |
| data | json | not null |
| environment | varchar | not null, default `''` |

---

### tenants

Stores tenant configurations.

| Column | Type | Constraints |
|--------|------|-------------|
| id | uuid | default uuid_generate_v4() |
| region | varchar | not null |
| major_version | smallint | not null |
| minor_version | smallint | not null |
| data | json | not null |
| environment | varchar | not null, default `''` |

---

### tenant_history

Stores historical snapshots of tenant configurations.

| Column | Type | Constraints |
|--------|------|-------------|
| id | uuid | default uuid_generate_v4() |
| tenant_id | uuid | |
| data | json | not null |
| created_at | timestamp | not null |
| environment | varchar | not null, default `''` |

---

### services

Stores service configurations.

| Column | Type | Constraints |
|--------|------|-------------|
| id | uuid | default uuid_generate_v4() |
| type | varchar | not null |
| data | json | not null |
| environment | varchar | not null, default `''` |

---

### service_history

Stores historical snapshots of service configurations.

| Column | Type | Constraints |
|--------|------|-------------|
| id | uuid | default uuid_generate_v4() |
| service_id | uuid | |
| type | varchar | not null |
| data | json | not null |
| created_at | timestamp | not null |
| environment | varchar | not null, default `''` |

---

### environments

Stores the list of execution environments.

| Column | Type | Constraints |
|--------|------|-------------|
| id | uuid | default uuid_generate_v4() |
| name | varchar | not null, unique |
| baseline | varchar | not null |
| namespace | varchar | not null |
| tenant | varchar | not null, default `''` |
| overrides | json | not null |
| phase | varchar | not null |

---

### outbox_entries

Stores the transactional outbox rows used to publish service, tenant, and environment configuration change events to Kafka.

| Column | Type | Constraints |
|--------|------|-------------|
| id | uint64 | primary key |
| topic | varchar | not null |
| message_key | bytea | not null |
| message_value | bytea | |
| headers | json | not null, default `{}` |
| enqueued_at | timestamp | not null, default current_timestamp |
| sent_at | timestamp | nullable |
| attempts | int | not null, default 0 |
| last_error | varchar | nullable |

## Relationships

- `tenant_history.tenant_id` references `tenants.id`
- `service_history.service_id` references `services.id`

## Indexes

- `outbox_entries_unsent_idx` on `outbox_entries.topic` where `sent_at IS NULL`
- `outbox_entries_sweeper_idx` on `outbox_entries.sent_at` where `sent_at IS NOT NULL`
- `idx_services_type_env` unique index on `services (type, environment)`

No other indexes are explicitly defined. GORM AutoMigrate creates default indexes.

## Migration Rules

Migrations are executed in the following order on startup:
1. templates
2. tenants and tenant_history
3. services and service_history
4. outbox_entries
5. environments
6. environment column backfill: every row in tenants, tenant_history, templates, services, and service_history with an empty or null `environment` is set to the baseline environment (`ATLAS_ENVIRONMENT` env var, default `main`); idempotent
7. servicesuniq: any (type, environment) group in `services` holding more than one row is deduplicated (extra rows deleted), then the `idx_services_type_env` unique index is created; must run after the environment column backfill
