# Config-Status Projection Adoption — Design

Task: task-090-config-projection-adoption
Status: Approved design
Created: 2026-06-12
PRD: `./prd.md`

---

## 1. Problem & Goal

`atlas-character-factory` and `atlas-world` load per-tenant config once at boot via
a `sync.Once` REST fetch (`configuration.Init → requestAllTenants`), cache it in a
package-level `map[uuid.UUID]tenant.RestModel`, and **never refresh it**. The
lookup path then calls `log.Fatalf("tenant not configured")` when a tenant is
absent — so provisioning a new tenant *after* a pod started crash-loops the pod
(observed 2026-06-12, GMS v84.1 Evan seed).

`atlas-login`/`atlas-channel` already solved this with a Kafka-backed
`configuration/projection` package that consumes the log-compacted
`EVENT_TOPIC_CONFIGURATION_TENANT_STATUS` topic, gates readiness on a one-shot
end-offset catch-up, applies live add/change/tombstone, and serves reads through
an error-returning registry (never `Fatalf`).

This task **copy-ports the tenant subset** of that pattern into the factory and
world. We do not modify `atlas-configurations` (the producer) nor the
login/channel reference implementations. We do not extract a shared lib (explicit
PRD non-goal).

## 2. Approach Selection

Three approaches were weighed:

1. **Copy-port the tenant subset per service (chosen).** Matches the existing
   login/channel precedent and the PRD non-goal that defers shared-lib
   extraction. Each service owns a self-contained `configuration/projection`
   package plus a rewritten `configuration/registry.go`. Cost: ~4 small ported
   files duplicated across two services.
2. **Extract `libs/atlas-config-projection`.** Rejected — explicit PRD non-goal;
   would touch four services (login/channel/factory/world) and their
   service-specific `tenant.RestModel`/key types, turning a crash-fix into a
   cross-service refactor.
3. **Subscriber pushes a callback on each apply (event-driven bridge).** Rejected
   in favor of login's proven ticker-based republish — a push callback couples
   the subscriber goroutine to rate-init/registry side effects and complicates
   the catch-up barrier. The ticker bridge is simpler and already battle-tested
   in login.

The remaining design describes approach 1.

## 3. Architecture

Each service gains the same three-layer shape (already proven in login):

```
Kafka topic ──▶ projection.Subscriber ──▶ projection.State        (authoritative in-memory snapshot)
EVENT_TOPIC_       (1 consumer,            (RWMutex map,
 CONFIGURATION_     FirstOffset,            ApplyTenant /
 TENANT_STATUS)     Observe→CaughtUp)       ApplyTenantTombstone)
                                                  │
                                       configuration bridge loop   (post-catch-up ticker)
                                                  │  diff prev→next
                                   ┌──────────────┼─────────────────────┐
                                   ▼                                    ▼
                       configuration.PublishSnapshot           onChange hook (world only):
                       (read API consumers use)                re-run initializeRatesFromConfig
                                   │                            for changed/new tenants
                                   ▼
                       GetTenantConfig(id) → (cfg, error)
                       GetTenantConfigs()  → (map, error)   [world]
                       ErrNotReady | ErrTenantNotConfigured  (never Fatalf)
```

Readiness is a separate one-way gate: `caughtUp.CaughtUpNow() && !shuttingDown`,
exposed at `GET /readyz` and declared as a k8s `readinessProbe`.

### 3.1 Ported `configuration/projection` package (both services)

Copy-port the **tenant subset** of login's package. Four files, trimmed of the
service-config half:

- **`envelope.go`** — tenant-only. `TenantEnvelope{ SchemaVersion, Id, Config
  json.RawMessage, EmittedAt }`, `const SupportedSchemaVersion = 1` (held in
  lockstep with login — NFR backward-compat), `ErrUnsupportedSchema`,
  `IsTombstone(value []byte) bool`, `DecodeTenantEnvelope(value) (TenantEnvelope,
  error)` which rejects `schema_version > SupportedSchemaVersion`. The
  `ServiceEnvelope` type and service decoders are dropped (dead code for these
  services).
