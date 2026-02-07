# Ban Domain

## Responsibility

The ban domain manages ban records for IP addresses, hardware IDs (HWID), and account IDs. It supports permanent and temporary bans with optional expiration, CIDR range matching for IP bans, and periodic cleanup of expired bans.

## Core Models

### Model

Immutable domain representation of a ban.

| Field | Type | Description |
|-------|------|-------------|
| tenantId | uuid.UUID | Tenant identifier |
| id | uint32 | Ban identifier |
| banType | BanType | Type of ban |
| value | string | Banned value (IP, HWID, or account ID) |
| reason | string | Ban reason |
| reasonCode | byte | Ban reason code |
| permanent | bool | Whether the ban is permanent |
| expiresAt | int64 | Unix timestamp of expiration (if not permanent) |
| issuedBy | string | Issuer of the ban |
| createdAt | time.Time | Creation timestamp |
| updatedAt | time.Time | Last update timestamp |

### BanType

Ban type enumeration.

| Value | Name | Description |
|-------|------|-------------|
| 0 | BanTypeIP | IP address ban |
| 1 | BanTypeHWID | Hardware ID ban |
| 2 | BanTypeAccount | Account ID ban |

## Invariants

- Value is required and cannot be empty
- A temporary ban is expired when the current time exceeds expiresAt
- A permanent ban never expires
- IP bans support both exact match and CIDR range matching
- CIDR range bans are checked against all active IP bans
- Ban checks evaluate in order: exact IP, CIDR IP, HWID, account
- Account bans store the account ID as a string value

## Processors

### Processor

Primary domain processor providing ban operations.

| Method | Description |
|--------|-------------|
| Create | Create a new ban |
| CreateAndEmit | Create ban and emit status event |
| Delete | Delete a ban by ID |
| DeleteAndEmit | Delete ban and emit status event |
| GetById | Retrieve ban by ID |
| GetByTenant | Retrieve all bans for tenant |
| GetByType | Retrieve bans filtered by type |
| CheckBan | Check if IP, HWID, or account is banned |
| ByIdProvider | Provider for ban by ID |

### ExpiredBanCleanup

Background task that periodically removes expired temporary bans. Operates across all tenants in a single global sweep rather than per-tenant.

---

# History Domain

## Responsibility

The history domain records login attempts from account session events. It tracks successful and failed logins with associated IP addresses, hardware IDs, and failure reasons. Records are automatically purged after a configurable retention period.

## Core Models

### Model

Immutable domain representation of a login history entry.

| Field | Type | Description |
|-------|------|-------------|
| tenantId | uuid.UUID | Tenant identifier |
| id | uint64 | History entry identifier |
| accountId | uint32 | Account identifier |
| accountName | string | Account name |
| ipAddress | string | IP address of login attempt |
| hwid | string | Hardware ID of login attempt |
| success | bool | Whether login succeeded |
| failureReason | string | Reason for failure (if failed) |
| createdAt | time.Time | Timestamp of login attempt |

## Invariants

- AccountId is required and cannot be zero
- Retention period is 90 days (RetentionDays constant)

## Processors

### Processor

Primary domain processor providing login history operations.

| Method | Description |
|--------|-------------|
| Record | Record a login attempt |
| GetByAccountId | Retrieve history by account ID |
| GetByIP | Retrieve history by IP address |
| GetByHWID | Retrieve history by hardware ID |
| GetByTenant | Retrieve all history for tenant |
| PurgeOlderThan | Remove records older than specified days |

### HistoryPurge

Background task that periodically removes login history records older than 90 days.
