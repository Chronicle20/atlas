# Connection Pool Configuration Plan

Last Updated: 2026-02-19

## Executive Summary

27 Atlas services use copy-pasted `database/connection.go` files with no connection pool settings. Go's `database/sql` defaults apply: unlimited open connections, 2 max idle, and no connection lifetime. With 25 services running 2 replicas each (plus atlas-data at 4 replicas), the PostgreSQL instance faces up to 54+ concurrent connection sources with no upper bound on connections per source. This plan adds sensible pool defaults to every service's `Connect()` function, with optional environment variable overrides for per-service tuning.

## Current State Analysis

### Connection Code Pattern

All 27 services follow an identical copy-pasted pattern in `database/connection.go`:

```go
func Connect(l logrus.FieldLogger, configurators ...Configurator) *gorm.DB {
    // ... DSN from env vars ...
    db, err = gorm.Open(postgres.Open(dsnBuilder.Build()), &gorm.Config{})
    // ... retry loop, migrations ...
    return db
}
```

No call to `db.DB()` to access the underlying `*sql.DB` and configure pool settings.

### Effective Defaults (Go `database/sql`)

| Setting | Default | Effect |
|---------|---------|--------|
| `MaxOpenConns` | 0 (unlimited) | No cap on simultaneous connections to PostgreSQL |
| `MaxIdleConns` | 2 | Only 2 connections kept warm; rest closed after use |
| `ConnMaxLifetime` | 0 (forever) | Connections never recycled; stale connections accumulate |
| `ConnMaxIdleTime` | 0 (forever) | Idle connections never cleaned up |

### Exception: atlas-data

`atlas-data` is the only service with pool settings (4 replicas):

```go
sqlDB.SetMaxOpenConns(30)
sqlDB.SetMaxIdleConns(10)
```

This is appropriate for atlas-data which handles heavy read traffic (WZ data extraction, game data queries). It already sets 30 max open / 10 max idle, but still lacks `ConnMaxLifetime`.

### Deployment Scale

| Replicas | Services | Total Instances |
|----------|----------|----------------|
| 4 | atlas-data | 4 |
| 2 | 22 services (account, ban, buddies, etc.) | 44 |
| 1 | 5 services (character, monsters, maps, cashshop, pets) | 5 |
| **Total** | **27 with DB** | **53 instances** |

### Worst Case: Connection Pressure

With `MaxOpenConns=0` (unlimited), each of the 53 instances can open connections without bound. Under load spikes or goroutine leaks, PostgreSQL can be overwhelmed. The `MaxIdleConns=2` default means most connections are opened and immediately closed, adding TCP overhead and preventing connection reuse.

### Services Without Database (no changes needed)

Services that don't use `database/connection.go`: atlas-login, atlas-channel, atlas-world, atlas-assets, atlas-wz-extractor, atlas-asset-expiration, atlas-query-aggregator, atlas-ui, atlas-redis, atlas-monster-death, atlas-messages.

## Proposed Future State

### Design: Sensible Defaults with Env Var Overrides

Add pool configuration directly inside each service's `Connect()` function after `gorm.Open()`:

```go
sqlDB, err := db.DB()
if err != nil {
    return true, err
}

sqlDB.SetMaxOpenConns(getIntEnv("DB_MAX_OPEN_CONNS", 10))
sqlDB.SetMaxIdleConns(getIntEnv("DB_MAX_IDLE_CONNS", 5))
sqlDB.SetConnMaxLifetime(getDurationEnv("DB_CONN_MAX_LIFETIME", 5 * time.Minute))
sqlDB.SetConnMaxIdleTime(getDurationEnv("DB_CONN_MAX_IDLE_TIME", 3 * time.Minute))
```

### Default Values

| Setting | Default | Rationale |
|---------|---------|-----------|
| `MaxOpenConns` | 10 | Conservative. 53 instances × 10 = 530 max connections. PostgreSQL default `max_connections=100` may need tuning, but most services won't hit 10 concurrent. |
| `MaxIdleConns` | 5 | Half of max open. Keeps warm connections ready without wasting resources. |
| `ConnMaxLifetime` | 5 minutes | Prevents stale connections after PostgreSQL restarts, network changes, or DNS updates. |
| `ConnMaxIdleTime` | 3 minutes | Cleans up idle connections faster than lifetime, reducing idle resource usage. |

### Environment Variable Overrides

Three new optional env vars per service, read from the atlas-env ConfigMap or per-service deployment YAML:

| Variable | Type | Default | Example |
|----------|------|---------|---------|
| `DB_MAX_OPEN_CONNS` | int | 10 | `"30"` for atlas-data |
| `DB_MAX_IDLE_CONNS` | int | 5 | `"10"` for atlas-data |
| `DB_CONN_MAX_LIFETIME` | duration string | `"5m"` | `"10m"` |
| `DB_CONN_MAX_IDLE_TIME` | duration string | `"3m"` | `"5m"` |

