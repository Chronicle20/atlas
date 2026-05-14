# task-062 Context

Quick reference for an engineer with zero familiarity with this repo. Read this once before starting; refer back when you hit a "where does X live?" question.

## What this task does

Replaces the in-pod, single-goroutine WZ extraction loop in `atlas-wz-extractor` with a Kafka-fanout job model:

1. **Within-pod fan-out** over the `for _, wzPath := range wzFiles` loop (bounded worker pool).
2. **Cross-pod sharding** — `POST /api/wz/extractions` becomes a dispatcher that emits one Kafka `START_EXTRACTION_UNIT` message per WZ file. Any consumer in the `wz-extractor-extraction` group picks them up.
3. **Redis tenant lock + job-state** replacing the in-process `extraction/mutex.go` registry, so multi-pod deployments cannot run two extractions for the same tenant and a `GET /jobs/{jobId}` endpoint returns coherent state regardless of which pod served it.

PRD: `prd.md` in this folder. Design: `design.md` in this folder. Read both before starting.

---

## Key files (existing)

| Path | Role |
|---|---|
| `services/atlas-wz-extractor/atlas.com/wz-extractor/main.go` | service entrypoint; today just wires REST + waitgroup. We add Redis client, Kafka producer manager teardown, and the new consumer registration. |
| `services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/processor.go` | `Processor` interface + `processorImpl`. Today: `Extract` runs the whole-list serial loop. We split into `ExtractUnit` (per-WZ) + `Extract` (now backed by `pool.go`). |
| `services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/resource.go` | route registration + `handleExtract` (today: spawns goroutine, returns 202). We extract the orchestration into `dispatcher.go`. |
| `services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/mutex.go` + `mutex_test.go` | in-process per-tenant mutex registry. **Deleted** — replaced by Redis lock. |
| `services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/map_render.go` | the existing `RenderMaps` worker pool (kept untouched — Map.wz remains one unit per design §4.3). |
| `services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/status.go` | filesystem-scan status endpoints (`/wz/extractions` GET, `/wz/input` GET). Unchanged; new job-status endpoint is separate. |
| `services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/tenant_path.go` | `TenantPath`/`ResolveTenantInputDir`/`ResolveTenantOutputDir` + `TenantKey` helpers. Reuse for Redis lock key composition. |
| `services/atlas-wz-extractor/atlas.com/wz-extractor/rest/handler.go` | thin re-exports of `server.RegisterHandler`. Use for new handlers. |
| `services/atlas-wz-extractor/atlas.com/wz-extractor/go.mod` | add `atlas-kafka`, `atlas-redis`, `redis/go-redis/v9`, `alicebob/miniredis/v2`, `kafka-go`. |

## Reference implementations to copy from

