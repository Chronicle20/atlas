# Multi-Version Tenant Provisioning — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-06-10
---

## 1. Overview

Atlas runs each game version as a *tenant* (`region` + `majorVersion` + `minorVersion`). The
`atlas-login` and `atlas-channel` services are multi-tenant per pod: each tenant gets its own
TCP socket listener and its own per-tenant Kafka consumers, bound from a single **`services`
configuration document** (stored in `atlas-configurations`) whose `tenants[]` array lists, for
each tenant, an `{id, port}` (login) or `{id, ipAddress, worlds[...]}` (channel) entry.

Running N versions side-by-side in one environment therefore requires two hand-maintained
places to agree, with no single source of truth tying them together:

1. the login/channel **`services` config `tenants[]`** list (drives socket bind + consumer
   registration), and
2. the **k8s LoadBalancer / Deployment port set** (`deploy/k8s/base/atlas-login.yaml`,
   `atlas-channel.yaml`) that exposes each version's port externally.

In ephemeral PR environments this actively breaks. `atlas-pr-bootstrap` is single-tenant: it
creates one canonical tenant from `REGION`/`MAJOR_VERSION`/`MINOR_VERSION` env, then on **every
(re)run rebuilds the `services` config from a static canonical template and PATCHes it back**,
overwriting the live `tenants[]` to contain only that canonical tenant. When a second version
(e.g. GMS v84 next to canonical v83) is added by hand via the web UI, the next bootstrap
reconciles it away. The dropped tenant's socket listener can linger long enough to accept a
connection and read the initial login packet, but its per-tenant consumers — the ones that
answer account-session events (license agreement, auth result) — are drained. The client
connects, logs in, then hangs with no server response. The failure is version-specific and
intermittent: whichever tenant won the last config write works; the other silently stops
responding.

This task makes N versions coexist durably in one environment (ephemeral and persistent)
without manual, clobberable edits, by (a) making the bootstrap **additive/idempotent** instead
of clobbering, (b) **deriving every port from the version** via one shared formula so the bind
side and the LB side cannot disagree, and (c) **generating the k8s LB/Deployment port set from
a single declared version list** at build time. PR #711 already backfilled the current LB ports
by hand (gms-84 8400/8401, gms-92 9201, gms-95 9501); this task removes the need to do that by
hand going forward.

## 2. Goals

Primary goals:

- The `atlas-pr-bootstrap` service-config step is **additive and idempotent**: it upserts its
  own canonical tenant entry into the live `services` config `tenants[]` (keyed by tenant id)
  and preserves every other entry untouched, across arbitrary re-runs.
- A second (third, Nth) co-resident version, once present in the `services` config, **survives
  every subsequent bootstrap run** and keeps both its socket listener and its per-tenant
  consumers.
- **Ports are derived from the version** by a single shared formula (`login = major×100`,
  `channel = login + 1`) wherever a port is written or exposed: bootstrap upsert, services
  config, and LB generation all consume the same derivation, so the bind side and the LB side
  are consistent by construction.
- The k8s LB/Deployment port set for `atlas-login` and `atlas-channel` is **generated at build
  time from one declared version set** rather than hand-edited per version.
- Adding a version to an environment is a **single declared-list edit + redeploy**; no
  per-version hand edits to two must-agree places.
- A documented operator runbook covers adding/removing a version end to end.

Non-goals:

- Changing the wire/packet layer or any version's packet structure.
- Changing the handler/writer opcode configuration projection internals (the per-tenant
  handler/writer config), beyond what the listener/consumer lifecycle requires.
- Adding new game-version *content* (maps, items, scripts) for any version.
- Rewriting the template-clone flow that creates a tenant's per-tenant configuration.
- Making login/channel consume `atlas-tenants`' `tenant.status` directly, or making
  `atlas-configurations` auto-project the `services.tenants[]` list from the registry. (These
  were considered and explicitly deferred in favor of the additive-bootstrap approach.)