### Design Decisions

1. **Inline in each service (not a shared library)** — Extracting to a shared library is tracked separately as "Low: Duplicated Database/REST Boilerplate" in `docs/architectural-improvements.md`. This change is minimal and mechanical — add ~15 lines per service. A shared library would be a larger refactor with its own plan.

2. **Helper functions for env var parsing** — Add `getIntEnv()` and `getDurationEnv()` helpers to each `connection.go`. These are 5-line functions, not worth a shared dependency.

3. **Defaults over ConfigMap** — Defaults are compiled in. No ConfigMap changes needed unless a service needs a custom value. This follows the existing pattern (DB_HOST/DB_PORT are in ConfigMap, but DB_NAME is per-service).

4. **atlas-data special case** — Update atlas-data to use the same env var pattern but with higher defaults (30/10) to preserve its current behavior while adding `ConnMaxLifetime` and `ConnMaxIdleTime`.

### PostgreSQL Consideration

With `MaxOpenConns=10` across 53 instances, worst-case concurrent connections is 530. PostgreSQL's default `max_connections=100` will need to be verified/increased. This is a one-time config check, not a code change.

## Implementation Phases

### Phase 1: Add Pool Configuration to All Services

**Scope:** Modify `database/connection.go` in all 27 services to add pool settings with env var overrides.

**Pattern for standard services (26 services):**

Add after `gorm.Open()` inside the `tryToConnect` function:

```go
sqlDB, err := db.DB()
if err != nil {
    return true, err
}

sqlDB.SetMaxOpenConns(getIntEnv("DB_MAX_OPEN_CONNS", 10))
sqlDB.SetMaxIdleConns(getIntEnv("DB_MAX_IDLE_CONNS", 5))
sqlDB.SetConnMaxLifetime(getDurationEnv("DB_CONN_MAX_LIFETIME", 5*time.Minute))
sqlDB.SetConnMaxIdleTime(getDurationEnv("DB_CONN_MAX_IDLE_TIME", 3*time.Minute))
```

Add helper functions:

```go
func getIntEnv(key string, defaultVal int) int {
    if v, ok := os.LookupEnv(key); ok {
        if i, err := strconv.Atoi(v); err == nil {
            return i
        }
    }
    return defaultVal
}

func getDurationEnv(key string, defaultVal time.Duration) time.Duration {
    if v, ok := os.LookupEnv(key); ok {
        if d, err := time.ParseDuration(v); err == nil {
            return d
        }
    }
    return defaultVal
}
```

**Pattern for atlas-data (preserves existing 30/10):**

Replace hardcoded values with env var overrides using higher defaults:

```go
sqlDB.SetMaxOpenConns(getIntEnv("DB_MAX_OPEN_CONNS", 30))
sqlDB.SetMaxIdleConns(getIntEnv("DB_MAX_IDLE_CONNS", 10))
sqlDB.SetConnMaxLifetime(getDurationEnv("DB_CONN_MAX_LIFETIME", 5*time.Minute))
sqlDB.SetConnMaxIdleTime(getDurationEnv("DB_CONN_MAX_IDLE_TIME", 3*time.Minute))
```

### Phase 2: Verification

- Build all 27 services
- Run tests for all 27 services (tests use SQLite in-memory, so pool settings are harmless)
- Verify PostgreSQL `max_connections` is sufficient for the deployment

### Phase 3: Documentation

- Update `docs/architectural-improvements.md` to mark this issue as RESOLVED
- Log pool configuration details

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| MaxOpenConns too low for a specific service under load | Low | Medium — queries queue waiting for a connection | Env var override per-service without code change |
| PostgreSQL max_connections exceeded | Medium | High — connection refused errors | Verify/increase PostgreSQL config before deploying |
| ConnMaxLifetime causes brief latency spike when connections recycle | Very Low | Low — new connection takes ~1ms | 5-minute lifetime spreads reconnections over time |
| Typo in mechanical edit breaks a service | Low | Low — caught by build/test | Build and test every service |

## Success Metrics

- All 27 services have explicit pool configuration
- No service uses Go defaults (unlimited open, 2 idle, forever lifetime)
- All services build and pass tests
- Environment variable overrides work for per-service tuning
- Zero additional ConfigMap/deployment YAML changes needed for defaults

## Required Resources and Dependencies

- **PostgreSQL config access** — Verify `max_connections` is adequate (recommend ≥600)
- **No new library dependencies** — Uses standard library `time.ParseDuration` and `strconv.Atoi`
- **No Kubernetes changes** — Defaults are compiled in; env vars are optional overrides

## Timeline Estimates

| Phase | Effort | Description |
|-------|--------|-------------|
| Phase 1 | S | Mechanical edit to 27 files — same pattern repeated |
| Phase 2 | S | Build + test all services |
| Phase 3 | S | Update architectural-improvements.md |
| **Total** | **S** | Single session |
