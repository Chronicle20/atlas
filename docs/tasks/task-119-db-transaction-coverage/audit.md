# DB Transaction Coverage — Full Write-Path Audit (Task 3)

Version: v1
Status: Complete (DL-4 closure evidence)
Design: docs/tasks/task-119-db-transaction-coverage/design.md (§3 survey, §5.2 taxonomy, §5.3 format)

This is a full sweep — not a spot-check — of all DB write statements in the 14 services that
had zero `database.ExecuteTransaction` call sites as of the PRD's grep premise. Every hit from

```
grep -rn "\.Create(\|\.Save(\|\.Update(\|\.Updates(\|\.Delete(\|\.Exec(" services/atlas-<svc> --include='*.go' | grep -v _test.go
grep -rn "\.Transaction(\|\.Begin()\|\.Commit()\|\.Rollback()" services/atlas-<svc> --include='*.go' | grep -v _test.go
```

is either classified below (write inventory) or accounted for as a non-DB exclusion, so the
sweep is checkable end-to-end.

**Taxonomy** (design §5.2): **A** multi-table (≥2 writes, ≥2 tables) → wrap. **B** multi-statement
single-table (≥2 write statements) → wrap. **C** single-statement → no change (RMW-with-one-write
is class C + a mandatory race annotation, never class B). **D** intentionally non-atomic → written
justification. Flags: **[T]** already-transactional via raw `db.Transaction`/manual `Begin` (needs
conversion to `ExecuteTransaction`). **[E]** emit-inside-transaction (publish precedes commit).

