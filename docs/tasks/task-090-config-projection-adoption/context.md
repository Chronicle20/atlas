# Context — task-090-config-projection-adoption

Companion to `plan.md`. Summarizes the key files, decisions, and dependencies an
engineer needs before executing the plan. Read `prd.md` and `design.md` for the
full rationale.

## Goal in one line

Replace the crash-prone one-shot REST config load in `atlas-character-factory`
and `atlas-world` with the Kafka-backed `configuration/projection` pattern already
proven in `atlas-login`, so a tenant provisioned *after* a pod started is picked
up live and a missing tenant returns an error instead of `log.Fatalf`-crashing the
pod.

## Reference implementation (copy-port source)

`services/atlas-login/atlas.com/login/configuration/`:

- `projection/envelope.go` — `ServiceEnvelope` + `TenantEnvelope` alias,
  `SupportedSchemaVersion = 1`, `ErrUnsupportedSchema`, `IsTombstone`,
  `DecodeServiceEnvelope`/`DecodeTenantEnvelope`. **Port the tenant subset only**
  (drop `ServiceEnvelope`).
- `projection/caughtup.go` — generic one-way readiness gate. **Copy verbatim.**
- `projection/state.go` — service+tenant snapshot. **Port the tenant subset only**
  (drop `service`, `ApplyService`, `ApplyServiceTombstone`).
- `projection/subscriber.go` — two consumers. **Port the tenant consumer only**
  (drop `ServiceTopic`, `ServiceId`, `handleService`).
- `projection/apply.go`, `projection/loop.go` — **NOT ported** (they drive a
  socket `listener.Registry` that neither service has).
- `registry.go` — error-returning, readiness-gated read API. **Template for the
  rewrite**, with one deviation (see below).
- `main.go:80-189` — projection wiring, catch-up gate, republish ticker,
  `MountReadiness("/readyz", ready)`, shutting-down teardown order,
  `parseProjectionCatchupTimeout`. **Wiring template.**

## Critical deviations from the login source (do not skip these)