| Path | What to copy |
|---|---|
| `services/atlas-data/atlas.com/data/kafka/consumer/data/consumer.go` | shape of `InitConsumers` + `InitHandlers` + `handleStart...` for a command-style consumer with tenant + span header parsers. |
| `services/atlas-data/atlas.com/data/kafka/consumer/data/kafka.go` | shape of `EnvCommandTopic` + `command[E]` + body struct. |
| `services/atlas-data/atlas.com/data/kafka/consumer/consumer.go` | `NewConfig(l)(name)(topicEnvKey)(groupId)` curried builder. **Copy verbatim into our service** (it's per-service code, not a shared lib). |
| `services/atlas-data/atlas.com/data/kafka/producer/producer.go` | `ProviderImpl(l)(ctx)(token)` curried producer with span+tenant decorators. **Copy verbatim.** |
| `services/atlas-data/atlas.com/data/main.go` (lines 60-122) | full main.go wiring: `tdm := service.GetTeardownManager()`; tracer; consumer manager + `InitConsumers/Handlers`; producer teardown; REST. We extend with Redis. |
| `services/atlas-buffs/atlas.com/buffs/main.go` (lines 41-79) | how to wire `atlas.Connect(l)` (Redis) + a registry, alongside Kafka. Closest existing combination of Redis + Kafka in the repo. |
| `services/atlas-buffs/atlas.com/buffs/character/registry.go` | example of an atlas-redis-backed registry with a `goredis.Client` field — useful pattern reference for `job.Store` and `lock.TenantLock`. |
| `services/atlas-buffs/atlas.com/buffs/kafka/message/buffer.go` | `Buffer.Put` + `Emit` semantics. Use the same pattern for emit-N-messages-atomically in the dispatcher. |
| `services/atlas-buffs/atlas.com/buffs/character/producer.go` | how a "command provider" function returns `model.Provider[[]kafka.Message]`. Mirror for our `startExtractionUnitCommandProvider`. |
| `services/atlas-portals/atlas.com/portals/blocked/cache_test.go` | how to use `miniredis.RunT(t)` to back unit tests. |

## Key shared libraries

| Lib | What we use |
|---|---|
| `libs/atlas-redis` | `redis.Connect(l)` (reads `REDIS_URL`/`REDIS_PASSWORD`), `redis.NewLockWithTTL(client, namespace, ttl)` with `Acquire/AcquireWithTTL/Release/Extend`, key helpers in `keys.go`. We **do not** use `Release`-with-owner-match from the lib — we add our own Lua compare-and-delete in `lock/tenant_lock.go` per design §5.5. |
| `libs/atlas-kafka` | `consumer.GetManager().AddConsumer(l, ctx, wg)`, `consumer.GetManager().RegisterHandler`, `consumer.SetHeaderParsers(SpanHeaderParser, TenantHeaderParser)`, `consumer.SetStartOffset(kafka.LastOffset)`. Producer side: `producer.GetManager().Close(l)`, `producer.SpanHeaderDecorator`, `producer.TenantHeaderDecorator`, `producer.Produce(l)(...)`, `producer.SingleMessageProvider`, `producer.CreateKey`. Topic: `topic.EnvProvider(l)(envName)()`. Message: `message.AdaptHandler(message.PersistentConfig(...))`. |
| `libs/atlas-tenant` | `tenant.MustFromContext(ctx)`, `tenant.WithContext(ctx, t)`. `t.Id()`, `t.Region()`, `t.MajorVersion()`, `t.MinorVersion()`. |
| `libs/atlas-rest/server` | `server.RegisterHandler`, `server.HandlerDependency`, `server.GetHandler`, `RouteInitializer`. Re-exported via `rest/handler.go`. |
| `libs/atlas-service` | `service.GetTeardownManager()` returns `tdm` with `Context()`, `WaitGroup()`, `TeardownFunc(...)`, `Wait()`. |

## Design choices already made (do NOT relitigate)

- **Map.wz stays one unit.** Internal `RenderMaps` pool keeps within-pod parallelism for it. (design §4.3)
- **Within-pod parallelism = partition assignment, not async handler pool.** Topic must be provisioned with `partitions ≥ 16`. The handler is synchronous. (design §4.2)
- **`WZ_EXTRACT_PARALLELISM`** is now (a) topic-provisioning hint and (b) pool size for the legacy `Extract`-the-method path used in tests. It is **not** consumed by the consumer at runtime. (design §4.10)
- **Tenant lock TTL = 60 min.** Refresh goroutine extends every 20 min. Compare-and-delete on release. (design §4.4 + §5.5)
- **Job hash keyed by jobId only**, not tenant-scoped — tenant ID is a field. Lock key **is** tenant-scoped. (design §4.7)
- **Single-replica clusters still go through Kafka.** No in-process fallback. (design §4.5)
- **Empty-input on POST → 400.** Was a regression-quality `202` previously. (design §4.8)
- **`wipeCharacterCache` runs once on the dispatcher** before any unit message is published. (design §4.9)
- **Idempotent finalize via WATCH/MULTI/EXEC** on the `:units` hash. Last-one-home does `SET status NX` for the terminal job state. (design §4.6)

## Things deliberately NOT in scope

- Removing the `*sync.WaitGroup` parameter from `extraction.InitResource`. It becomes effectively a no-op under the new model; design §11 punts removal to a follow-up.
- Subdividing Map.wz.
- HPA configuration.
- Multi-tenant fairness / queueing / cancellation.
- Streaming a single image's parse across goroutines.
- New shared library. Everything new is service-local.

## Build & test commands

From inside `services/atlas-wz-extractor/atlas.com/wz-extractor/`:

```bash
go build ./...
go test ./...
```

Service builds against `go.work` so the local lib paths resolve without explicit replace directives.

---

## Glossary

- **Unit** = one `(jobId, wzFile, stage flags)` Kafka message. The minimum granularity of work.
- **Job** = one `POST /api/wz/extractions` invocation; aggregates all units for one extraction.
- **Dispatcher** = the pod that received the POST. Acquires lock, wipes character cache, creates Redis records, publishes N unit messages, returns 202. Does **not** itself execute any unit.
- **Last one home** = whichever consumer brings `unitsCompleted + unitsFailed` to `unitsTotal`. That consumer declares the terminal job status and releases the tenant lock.
