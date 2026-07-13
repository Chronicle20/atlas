# task-119-db-transaction-coverage — Execution Context

Companion to `plan.md`. Key files, decisions, and dependencies an implementer needs; everything below was verified against this worktree on 2026-07-02 (pre-rebase — Task 4 re-verifies line numbers).

## The headline discovery (drives everything)

`database.ExecuteTransaction` (libs/atlas-database/transaction.go:9) **has never opened a transaction**: `isTransaction` checks `Statement.ConnPool != nil`, which is true for every `*gorm.DB`, so `fn(db)` always runs un-transacted. All 53 call sites across 18 services are non-atomic today. The fix (`gorm.TxCommitter` type-assert, design §2.4) is commit 1 and a prerequisite for every rollback test in this task. It also silently voids task-114's outbox enqueue-in-tx atomicity — the design recommends rebase-cutting commit 1 into an immediate standalone PR (owner decides; surface it, don't block).

## Key files

| Area | File | Notes |
|---|---|---|
| Lib fix | `libs/atlas-database/transaction.go` | 18 lines; only `isTransaction` changes |
| Test infra | `libs/atlas-database/databasetest/testdb.go` | `NewInMemoryTenantDB(t, migrations...)`, `TenantContext(id)` — the idiom every rollback test uses |
| New helper | `libs/atlas-database/databasetest/failwrites.go` | `FailWritesOn(t, db, table, verbs...)` — Task 2 creates it |
| Exemplar | `services/atlas-guilds/atlas.com/guilds/guild/processor.go:253` | Canonical composition: `Emit` outside, `ExecuteTransaction` inside, buffer-puts inside the tx |
| keys | `services/atlas-keys/atlas.com/keys/key/processor.go:72,91,105,117` | 4× raw `db.Transaction` → one-line swaps |
| families | `services/atlas-families/atlas.com/family/family/processor.go:173,245,318` | 3× raw; has `WithTransaction` (line 73); emits via caller-owned buffer (already correct) |
| npc-conversations | `services/atlas-npc-conversations/atlas.com/npc/conversation/npc/processor.go:130,153,177,200,219,271` | 6× raw; no emits in these flows |
| monster-book | `kafka/consumer/monsterbook/consumer.go:56` (emit INSIDE tx — [E]), `kafka/consumer/character/consumer.go:49` | Card/collection processors already have `WithTransaction` — untouched |
| marriages | `marriage/processor.go:1581` (Accept flow), `:1688` (`executeInTransaction` — the repo's only manual `Begin/Commit`; emit inside tx — [E]) | `AcceptProposalWithTransactionAndEmit` is on the `Processor` interface (line 47) |
| storage | `storage/processor.go:720` (`ExpireAndEmit`), `:483` (`MergeAndSort`), `asset/processor.go:50` (`GetOrCreateStorageId`) | The genuine unwrapped gaps; neither processor has `WithTransaction` yet |
| Audit artifact | `docs/tasks/task-119-db-transaction-coverage/audit.md` | Task 3 creates; Tasks 5-14 fill remediation pointers |

## Verified facts implementers should not re-derive

- All six remediation services already depend on `libs/atlas-database` in go.mod (import as `database "github.com/Chronicle20/atlas/libs/atlas-database"`).
- Table names: `keys`, `family_members`, `conversations` (+`recipes`), `monster_book_cards`/`monster_book_collections`, `marriages`/`proposals`, `storages`/`storage_assets`.
- Migrations: `key.Migration`, `family.Migration`, `npc.MigrateTable`+`recipe.MigrateTable`, `card.Migration`+`collection.Migration`, `marriage.Migration` (covers all 3 entities), `storage.Migration`+`asset.Migration`.
- Tenant create-callback stamps `tenant_id` (`libs/atlas-database/tenant_scope.go:110-124`); reads/writes without a tenant ctx are not blocked (existing provider tests rely on this).
- npc-conversations deletes go through GORM `.Delete` (soft for conversations, `Unscoped()` hard for delete-all, hard for recipes) — all fire the Delete callback chain, so `FailWritesOn` intercepts them. Raw `.Exec` would not (only relevant if task-116 changes shapes).
- `getSlotMaxByTemplateId` (storage/processor.go:630) does atlas-data REST lookups → must be prefetched BEFORE the `MergeAndSort` tx (no network I/O inside a tx). In tests those lookups fail → code falls back to slotMax=100 (existing behavior, handy for fixtures).
- `mbmsg.Command` fields: `CharacterId uint32`, `EventId uuid.UUID`, `Type string`, `Body`; `card.UpsertResult` has `Inserted`/`Duplicate` flags.
- Marriages `ProcessorImpl` fields to clone in the shadow processor: `log, ctx, db, producer, characterProcessor` (re-check after task-116).
- The 18 `ExecuteTransaction`-calling services (fleet test scope, Task 15): buddies, cashshop, character, configurations, data, drop-information, fame, gachapons, guilds, inventory, merchant, mounts, notes, npc-shops, pets, quest, skills, tenants.

## Decisions locked at design time

1. **D0 fix via `gorm.TxCommitter`**, not unconditional `db.Transaction(fn)` (savepoint semantics rejected — nested composition must JOIN the outer tx).
2. **Refined taxonomy**: RMW with exactly ONE write is class C + race annotation, NOT class B (no rollback test is definable; wrapping doesn't close the race). Applies to account `GetOrCreate`-style flows.
3. **Seeder cycle is class D** (shared `libs/atlas-seeder` semantics: continue-on-error, per-file accounting, per-tenant lock). Changing it affects out-of-scope consumers — recorded as follow-up candidate in audit.md, not smuggled in.
4. **Emit convention**: buffer + publish-after-commit (guilds shape). task-114's outbox migration list does NOT include these 14 services; re-check per service at the Task 4 rebase gate — if one was migrated, swap the provider, don't restructure.
5. **[E] fixes change failure-mode behavior only** (no event for a rolled-back write — that's FR-2.2's requirement); happy-path events must be byte-identical (FR-2.4).
6. **storage `ExpireAndEmit`**: replacement-create failure becomes fatal-and-rollback (was Warn-and-keep-delete). Intentional — this IS the atomicity gap; document in audit.md.
7. **Category-1 rollback tests are green-before-and-green-after** (raw tx already rolls back) — they are regression locks, not red-green gates. Only storage Tasks 11-12 are true TDD red→green.

## Sequencing dependencies

- Tasks 1–3 run now. **Tasks 5–13 are blocked until task-114-outbox-adoption AND task-116-processor-gen3-unification merge into main** (PRD §7) and the branch is rebased (Task 4 checkpoint). As of 2026-07-02 both are unmerged (in-flight worktrees `.worktrees/task-114-outbox-adoption`, `.worktrees/task-116-processor-gen3-unification`).
- task-116 rewrites processor files in these same services — port diffs to the post-116 shape if lines moved; the composition rule is the invariant.
- Verification gate (CLAUDE.md): per-module `go test -race` / `go vet` / `go build`, `docker buildx bake all-go-services` (lib is COPY'd into every image), `tools/redis-key-guard.sh`.
- Code review (`superpowers:requesting-code-review`) runs BEFORE any PR (repo rule).