Remediation-commit placeholders are `_(commit: pending — Task N)_` per the brief; Tasks 5–13 filled
these in after the task-114/task-116 rebase gate (Task 4). **Pointer convention (finalized at Task 14):**
each remediation pointer references its commit by *task number + commit subject* rather than a raw SHA.
This is deliberate — the branch rebases onto `main` again before merge (PR #961), which rewrites every
SHA; the task+subject reference is rebase-stable and unambiguously resolvable via `git log --grep`.
No `_(commit: pending …)_` placeholder remains.

---

## atlas-npc-conversations

### Write inventory

| file:line | Entry point | Writes | Class | Flags |
|---|---|---|---|---|
| `conversation/npc/processor.go:130` `createWithSkipTracking` | seed-time helper invoked from the reindex/seed loop | `conversations` + `recipes` | A | [T] |
| `conversation/npc/processor.go:153` `Create` | REST `POST /npcs/conversations` (`conversation/npc/resource.go:98`) | `conversations` (`npc/administrator.go:21`) + `recipes` (`recipe/administrator.go:21` via `recipe/processor.go:141` `RebuildForConversation`) | A | [T] |
| `conversation/npc/processor.go:177` `Update` | REST `PATCH /npcs/conversations/{id}` (`conversation/npc/resource.go:134`) | `conversations` (`npc/administrator.go:52`) + `recipes` (rebuild) | A | [T] |
| `conversation/npc/processor.go:200` `Delete` | REST `DELETE /npcs/conversations/{id}` (`conversation/npc/resource.go:162`) | `conversations` (`npc/administrator.go:75`) + `recipes` (`recipe/administrator.go:33`) | A | [T] |
| `conversation/npc/processor.go:219` `DeleteAllForTenant` | tenant-teardown/admin path (Processor interface method, not directly REST-routed) | `conversations` (`npc/administrator.go:82`) + `recipes` (`recipe/administrator.go:41`) | A | [T] |
| `conversation/npc/processor.go:271` `ReindexAllRecipes` | REST `POST /npcs/conversations/reindex-recipes` (`conversation/npc/resource.go:253,263`) | `conversations` (read) + `recipes` (delete-all + bulk-insert loop) | A | [T] |
| `conversation/quest/administrator.go:11,32,74,82` (`quest/processor.go:87,99,111,123`) | REST `POST/PATCH/DELETE /quests/conversations{,/id}` (`quest/resource.go:121,157,185`) | `quest_conversations` | C | no [T] needed — single write each |
| `conversation/npc/subdomain.go:28` `DeleteAllForTenant` (seeder `Subdomain` impl) | `libs/atlas-seeder/seed.go:87` via `POST /npcs/conversations/seed` | `conversations` only | D (seeder cycle) | see completeness note below |
| `conversation/npc/subdomain.go:67` `BulkCreate` (seeder `Subdomain` impl) | `libs/atlas-seeder/seed.go:112` | `conversations` only (single bulk INSERT) | D (seeder cycle) | see completeness note below |
| `conversation/quest/subdomain.go:28,67` `DeleteAllForTenant`/`BulkCreate` | seeder cycle, same as above | `quest_conversations` | D (seeder cycle) | single table, no derived rows — benign |

### Exclusions (non-DB write-verb hits)

| file:line | Reason |
|---|---|
| `conversation/operation_executor.go:261,307` | `e.sagaP.Create(s)` — `saga.Processor.Create` (`npc/saga/processor.go:27`) is a Kafka publish to atlas-saga-orchestrator's command topic via `producer.ProviderImpl`, not a DB write. |
| `conversation/processor.go:718,780,848,909,974` | `saga.NewProcessor(p.l,p.ctx).Create(s)` — same `atlas-npc-conversations/saga` package, same Kafka-publish mechanism. |
| `conversation/quest/resource.go:121,157,185` | REST handler bodies delegating to `NewProcessor(...).Create/Update/Delete`; the write itself is at `quest/administrator.go`, listed once above, not duplicated here. |
| `conversation/npc/resource.go:98,134,162` | Same pattern — REST delegator, not a separate write. |
| `test/tenant.go:10` | `tenant.Create(uuid.New(), "GMS", 83, 1)` — test helper, builds an in-memory `tenant.Model`, no DB access. |

### Verdicts

- `Create`/`Update`/`Delete`/`DeleteAllForTenant`/`ReindexAllRecipes`/`createWithSkipTracking` (npc pkg, 6 sites): **A**, already wrapped via raw `db.Transaction` — confirmed emit-free (no `message.Emit`/producer call inside any of the six closures; re-confirmed at remediation time via `grep -n "Emit\|producer\." conversation/npc/processor.go` → no hits). Remediation: converted to `ExecuteTransaction`; added a characterization test (`processor_rollback_test.go`) proving `DeleteAllForTenant`'s recipe-clear-then-conversation-delete ordering rolls back atomically (fails the conversation delete via `FailWritesOn`, asserts the recipe clear is restored). `_(commit: "refactor(atlas-npc-conversations): standardize raw db.Transaction onto database.ExecuteTransaction" — Task 7, done)_`
- Quest-conversation CRUD (4 sites): **C** — no change; each administrator function is a single INSERT/UPDATE/DELETE. No transaction needed.
- Seeder-cycle paths (npc + quest subdomains): **D** — justified per the shared `libs/atlas-seeder` semantics (continue-on-error, per-file accounting, per-`(tenant,group)` mutex, `libs/atlas-seeder/seed.go:22-39,85-120`); changing it is an out-of-scope `atlas-seeder` design change (candidate follow-up, not a task here). **Completeness note (not a transaction-coverage finding, flagged for awareness):** the npc seeder's `DeleteAllForTenant` (subdomain.go:28) does not clear `recipes`, and its `BulkCreate` (subdomain.go:67) does not derive recipe rows — only the manual `POST /npcs/conversations/reindex-recipes` endpoint (`processor.go:271`) rebuilds them, and no automatic call site invokes it after a seed. This is a data-completeness gap adjacent to, but distinct from, the transaction-coverage question audited here.

---

## atlas-keys

### Write inventory

| file:line | Entry point | Writes | Class | Flags |
|---|---|---|---|---|
| `key/processor.go:72` `Reset` | REST `DELETE /characters/{id}/keys` (`character/resource.go:58`) | `keys` — delete-all-for-character then a 40-entry create loop | B | [T] |
| `key/processor.go:91` `CreateDefault` | Kafka consumer, character-created status event (`kafka/consumer/character/consumer.go:49`) | `keys` — 40-entry create loop | B | [T] |
| `key/processor.go:105` `Delete` | Kafka consumer, character-deleted status event (`kafka/consumer/character/consumer.go:63`) | `keys` — single delete-by-character | C (wrapped anyway, harmless) | [T] |
| `key/processor.go:117` `ChangeKey` | REST `PATCH /characters/{id}/keys/{keyId}` (`character/resource.go:41`) | `keys` — read-then-create-or-update (RMW) | B | [T] |

`key/administrator.go:17` (create), `:25` (update), `:29` (delete) are the raw statement helpers invoked only via `tx` from inside the four transactions above.

### Exclusions (non-DB write-verb hits)

None — every write-verb hit in this service traces to a real `keys`-table write via the four `processor.go` transactions above.

### Verdicts

- `Reset`, `CreateDefault`: **B** — delete-all+create-loop / create-loop, single table (`keys`), already wrapped in raw `db.Transaction`. Convert to `ExecuteTransaction`. `_(commit: "refactor(atlas-keys): standardize raw db.Transaction onto database.ExecuteTransaction" — Task 5, done)_`
- `Delete`: **C** in substance (single statement) but already wrapped — harmless; convert wrapper for consistency as part of the same Task 5 commit.
- `ChangeKey`: **B** (read-then-branch-to-create-or-update, single table `keys`) — already wrapped. `_(commit: "refactor(atlas-keys): standardize raw db.Transaction onto database.ExecuteTransaction" — Task 5, done)_`. No emit calls found anywhere in `key/processor.go` (no Kafka producer import in the file), so no [E] flag applies.

---

## atlas-families

### Write inventory

| file:line | Entry point | Writes | Class | Flags |
|---|---|---|---|---|
| `family/processor.go:173` `AddJunior` | Kafka command `AddJuniorCommand` (`kafka/consumer/family/consumer.go:83`) + REST `POST /families/{id}/juniors` (`family/resource.go:48`) | `family_members` — senior row save + junior row save (2 writes, 1 table) | B | [T] |
| `family/processor.go:245` `RemoveMember` | Kafka command `RemoveMemberCommand` (`consumer.go:111`) + character-deleted status event | `family_members` — senior save + N junior saves + target delete | B | [T] |
| `family/processor.go:318` `BreakLink` | Kafka command `BreakLinkCommand` (`consumer.go:138`) + REST `DELETE /families/links/{id}` (`resource.go:93`) | `family_members` — senior save + self save + N junior saves | B | [T] |
| `family/administrator.go:61` `BatchResetDailyRep` | Scheduler `ReputationResetJob.executeResetJob` (`scheduler/reputation_reset.go:134`) | `family_members` — single bulk `UPDATE ... WHERE daily_rep > 0` | C | no [T] needed |
| `family/administrator.go:88` `SaveMember` (standalone use) | `AwardRep`/`DeductRep` (`processor.go:458,510`) ← Kafka commands (`consumer.go:166,194`) | `family_members` — single `db.Save` | C | no [T] needed (this function also runs inside the 3 TXs above via `tx`, where it is a sub-step, not a separate finding) |
| `family/administrator.go:31` `CreateMember` (auto-provision branch) | `AddJunior` (`processor.go:106`), called with raw `p.db`, **not** `tx` — the surrounding branch always returns `ErrSeniorNotFound` | `family_members` — single insert | C | **flag**: runs outside the enclosing transaction, on a caller path that reports failure (see note below) |

### Exclusions (non-DB write-verb hits)

| file:line | Reason |
|---|---|
| `family/entity.go:42,58,63,74,88` | `db.Exec(...)` inside `Migration(db *gorm.DB) error` — one-time schema DDL (`CREATE INDEX`, `ALTER TABLE ... ADD CONSTRAINT`) run at service startup, not a request-time data write. |

### Verdicts

- `AddJunior`, `RemoveMember`, `BreakLink`: **B** (2+ writes, single table `family_members`), already wrapped via raw `db.Transaction`, buffered emits deferred until after the transaction returns (confirmed no `message.Emit` inside any of the three closures). Convert to `ExecuteTransaction`. `_(commit: "refactor(atlas-families): standardize raw db.Transaction onto database.ExecuteTransaction" — Task 6, done)_` — the three sites were converted 1:1 (same boundaries, writes, order); a rollback regression test (`TestAddJunior_RollsBackSeniorSaveWhenJuniorSaveFails`, `family/processor_rollback_test.go`) was added and confirmed PASS both before and after the conversion, characterizing `AddJunior`'s two-write senior/junior save as atomic.
- `BatchResetDailyRep`, standalone `SaveMember` calls: **C** — no change.
- **Informational flag (not a transaction-coverage classification change, not remediated by Task 6):** `AddJunior`'s auto-provisioning of a missing senior (`family/administrator.go:31` via `processor.go:106`) executes against `p.db` directly, outside the subsequent transaction, on a branch that always returns `ErrSeniorNotFound` to the caller. A request reported as "failed" therefore has a persisted side effect. Task 6's scope was conversion-only (no behavior change), so this was left as-is; still worth a follow-up (move the auto-provision inside the transaction, or drop it) even though it doesn't change the class-B verdict above.

---

## atlas-marriages

### Write inventory

| file:line | Entry point | Writes | Class | Flags |
|---|---|---|---|---|
| `marriage/processor.go:266,273,292` (inside `AcceptProposal`, called from `AcceptProposalAndEmit` at `processor.go:314`) | Kafka command `MarriageAccept` — **the live, wired path** (`kafka/consumer/marriage/consumer.go:154` `handleAccept`) | `proposals` (`administrator.go:52` `UpdateProposal`) + `marriages` (`administrator.go:83` `CreateMarriage`, `administrator.go:99` `UpdateMarriage`) — 2 tables, 3 writes | **A — genuinely unwrapped** | **no [T]** — plain sequential `p.db.WithContext(p.ctx)` calls, no `Begin`/`Transaction` anywhere in this call chain |
| `marriage/processor.go:1609,1616,1635` (inside `executeInTransaction`, reached only via `AcceptProposalWithTransactionAndEmit` at `processor.go:1581`) | **Dead code** — `grep -rn "AcceptProposalWithTransactionAndEmit" services/atlas-marriages` shows zero callers outside its own definition/interface declaration | `proposals` + `marriages` (identical writes to the live path above) | A | [T][E] — manual `Begin`(`:1690`)/`Rollback`(`:1708`)/`Commit`(`:1713`); `message.EmitWithResult` buffer built at `:1606`, `buf.Put` calls at `:1655,1670`, actual publish (`kafka/message/message.go:55` inside `EmitWithResult`) happens as part of `f(buf)(input)` returning at `processor.go:1682`, which completes **before** `tx.Commit()` at `:1713` — confirmed publish-before-commit |
| `marriage/administrator.go:36` `CreateProposal` | `handlePropose` (`kafka/consumer/marriage/consumer.go:90`) → `ProposeAndEmit` | `proposals` | C | no [T] needed |
| `marriage/administrator.go:140` `CreateCeremony` | `handleScheduleCeremony` (`consumer.go:280`) → `ScheduleCeremonyAndEmit` | `ceremonies` | C | no [T] needed |
| `marriage/administrator.go:156` `UpdateCeremony` | Start/Complete/Cancel/Postpone/Reschedule/AddInvitee/RemoveInvitee/AdvanceState ceremony handlers | `ceremonies` | C | no [T] needed |
| `marriage/administrator.go:52,99` (standalone uses outside the Accept flow — e.g. Divorce, Decline, Cancel, Expire, character-deletion cascade) | various Kafka consumers | `proposals` / `marriages` | C | no [T] needed |

### Exclusions (non-DB write-verb hits)

| file:line | Reason |
|---|---|
| `scheduler/proposal_expiry.go:126` | `tenant.Create(tenantId, "background-scheduler", 1, 0)` — builds an in-memory `tenant.Model` via `libs/atlas-tenant/processor.go:30` (`Creator(...)` + `model.FixedProvider`), no DB/network access. |
| `scheduler/ceremony_timeout.go:126` | Same `tenant.Create` bootstrap pattern. |
| `test/tenant.go:10` | Test helper, excluded. |

### Verdicts

- **`AcceptProposalAndEmit` (live path, `processor.go:314`, wired via `consumer.go:154`): class A, genuinely unwrapped.** This corrects the design-phase hypothesis, which characterized marriages as already-transactional via manual `Begin`/`Commit`. That manual-tx code (`executeInTransaction`/`AcceptProposalWithTransactionAndEmit`) existed but was **dead code — zero callers found anywhere in the service, including tests**. The actual production accept-flow performed the identical 2-table/3-write sequence (`UpdateProposal` → `CreateMarriage` → `UpdateMarriage`) with no transaction wrap at all: a crash between any two of the three writes left a proposal marked accepted with no (or a half-updated) marriage. Kafka emission in the live path (`message.Emit` at `processor.go:321`, after `AcceptProposal()` returns at `:315`) already happened after all writes complete, so there was no [E] defect on the live path — the defect was the missing transaction wrap itself. `_(commit: "fix(atlas-marriages): wrap live proposal-accept writes in ExecuteTransaction, delete dead manual-tx twin" — Task 9, done)_` — `AcceptProposal`'s body (`processor.go:242`) now wraps its three writes (`GetProposalByIdProvider`, `UpdateProposal`, `CreateMarriage`, `UpdateMarriage`, all now bound to `tx`) in `database.ExecuteTransaction`. A regression test (`TestAcceptProposal_RollsBackProposalUpdateWhenMarriageCreateFails`, `marriage/processor_rollback_test.go`) forces the `marriages` create to fail and confirms the `proposals` update rolls back to pending — RED (proposal left `Accepted`, 0x1) before the fix, GREEN after. The dead `AcceptProposalWithTransactionAndEmit`/`executeInTransaction` pair (and its `Processor` interface declaration) was deleted rather than retired, so there is now a single accept-flow implementation. `AcceptProposalAndEmit` (`processor.go:325`) is unchanged — it still calls `AcceptProposal(...)()` then publishes via `message.Emit`, which is publish-after-commit both before and after this fix; its stale "single transaction" comment was reworded to reflect that the DB transaction is already committed by the time it runs.
- Dead-code twin (`executeInTransaction`): structurally **A, [T][E]** (manual Begin/Commit, publish-before-commit) but unreachable in production. Recorded for completeness; remediation is to delete it as part of Task 9 rather than "fix" unreachable code.
- All other single-table CRUD (`CreateProposal`, `CreateCeremony`, `UpdateCeremony`, standalone `UpdateProposal`/`UpdateMarriage`): **C** — no change.

---

## atlas-monster-book

### Write inventory

| file:line | Entry point | Writes | Class | Flags |
|---|---|---|---|---|
| `kafka/consumer/monsterbook/consumer.go:56` `handleCardPickedUp` | Kafka command, card picked up | `monster_book_cards` (`card/administrator.go:43,60-64,69-73` insert/update via `upsertCard`) + `monster_book_collections` (`collection/administrator.go:46` insert via `upsertStats`, called from `RecomputeAndEmit`) — 2 tables | A | [T][E] |
| `kafka/consumer/character/consumer.go:49` `handleStatusEventDeletedFunc` | Kafka character-deleted status event | `monster_book_cards` (`card/administrator.go:78` `deleteByCharacter`) + `monster_book_collections` (`collection/administrator.go:91` `deleteByCharacter`) — 2 tables, delete-only cascade | B* (see note) | [T], no [E] |
| `collection/administrator.go:59-63` `setCover` | `SetCoverAndEmit` (`collection/processor.go:236`) ← Kafka `handleSetCover` (`kafka/consumer/monsterbook/consumer.go:79-88`) **and** REST `PATCH /characters/{id}/monster-book` (`character/resource.go:76`) | `monster_book_collections` — single update | C | no [T] needed |

*Note: `handleStatusEventDeletedFunc` writes 2 tables (`monster_book_cards`, `monster_book_collections`) via delete-only statements; taxonomy-wise this satisfies class A's "≥2 writes, ≥2 tables" definition. Recorded as A in the closing matrix; listed as B* here only to flag that both writes are the same verb (delete) with no cross-referential-integrity risk beyond "both must go together," a lighter-weight case than the create/create+update mix in `handleCardPickedUp`.

### Exclusions (non-DB write-verb hits)

None — every write-verb hit in this service is a genuine `monster_book_cards`/`monster_book_collections` write.

### Verdicts

- `handleCardPickedUp`: **A**. **Superseded finding**: the `[E]` defect described above (`message.Emit(producer.ProviderImpl(l)(ctx))(...)` wrapping a raw `db.Transaction`, publish-before-commit) was the pre-rebase state this survey originally documented. Since then, **task-114's fleet-wide transactional-outbox migration (merged to main, `d2e13ba3d`) already rewrote this handler** to the canonical composition: `database.ExecuteTransaction(db.WithContext(ctx), func(tx) error { return message.Emit(outbox.EmitProvider(l, ctx, tx))(...) })` (`kafka/consumer/monsterbook/consumer.go:57-73`) — `ExecuteTransaction` outer, outbox enqueue-in-tx inner. The card write, collection write, and outbox-row enqueue commit atomically on `tx`; `message.Emit` (`kafka/message/message.go:33-48`) returns before flushing on any inner error, so a failed collection write both rolls back the card upsert and never enqueues the outbox row. This already satisfies both the `[T]` conversion and the `[E]` fix — no code change was needed or made here. Task 8's actual contribution: a new rollback test (`consumer_rollback_test.go`) that locks this atomicity by asserting `monster_book_cards` stays empty when the `monster_book_collections` write is forced to fail. That atomicity only became *real* once task-119 Task 1 fixed `database.ExecuteTransaction` (previously a no-op per `bug_execute_transaction_noop`) — before Task 1, this same code would not have rolled back. `_(commit: "refactor(atlas-monster-book): convert cascade-delete to ExecuteTransaction + lock CARD_PICKED_UP atomicity" — Task 8, done; rollback test only, handler code untouched)_`
- `handleStatusEventDeletedFunc`: **A** (2 tables, delete-only), **converted** from raw `db.WithContext(ctx).Transaction(...)` to `database.ExecuteTransaction(db.WithContext(ctx), ...)` (`kafka/consumer/character/consumer.go:49`), **no emit inside this handler at all** (no `message`/producer import used for publishing in this file beyond the kafka handler-adapter types) — this corrects the design-phase assumption that both monster-book consumer sites share the same `[E]` defect; this second site is clean. Wrapper conversion only, closure body unchanged. `_(commit: "refactor(atlas-monster-book): convert cascade-delete to ExecuteTransaction + lock CARD_PICKED_UP atomicity" — Task 8, done)_`
- `setCover`: **C** — no change; correctly unwrapped single-statement update, reachable from both a Kafka command and a REST PATCH with no additional writes in either path.

---

## atlas-storage

**No raw `db.Transaction`/`ExecuteTransaction`/`.Begin(`/`WithTransaction` hits anywhere in this service** — confirmed via `grep -rn "db.Transaction\|ExecuteTransaction\|\.Begin(\|WithTransaction" services/atlas-storage --include='*.go' | grep -v _test.go` (zero output). This is the substantive finding of the whole audit: every multi-write flow below runs fully unwrapped.

Tables: `storages` (`storage/entity.go:18-20`), `storage_assets` (`asset/entity.go:55-57`).

### Write inventory

| file:line | Entry point | Writes | Class | Flags |
|---|---|---|---|---|
| `storage/processor.go:720-760` `ExpireAndEmit` | Item-expiry flow (NPC/UI-driven; not a per-packet hot path) | `storage_assets` — delete expiring asset (`:729`) → **emit** (`:735`, mid-flow, before the replacement branch) → conditionally create replacement asset (`:757`); same table twice, separated by an externally-visible Kafka publish | **A** (same-table-twice-with-intervening-externally-visible-emit is treated as A, not B, per the elevated risk: a crash after `:729` leaves the asset gone and an EXPIRED event already published/in-flight with no replacement created) | unwrapped — no [T] |
| `storage/processor.go:514` `MergeAndSort` (+ `sortAssets` helper) | Storage-merge flow (NPC/UI-driven) | `storage_assets` — per stackable group: N `UpdateQuantity` + N `Delete` calls, then a second independent pass of `UpdateSlot` calls across all assets; single table, unbounded count of write statements, early-return-on-error leaves prior groups already committed and later groups untouched | **B** (multi-statement, single table, no atomicity — practical risk equals a class-A finding despite being one table) | **remediated — `[T]`, Task 12**: all writes now run inside one `database.ExecuteTransaction`; `getSlotMaxByTemplateId` (atlas-data REST) is prefetched into a `map[templateId]slotMax` before the tx opens, so no network I/O runs inside the tx |
| `asset/processor.go:50-78` `GetOrCreateStorageId` | REST `GET /storage/accounts/{accountId}/assets` (`asset/resource.go:34`) | `storage_assets` — `First` (read) then exactly one `Create` (`:70`) if not found | **C + mandatory race annotation** | unwrapped |
| `storage/processor.go:42-51` `GetOrCreateStorage` | `Deposit`, `Accept`, `ExpireAndEmit`'s replacement branch, REST `handleGetStorageRequest`/`handleCreateStorageRequest` | `storages` — `First` (read) then exactly one `Create` if not found | **C + mandatory race annotation** (same shape as above, different table/package) | unwrapped |
| `storage/administrator.go:20` `Create` | `CreateStorage` (REST) | `storages` | C | — |
| `storage/administrator.go:29-35` `UpdateMesos` | `UpdateMesosAndEmit` ← Kafka command (`kafka/consumer/storage/consumer.go:94`) | `storages` | C | — |
| `storage/administrator.go:47-51` `Delete` | `DeleteByAccountId` (`storage/processor.go:830`) ← Kafka character-deleted event (`account/consumer.go:46`), paired in the same call with `asset.DeleteByStorageId` below | `storages` | part of the class-A pairing below | **remediated — `[T]`, Task 12b**: both writes now run inside one `database.ExecuteTransaction` |
| `asset/administrator.go:61-65` `DeleteByStorageId` | Same `DeleteByAccountId` call as above | `storage_assets` | **A** (2 tables, `storages`+`storage_assets`, one logical delete-by-account operation) | **remediated — `[T]`, Task 12b**: per-storage `asset.DeleteByStorageId` + `Delete` loop now runs inside one `database.ExecuteTransaction`; a failed storage delete rolls back that storage's already-executed asset delete |
| `asset/administrator.go:47` `Create` | `Deposit` (`storage/processor.go:109`) | `storage_assets` | C (sub-step; multi-write risk already captured under `ExpireAndEmit`/`MergeAndSort` where relevant) | — |
| `asset/administrator.go:55-58` `Delete` | `Withdraw`/`Release`/`MergeAndSort` loop/`ExpireAndEmit` | `storage_assets` | C standalone; part of A/B above where applicable | — |
| `asset/administrator.go:75-81` `UpdateQuantity` | `Withdraw` (partial), `Accept` (merge), `Release` (partial), `MergeAndSort` loop | `storage_assets` | C standalone; part of B (`MergeAndSort`) where applicable | — |
| `asset/entity.go:61` raw `db.Exec` `UPDATE storage_assets SET flag = flag \| ...` | bitwise flag-update helper | `storage_assets` | C (single statement) | — |
| `storage/administrator.go:38-44` `UpdateCapacity` | **no call site found** in any processor/consumer/resource file read | `storages` | N/A — dead/unreferenced code, not a live write path | — |

### Exclusions (non-DB write-verb hits)

| file:line | Reason |
|---|---|
| `kafka/consumer/storage/consumer.go:145,172` | `projection.GetManager().Create(...)`/`.Delete(...)` — `projection.Manager` (`projection/manager.go:21-30,52-73`) is backed by an `atlas-redis` `TenantRegistry`, a Redis-backed session cache, not `*gorm.DB`. |
| `kafka/consumer/compartment/consumer.go:104,136` | `projection.GetManager().Update(...)` — same Redis-backed cache. |
| `kafka/consumer/character/consumer.go:91` | `projection.GetManager().Delete(...)` — same Redis-backed cache. |
| `kafka/consumer/storage/consumer.go:168`, `kafka/consumer/character/consumer.go:87` | `storage.GetNpcContextCache()...Remove(characterId)` — in-memory/legacy NPC-context cache by usage shape (not opened line-by-line; consistent with the projection cache pattern), not a DB write. |

### Verdicts

- `ExpireAndEmit`: **A**, remediated — delete + replacement-create now run inside one `database.ExecuteTransaction`, and `emitExpiredEvent` is published strictly after commit (moved out of the mid-flow position it previously held). **Failure-mode change (intentional, per design §context.md decision 6):** previously a failed replacement-create was Warn-and-continue — the delete stayed committed and the EXPIRED event had already been published mid-flow, desyncing the pair. Now every error branch inside the transaction closure returns the error, rolling back the delete; `ExpireAndEmit` returns the error to the caller and the event is never published for a rolled-back expiry. Locked by `TestExpireAndEmit_RollsBackDeleteWhenReplacementCreateFails` (`storage/processor_rollback_test.go`), which forces the replacement `asset.Create` to fail via `databasetest.FailWritesOn` and asserts both an error return and that the deleted asset is restored (`storage_assets` count == 1). `_(commit: "fix(atlas-storage): expire+replace is one transaction, event publishes after commit" — Task 11, done)_`
- `MergeAndSort`: **B**, remediated — the merge loop (`UpdateQuantity`/`Delete`) and the `sortAssets` re-slotting pass (`UpdateSlot`) now run inside one `database.ExecuteTransaction`. `getSlotMaxByTemplateId` (atlas-data REST call per unique templateId) is hoisted into a `slotMaxByTemplate` map built before the transaction opens — genuine RED→GREEN TDD confirmed the gap first (forcing `storage_assets` deletes to fail left the first stack's quantity update committed as 60 while the second stack stayed at 30, i.e. `[60 30]` instead of the pre-merge `[30 30]`), then the restructure made the delete failure roll back the earlier quantity update in the same group. Tx bound: at most storage-capacity rows (~100) per call, and zero network I/O inside the tx (design §8 requirement). `sortAssets` signature changed to `func (p *ProcessorImpl) sortAssets(db *gorm.DB, assets []asset.Model) error` — its only two call sites are both inside `MergeAndSort`, no external callers affected. Locked by `TestMergeAndSort_RollsBackQuantityUpdatesWhenDeleteFails` (`storage/processor_rollback_test.go`). `_(commit: "fix(atlas-storage): MergeAndSort merge/compact/sort writes are one transaction" — Task 12, done)_`
- `GetOrCreateStorageId` / `GetOrCreateStorage`: **C + mandatory race annotation** — not class B (exactly one write each). Race: `world_id`+`account_id`(+`tenant_id`) is a `uniqueIndex` on `storages` (`storage/entity.go:10-13`), but the SELECT-then-INSERT here has no row lock and no serializable transaction; two concurrent first-time accesses for the same key can both miss the `First` and both attempt `Create`, with the loser surfacing a raw unique-constraint-violation 500 rather than a graceful retry-as-lookup. Closing this needs a DB-level fix (unique constraint is already present; add conflict-safe upsert or retry-on-conflict logic), not a transaction wrap — documented, not fixed here per the refined taxonomy. `_(commit: "feat(atlas-storage): WithTransaction plumbing; GetOrCreateStorageId joins caller transactions" — Task 10, done)_` — Task 10 added the `WithTransaction` plumbing and wrapped `GetOrCreateStorageId` in `ExecuteTransaction` (join-capable); the race annotation is documented here and needs a schema/upsert change (out of scope), not a transaction wrap.
- `storage.Delete` + `asset.DeleteByStorageId` (account-deletion cascade, `DeleteByAccountId`): **A**, remediated — the per-storage `asset.DeleteByStorageId` + `Delete` loop in `DeleteByAccountId` (`storage/processor.go:830`) now runs inside one `database.ExecuteTransaction`; the initial `GetByAccountId` read stays outside the tx. **Failure-mode change:** previously each delete ran on the raw handle with Warn/Error-and-continue and the function unconditionally returned `nil`, so a mid-cascade failure left storage data half-deleted while reporting success; now a failure in either delete returns the error and rolls back that storage's asset deletes (and any earlier storages in the same call, since the whole loop is one transaction). Locked by `TestDeleteByAccountId_RollsBackAssetDeletesWhenStorageDeleteFails` (`storage/processor_rollback_test.go`), which forces the storage `Delete` to fail via `databasetest.FailWritesOn` and asserts both an error return and that the storage row + its asset survive (`storages` count == 1, `storage_assets` count == 1). `_(commit: "fix(atlas-storage): wrap account-deletion cascade (DeleteByAccountId) in one transaction" — Task 12b, done)_`
- All other single-statement administrator functions: **C** — no change.
- `UpdateCapacity`: unreferenced dead code — footnote only, not a transaction-coverage finding.

---

## atlas-account

### Write inventory

| file:line | Entry point | Writes | Class | Flags |
|---|---|---|---|---|
| `account/administrator.go:18` `create` | `GetOrCreate` (`processor.go:141-153`, RMW) ← `AttemptLogin` auto-register; also standalone `Create` (`processor.go:153`) ← REST create handler | `accounts` | **C + mandatory race annotation** | — |
| `account/administrator.go:36` `update` | `Update` (`processor.go:189` — actually the RMW body reading `GetById` then diffing changed columns) ← REST `PATCH` (`resource.go:50`) | `accounts` | C (single statement; minor documented RMW note, not a required-fix race) | — |
| `account/administrator.go:43` `deleteById` | `Delete` (`processor.go:238`) ← `DeleteAndEmit`, guarded by a login-state check via `GetById` | `accounts` | C (single statement; minor documented TOCTOU note) | — |

`account/entity.go` confirms `Entity.Name string` has **no `unique`/`uniqueIndex` gorm tag** — the race below is real, not hypothetical.

### Exclusions (non-DB write-verb hits)

None — `processor.go:153,238` and `resource.go:50` are entry-point delegators into the three administrator writes above, not separate writes.

### Verdicts

- `GetOrCreate`→`Create`: **C + mandatory race annotation.** `GetByName(name)` is checked for `ErrRecordNotFound` before falling through to `Create`; two concurrent auto-register attempts for the same `name` can both observe "not found" and both `INSERT`, producing duplicate `accounts` rows with the same name. **This is not closeable by a transaction wrap alone** — a transaction does not prevent two concurrent transactions from both reading "no row" under normal isolation; the actual fix is a unique index on `accounts.name` (schema change, out of scope for this task, documented per design §5.2's refinement). Re-confirmed post-rebase: `account/processor.go:144-156` (`GetOrCreate`) and `account/administrator.go:18` (`create`, single `db.Create` statement) unchanged in shape. `_(commit: "fix(atlas-maps): atomic character-deletion cleanup; finalize account/ban class-C verdicts (task-119)" — Task 13, done)_` (documents the annotation; no code change).
- `Update`: **C** — single statement. Documented note: concurrent `Update` calls targeting the *same* field (e.g. two concurrent `pinAttempts` bumps) can lose an update under the current read-then-diff-then-write shape; not required to fix here (single write, no rollback test is definable), noted for awareness.
- `Delete`: **C** — single statement. Documented note: the preceding login-state check (`GetById`) is not re-verified atomically against the delete (`DELETE ... WHERE id=?` has no state predicate); low blast radius (administrative/rare path), and closing it would require moving login-state out of the in-process registry and into the row itself — out of scope.

---

## atlas-ban

### Write inventory

| file:line | Entry point | Writes | Class | Flags |
|---|---|---|---|---|
| `ban/administrator.go:26` `create` | `CreateAndEmit`/`Create` (`ban/processor.go:63`) ← Kafka command `CommandTypeCreate` (`kafka/consumer/ban/consumer.go:44`) | `bans` | C | — |
| `ban/administrator.go:37` `deleteById` | `Delete` (`ban/processor.go:86`) ← Kafka command `CommandTypeDelete` (`consumer.go:66`) | `bans` | C | — |
| `ban/administrator.go` `updateExpiresAt` | `ExpireBanAndEmit` ← REST | `bans` | C (RMW: `GetById` + `Permanent()` guard, one write) | — |
| `history/administrator.go:22` `create` | `Record` (`history/processor.go:38`) ← Kafka `account_session_status_event` (`kafka/consumer/account/consumer.go:41,53`) | `login_history` | C | — |
| `ban/task.go:31` ticker delete | `ExpiredBanCleanup.Run()` (registered `main.go:78`, own goroutine per `tasks/task.go:16-27`) | `bans` | D | single-statement sweep, independent of history |
| `history/task.go:33` ticker delete | `HistoryPurge.Run()` (registered `main.go:79`, own goroutine) | `login_history` | D | single-statement sweep, independent of bans |

### Exclusions (non-DB write-verb hits)

None — REST/resource files delegate to the same processor/administrator writes already inventoried above, not separate writes.

### Verdicts

- **`ban`↔`history` pairing: refuted.** Design survey flagged this as an open question ("possible ban↔history pairing to confirm"). Traced both call chains fully: `kafka/consumer/ban/consumer.go:42-61` (`handleCreateBanCommand`, triggered by the `ban_command` Kafka topic) calls `ban.Processor.CreateAndEmit`→`Create`→`create` (`ban/administrator.go:26`) with **no reference anywhere in `ban/processor.go` or `ban/administrator.go` to the `history` package** (confirmed via import block and `grep -rln "atlas-ban/ban" services/atlas-ban/atlas.com/ban/history/*.go` → no hits, and vice versa). History rows are written exclusively by `kafka/consumer/account/consumer.go:48,61` (`handleCreatedSessionEvent`/`handleErrorSessionEvent`), triggered by the **separate** `account_session_status_event` Kafka topic emitted by **atlas-account** on login attempts — an entirely independent trigger from ban creation/deletion. Registered as two independent consumers in `main.go` subscribing to different topics. **No entry point writes both a `bans` row and a `login_history` row.** Both remain class C, single-table, single-statement. Re-confirmed post-rebase with updated line numbers: `ban/processor.go:65` (`p.Create` inside `CreateAndEmit`) and `:88` (`p.Delete` inside `DeleteAndEmit`) each write only `bans`; `kafka/consumer/account/consumer.go:48,61` each write only `login_history` via `history.NewProcessor(...).Record(...)`.
- All ban/history CRUD: **C** — no change. `_(commit: "fix(atlas-maps): atomic character-deletion cleanup; finalize account/ban class-C verdicts (task-119)" — Task 13, done)_` documents this refutation formally; no code change needed since both are already correctly single-statement.
- Delete-sweep tickers (`ban/task.go`, `history/task.go`): **D** — justified; single-statement, independently-scheduled background sweeps with no cross-table coupling.

---

## atlas-maps

### Write inventory

| file:line | Entry point | Writes | Class | Flags |
|---|---|---|---|---|
| `character/location/administrator.go:19` `upsertLocation` (`db.Save`) | 5 call sites in `kafka/consumer/character/consumer.go` (login/map-change/channel-change events) + `channel_change_request.go:27` + `character/warp/processor.go:57`, all via `location.Processor.Set()` | `character_locations` | C | — |
| `kafka/consumer/character/consumer.go` `handleStatusEventDeletedFunc` → `location.NewProcessor(fl, ctx, tx).Delete(event.CharacterId)` | Kafka character-deleted status event | `character_locations` (`character/location/administrator.go:27-34` `deleteLocation`) | part of the class-A pairing below | **remediated — `[T]`, Task 13** |
| `kafka/consumer/character/consumer.go` (same `handleStatusEventDeletedFunc`) → `visit.NewProcessor(fl, ctx, tx).DeleteByCharacterId(...)` | Same Kafka character-deleted status event | `character_map_visits` (`visit/administrator.go:24-31` `deleteByCharacterId`) | **A** (2 tables, `character_locations` + `character_map_visits`, one logical character-deletion cascade) | **remediated — `[T]`, Task 13**: both deletes now run inside one `database.ExecuteTransaction` |
| `kafka/consumer/mist/consumer.go:64` `processorFactory(...).Create(c.Body)` | Kafka mist-spawn event | *(see exclusions — not a DB write)* | — | — |
| `visit/administrator.go:9-22` `recordVisit` (`FirstOrCreate`) | `map/processor.go:66` character map-entry flow | `character_map_visits` | C | backed by a genuine DB unique index `idx_visits_tenant_char_map` (`visit/entity.go:13-15`) on `(tenant_id, character_id, map_id)` — unlike account's `GetOrCreate`, concurrent racers get a constraint violation rather than silent duplication; **not** a race-annotation case |

### Exclusions (non-DB write-verb hits)

| file:line | Reason |
|---|---|
| `map/weather/registry.go:81` | `Registry.Delete(key)` — in-memory only: `sync.RWMutex` + `map[FieldKey]WeatherEntry` singleton (`registry.go:22-34`), no `gorm`/`db` import in the file. |
| `kafka/consumer/mist/consumer.go:64` | `processorFactory(...).Create(c.Body)` — `mist/processor.go`/`mist/registry.go` have zero `gorm`/`db` references; `mist/registry.go` is a `sync.RWMutex`-protected `map[string]*tenantBucket` singleton, same in-memory pattern as `weather`. |

### Verdicts

- `character_locations` upsert (`upsertLocation`), `character_map_visits` `recordVisit`: **C** — no change; `recordVisit`'s FirstOrCreate is race-safe via a genuine unique DB index (positive contrast to atlas-account's unguarded `GetOrCreate`).
- **`handleStatusEventDeletedFunc`'s dual delete (visits then location): class A, remediated** — this is a real finding not in the design-phase survey (which characterized maps as purely class C). Both deletes target different tables in one logical "character removed, clean up map state" operation; previously executed sequentially with independent error-and-continue handling and no transaction, so one could succeed while the other failed, leaving `character_locations` and `character_map_visits` inconsistent for a deleted character. Both writes now run inside one `database.ExecuteTransaction(db.WithContext(ctx), func(tx *gorm.DB) error {...})`, threading `tx` into both `visit.NewProcessor` and `location.NewProcessor`, with each error returned (no more error-and-continue) so a failure rolls back both. `mapcharacter.NewProcessor(fl, ctx).ExitAll(event.CharacterId)` (in-memory registry cleanup, not a DB write) stays outside and after the tx block, unconditional, preserving prior behavior. Locked by `TestHandleStatusEventDeleted_RollsBackVisitDeleteWhenLocationDeleteFails` (`kafka/consumer/character/consumer_rollback_test.go`), which forces the `character_locations` delete to fail via `databasetest.FailWritesOn` and asserts the visit row survives (`character_map_visits` count == 1). `_(commit: "fix(atlas-maps): atomic character-deletion cleanup; finalize account/ban class-C verdicts (task-119)" — Task 13, done)_`
- In-memory registry hits (`weather`, `mist`): excluded, not DB writes.

---

## atlas-map-actions

### Write inventory

| file:line | Entry point | Writes | Class | Flags |
|---|---|---|---|---|
| `script/administrator.go:21` `createMapScript` | REST `POST /maps/actions` (`resource.go:109`) | `map_scripts` | C | live REST CRUD, independent of the seeder cycle below |
| `script/administrator.go:48` `updateMapScript` | REST `PATCH /maps/actions/{scriptId}` (`resource.go:141`) | `map_scripts` | C | — |
| `script/administrator.go:71` `deleteMapScript` | REST `DELETE /maps/actions/{scriptId}` (`resource.go:166`) | `map_scripts` | C | — |
| `script/administrator.go:78,86-88` `deleteAllMapScripts`/`DeleteAllByType` | Seeder subdomains `OnUserEnterSubdomain`/`OnFirstUserEnterSubdomain` (`subdomain_on_user_enter.go:29`, `subdomain_on_first_user_enter.go:28`) ← `libs/atlas-seeder/seed.go:87` via `POST /maps/actions/seed` | `map_scripts` (scoped by `script_type`) | D | seeder cycle |
| `script/administrator.go:105` `BulkCreate` | Same two seeder subdomains (`subdomain_on_user_enter.go:45-47`, `subdomain_on_first_user_enter.go:44-46`) ← `libs/atlas-seeder/seed.go:112` | `map_scripts` | D | seeder cycle |

### Exclusions (non-DB write-verb hits)

| file:line | Reason |
|---|---|
| `script/executor.go:88,116,190,211,245` | `e.sagaP.Create(s)` — `mapactionsaga.ProcessorImpl.Create` (`saga/processor.go:28-30`) is a **Kafka producer publish** to `EnvCommandTopic` (`producer.ProviderImpl(...)`), consumed asynchronously by atlas-saga-orchestrator. **Correction to the design-phase hypothesis**, which labeled this "REST to saga-orchestrator" — verified mechanism is Kafka, not HTTP; the exclusion classification (not a DB write) still holds. |
| `script/resource.go:109,141,166` | REST handler delegators into `administrator.go` writes already inventoried above. |

### Verdicts

- Live REST single-script CRUD (`createMapScript`/`updateMapScript`/`deleteMapScript`): **C** — no change; reachable independently of seeding.
- Seeder-cycle delete-all + bulk-create: **D** — justified per shared `libs/atlas-seeder` semantics (`seed.go:22-39,85-120`); out-of-scope follow-up candidate if `atlas-seeder` itself is ever revisited, not a task here.

---

## atlas-portal-actions

### Write inventory

| file:line | Entry point | Writes | Class | Flags |
|---|---|---|---|---|
| `script/administrator.go:21` `createPortalScript` | REST `POST /portals/scripts` (`resource.go:110-142`) | `portal_scripts` | C | live REST CRUD |
| `script/administrator.go:52` `updatePortalScript` | REST `PATCH /portals/scripts/{scriptId}` (`resource.go:145-178`) | `portal_scripts` | C | — |
| `script/administrator.go:76` `deletePortalScript` | REST `DELETE /portals/scripts/{scriptId}` (`resource.go:181-196`) | `portal_scripts` | C | — |
| `script/subdomain.go:28` `PortalSubdomain.DeleteAllForTenant` | seeder cycle ← `libs/atlas-seeder/seed.go:87` via `POST /portals/scripts/seed` | `portal_scripts` | D | seeder cycle; `script/administrator.go:83` `deleteAllPortalScripts` is defined but **unreferenced** — the seeder uses its own inline delete in `subdomain.go`, not this function |
| `script/subdomain.go:83` `PortalSubdomain.BulkCreate` | seeder cycle ← `seed.go:112` | `portal_scripts` | D | seeder cycle |

### Exclusions (non-DB write-verb hits)

| file:line | Reason |
|---|---|
| `script/executor.go:116,163,198,247,291,360,429,473,507,558,587,621,663` | `e.sagaP.Create(s)` — Kafka producer publish (`portalsaga.ProcessorImpl.Create`, `saga/processor.go:28-30`, `EnvCommandTopic`), not REST, not a DB write. Same correction as map-actions. |
| `script/resource.go:121,157,185` | REST handler delegators, not separate writes. |

### Verdicts

- Live REST CRUD: **C** — no change.
- Seeder cycle: **D** — justified, same shared-lib semantics. `script/administrator.go:83` (`deleteAllPortalScripts`) is dead code — flagged as an inconsistency vs. map-actions (which does route through `administrator.go`), not a transaction-coverage finding.

---

## atlas-reactor-actions

### Write inventory

| file:line | Entry point | Writes | Class | Flags |
|---|---|---|---|---|
| `script/administrator.go:21` `createReactorScript` | REST `POST /reactors/actions` (`resource.go:109-142`) | `reactor_scripts` | C | live REST CRUD |
| `script/administrator.go:52` `updateReactorScript` | REST `PATCH /reactors/actions/{scriptId}` (`resource.go:145-178`) | `reactor_scripts` | C | — |
| `script/administrator.go:75` `deleteReactorScript` | REST `DELETE /reactors/actions/{scriptId}` (`resource.go:181-196`) | `reactor_scripts` | C | — |
| `script/subdomain.go:28` `ReactorSubdomain.DeleteAllForTenant` | seeder cycle ← `seed.go:87` via `POST /reactors/actions/seed` | `reactor_scripts` | D | seeder cycle; `script/administrator.go:82` `deleteAllReactorScripts` is defined but **unreferenced**, same pattern as portal-actions |
| `script/subdomain.go:90` `ReactorSubdomain.BulkCreate` | seeder cycle ← `seed.go:112` | `reactor_scripts` | D | seeder cycle |

### Exclusions (non-DB write-verb hits)

| file:line | Reason |
|---|---|
| `script/executor.go:180,241,346,396,427,473,497` | `e.sagaP.Create(s)` — Kafka producer publish (`reactorsaga.ProcessorImpl.Create`, `saga/processor.go:28-30`), confirmed zero `db`/`gorm` usage anywhere in `executor.go`. Same correction as map-actions/portal-actions. |
| `script/resource.go:121,157,185` | REST handler delegators, not separate writes. |

### Verdicts

- Live REST CRUD: **C** — no change.
- Seeder cycle: **D** — justified, same shared-lib semantics.

---

## atlas-party-quests

### Write inventory

| file:line | Entry point | Writes | Class | Flags |
|---|---|---|---|---|
| `definition/administrator.go:20` `createDefinition` | REST `POST /party-quests/definitions` (`resource.go:114`) | `definitions` | C | live REST CRUD |
| `definition/administrator.go:46` `updateDefinition` | REST `PATCH /party-quests/definitions/{id}` (`resource.go:145`) | `definitions` | C | — |
| `definition/administrator.go:67` `deleteDefinition` | REST `DELETE /party-quests/definitions/{id}` (`resource.go:169`) | `definitions` | C | — |
| `definition/administrator.go:73` `deleteAllDefinitions` | `definition/subdomain.go:59`-equivalent `DeleteAllForTenant` ← `libs/atlas-seeder/seed.go:87` via `POST /party-quests/definitions/seed` | `definitions` | D | seeder cycle |
| `definition/subdomain.go:59` `BulkCreate` | seeder cycle ← `seed.go:112` | `definitions` | D | **per-model loop of individual `db.Create` calls, not one bulk-slice INSERT** — a mid-loop error leaves a partial subset of that file's definitions committed with no rollback within the loop itself, a slightly worse partial-write shape than the atomic-bulk-insert used by map/portal/reactor-actions, but still governed by the same `runSubdomain` per-file continue-on-error accounting (`seed.go:106-116`) — same class-D justification applies |

### Exclusions (non-DB write-verb hits)

| file:line | Reason |
|---|---|
| `instance/processor.go:271,321,355,454,526,670,735,757,802,846,883,945,987,1001,1122,1452,1499,1539` (and all other `GetRegistry()` call sites in the file, 43 total including reads) | **Entire `instance/processor.go` state machine is in-memory, not DB-backed.** `instance/registry.go:19-22` — `Registry{ lock sync.Mutex; tenants map[tenant.Model]*tenantData }`, singleton via `sync.Once`, no `gorm`/DB import. `instance/processor.go` does hold a `*gorm.DB` field, but every one of its 15 uses is a read-only quest-definition lookup (`definition.NewProcessor(...).ByIdProvider/ByQuestIdProvider`), never a write. |
| `definition/resource.go:114,145,169` | REST handler delegators into `administrator.go` writes already inventoried above. |

### Verdicts

- Live REST CRUD on `definitions`: **C** — no change.
- Seeder cycle (delete-all + per-file bulk-create): **D** — justified, shared-lib semantics; noted the per-model-loop-vs-bulk-insert variance as an informational difference from the sibling `-actions` services, not a separate remediation item.
- `instance/*`: entirely in-memory — excluded from DB-transaction scope; no class applies.

---

## atlas-saga-orchestrator

### Write inventory

| file:line | Entry point | Writes | Class | Flags |
|---|---|---|---|---|
| `saga/store.go:122-131` `Put` (existing-saga branch) | REST `POST /sagas` (`saga/resource.go:88`) and Kafka `kafka/consumer/saga/consumer.go:63`, both → `Processor.Put` → `store.Put` | `sagas` — single conditional `Model(&Entity{}).Where("transaction_id = ? AND version = ?", txId, ver).Updates(...)` | D | optimistic-version guard (in-memory `s.ver` map) is the concurrency mechanism, not a transaction |
| `saga/store.go:174-177` `Put` (new-saga branch) | Same entry points | `sagas` — single `Clauses(clause.OnConflict{...}).Create(&e)` (Postgres upsert) | D | DB-level `ON CONFLICT DO UPDATE` provides the atomicity a transaction would add; single statement |
| `saga/store.go:194-199` `Remove` | Saga-completion path | `sagas` — single unconditional status-update | D | single statement, no invariant to protect |
| `saga/store.go:240-246` `UpdateStatusFailed` | Saga-failure/compensation path | `sagas` — single unconditional status-update | D | single statement |
| `saga/store.go:299-304` `TryTransition` | Saga state-machine transitions | `sagas` — single conditional `Where("transaction_id = ? AND status = ?", ...).Updates(...)` (explicit compare-and-swap, comment at `store.go:286-289`) | D | textbook single-statement CAS — the guard *is* the concurrency control |

### Exclusions (non-DB write-verb hits)

| file:line | Reason |
|---|---|
| `main.go:197,249` | `tenant.Create(...)` — builds an in-memory `tenant.Model` via `libs/atlas-tenant/processor.go:19-32`, no DB/network I/O. |
| `saga/handler.go:1397` | `h.inviteP.Create(...)` — `invite.ProcessorImpl.Create` (`invite/processor.go:32-41`) is a Kafka command emission (`producer.ProviderImpl(...)(invite2.EnvCommandTopic)`), not a local DB write. |
| `saga/handler.go:2373` | `h.savedLocationP.Delete(...)` — `saved_location.ProcessorImpl.Delete` (`saved_location/processor.go:44-50`) calls `requests.DeleteRequest(url)` — a genuine outbound REST HTTP call to another service, not a local DB write. |

Full-service reconciliation: re-ran `grep -rn "\.Create(\|\.Save(\|\.Update(\|\.Updates(\|\.Delete(\|\.Exec(" services/atlas-saga-orchestrator --include='*.go' | grep -v _test.go` — exactly 5 hits (`handler.go:1397`, `handler.go:2373`, `store.go:177`, `main.go:197`, `main.go:249`), all accounted for above; the broader `.Updates(` pattern additionally surfaces `store.go:124,194,241,301` (also reconciled above). No other DB-write call sites exist in this service.

### Verdicts

- All `sagas`-table writes: **D** — intentionally non-atomic. The optimistic-version guard (`s.ver` in-memory map, compare-and-swap `WHERE version=?`/`WHERE status=?` clauses) is the concurrency mechanism; saga compensation (not a DB transaction) is how partial cross-service state is handled when a step fails. A transaction wrap adds nothing here (single statement per write, DB already guarantees per-row atomicity for the UPDATE/upsert). No remediation task — documented as-is.
- Outbound calls (`inviteP.Create`, `savedLocationP.Delete`): excluded, not DB writes.

---

## Closing 14-service matrix

**D0 — the enabling fix (`libs/atlas-database`).** Before any per-service wrap could matter, `database.ExecuteTransaction` had to actually open a transaction. Task 1 (`fix(atlas-database): ExecuteTransaction never opened a transaction`) replaced the always-true `isTransaction` heuristic (`db.Statement.ConnPool != nil`, true even on the root pool) with GORM's own `gorm.TxCommitter` type check. **Blast radius: this single fix activates real `BEGIN`/`COMMIT`/`ROLLBACK` at all ~53 existing `ExecuteTransaction` call sites across the ~18 services already using the helper** (including every flow task-114's outbox migration coupled to `ExecuteTransaction` — which were silently non-atomic in `main` until this lands; see `docs/architectural-improvements.md` CD-2 caveat). Every per-service row below is only genuinely atomic because of D0. Task 2 (`databasetest.FailWritesOn`) is the fault-injection helper every rollback test below depends on.

| Service | Classes found | Action | Remediation task | Status |
|---|---|---|---|---|
| atlas-keys | B ×4, all [T] | Standardize (`db.Transaction` → `ExecuteTransaction`) | Task 5 | `_(commit: "refactor(atlas-keys): standardize raw db.Transaction onto database.ExecuteTransaction" — Task 5, done)_` |
| atlas-families | B ×3, all [T] | Standardize | Task 6 | `_(commit: "refactor(atlas-families): standardize raw db.Transaction onto database.ExecuteTransaction" — Task 6, done)_`; informational flag on `AddJunior`'s untransacted auto-provision side effect (not remediated — no schema/behavior change was in scope for this task) |
| atlas-npc-conversations | A ×6 (npc pkg, [T]); C ×4 (quest pkg); D (seeder ×2) | Standardize the 6 [T] sites; no change for quest CRUD; seeder is out-of-scope follow-up candidate | Task 7 | `_(commit: "refactor(atlas-npc-conversations): standardize raw db.Transaction onto database.ExecuteTransaction" — Task 7, done)_`; completeness note on seeder-path recipe orphaning (not a transaction-coverage defect) |
| atlas-monster-book | A ×2 (`handleCardPickedUp` already `[T]`+`[E]`-fixed by task-114's outbox migration; `handleStatusEventDeletedFunc` `[T]` only) | Convert `handleStatusEventDeletedFunc` to `ExecuteTransaction`; add rollback test locking `handleCardPickedUp`'s already-correct atomicity (no handler code change — task-114 pre-empted the `[E]` fix) | Task 8 | `_(commit: "refactor(atlas-monster-book): convert cascade-delete to ExecuteTransaction + lock CARD_PICKED_UP atomicity" — Task 8, done)_` |
| atlas-marriages | A (live path — genuinely unwrapped, corrects design hypothesis); A [T][E] (dead-code twin) | Wrap the live `AcceptProposalAndEmit` writes in `ExecuteTransaction`; delete or retire the unreachable manual-tx twin | Task 9 | `_(commit: "fix(atlas-marriages): wrap live proposal-accept writes in ExecuteTransaction, delete dead manual-tx twin" — Task 9, done)_` |
| atlas-storage | A ×2 (`ExpireAndEmit`, account-deletion cascade), B ×1 (`MergeAndSort`), C+race ×2 (`GetOrCreateStorageId`, `GetOrCreateStorage`) | Wrap all 3 genuine gaps; document the 2 race annotations (no schema change in scope) | Tasks 10–12b | `ExpireAndEmit`: `_(commit: "fix(atlas-storage): expire+replace is one transaction, event publishes after commit" — Task 11, done)_`; `GetOrCreateStorageId`/`WithTransaction` plumbing: `_(commit: "feat(atlas-storage): WithTransaction plumbing; GetOrCreateStorageId joins caller transactions" — Task 10, done)_`; `MergeAndSort`: `_(commit: "fix(atlas-storage): MergeAndSort merge/compact/sort writes are one transaction" — Task 12, done)_`; race annotations (`GetOrCreateStorageId`/`GetOrCreateStorage`) documented above, no schema change in scope; account-deletion cascade (`DeleteByAccountId`): `_(commit: "fix(atlas-storage): wrap account-deletion cascade (DeleteByAccountId) in one transaction" — Task 12b, done)_`. All 3 genuine class-A/B gaps for this service are now remediated. |
| atlas-account | C ×3, one with mandatory race annotation (`GetOrCreate`→`Create`, name-uniqueness) | No change; document race annotation | Task 13 | `_(commit: "fix(atlas-maps): atomic character-deletion cleanup; finalize account/ban class-C verdicts (task-119)" — Task 13, done)_` — audit.md-only, no code change |
| atlas-ban | C ×4 (CRUD) + D ×2 (tickers); ban↔history pairing **refuted** | No change; document the refutation | Task 13 | `_(commit: "fix(atlas-maps): atomic character-deletion cleanup; finalize account/ban class-C verdicts (task-119)" — Task 13, done)_` — audit.md-only, no code change |
| atlas-maps | C ×3; **A ×1 (new finding: `handleStatusEventDeletedFunc` dual delete, not in the design survey)** — remediated | Wrapped the character-deletion dual-delete in one `database.ExecuteTransaction`; no change to the C flows | Task 13 | `_(commit: "fix(atlas-maps): atomic character-deletion cleanup; finalize account/ban class-C verdicts (task-119)" — Task 13, done)_` |
| atlas-map-actions | C ×3 (live REST CRUD) + D ×2 (seeder) | No change; seeder is out-of-scope follow-up candidate | — (no task; already correct) | Confirmed, no remediation needed |
| atlas-portal-actions | C ×3 (live REST CRUD) + D ×2 (seeder) | No change; seeder is out-of-scope follow-up candidate | — (no task; already correct) | Confirmed, no remediation needed |
| atlas-reactor-actions | C ×3 (live REST CRUD) + D ×2 (seeder) | No change; seeder is out-of-scope follow-up candidate | — (no task; already correct) | Confirmed, no remediation needed |
| atlas-party-quests | C ×3 (live REST CRUD, `definitions`) + D ×2 (seeder); `instance/*` entirely in-memory (excluded) | No change; seeder is out-of-scope follow-up candidate | — (no task; already correct) | Confirmed, no remediation needed |
| atlas-saga-orchestrator | D ×5 (optimistic-version store) | No change — justified as-is | — (no task; already correct) | Confirmed, no remediation needed |

**Follow-up candidate (out of scope for this task, not assigned a task number):** the shared `libs/atlas-seeder` delete-all+bulk-create cycle (`seed.go:85-120`) underlies every class-D verdict in atlas-map-actions, atlas-portal-actions, atlas-reactor-actions, and atlas-party-quests' `definition/*`. Wrapping it in a transaction would change semantics (continue-on-error per-file accounting, §5.2) for every consumer of the shared lib, including services outside this task's scope (e.g. atlas-npc-conversations' and atlas-npc-shops'-style seeders elsewhere in the monorepo, not audited here). Any such change belongs to a dedicated `libs/atlas-seeder` design task, not to task-119's remediation commits.

**Corrections to the design-phase survey (§3) surfaced by this full sweep**, for the record:
1. **atlas-marriages** is *not* already-transactional on its live path — the design's `[T][E]` finding describes dead code (`executeInTransaction`/`AcceptProposalWithTransactionAndEmit`, zero callers). The actual production accept-flow (`AcceptProposalAndEmit`) is a genuine unwrapped class-A gap.
2. **ban↔history pairing** is refuted — the two are fully independent write paths, no entry point writes both.
3. **atlas-maps** has one class-A finding (`handleStatusEventDeletedFunc`'s dual delete) that the design survey did not anticipate (it characterized maps as purely class C).
4. **atlas-monster-book**'s second consumer site (`kafka/consumer/character/consumer.go`) has **no** emit inside it — the design implied both monster-book sites might share the `[E]` defect; only `handleCardPickedUp` does.
5. **map/portal/reactor-actions' `sagaP.Create`** calls are Kafka producer publishes, not REST calls to atlas-saga-orchestrator as the design phrased it — exclusion classification is unaffected, transport label is corrected.
6. **atlas-storage**'s account-deletion cascade (`storage.Delete` + `asset.DeleteByStorageId`, both invoked from the same Kafka handler) is an additional class-A finding beyond the three flows (`ExpireAndEmit`, `MergeAndSort`, `GetOrCreateStorageId`) the design survey named.

---

## Rebase gate (Task 4) — 2026-07-12

- **Rebased onto** `origin/main` @ `e15b343b1` (clean; all 7 branch commits replayed, no conflicts — main did not touch `libs/atlas-database/`).
- **Dependency merge commits confirmed in main:**
  - task-114 (fleet-wide transactional outbox): `d2e13ba3d` (#903)
  - task-116 (Gen3 processor unification): `e15b343b1` (#967)
- **CRITICAL — main's `isTransaction` is still the buggy `ConnPool != nil` form.** task-114 did *not* rebase onto this branch's Task 1 fix (the design §2.4 standalone-PR recommendation was not acted on before task-114 merged). Consequence: task-114's fleet-wide outbox `Emit` currently runs on a no-op `ExecuteTransaction` in main — enqueue-in-tx is presently **non-atomic in production** until this branch's Task 1 fix (`b3c85d638`) lands. The remediation tasks below only become effective once that fix merges. No conflict to reconcile; the two just never met.
- **Remediation targets re-verified post task-116 rewrites** — all present, shapes intact:
  - npc-conversations `conversation/npc/processor.go` — 6 `db...Transaction` sites (132/155/179/202/221/273).
  - keys `key/processor.go` — 4 sites (74/93/107/119).
  - families `family/processor.go` — 3 sites (175/247/320).
  - marriages `marriage/processor.go:1692` — manual `Begin()` (the dead-code twin; live `AcceptProposalAndEmit` still unwrapped per §3 correction #1).
  - monster-book `kafka/consumer/character/consumer.go:49` — raw `Transaction`.
  - storage `storage/processor.go` — `MergeAndSort` (508), `ExpireAndEmit` (745); `asset/processor.go` — `GetOrCreateStorageId` (58).
- **Emit-convention decision per service (design §6.5 re-check):** grep for `outbox` across the 6 touched services shows **only atlas-monster-book was migrated to the outbox by task-114** (`card/processor.go`, `collection/processor.go`, `main.go`, `kafka/consumer/monsterbook/consumer.go`). The established shape is `message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(...)` inside the tx (`card/processor.go:104`).
  - **atlas-monster-book (Task 8):** use the outbox `EmitProvider(l, ctx, tx)` enqueue-in-tx wrapper — a provider swap per design §6.1, not buffer+publish-after-commit. This also subsumes the `[E]` fix (events enqueue in the same tx, so a rollback drops them).
  - **atlas-keys, atlas-families, atlas-npc-conversations, atlas-marriages, atlas-storage (Tasks 5–7, 9–13):** not migrated — use buffer + publish-after-commit exactly as the plan's diffs specify.

---

## D0 blast-radius: fleet-wide `ExecuteTransaction`-callers audit (task-119, 2026-07-13)

The D0 fix (Task 1) made `database.ExecuteTransaction` actually open transactions
fleet-wide. Because the helper was previously a verified no-op, every service that
already used it ran its "transactions" as plain root-handle calls — which masked
two classes of latent problem that only surface once real transactions open:

1. **Test-harness fragility.** A sqlite `:memory:` database is *per-connection*. A
   read/write issued on the root `p.db` handle *inside* a real transaction lands on
   a second pooled connection whose in-memory schema is empty → `no such table`.
2. **Latent atomicity bugs.** A processor method that itself opens
   `ExecuteTransaction`, called on the root `p` (not `p.WithTransaction(tx)`) from
   inside another transaction closure, opens a *separate* transaction on a different
   connection — its writes commit independently and survive an outer rollback.

### Scope and method

Every one of the **26 services** that call `database.ExecuteTransaction` was audited
(19 non-task-119 callers via per-service read-only audit agents; the 7 task-119
remediation services carry rollback tests that already prove their wrapped flows are
atomic). For each service, every `ExecuteTransaction`/`db.Transaction` closure was
traced: every DB-touching call inside must ride the closure's `tx` (via
`WithTransaction(tx)`, a `tx` handle passed to a package-level administrator/provider,
or a helper that receives only `tx`). Per-service reports live in
`docs/tasks/task-119-db-transaction-coverage/` scratch during the audit; the outcomes:

### Findings and remediation

| Service | Verdict | Action |
|---|---|---|
| **atlas-inventory** | **5 CRITICAL nested-tx** | Fixed. `AttemptItemPickUp` called `p.CreateAsset` un-bound (2 sites); `MergeAndCompact`/`CompactAndSort` called `p.Move` un-bound (4 sites) — each opens its own `ExecuteTransaction`, so the picked-up asset / slot moves committed outside the enclosing tx. Bound all via `p.WithTransaction(tx)`; also bound the compaction-loop compartment re-fetches to `tx` (correctness — they must observe the in-tx slot moves) and fixed `inventory.WithTransaction` to rebind its compartment sub-processor. Compaction + pickup tests green. |
| **atlas-pets** | **1 CRITICAL nested-tx** | Fixed. `Despawner` was a Go method value bound to the original processor at `NewProcessor` time; `With(WithTransaction(tx))`'s shallow copy never rebound it, so `Despawn` (in `DespawnAndEmit` and `EvaluateHunger`) always opened a separate transaction on the root pool. Removed the construction-time binding so `Despawn` dispatches to the receiver's own `defaultDespawn` (tx-bound on clones); the field remains an optional test-mock override. Added a rollback regression test (RED→GREEN). |
| **atlas-quest** | 3 MINOR (benign) | No fix. `completeCore`/`startCore`/`startChainedCore` read via root `p.db` inside a tx closure — reads return committed data; harmless in Postgres. Test harness switched to shared-cache in-memory sqlite so the reads see the schema under real transactions. |
| **atlas-character** | 1 MINOR (benign) | No fix. `Delete`'s pre-read uses root `p.GetById` (feeds only the emitted event's `WorldId`, not the delete). Test harness switched to shared-cache in-memory sqlite. |
| 15 others | CLEAN | atlas-buddies, atlas-cashshop, atlas-configurations, atlas-data, atlas-drop-information, atlas-fame, atlas-gachapons, atlas-guilds, atlas-merchant, atlas-mounts, atlas-mts, atlas-notes, atlas-npc-shops, atlas-skills, atlas-tenants — every in-closure DB op rides the tx (via `WithTransaction(tx)` or a `tx`-seeded constructor; GORM correctly downgrades genuinely-nested `db.Transaction` to a SAVEPOINT). |
| 7 task-119 svcs | CLEAN | keys, families, npc-conversations, monster-book, marriages, storage, maps — rollback-test-proven atomic. |

**Test-harness fix (no production impact):** atlas-quest, atlas-character, and
atlas-inventory built a bare sqlite `:memory:` test DB. Switched each to a
uniquely-named shared-cache in-memory DB (`file:<uuid>?mode=memory&cache=shared`,
one idle connection pinned open) so every pooled connection shares one schema under
real transactions. `SetMaxOpenConns(1)` was rejected — it deadlocks when a flow
legitimately uses two connections (tx + a root read).

**Production safety of D0:** the fix is *not* a regression. Before it, no service's
transactions were real, so the fleet had zero atomicity everywhere; after it,
correctly-written flows become atomic and the two nested-tx outliers (inventory,
pets) are fixed. In Postgres the benign MINOR root-reads return committed data and
never error.

**Adjacent follow-up candidates (out of scope — a different class, pre-existing, not
D0-exposed):** several services have multi-write methods with *no* transaction
wrapping at all (`atlas-npc-shops` `CreateShop`, `atlas-merchant` `CreateShop`,
`atlas-tenants` `Seed*`, `atlas-mts` dead-code `UpdateAuction`). These are genuine
atomicity gaps of the same shape task-119 remediated in its 14 services, but in
services outside this task's scope; they belong to a dedicated follow-up, not this
branch.
