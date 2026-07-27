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

---

### tenant_history

Stores historical snapshots of tenant configurations.

| Column | Type | Constraints |
|--------|------|-------------|
| id | uuid | default uuid_generate_v4() |
| tenant_id | uuid | |
| data | json | not null |
| created_at | timestamp | not null |

---

### services

Stores service configurations.

| Column | Type | Constraints |
|--------|------|-------------|
| id | uuid | default uuid_generate_v4() |
| type | varchar | not null |
| data | json | not null |

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

---

### outbox_entries

Stores the transactional outbox rows used to publish service and tenant configuration change events to Kafka.

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

No other indexes are explicitly defined. GORM AutoMigrate creates default indexes.

## Migration Rules

Migrations are executed via GORM AutoMigrate on startup in the following order:
1. templates
2. tenants and tenant_history
3. services and service_history
4. outbox_entries