- **`caughtup.go`** — copied verbatim. Generic; the one-way `CaughtUp` gate
  (`SetEndOffsets`, `Observe`, `CaughtUpNow`, `WaitCaughtUp`) is service-agnostic.
  An empty/absent topic is trivially caught up.
- **`state.go`** — tenant-only. Drops `service *configuration.RestModel`,
  `ApplyService`, `ApplyServiceTombstone`. Keeps `tenants map[uuid.UUID]tenant.
  RestModel` under `sync.RWMutex`, `ApplyTenant(env)`, `ApplyTenantTombstone(id)`,
  and `Snapshot() map[uuid.UUID]tenant.RestModel` (returns a copy).
  **Deviation from login (resolves PRD Q4):** `ApplyTenant` sets
  `cfg.Id = env.Id` (the parsed uuid string) before storing. Login's
  `ApplyTenant` leaves `Id` empty because login keys only on the map; the old
  factory/world REST path keyed on `tc.Id` and populated it via JSON:API
  `SetID`, so we set it explicitly to keep the snapshot model byte-identical to
  the previously-loaded one. The envelope `config` payload round-trips into the
  factory/world `tenant.RestModel` via plain `json.Unmarshal` — proven by login,
  whose `tenant.RestModel` shares the same json tags (`Id` is `json:"-"`,
  populated out-of-band).
- **`subscriber.go`** — tenant-only. `Subscriber{ State, CaughtUp, TenantTopic
  string }`. `Start(ctx, l, wg, groupId)` snapshots the single tenant topic's end
  offsets (`offsetsOrEmpty` — empty on missing topic, never fatal), calls
  `CaughtUp.SetEndOffsets`, registers **one** `FirstOffset` consumer with
  `handleTenant`. `handleTenant` calls `CaughtUp.Observe(topic, partition,
  offset)` per message, then: tombstone (`IsTombstone` + key `tenant:<uuid>`) →
  `ApplyTenantTombstone(id)`; otherwise `DecodeTenantEnvelope` → `ApplyTenant`.
  Decode/apply failures and `ErrUnsupportedSchema` log at WARN and skip (FR-3,
  FR-4). The service-topic consumer and `handleService` are dropped (FR-5,
  FR-18 non-goal).

login's `apply.go` and `loop.go` are **not** ported — they drive a socket
`listener.Registry` that neither the factory nor world has.

### 3.2 Rewritten `configuration/registry.go` (both services)

Replace the `sync.Once` REST loader with login's error-returning, readiness-gated
read API (login `configuration/registry.go` is the template):

- Package vars: `configMu sync.RWMutex`, `tenantConfig map[uuid.UUID]tenant.
  RestModel`, `readyCh chan struct{}` (closed on first publish), `readyOnce`.
- `var ErrNotReady` (transient, pre-first-snapshot) and
  `var ErrTenantNotConfigured` (persistent, absent in a ready snapshot).
- `PublishSnapshot(tenants map[uuid.UUID]tenant.RestModel)` — copies the map under
  lock, closes `readyCh` once (FR-12).
- `GetTenantConfig(tenantId) (tenant.RestModel, error)` — waits briefly for first
  readiness; returns `ErrNotReady` before the first snapshot, the value if
  present, `ErrTenantNotConfigured` if absent. **No `log.Fatalf` on any path**
  (FR-10).
- Delete `Init`, the `sync.Once`, `requestAllTenants`, and the `configuration/
  requests.go` REST helper (no remaining callers; `preset_requests.go` keeps
  calling `GetTenantConfig`).

**world-only additions:**
- `GetTenantConfigs() (map[uuid.UUID]tenant.RestModel, error)` — returns
  `ErrNotReady` before the first snapshot, otherwise the (possibly empty) map. No
  `Fatalf` (FR-11).
