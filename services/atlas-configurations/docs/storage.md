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
| id | uuid | |
| type | varchar | |
| data | json | not null |

## Relationships

- `tenant_history.tenant_id` references `tenants.id`

## Indexes

None explicitly defined. GORM AutoMigrate creates default indexes.

## Migration Rules

Migrations are executed via GORM AutoMigrate on startup in the following order:
1. templates
2. tenants and tenant_history
3. services
