# Risks — task-016 Kafka Consumer Self-Healing + Visibility

Enumerates technical and operational risks, likelihood, and proposed mitigations. These are the items a reviewer should push on before implementation starts, and the items the implementer should verify during `/dev-docs` plan drafting.

---

## R1. Rebalance storm during broker instability

**Risk.** Each reader recreation forces a fresh consumer-group join, which triggers a rebalance on the broker side. If the broker is flapping (intermittently reachable), the recreate loop will drive rebalances at the backoff cadence. Across 49 pods using this library, a coordinated broker hiccup could produce a rebalance burst.

**Likelihood.** Low in the current dev-cluster deployment (1 pod per service per consumer group → rebalances are trivially cheap). Medium-to-high if atlas deploys multi-replica services sharing consumer groups in the future.

**Mitigation.** Capped exponential backoff (500ms → 10s) bounds the rebalance rate per consumer at ≥1 per 10s at worst. At the monorepo scale (49 consumers × 10s cap) that is ~5 rebalances/second across the fleet in a worst-case storm, which single-broker Kafka tolerates. Revisit if we add replicas per group — add jitter to backoff at that point.

**Indicator to watch post-deploy.** `recreateCount` on the debug route increasing without a corresponding broker outage signal. If we observe recreate churn against a healthy broker, the fetch loop may be misclassifying a recoverable transient as a fatal error.

---

## R2. Mid-commit reader swap races

**Risk.** `processMessage` (manager.go:199–203) calls `c.reader.CommitMessages(...)` after a successful handler run. If the outer recreate loop swaps `c.reader` out between the fetch and the commit, the commit may target the wrong reader — either panic (closed reader) or commit succeeds against a new reader with a stale offset.

**Likelihood.** Low under current synchronous processing; handlers complete before the fetcher can cycle. Medium if handlers are async or the reader is rebuilt because of a mid-commit error.

**Mitigation.** Two options, decide at implementation time:
- (a) Mutex-guard `c.reader` for both reads and writes; commit acquires the mutex and uses whatever reader is current (accepts that a mid-swap commit may silently drop a message in exchange for simplicity).
- (b) Thread the fetch-time reader through `processMessage` as an explicit parameter; commit goes to the original reader, which may already be `Close()`d → commit errors are already logged at `Warn` today.

**Prefer (b)** — commits on a closed reader produce a visible error; commits routed to a fresh reader with a stale offset silently risk duplicate delivery, which is harder to diagnose. Document the chosen approach in the implementation plan.

---

## R3. `ReaderConfig` drift across recreations

**Risk.** The current code builds `kafka.ReaderConfig` inline in `AddConsumer` (manager.go:93–99) from the `Config` struct. Moving that construction into the `Consumer` struct for later rebuild means capturing config at registration time. If any field in `kafka.ReaderConfig` is later expected to change at runtime (e.g., broker list rotation, group-id override), recreations will use the stale config.

**Likelihood.** Low — the config is intended to be fixed per consumer lifetime.

**Mitigation.** Document the invariant on the `Consumer` struct: "ReaderConfig is captured once at registration and never mutated." If a runtime change is ever needed, add an explicit API (e.g., `UpdateConsumerConfig`) rather than surprising callers.

---

## R4. Cross-service rollout coordination

**Risk.** The library change is breaking at the *behavior* layer even though the API is source-compatible — services linking the old library version against the new one do not co-exist; each service rebuilds and redeploys. If a subset of services are rebuilt and deployed without the rest, the new consumer behavior runs only in that subset.

**Likelihood.** Medium during an incremental rollout.

**Mitigation.** No coordination required for correctness (old behavior is strictly worse, not different-shaped), but the debug route will return 404 on services that have not yet been rebuilt. Document this in the release notes. The 8 HTTP-less services need their manifest changes deployed in lockstep with their code changes, or they will start advertising a `REST_PORT` that the container is not listening on — manageable with a single coordinated deploy.

---

## R5. Debug route leaking internal state

**Risk.** The debug route lists topic names, consumer group ids, broker hostnames, and internal error strings. If the route is ever promoted to the public ingress without further review, an external attacker can enumerate the service's event topology and use error messages to fingerprint library versions.

**Likelihood.** Low today (route is cluster-internal). Medium if a future follow-up adds ingress exposure without re-reviewing auth.

