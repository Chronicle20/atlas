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
| expiresAt | time.Time | Expiration time (if not permanent) |
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
| ExpireBan | Expire a temporary ban early |
| ExpireBanAndEmit | Expire ban and emit status event |
| GetById | Retrieve ban by ID |
| ByIdProvider | Provider for ban by ID |
| AllProvider | Provider for paged bans for tenant |
| ByTypePagedProvider | Provider for paged bans filtered by type |
| CheckBan | Check if IP, HWID, or account is banned |

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
| ByAccountIdProvider | Provider for paged history by account ID |
| ByIPPagedProvider | Provider for paged history by IP address |
| ByHWIDPagedProvider | Provider for paged history by hardware ID |
| AllProvider | Provider for paged history for tenant |
| PurgeOlderThan | Remove records older than specified days |

### HistoryPurge

Background task that periodically removes login history records older than 90 days.

---

# Report Domain

## Responsibility

The report domain persists player-submitted reports against another player — `sue` (in-game report of general misconduct) and `claim` (chat-log-corroborated report submitted through the claim UI). A report snapshots the reporter and accused identity, a reason code, an optional description, and — for `claim` reports — the client-submitted chat log plus a best-effort server-captured transcript, so GMs can review a report without depending on data that may since have changed or expired.

## Core Models

### Model

Immutable domain representation of a report.

| Field | Type | Description |
|-------|------|-------------|
| id | uuid.UUID | Report identifier (surrogate, generated in Go at create time — never a business-value PK) |
| tenantId | uuid.UUID | Tenant identifier |
| kind | Kind | `sue` or `claim` |
| reporterId | uint32 | Character ID of the reporter |
| reporterName | string | Character name of the reporter |
| accusedId | uint32 | Character ID of the accused |
| accusedName | string | Character name of the accused |
| reasonType | byte | Client-supplied reason code |
| description | string | Reporter-supplied description, capped at 2000 characters (runes) |
| chatLog | *string | Client-submitted chat log for `claim` reports; nil for `sue` reports, capped at 16384 bytes |
| serverTranscript | []TranscriptLine | Server-captured chat lines involving reporter and accused, snapshotted at creation; nil when atlas-messages was unreachable (best-effort corroboration, not a required field) |
| status | Status | `open`, `reviewed`, or `actioned` |
| createdAt | time.Time | Creation timestamp |
| updatedAt | time.Time | Last update timestamp |

### Kind

| Value | Name | Description |
|-------|------|-------------|
| "sue" | KindSue | In-game general-misconduct report |
| "claim" | KindClaim | Chat-log-corroborated report submitted through the claim UI |

### Status

| Value | Name | Description |
|-------|------|-------------|
| "open" | StatusOpen | Newly created, not yet reviewed |
| "reviewed" | StatusReviewed | A GM has looked at it |
| "actioned" | StatusActioned | A GM has taken action on it |

### TranscriptLine

One server-captured chat line attached to a report's `serverTranscript`.

| Field | Type | Description |
|-------|------|-------------|
| timestamp | int64 | Unix-milli capture time |
| senderId | uint32 | Character ID of the line's author |
| senderName | string | Character name of the line's author |
| chatType | string | Chat channel/type the line was captured from |
| text | string | Line content |

## Invariants

- Kind must be `sue` or `claim`; Status must be `open`, `reviewed`, or `actioned`
- The accused must resolve to a real character in the tenant (by id or by name) or creation is rejected with `NOT_FOUND`, never persisted
- A reporter may create at most `MaxClaimsPerWindow` (100) `claim` reports whose `createdAt` falls at or after a rolling `ClaimQuotaWindow` (7 days) before now; exceeding the cap rejects creation with `QUOTA_EXCEEDED` before the accused is resolved. `sue` reports are not subject to this cap.
- Description is truncated (never rejected) at 2000 runes; the cut always lands on a full rune so the stored value is valid UTF-8
- ChatLog is truncated (never rejected) at 16384 bytes; the cut walks rune-by-rune so it never splits a multi-byte sequence
- ServerTranscript is best-effort: an atlas-messages outage persists the report with a nil transcript rather than failing the report
- A report's status transitions are not otherwise constrained (no enforced state machine beyond the three valid values)

## Processors

### Processor

Primary domain processor providing report operations.

| Method | Description |
|--------|-------------|
| CreateFromCommand | Check the reporter's claim quota (for `claim` kind), resolve reporter/accused, snapshot the chat transcript, persist the report, and buffer exactly one status event (CREATED or ERROR) |
| CreateFromCommandAndEmit | CreateFromCommand and emit the buffered event |
| UpdateStatus | Update a report's status by ID |
| GetById | Retrieve a report by ID |
| ByIdProvider | Provider for a report by ID |
| GetByTenant | Retrieve all reports for the tenant |
| GetByStatus | Retrieve reports filtered by status |

Reporter/accused resolution goes through `atlas-ban`'s `character` REST client; the server-captured transcript goes through the `chat` REST client (`atlas-messages` `/api/chat/history` — see that service's docs for the exposure caveat on this endpoint).