- `initializeRatesFromConfig(l, tenantId, tc)` stays in the `configuration`
  package but is **no longer called from `Init`** (deleted) — it is invoked by
  the bridge's onChange hook and initial apply (§3.4). Its body is unchanged
  (reads `wc.GetExpRate()` etc., `InitWorldRates` = unconditional `Put`).

### 3.3 The bridge loop (`configuration/bridge.go`, both services)

A small per-service loop bridges `projection.State` → `configuration.
PublishSnapshot`, mirroring login's 1s republish ticker (login main.go
lines 178-189). Shape:

```
RunBridge(ctx, l, state, caughtUp, interval, onChange):
    caughtUp.WaitCaughtUp(ctx)            // block until catch-up; return on ctx done
    prev := {}
    publish := func():
        next := state.Snapshot()
        onChange(prev, next)              // diff-driven side effects
        configuration.PublishSnapshot(next)
        prev = next
    publish()                             // first publish immediately after catch-up
    ticker(interval):                     // default 1s
        publish()
```

- **factory** passes `onChange = nil` (no side effects — `PublishSnapshot` is
  idempotent; a blind republish is harmless). Resolves PRD Q3.
- **world** passes `onChange = reinitChangedRates`, which diffs `prev` vs `next`
  and calls `initializeRatesFromConfig` **only for tenants whose config changed
  or newly appeared** (compare by value; new key, or different `Worlds` rate
  payload). This is the user-confirmed semantic (PRD Q1): a genuine config change
  replaces that tenant's rates from config — clobbering any live `SetWorldRate`
  override — while unchanged tenants are left untouched so live overrides survive
  between config changes. Blind per-tick re-init is explicitly rejected (would
  stomp overrides every second). On the first post-catch-up pass `prev` is empty,
  so every caught-up tenant is treated as "new" and gets its rates initialized
  exactly as boot did today.

A tenant **tombstone** removes the key from the snapshot; `onChange` sees it
disappear from `next`. For v1 (PRD Q2) world performs **no active rate teardown**
— the stale rate entry is left in the registry (harmless; the tenant no longer
serves requests because `GetTenantConfig` now returns `ErrTenantNotConfigured`).
The diff logic MUST NOT panic on a removed key.

### 3.4 `main.go` wiring

**Common to both services** (replacing the `configuration.Init(...)` call):

1. `state := projection.NewState()`, `caughtUp := projection.NewCaughtUp()`.
2. Resolve `tenantTopic := os.Getenv("EVENT_TOPIC_CONFIGURATION_TENANT_STATUS")`;
   if empty, log a clear WARN ("tenant config updates will not propagate live")
   and continue (FR-5).
3. `sub := &projection.Subscriber{State: state, CaughtUp: caughtUp, TenantTopic:
   tenantTopic}`; `groupId := fmt.Sprintf("%s - projection - %s",
   consumerGroupId, uuid.New().String())` (per-replica replay, NFR startup
   ordering); `sub.Start(tdm.Context(), l, tdm.WaitGroup(), groupId)`.
4. Gate on catch-up: `ctx, cancel := context.WithTimeout(tdm.Context(),
   parseProjectionCatchupTimeout())`; `if err := caughtUp.WaitCaughtUp(ctx); err
   != nil { l.WithError(err).Fatal(...) }`. This mirrors login: a **startup**
   catch-up timeout fails loudly (no traffic served yet, k8s restarts) — distinct
   from the **request-time** crash this task eliminates. `parseProjectionCatchup
   Timeout` is ported from login (`PROJECTION_CATCHUP_TIMEOUT_S`, default 5m).
5. `var shuttingDown atomic.Bool`; `ready := func() bool { return caughtUp.
   CaughtUpNow() && !shuttingDown.Load() }`.
6. `tdm.TeardownFunc(func(){ shuttingDown.Store(true) })` registered **first** so
   `/readyz` flips not-ready before the rest of shutdown (FR-7).
7. `go configuration.RunBridge(tdm.Context(), l, state, caughtUp, time.Second,
   onChange)` (onChange nil for factory).
