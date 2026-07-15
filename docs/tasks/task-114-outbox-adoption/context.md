# task-114-outbox-adoption — Execution Context

Companion to `plan.md`. Key files, decisions, dependencies, and the planning-time fleet sweep the plan's task boundaries are built on.

## Key files

| Area | Path | Why it matters |
|---|---|---|
| Outbox lib | `libs/atlas-outbox/{outbox,drainer,entity,migration,backfill,lock,notify}.go` | Existing enqueue/drain/leadership; Tasks 1–5 extend it |
| Publisher (pre-move) | `services/atlas-configurations/atlas.com/configurations/outbox/publisher.go` | Moves verbatim to `libs/atlas-outbox/publisher.go` (Task 2) |
| Wiring template | `services/atlas-configurations/atlas.com/configurations/main.go:52-65` | The migration + drainer + teardown pattern every service copies |
| Emit seam | `services/atlas-character/atlas.com/character/kafka/message/message.go:44-77`, `kafka/producer/producer.go:10-20` | `Provider = func(token string) producer.MessageProducer` is the single seam; `EmitProvider` returns its unnamed underlying type |
| Header decoration | `libs/atlas-kafka/producer/header.go` (decorators), `producer.go:74-83` + `message.go:14-27` (fold) | Parity reference for `headerMap` |
| Token resolution | `libs/atlas-kafka/topic` (`EnvProvider`) | Env-var token → real topic; falls through to the token itself |
| Tx re-entrancy | `libs/atlas-database/transaction.go:9-14` | `ExecuteTransaction` on an existing tx runs the closure directly — the recipe's safety basis |
| Tenant scoping | `libs/atlas-database/tenant_scope.go:64` (`hasTenantColumn`) | Callbacks skip tables without `tenant_id`; outbox_entries is safe in tenant-scoped services |
| FR-1 hot paths | `services/atlas-character/atlas.com/character/character/processor.go:733-910` | Meso/fame/AP defects + in-tx emits (lines 742,750,751,768,785,812,813,826,830,892,901,905) |
| Guard model | `tools/rediskeyguard/`, `tools/redis-key-guard.sh`, `.github/workflows/pr-validation.yml:84-98,480,496` | outboxguard mirrors all three |
| Character test harness | `character/processor_test.go:20-48` (`testDatabase`/`testTenant`/`testLogger`), `testmain_test.go` (`producertest.InstallNoop`) | Reused by the new outbox tests |

## Decisions (from design.md, plus planning-time deviations)

1. **D1 seam**: `outbox.EmitProvider(l, ctx, tx)` — outbox-backed `producer.Provider`; migrations are structural inversions (Emit moves inside `ExecuteTransaction`). Service-local `message`/`producer` packages untouched.
2. **D2 headers**: decorate from request ctx at enqueue; drainer re-attaches at publish. **Planning deviation (binding)**: stored header *values* are base64-encoded in the jsonb. Reason: `TenantHeaderDecorator` emits raw big-endian uint16 version bytes (libs/atlas-kafka/producer/header.go:38-39) — always contain NUL (Postgres jsonb rejects it) and can be invalid UTF-8 (v185 = 0xB9; `encoding/json` mangles to U+FFFD). Base64 gives a byte-exact round trip. Existing rows (`{}` only, from atlas-configurations) unaffected.
3. **D3 ordering**: drainer `ORDER BY id ASC` (enqueued_at is tx-stable in Postgres → intra-tx ties). Backfill audited: enqueues row-by-row, no ordering query — no change.
4. **D4 publisher**: straight move, no alias. atlas-configurations keeps its local `outbox` package for envelopes only.
5. **D5 guard**: in scope — `tools/outboxguard` (go/analysis), `tools/outbox-guard.sh`, CI job next to redis-key-guard. Lexical rule: `producer.ProviderImpl` inside a func literal passed to `database.ExecuteTransaction` / `.Transaction`. No baseline file; tree is clean when it lands (Task 25 runs after all migrations).
6. **D6 wiring**: lib defaults everywhere (poll 1s, batch 100, retention 7d); drainer + `TopicWriterPool` + teardown per service main.
7. **D7 classification**: migrate emits asserting DB state changes; leave direct (inventoried) rejection/no-change events, command emits, relays, non-DB tickers. Rejection emits move *outside* tx closures via a captured `rejectEmit` closure (Task 7/8 device).
8. **Planning deviation — atlas-data**: design §7 anticipated `EnqueueBuffer` use; the authoritative FR-3.1 sweep found **zero** tx-coupled sites (`data/processor.go:85` = pure command; `:287` = post-worker aggregate across many txs, TTL-guarded). Inventory-only, no code change (PRD §7 rule). Same for gachapons/drop-information (no producer at all).
9. **Planning finding — only atlas-character has in-tx direct emits.** Every other service's exposure is the post-commit-crash window (`Emit` after tx); their migrations are pure Pattern A/B/C inversions.
10. **atlas-quest divergence**: no `message.Emit`/`ProviderImpl`; emits via `EventEmitter` interface. Migration = `OutboxEventEmitter` + `txEmitter func(tx) EventEmitter` processor field (mock-injection preserved via wrapper).

