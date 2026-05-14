# Task-040 — Implementation Context

Quick reference for the executing agent. The authoritative artifacts are `prd.md`, `design.md`, `risks.md`, and `plan.md` in this folder; `context.md` exists to surface the file paths, symbols, and gotchas an executor would otherwise rediscover.

## Companion artifacts (read first)

- `prd.md` — Goals, non-goals, functional requirements, acceptance criteria.
- `design.md` — Architectural choices and the "why pathway (a)" decision.
- `risks.md` — R1–R7 risks (cardinality, sampling skew, etc.). Design adds R8–R9.
- `migration-runbook.md` — Per-service edit pattern (created by Task 14).
- `cluster-changes.md` — Out-of-tree cluster YAML edits (created by Task 18).

## Key files (existing, read-only context)

| Path | Purpose | Notes for task-040 |
|---|---|---|
| `services/atlas-channel/atlas.com/channel/session/processor.go:168-184` | `Announce` curried function | Wrap inner lambda (line 173–179) with manual span. |
| `services/atlas-channel/atlas.com/channel/session/model.go:108-118` | `announceEncrypted` | Called from inside `Announce`'s inner lambda. |
| `services/atlas-channel/atlas.com/channel/session/model.go:164-166` | `s.WorldId()` | Source for `world.id` span attribute. Returns `world.Id` (alias for `byte`). |
| `services/atlas-channel/atlas.com/channel/socket/handler/handle.go:51-74` | `AdaptHandler` | Already creates a span named after the handler `name`. Confirms `CharacterItemUseHandle` becomes a span (panel 3). |
| `services/atlas-channel/atlas.com/channel/socket/handler/character_item_use.go:17-23` | `CharacterItemUseHandleFunc` | Entry point for client item-use; not modified by this task. |
| `libs/atlas-packet/inventory/serverbound/item_use.go:13` | `CharacterItemUseHandle = "CharacterItemUseHandle"` | Confirms the literal string used as span name. |
| `libs/atlas-kafka/consumer/manager.go:420` | `wctx, span = otel.GetTracerProvider().Tracer("atlas-kafka").Start(wctx, c.name)` | Kafka consumer auto-instrumentation; emits `c.name` (e.g. `compartment_command`, `consumable_command`) as span name. |
| `libs/atlas-kafka/consumer/manager_test.go:67-111` | `MockSpan` / `MockTracer` / `MockTracerProvider` pattern | Reuse this pattern for `TestAnnounce_StartsSpan` (copy into atlas-channel test, do not import). |
| `services/atlas-inventory/atlas.com/inventory/kafka/consumer/compartment/consumer.go:26` | Consumer name `"compartment_command"` | Panel 4 query depends on this. |
| `services/atlas-consumables/atlas.com/consumables/kafka/consumer/consumable/consumer.go:22` | Consumer name `"consumable_command"` | Panel 5 query depends on this. **Note:** lives under atlas-consumables, not atlas-inventory — design §4 panel 5 originally said "Inventory `consumable_command`"; corrected service_name to `atlas-consumables` in the dashboard JSON in Task 11. |
| `services/atlas-channel/atlas.com/channel/tracing/tracing.go` | Existing per-service tracer setup | Byte-identical with the other 53 copies; deleted in Task 5. |
| `services/atlas-channel/atlas.com/channel/main.go:136,382` | `tracing.InitTracer` / `tracing.Teardown` call sites | Import path swap in Task 5; call signatures unchanged. |
| `services/atlas-channel/Dockerfile` | Reference Dockerfile shape | Three-line additions for libs/atlas-tracing (per-lib `COPY` + inline `go.work` + source `COPY`). |
| `libs/atlas-tenant/go.mod` | Reference shared lib `go.mod` shape | Mimic for libs/atlas-tracing. |
| `go.work` | Workspace root | Add `./libs/atlas-tracing` in alphabetical position (between `./libs/atlas-socket` and `./libs/atlas-tenant`). |
| `deploy/k8s/env-configmap.yaml:13` | `TRACE_ENDPOINT: "tempo.home:4317"` | Insert `TRACE_SAMPLING_RATIO: "1.0"` immediately after. |
| `deploy/compose/.env:11` & `deploy/compose/.env.example:11` | `TRACE_ENDPOINT=tempo.home:4317` | Insert `TRACE_SAMPLING_RATIO=1.0` immediately after. |

## Symbols introduced by this task

| Symbol | Location | Purpose |
|---|---|---|
| `tracing.InitTracer(serviceName string)` | `libs/atlas-tracing/tracing.go` | Replaces 54 byte-identical per-service copies. |
| `tracing.Teardown(l logrus.FieldLogger)` | `libs/atlas-tracing/tracing.go` | Replaces 54 byte-identical per-service copies. |
| `tracing.parseSamplingRatio(l logrus.FieldLogger)` (unexported) | `libs/atlas-tracing/sampling.go` | Reads `TRACE_SAMPLING_RATIO`, validates `[0.0, 1.0]`, defaults to 1.0 with warn. |
| Span `session.Announce` | `services/atlas-channel/atlas.com/channel/session/processor.go` | Manual OTel span around `s.announceEncrypted`. Attributes: `writer.name`, `tenant.id`, `world.id`. |
| Configmap `grafana-dashboards-atlas` | Created by `deploy/grafana/apply.sh` | Mounted into Grafana at `/etc/grafana/provisioning/dashboards` (cluster-side mount in Task 18). |
| Dashboard UID `atlas-latency` | `deploy/grafana/dashboards/atlas-latency.json` | 7 panels + 2 template vars + 1 annotation. |

## Decisions locked

