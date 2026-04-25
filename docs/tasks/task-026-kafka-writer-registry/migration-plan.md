# Migration Plan — Kafka Writer Registry

This document supplements `prd.md`. It describes the order of operations for the all-at-once migration so the implementation phase can execute it as a single PR without leaving the build broken at intermediate steps.

## Pre-flight: enumerate the real surface area

Before editing any files, confirm the actual list of services and callsites:

```
# Per-service producer wrappers (expected ~63)
find services -name producer.go -path "*/kafka/producer/*"

# All producer callsites (expected ~163)
grep -rn "producer.ProviderImpl" services/

# Services missing a kafka/producer/producer.go but using the library directly
grep -rln "github.com/Chronicle20/atlas/libs/atlas-kafka/producer" services/ \
  | xargs -I{} dirname {} | sort -u
```

The PRD lists services from a snapshot survey; the implementation phase MUST trust the live grep over the snapshot.

## Phase 1 — Library changes (one PR, one commit)

1. Add `Manager` type and singleton accessor in `libs/atlas-kafka/producer/`. Most likely a new file `manager.go`.
2. Modify `producer.go`:
   - Remove `w.Close()` from the per-publish path inside `Produce`.
   - Update `WriterProvider` to either delegate to the registry or be deprecated. If deprecated, a `// Deprecated:` doc comment plus a passthrough that fetches from the registry.
3. Add unit tests for the registry: concurrent first-touch, idempotent close, error propagation.
4. Update `producer_test.go` to the new lifecycle.
5. Run `go test ./...` inside `libs/atlas-kafka/`.

After Phase 1, the library compiles and tests pass, but services are still on the old shape — they pass through the deprecated `WriterProvider`, which now delegates. **The registry would technically work end-to-end at this point, but `main.go` files don't yet register `Close()` for graceful shutdown.** That gap is closed in Phase 2.

## Phase 2 — Per-service mechanical updates (same PR)

The 63 services share an almost-identical `kafka/producer/producer.go` and a similarly-shaped `main.go`. The edits are:

### 2a — Update each `services/*/atlas.com/*/kafka/producer/producer.go`

For each file, replace the body of `ProviderImpl` so it pulls a Writer from the registry instead of through `producer.WriterProvider(topic.EnvProvider(l)(token))`. The exact code shape will be settled in the design phase, but the diff is mechanical and identical across all 63 services. A scripted find-and-replace (e.g. `gopls rename` or a `sed` script with tested anchors) is acceptable as long as the result is reviewed.

### 2b — Update each `services/*/atlas.com/*/main.go`

Add two lines in a consistent location: after the consumer manager block, before the `server.New(l)...Run()` call. Proposed pattern (final shape settled in design):

```go
producer.GetManager().Init(l)
tdm.TeardownFunc(producer.GetManager().Close)
```

This places the producer teardown *after* the consumer teardown registration, so by the natural ordering of `tdm.TeardownFunc(...)` slices, consumers stop first and producers flush last.

### 2c — Verify per-service builds

Run `go build ./...` from each service's module root. Run that service's existing test suite. Both must pass before merging.

## Phase 3 — End-to-end verification (same PR)

1. **Smoke test on atlas-data** (the service where the symptom was originally discovered):
   - Bring up the stack with `COMMAND_TOPIC_DATA` configured for at least 4 partitions.
   - Run `POST /data/process`.
   - Inspect partition assignments with `kafka-consumer-groups.sh --describe --group "Data Service" --bootstrap-server <broker>`.
   - Confirm multiple partitions show current-offset advancement, indicating real fan-out.
2. **Graceful shutdown test on any one service:**
   - `kubectl rollout restart deployment/<svc>` (or local equivalent).
   - Confirm logs include the registry's "shut down N writers" line before the pod exits.

## Rollback

Because this is a single-PR change touching the shared library, rollback is `git revert <pr-merge-commit>`. There is no schema migration, no data migration, and no deploy-manifest change to undo. Topic partition counts (if previously raised on the broker) are not part of this rollback — they're external state and unaffected.

## Risks specific to the migration

- **Diff size.** ~63 service files + ~63 `main.go` files + library + tests. The diff will be large but mechanical. Review burden mitigated by the per-service edits being identical templates.
- **Hidden producer callsites that don't go through `ProviderImpl`.** Some services may construct `kafka.Writer` directly or use `producer.Produce(...)` with a custom `Writer` provider. Pre-flight grep MUST surface these; each gets a one-off review.
- **First-publish latency.** Lazy Writer construction means the first message to a new topic incurs a TCP/metadata round-trip on the request path. For request-response services this is one-time and acceptable. If any service had previously been benefiting from connection pre-warming via the per-call construct (it wasn't, but perception matters), expect to clarify in code review.
- **Silent test breakage.** Some service tests may have been incidentally relying on per-call Writer close (e.g. asserting on `Close()` being called). Pre-flight grep for `WriteMessages` and `Close()` in test files MUST flag these.