1. **`tenant.RestModel.Id` is a `string` with `json:"-"`** in both factory and
   world (`configuration/tenant/rest.go`). The envelope `config` payload does NOT
   carry `id` in a way that unmarshals into this field. So `State.ApplyTenant`
   MUST set `cfg.Id = env.Id` (the envelope's id string) after `json.Unmarshal`,
   before storing. This is design Q4. Login's tenant `RestModel.Id` is a
   `uuid.UUID` populated differently — do not copy login's `ApplyTenant` blindly.

2. **No service config in these services.** Login's `PublishSnapshot(svc, tenants)`
   closes `readyCh` only `if svc != nil`. Factory/world `PublishSnapshot(tenants)`
   takes tenants only and MUST close `readyCh` (via `readyOnce`) on the **first
   call unconditionally**. Drop `serviceConfig`/`GetServiceConfig` entirely.

3. **World `main.go` ordering.** Today `configuration.Init` runs at
   `main.go:86` — AFTER `server.New(...).Run()` (line 74-83). `Run()` is
   non-blocking (it registers the HTTP server on the teardown WaitGroup and
   returns; `tdm.Wait()` blocks). The projection subscriber + `caughtUp` must be
   created BEFORE `server.New(...).Run()` so `MountReadiness("/readyz", ready)`
   can close over `caughtUp`; the catch-up wait, first publish, and boot status
   sweep (`main.go:89`) move to AFTER `Run()` returns but operate on the populated
   snapshot. See plan Task W7 for the exact ordering.

## Files touched

### atlas-character-factory (`services/atlas-character-factory/atlas.com/character-factory/`)
- `configuration/projection/envelope.go` — NEW (tenant subset)
- `configuration/projection/caughtup.go` — NEW (verbatim copy)
- `configuration/projection/state.go` — NEW (tenant subset, `Id` set)
- `configuration/projection/subscriber.go` — NEW (tenant consumer only)
- `configuration/projection/projection_test.go` — NEW (ported tenant tests)
- `configuration/registry.go` — REWRITE (error-returning, readiness-gated; no `Init`/`sync.Once`)
- `configuration/registry_test.go` — NEW (ErrNotReady/value/ErrTenantNotConfigured)
- `configuration/bridge.go` — NEW (republish ticker, `onChange = nil`)
- `configuration/requests.go` — DELETE (only `requestAllTenants`, now unused)
- `main.go` — replace `configuration.Init(...)` with projection wiring + `/readyz`
- `../../../deploy/k8s/base/atlas-character-factory.yaml` — add `readinessProbe`

### atlas-world (`services/atlas-world/atlas.com/world/`)
- `configuration/projection/{envelope,caughtup,state,subscriber}.go` — NEW
- `configuration/projection/projection_test.go` — NEW
- `configuration/registry.go` — REWRITE; keep `initializeRatesFromConfig` (now called from bridge); `GetTenantConfigs()` returns `(map, error)`; no `Init`/`sync.Once`
- `configuration/registry_test.go` — NEW
- `configuration/bridge.go` — NEW (diff loop, `onChange = reinitChangedRates`)
- `configuration/bridge_test.go` — NEW (diff re-inits only changed/new; removed tenant no panic)
- `configuration/requests.go` — DELETE
- `main.go` — projection wiring; sequence `main.go:89` sweep after catch-up; handle `GetTenantConfigs` error
- `world/processor.go:78` — already returns provider error on `GetTenantConfig` error; verify, no change expected
- `../../../deploy/k8s/base/atlas-world.yaml` — add `readinessProbe`

### No change
`atlas-configurations` (producer), `atlas-login`, `atlas-channel` (reference impls).

## Key dependencies / helpers (already exist)
- `<svc>/kafka/consumer.LookupBrokers()` — broker list (both services have it).
- `consumer.GetManager().AddConsumer(l, ctx, wg)` / `.RegisterHandler` — shared
  atlas-kafka manager.
- `consumer.ReadEndOffsets(ctx, brokers, topic)` (`libs/atlas-kafka/consumer/offsets.go`).
- `consumer.NewConfig`, `consumer.SetHeaderParsers`, `consumer.SetStartOffset`,
  `kafka.FirstOffset`.
- `server.MountReadiness(path, func() bool)` (`libs/atlas-rest/server/server.go:34`).
- `service.GetTeardownManager()` → `tdm.Context()`, `tdm.WaitGroup()`, `tdm.TeardownFunc()`.

## Env / config
- `EVENT_TOPIC_CONFIGURATION_TENANT_STATUS` — already injected into both services
  via `envFrom: configMapRef: atlas-env`. No new topic env key.
- `PROJECTION_CATCHUP_TIMEOUT_S` — optional, default 5m (port `parseProjectionCatchupTimeout`).

## Build & Verification (CLAUDE.md — both go.mods touched)
1. `go test -race ./...` clean in both modules.
2. `go vet ./...` clean in both modules.
3. `go build ./...` clean in both modules.
4. `docker buildx bake atlas-character-factory` and `docker buildx bake atlas-world`
   from the worktree root.
5. `tools/redis-key-guard.sh` clean from repo root.
6. Acceptance `grep`: no `log.Fatalf("tenant not configured")` in either service.

## Resolved open questions (from design §10)
- Q1: World rates — diff loop, **config wins on change**; re-init only changed/new
  tenants; unchanged tenants keep live overrides.
- Q2: Tenant tombstone — no active rate teardown for v1; stop serving via
  `ErrTenantNotConfigured`; must not panic on removed key.
- Q3: Bridge — ticker republish (mirrors login). Factory blind republish; world
  diff-driven with rate-reinit `onChange`.
- Q4: `ApplyTenant` sets `cfg.Id = env.Id` explicitly.