**Mitigation.** §4.8 and §8 of the PRD explicitly scope this task to cluster-internal access and call out that ingress exposure requires an auth layer to be added at that time. When a follow-up task proposes ingress exposure, that task owner is responsible for:
- Adding authentication (tenant header + shared secret, or mTLS, or a service-mesh sidecar).
- Reviewing error-string content to ensure no credentials leak (e.g., a kafka-go error containing a SASL password).
- Considering whether the route should redact broker hostnames in non-dev environments.

---

## R6. Shutdown ordering regression

**Risk.** The old implementation used an inner `done` channel + outer `<-ctx.Done()` block. The new implementation folds both into a single loop that observes `ctx` directly. If the new loop returns before `WaitGroup.Done()` on a path we don't anticipate, `tdm.Wait()` hangs indefinitely at process shutdown.

**Likelihood.** Low — the new loop has `defer wg.Done()` at the top.

**Mitigation.** The test suite must include a shutdown-ordering test: script a fake reader, cancel the parent context, assert that `start` returns within a short bound (<100ms), assert `WaitGroup` counts to zero. This is a must-have in `manager_test.go` per the PRD's test checklist.

---

## R7. Minimal REST scaffold breaking socket-heavy services

**Risk.** The 8 services getting a minimal REST scaffold include atlas-channel, which today is a socket-heavy game server. Adding an HTTP listener on a new port could (a) fight for a port that's already bound, (b) introduce unexpected goroutine/lifecycle interactions with the existing socket server, or (c) pass startup but silently fail to serve requests if the builder's graceful-shutdown doesn't mesh with the existing teardown manager.

**Likelihood.** Low. `libs/atlas-rest/server.Builder` is already used by the other 41 services and participates in the same `TeardownManager` / `WaitGroup` pattern that atlas-channel uses for its socket server. The port (`REST_PORT`, default `8080`) doesn't collide with atlas-channel's socket port (`8302` per config).

**Mitigation.** For each of the 8 services, during implementation: (1) verify that adding the REST builder does not alter the existing startup sequence by reading main.go before editing; (2) smoke-test the debug route after startup; (3) smoke-test existing behavior (socket connections for atlas-channel, consumer processing for atlas-fame etc.) to confirm no regression. Include this as an acceptance-criteria checkbox in the implementation plan.

---

## R8. `api2go/jsonapi` overhead for a debug route

**Risk.** The `api2go/jsonapi` serializer has specific struct-shape requirements (`GetName()`, `GetID()`, etc.) that may be awkward for a synthetic "list of consumers" where each entry is a point-in-time snapshot rather than a domain entity. A naïve integration may require boilerplate that feels heavy for a read-only debug surface.

**Likelihood.** Medium during implementation; cosmetic, not functional.

**Mitigation.** Flagged as an open question in PRD §9. If `api2go/jsonapi` wiring feels disproportionate, fall back to a hand-rolled JSON:API response — the spec locks in the response shape and content-type, not the serializer. The cost of inconsistency with other routes is low because this is a debug-only route.

---

## R9. `recreateCount` resets on process restart

**Risk.** `recreateCount` is in-memory and resets on process restart. An operator inspecting the debug route after a pod restart will see `recreateCount: 0` even if the pod was restarted *because* of consumer death in the previous incarnation.

**Likelihood.** Not a bug — it's the correct semantics for "are we having trouble right now?"

**Mitigation.** Document in the debug route's response (or in a comment in `debug.go`) that `recreateCount` is process-local. If historical cross-restart tracking is ever needed, it's a separate feature (metrics backend, log aggregation) explicitly out of scope for this task.

---

## R10. Go module version skew

**Risk.** `libs/atlas-kafka` is consumed by every service via Go modules (`go.mod`). The library change ships as a new version, and each service's `go.mod` / `go.sum` needs updating. If a service is missed or its `go.sum` update is incomplete, the service will build against an older library version without the fix.

**Likelihood.** Medium — mechanical but easy to miss across 49 services.

**Mitigation.** Per the project CLAUDE.md: "After making changes across multiple services, always run builds and tests for ALL affected services before reporting completion." The implementation plan must include a `go work` or per-service `go get` update step, and the Docker-build acceptance criteria catches any service still on the stale version.