- **Pipeline pathway:** (a) Tempo `metrics_generator` with curated dimensions. Reversible to (b) collector via env-var swap if needed later (no Atlas code changes required).
- **Span name:** `session.Announce`. Stable; `writer.name` is the variable axis.
- **Tracer name on `otel.GetTracerProvider().Tracer(...)`:** `"atlas-channel"` — same as existing manual spans (`teardown`, `session-destroy`) and the auto-instrumented socket handler at `handle.go:57`.
- **Sampling default:** `1.0` (always sample). Warning log on parse failure; silent on unset.
- **Cardinality allowlist for spanmetrics:** `service_name`, `span_name`, `span_kind`, `status_code`, `writer.name`, `tenant.id`, `world.id`. Anything else is forbidden — and Tempo's `dimensions:` list is the single point of enforcement.
- **Migration scope:** all 54 services in this task. Phased PRs allowed; partial migration is not (the lib should not coexist with byte-identical copies long-term).

## Open items deferred from design (verified during planning)

| Open item | Resolution |
|---|---|
| Use-item entry-handler span name | `CharacterItemUseHandle` (literal string in `libs/atlas-packet/inventory/serverbound/item_use.go:13`, used by `AdaptHandler` at `handle.go:57`). |
| atlas-inventory consumer span name | `compartment_command` (`services/atlas-inventory/atlas.com/inventory/kafka/consumer/compartment/consumer.go:26`). |
| atlas-consumables consumer span name | `consumable_command` (`services/atlas-consumables/atlas.com/consumables/kafka/consumer/consumable/consumer.go:22`). Lives in atlas-consumables, not atlas-inventory — dashboard panel 5's `service_name` corrected accordingly. |
| `replace`-directive convention | Existing services use `replace github.com/Chronicle20/atlas/libs/<name> => ../../../../libs/<name>`. Mirror this for libs/atlas-tracing. |
| Dockerfile build pattern | Each Dockerfile copies per-lib `go.mod`/`go.sum`, inlines a `go.work` block, then copies per-lib source. Three line additions per Dockerfile. |

## Gotchas

- **`go.mod` `replace` paths.** `services/atlas-*/atlas.com/<dir>/go.mod` is **four** levels deep from `libs/`. The path is `../../../../libs/atlas-tracing` (not three, not five). Off-by-one here is the most common Dockerfile-build failure mode.
- **`go.work` in Dockerfiles is regenerated inline.** Each service's Dockerfile has its own `RUN echo ... go.work` block. Adding the lib to the repo-root `go.work` is necessary but not sufficient — every Dockerfile needs the corresponding `echo '    ./libs/atlas-tracing' >> go.work && \` line too.
- **`world.id` is a `byte`, not a string.** Use `attribute.Int("world.id", int(s.WorldId()))`, not `attribute.String`.
- **`tenant.MustFromContext(ctx).Id().String()`** — the `.String()` is mandatory; `Id()` returns a `uuid.UUID` value type and OTel attributes need primitives.
- **Don't import `attribute`/`codes` from anywhere other than `go.opentelemetry.io/otel/attribute` and `go.opentelemetry.io/otel/codes`.** Easy to autocomplete the wrong package and end up with a compile error.
- **Don't add `character.id`, `account.id`, `session.id`, `transaction.id`, `item.id`, or any free-form string as a span attribute** on `session.Announce` — even though they could be added later via the dimensions allowlist, omitting them at the span level is defence-in-depth (design §3.2).
- **`writer.Producer` signature.** `services/atlas-channel/atlas.com/channel/socket/writer/` defines this; the test in Task 7 mocks it. Inspect the actual signature before writing the mock — it may have shifted from what the test stub assumes.
- **Test file existing imports.** `processor_test.go` doesn't currently import `context`, `errors`, or any OTel packages. Add what you need; don't accidentally import the production tracer (would couple test to global state across tests).
- **Test isolation.** `TestAnnounce_StartsSpan` sets the global tracer provider via `otel.SetTracerProvider`. `t.Cleanup` to restore the previous provider, otherwise neighbouring tests in the same package leak the mock.
- **Dashboard datasource UIDs.** The JSON uses conventional UIDs `prometheus` and `loki`. The cluster Grafana might use different ones. Verify via `kubectl get configmaps -n observability` and inspect the datasources configmap, or accept that panels show "datasource not found" until UIDs are corrected.
- **`CLAUDE.md` rule:** "always verify Docker builds when changing shared libraries". Tasks 6 and 15 are the canary docker builds; Task 16 step 3 fans out the full matrix.

## Verification commands cheat-sheet

```bash
# Library tests
cd libs/atlas-tracing && go test ./...

# atlas-channel build + tests
cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./...

# Docker build (run from repo root)
docker build -f services/atlas-channel/Dockerfile -t atlas-channel:task040 .

# Find all remaining tracing/tracing.go (should be 0 after Task 16)
find services -name "tracing.go" -path "*/tracing/*"

# Confirm every service references the lib
grep -l "libs/atlas-tracing" services/atlas-*/atlas.com/*/go.mod | wc -l   # expect 54

# JSON validity for the dashboard
python3 -m json.tool deploy/grafana/dashboards/atlas-latency.json > /dev/null && echo OK
```

## Out-of-scope (don't get pulled into these)

- Manual span instrumentation of any service other than atlas-channel's `session.Announce`.
- Migrating `TRACE_ENDPOINT` to a non-Tempo target.
- SLO definitions / alerting rules.
- Refactoring `Announce`'s curried signature (it stays a 5-level curry; add the span inside the inner lambda only).
- Promoting `character.id`, `account.id`, `session.id`, `transaction.id`, `item.id` to spanmetrics dimensions (forbidden by §8.2).
- Touching atlas-ui (frontend, no Go tracing).