## Dependencies & ordering

- Tasks 1–5 (lib) strictly first; Task 4 needs Task 3; Task 2 independent of 1/3/4.
- Task 6 (character wiring) before 7–10. Tasks 11–24 (per-service) are mutually independent, each depends on Phase 1.
- Task 25 (guard) after all migrations (starts clean, no baseline). Task 26 last.
- New lib deps: `libs/atlas-outbox` gains `atlas-kafka` + `atlas-model` requires **plus replace lines for atlas-kafka's transitive Chronicle20 deps (atlas-retry, atlas-tenant)** — replace directives don't propagate from dependency go.mods.
- Build infra already handles atlas-outbox: go.work line 11; Dockerfile COPY lines 39/68/92. Per-service change = go.mod require+replace only; single `docker buildx bake all-go-services` in Task 26 covers the bake gate.
- Module-name gotchas: atlas-npc-shops module is `atlas-npc`; atlas-drop-information module is `atlas-drops-information`; module dirs are `services/<svc>/atlas.com/<shortname>` (`npc`, `dis`, etc.).

## Fleet sweep snapshot (2026-07-02, this worktree)

Per-service tx/emit counts the task boundaries were drawn from (re-verify in each task's enumerate step):

| Service | ExecuteTransaction | Emit / EmitWithResult | In-tx direct emits | Notes |
|---|---|---|---|---|
| character | 25 (processor.go) | 23 / 0 | **12** | reference impl |
| inventory | 26 | 25 / 0 | 0 | struct-init provider compartment/processor.go:108 |
| cashshop | 4 | 13 / 4 | 0 | helper emits :234,239 outside tx |
| fame | 2 | 2 / 0 | 0 | provider captured to local var |
| buddies | 8 | 8 / 0 | 0 | |
| guilds | 13 | 18 / 0 | 0 | |
| notes | 4 (administrator.go) | 3 / 2 | 0 | |
| pets | 13 | 11 / 1 | 0 | |
| mounts | 3 | 3 / 0 | 0 | Emits in task.go + 2 consumers (Pattern C) |
| skills | 5 | 5 / 0 | 0 | :229 registry-only, left direct |
| quest | 6 | 0 / 0 | 0 | EventEmitter at :216,361,461,535,580,586,866,936 |
| merchant | 9 | 11 / 0 | 0 | no EmitWithResult in message pkg |
| npc-shops | 4 | 5 / 0 | 0 | module `atlas-npc` |
| tenants | 7 (administrators) | 4 / 8 | 0 | EmitWithResult-heavy |
| gachapons | 3 | — | — | no producer at all |
| drop-information | 3 | — | — | no producer at all |
| data | 18 | — | — | 2 non-tx ProviderImpl sites |

No service go.mod requires atlas-outbox yet (except configurations); no service main.go runs a drainer (except configurations).

## Verification gates (every claim of done)

`go test -race ./...` + `go vet ./...` + `go build ./...` per changed module; `docker buildx bake all-go-services` once (Task 26); `tools/redis-key-guard.sh`; new `tools/outbox-guard.sh`. Code review (`superpowers:requesting-code-review`) before any PR.