- A runtime port-reconciler controller that patches the LB live from the tenant registry
  (deferred in favor of the build-time generator).
- Region/version filtering on the `atlas-tenants` REST API.

## 3. User Stories

- As a **platform operator**, I want to add a new version to a running environment by editing
  one declared list and redeploying, so that I don't have to hand-edit two places that can
  silently disagree.
- As a **PR author**, I want my ephemeral PR environment's canonical version to come up
  correctly **and** any additional version I add by hand to keep working across bootstrap
  re-runs, so that multi-version testing isn't randomly broken.
- As an **operator debugging a hung login**, I want co-resident versions to never drain each
  other's consumers, so that "client logs in then hangs" stops being a provisioning artifact.
- As a **developer adding a version**, I want the LB/Deployment ports generated from the same
  version list and port formula the services use, so that I can't forget the second place.

## 4. Functional Requirements

Organized by capability area. Requirements are testable.

### 4.1 Port derivation (shared formula)

- **FR-1.1** There is exactly one canonical derivation of a version's ports:
  `loginPort = majorVersion × 100`, `channelPort = loginPort + 1`. (Matches the existing
  convention and PR #711.)
- **FR-1.2** Every producer of a port value consumes this single derivation: the bootstrap
  upsert (FR-2), and the LB generator (FR-3). No port is independently hardcoded in more than
  one place.
- **FR-1.3** The derivation is a function of `majorVersion` only; `minorVersion` and `region`
  do not change the port. (Co-residency of two tenants that share a `majorVersion` is out of
  scope and MAY be rejected/flagged by tooling, but is not a supported configuration.)

### 4.2 Additive / idempotent bootstrap (`atlas-pr-bootstrap`)

- **FR-2.1** The service-config step MUST NOT rebuild the live `services` config `tenants[]`
  from a static template and overwrite it. It MUST read the current live `services` config,
  upsert its canonical tenant entry, and write back the **merged** list.
- **FR-2.2** The upsert is **keyed by tenant id**: the entry whose `id` equals the resolved
  canonical tenant UUID is created-or-updated; every entry with a different `id` is left
  byte-for-byte untouched.
- **FR-2.3** The canonical tenant's port in the upserted entry is **version-derived** (FR-1),
  computed from the bootstrap's `MAJOR_VERSION` env — not the fixed 8300/8301 baked into the
  canonical template. (Today bootstrap binds 8300/8301 regardless of the PR's actual version.)
- **FR-2.4** For the channel service, the upserted entry's `ipAddress` is set to the discovered
  LoadBalancer IP, as today; other entries' `ipAddress` values are preserved.
- **FR-2.5** The step is **idempotent**: running bootstrap twice with the same inputs produces
  the same live `services` config and emits no spurious config-status churn on the second run
  (the existing "skip PATCH when attributes already match" guard MUST continue to hold for the
  merged result).
- **FR-2.6** The behavior applies to both the login and channel `services` documents. The
  tenant-agnostic `drops` service config is unaffected (no `tenants[]`).
- **FR-2.7** If the live `services` config does not yet exist (first run), bootstrap creates it
  containing the canonical tenant entry (POST path), as today.

### 4.3 LB / Deployment port generation (build-time)

- **FR-3.1** There is a single **declared version set** (e.g. a `versions.json` / version
  manifest, or an addition to the existing `.github/config/services.json` single-source-of-truth
  ecosystem) enumerating the versions an environment exposes (e.g. `[83, 84, 87, 92, 95, 185]`
  with region/minor as needed).
- **FR-3.2** A build-time generator consumes the declared version set and the port derivation
  (FR-1) to emit the `containerPorts` and `Service.ports` entries for `atlas-login.yaml` and
  `atlas-channel.yaml`, following the existing naming convention
  (`atlas-login-<region>-<major><minor>`).
- **FR-3.3** The generated manifests reproduce the current hand-maintained port set (the post
  PR-#711 state) exactly for the existing versions — i.e. introducing the generator is a no-op
  diff against today's checked-in ports for `[12, 83, 84, 87, 92, 95, 185]`.
- **FR-3.4** Adding a version to the declared set, regenerating, and redeploying is sufficient
  to expose that version's login+channel ports; no other manifest edit is required.
- **FR-3.5** The generator (or a CI check) fails the build if the checked-in manifests drift
  from what the declared version set would generate, so the two cannot silently diverge.

### 4.4 Coexistence correctness

- **FR-4.1** With two versions present in the `services` config (e.g. v83 + v84), both tenants'
  socket listeners bind on their derived ports AND both tenants' per-tenant Kafka consumers are
  registered and remain registered after a bootstrap re-run.
- **FR-4.2** A bootstrap re-run that targets the canonical version MUST NOT drain, drop, or
  reset the consumers or listener of any other co-resident version (the root-cause failure mode).
- **FR-4.3** Removing a tenant from the `services` config drains exactly that tenant's listener
  and consumers and leaves others intact (existing projection behavior; verified, not changed).

### 4.5 Operator workflow

- **FR-5.1** The supported flow to add a version to an environment is: add the version to the
  one declared version set, then redeploy. Bootstrap (additive upsert), the derived ports, and
  the LB generator reconcile the rest.
- **FR-5.2** A runbook documents the full operator procedure for both ephemeral and persistent
  environments, including how a hand-added second version in an ephemeral env now survives
  bootstrap, and the safe restart/verify sequence (watch for `projection.applied op=add`).

## 5. API Surface

No new or modified REST endpoints are required by this task.

- `atlas-pr-bootstrap` continues to call the existing `atlas-configurations` services-config
  API (`GET/POST/PATCH /api/configurations/services[/{serviceId}]`); only the request body
  construction changes (read-merge-write instead of template-rewrite). Note the known
  `atlas-configurations` PATCH panic on tenant-agnostic configs
  (`reflect.Value.Set using unaddressable value`) is worked around today by the "skip no-op
  PATCH" guard; the merged-body path MUST preserve that guard.
- `atlas-tenants` `GET /api/tenants` remains the authoritative tenant registry but is **not**
  newly consumed by login/channel as part of this task (non-goal).

## 6. Data Model

No database schema changes.

- The `services` configuration document (atlas-configurations, keyed by service UUID) is the
  mutated artifact; its `tenants[]` shape is unchanged — login `{id, port}`, channel
  `{id, ipAddress, worlds[{channels[{id, port}]}]}`. Only **how** bootstrap mutates it changes.
- A new **declared version set** artifact is introduced for LB generation (file under
  `.github/config/` or `deploy/`, format TBD in design). It is build/deploy-time config, not a
  runtime DB entity.
- Port values are derived, not stored as an independent source: `major×100` (+1 channel).

## 7. Service Impact

- **atlas-pr-bootstrap** (`scripts/bootstrap.sh`): rework `upsert_service_config` from
  template-rewrite-and-overwrite to read-live → upsert-by-id → write-merged; derive the
  canonical port from `MAJOR_VERSION`; preserve other entries' `ipAddress`; keep the no-op-PATCH
  idempotency guard. Canonical service templates (`canonical/services/{login,channel}-service.json`)
  may need the fixed port replaced by a derived value.
- **deploy/k8s/base** (`atlas-login.yaml`, `atlas-channel.yaml`): `containerPorts` and
  `Service.ports` become generator output from the declared version set instead of hand-edited
  lists. New generator script/kustomize step + CI drift check.
- **.github/config** (or `deploy/`): new declared version-set manifest; possibly extend
  `services.json` ecosystem / schema.
- **atlas-login** / **atlas-channel**: no code change expected — the projection already binds
  per-tenant listeners and registers per-tenant consumers from the `services` config and
  reconciles adds/drops. This task makes the *inputs* stable; coexistence behavior must be
  **verified** (FR-4) but is not re-architected. (If verification surfaces a real
  drain-on-rewrite bug in the projection itself, that is in scope to fix.)
- **Docs**: `docs/onboarding.md`, `docs/runbooks/ephemeral-pr-deployments.md` updated with the
  new add-a-version workflow and the additive-bootstrap guarantee.

## 8. Non-Functional Requirements

- **Multi-tenancy**: all changes preserve tenant isolation; per-tenant consumers and listeners
  remain keyed by tenant id (channel: tenant/world/channel triple).
- **Idempotency**: bootstrap is safe to run arbitrarily many times; convergent, no churn on
  re-run (FR-2.5).
- **Determinism**: LB generation is pure-function of the declared version set + port formula;
  same inputs → identical manifests (FR-3.3, FR-3.5).
- **Observability**: adding/removing a version produces clear, greppable signals
  (`projection.applied op=add`/`op=drain`); the runbook references them.
- **Backward compatibility**: introducing the generator is a no-op diff against the current
  checked-in ports; existing single-tenant ephemeral environments behave identically.
- **Security**: no new external surface; LB exposes only derived per-version ports as today.

## 9. Open Questions

- **Declared version-set location/format**: standalone `versions.json` vs. extending
  `.github/config/services.json` (+ its JSON schema) vs. a `deploy/`-local manifest. Region and
  `minorVersion` representation in that set. (Design phase.)
- **Generator mechanism**: kustomize generator/transformer vs. a script that writes the base
  YAML vs. a patch overlay. Where the CI drift check (FR-3.5) lives. (Design phase.)
- **Canonical template port**: whether to keep a placeholder in
  `canonical/services/*-service.json` and substitute at bootstrap time, vs. construct the
  tenant entry entirely in `bootstrap.sh`. (Design phase.)
- **Same-major coexistence**: behavior if two tenants share a `majorVersion` (port collision) —
  reject, warn, or document as unsupported. (FR-1.3; confirm in design.)
- **Persistent-env tenant creation**: confirm the persistent-environment path for *creating*
  the tenant row + per-tenant config (UI template-clone) and how the declared version-set edit
  relates to it, so "one declared list + redeploy" is coherent end-to-end.

## 10. Acceptance Criteria

- [ ] A single port-derivation formula (`major×100`, `+1` channel) exists and is the sole
      source consumed by both bootstrap and LB generation (FR-1).
- [ ] `atlas-pr-bootstrap` upserts its canonical tenant into the live `services` config keyed by
      tenant id, preserving all other `tenants[]` entries, for both login and channel (FR-2.1,
      FR-2.2, FR-2.6).
- [ ] The canonical tenant's port is version-derived from `MAJOR_VERSION`, not hardcoded
      8300/8301 (FR-2.3).
- [ ] Bootstrap is idempotent: a second identical run leaves the live `services` config and
      config-status stream unchanged (FR-2.5).
- [ ] Integration/repro: with v83 (canonical) + v84 (hand-added) present, a bootstrap re-run
      leaves v84's listener AND consumers intact; a v84 client can complete the login handshake
      without hanging (FR-4.1, FR-4.2).
- [ ] A declared version set drives a build-time generator that emits the login+channel
      `containerPorts` and `Service.ports` (FR-3.1, FR-3.2).
- [ ] Generating against the current version set produces a no-op diff vs. the post-#711
      checked-in manifests (FR-3.3).
- [ ] A CI/build check fails on drift between the declared version set and the checked-in port
      manifests (FR-3.5).
- [ ] Adding a version = one declared-list edit + redeploy, with no other manifest or config
      hand-edit, documented in the runbook (FR-5).
- [ ] `docs/onboarding.md` and `docs/runbooks/ephemeral-pr-deployments.md` updated.
- [ ] `go test -race ./...`, `go vet ./...`, `go build ./...` clean in any changed Go module;
      `docker buildx bake` for any service whose `go.mod` changed; `tools/redis-key-guard.sh`
      clean. (Expected: no Go module changes; bootstrap is shell + k8s/CI config.)