8. Mount readiness on the existing `server.New(l)` builder via the same
   atlas-rest readiness helper login uses (`MountReadiness("/readyz", ready)`),
   alongside the existing route initializers (FR-7).

**world-only**, additionally:
- The boot status sweep at `main.go:89`
  (`model.ForEachMap(model.FixedProvider(configuration.GetTenantConfigs()),
  channel.RequestStatus(l)(ctx))`) moves to **after** step 7's first publish, and
  `GetTenantConfigs()` now returns `(map, error)` — on `ErrNotReady`/error, log
  and skip the sweep (do not `Fatal`) (FR-16, FR-11). Because the bridge's first
  `publish()` runs synchronously inside `RunBridge` before its ticker, sequence
  the sweep after a `caughtUp.WaitCaughtUp` + an explicit initial publish (or run
  the sweep from inside the bridge's first pass) so it operates on a populated
  snapshot.
- `world/processor.go:78` already calls `GetTenantConfig(id) (tc, err)`; verify
  its error handling now tolerates `ErrTenantNotConfigured`/`ErrNotReady` by
  returning a request failure rather than assuming success.

**factory-only:** the seed/preset failure path is already correct —
`factory/processor.go:100-105` logs and returns the error from `GetTenantConfig`,
which `atlas-login` surfaces as an `AddCharacter` client error (FR-13, FR-14). No
call-site change beyond the registry now actually returning
`ErrTenantNotConfigured` instead of crashing.

## 4. Data Flow Examples

- **New tenant provisioned while factory runs:** configurations emits a tenant
  envelope → subscriber `ApplyTenant` inserts it into `State` → next bridge tick
  `PublishSnapshot` copies it into the registry → a seed request for that tenant
  now finds it. No restart, `RESTARTS` unchanged (PRD acceptance repro).
- **Tenant config change (world rates):** envelope replaces the tenant in `State`
  → bridge diff detects the changed `Worlds` payload → `initializeRatesFromConfig`
  re-`Put`s that world's rates → `PublishSnapshot`. Unchanged tenants untouched.
- **Tenant tombstone:** nil-value message, key `tenant:<uuid>` → `ApplyTenant
  Tombstone` deletes from `State` → bridge republishes without it →
  `GetTenantConfig` returns `ErrTenantNotConfigured` (no crash).

## 5. Error Handling

| Condition | Behavior |
|---|---|
| `schema_version > SupportedSchemaVersion` | WARN + skip (forward-compatible, FR-3) |
| Envelope decode / apply failure | WARN + skip; consumer continues (FR-4) |
| `EVENT_TOPIC_..._TENANT_STATUS` unset | WARN; empty topic → trivially caught up; serve degraded, never crash (FR-5) |
| Catch-up exceeds timeout | startup `Fatal` (no traffic served; k8s restarts), mirrors login (FR-9) |
| `GetTenantConfig` before first snapshot | `ErrNotReady` (caller DEBUG + skip, FR-10) |
| `GetTenantConfig` absent in ready snapshot | `ErrTenantNotConfigured` (caller ERROR + request failure, FR-10) |
| world `GetTenantConfigs` before snapshot | `ErrNotReady`; caller skips boot sweep (FR-11/16) |

No `log.Fatalf("tenant not configured")` remains in either service (acceptance
`grep` returns nothing).

## 6. Testing Strategy

- **Unit (ported from login's `projection_test.go`, tenant subset):**
  `State.ApplyTenant`/`ApplyTenantTombstone`/`Snapshot` (incl. `Id` is set);
  `CaughtUp` gate (empty topic trivially caught up; flips once consumed ≥ end-1;
  one-way); `DecodeTenantEnvelope` rejects high `schema_version`; `IsTombstone`.
- **Bridge diff (world):** `reinitChangedRates` re-inits only changed/new
  tenants; unchanged tenant's rates are not re-`Put` (assert via a fake/recording
  rate registry or `InitWorldRates` call count); removed tenant does not panic.
- **Registry:** `ErrNotReady` before publish; value after publish for present id;
  `ErrTenantNotConfigured` for absent id; world `GetTenantConfigs` error/empty
  paths; no `Fatalf` reachable.
- **Repro (documented manual, PRD acceptance):** with a factory pod running,
  provision a new GMS tenant; within seconds a seed for it succeeds (or fails
  gracefully on validation) with `RESTARTS` unchanged. Tombstone a tenant →
  subsequent request returns `ErrTenantNotConfigured`.
- Use the project Builder pattern for any model setup; no `*_testhelpers.go`.

## 7. Deployment & Config

- Add a `readinessProbe` targeting `GET /readyz` to
  `deploy/k8s/base/atlas-character-factory.yaml` and
  `deploy/k8s/base/atlas-world.yaml` (FR-8).
- `EVENT_TOPIC_CONFIGURATION_TENANT_STATUS` is already injected into both services
  via `envFrom: configMapRef: atlas-env` — no new env key for the topic.
- `PROJECTION_CATCHUP_TIMEOUT_S` is optional (default 5m); add only if a
  non-default is wanted.

## 8. Files Touched

**atlas-character-factory** (`services/atlas-character-factory/atlas.com/character-factory/`):
- `configuration/projection/{envelope,caughtup,state,subscriber}.go` — new (ported, tenant subset)
- `configuration/projection/projection_test.go` — new (ported)
- `configuration/registry.go` — rewrite (error-returning, readiness-gated; remove `Init`/`sync.Once`)
- `configuration/bridge.go` — new (republish loop, `onChange = nil`)
- `configuration/requests.go` — remove `requestAllTenants` (delete file if no other users)
- `main.go` — replace `Init` call with projection wiring + `/readyz` + shutdown not-ready
- `../../../deploy/k8s/base/atlas-character-factory.yaml` — `readinessProbe`

**atlas-world** (`services/atlas-world/atlas.com/world/`):
- `configuration/projection/{envelope,caughtup,state,subscriber}.go` + `projection_test.go` — new (ported)
- `configuration/registry.go` — rewrite; keep `initializeRatesFromConfig` (called from bridge); `GetTenantConfigs` returns `(map, error)`; remove `Init`/`sync.Once`
- `configuration/bridge.go` — new (diff loop, `onChange = reinitChangedRates`)
- `configuration/requests.go` — remove `requestAllTenants`
- `main.go` — projection wiring; sequence `main.go:89` sweep after catch-up; handle `GetTenantConfigs` error
- `world/processor.go` — confirm `GetTenantConfig` error handling tolerates new errors
- `../../../deploy/k8s/base/atlas-world.yaml` — `readinessProbe`

**No change:** `atlas-configurations`, `atlas-login`, `atlas-channel`.

## 9. Verification (CLAUDE.md Build & Verification)

Both services' `go.mod` are touched, so all are mandatory:
- `go test -race ./...`, `go vet ./...`, `go build ./...` clean in both modules.
- `tools/redis-key-guard.sh` clean from repo root.
- `docker buildx bake atlas-character-factory` and `docker buildx bake
  atlas-world` succeed from the worktree root.

## 10. Resolved Open Questions

- **Q1 (world rates on change):** Diff loop, **config wins on change** — re-init
  rates only for changed/new tenants; a config change clobbers live overrides for
  that tenant, unchanged tenants keep theirs. (User-confirmed.)
- **Q2 (world tombstone):** No active teardown for v1 — stop serving via
  `ErrTenantNotConfigured`; stale rate entry left in place; must not crash.
- **Q3 (bridge shape):** Ticker-based republish loop (mirrors login). factory =
  blind republish; world = diff-driven with rate-reinit `onChange`.
- **Q4 (Id population):** `ApplyTenant` explicitly sets `cfg.Id = env.Id`;
  payload round-trips via plain `json.Unmarshal` (proven by login). This is the
  one intentional deviation from the login source.
