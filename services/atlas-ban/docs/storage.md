# Storage

## Tables

### bans

| Column | Type | Constraints |
|--------|------|-------------|
| tenant_id | uuid | NOT NULL |
| id | uint32 | PRIMARY KEY, AUTO INCREMENT, NOT NULL |
| ban_type | byte | NOT NULL |
| value | string | NOT NULL |
| reason | string | NOT NULL, DEFAULT '' |
| reason_code | byte | NOT NULL, DEFAULT 0 |
| permanent | bool | NOT NULL, DEFAULT false |
| expires_at | time.Time | NOT NULL |
| issued_by | string | NOT NULL, DEFAULT '' |
| created_at | time.Time | GORM managed |
| updated_at | time.Time | GORM managed |

### login_history

| Column | Type | Constraints |
|--------|------|-------------|
| tenant_id | uuid | NOT NULL |
| id | uint64 | PRIMARY KEY, AUTO INCREMENT, NOT NULL |
| account_id | uint32 | NOT NULL |
| account_name | string | NOT NULL |
| ip_address | string | NOT NULL, DEFAULT '' |
| hwid | string | NOT NULL, DEFAULT '' |
| success | bool | NOT NULL, DEFAULT false |
| failure_reason | string | NOT NULL, DEFAULT '' |
| created_at | time.Time | GORM managed |

## Relationships

None.

## Indexes

### bans

Primary key on `id` column (auto-generated).

### login_history

| Index | Column |
|-------|--------|
| idx_login_history_account_id | account_id |
| idx_login_history_ip_address | ip_address |
| idx_login_history_hwid | hwid |
| idx_login_history_created_at | created_at |

## Migration Rules

- Migration is performed via GORM AutoMigrate on Entity structs for both bans and login_history tables
- Schema changes are applied automatically on service startup
