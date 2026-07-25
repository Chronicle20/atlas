# DB Transaction Coverage for Multi-Entity Mutations — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix the no-op `database.ExecuteTransaction` helper, audit all 14 database-backed services with zero call sites, standardize/wrap every multi-statement mutation onto the fixed helper, and prove every boundary with a rollback test.

**Architecture:** Three layers (design §4): (1) a one-function fix in `libs/atlas-database` plus a shared `FailWritesOn` fault-injection test helper; (2) service write paths converted to the canonical composition — `message.Emit`/buffer outside, `database.ExecuteTransaction` inside (exemplar: `services/atlas-guilds/atlas.com/guilds/guild/processor.go:253`); (3) a committed `audit.md` classifying every write path in the 14 services (DL-4 closure evidence).

**Tech Stack:** Go, GORM (sqlite in-memory for tests via `databasetest.NewInMemoryTenantDB`), testify, Kafka message buffers (`libs/atlas-kafka/message`).

## Global Constraints

- All work happens in this worktree on branch `task-119-db-transaction-coverage`. Verify with `git branch --show-current` after every commit.
- **No Kafka emit inside any transaction** (`message.Emit`/`EmitWithResult`/direct producer calls publish only after the tx commits). `mb.Put(...)`/`buf.Put(...)` is buffering, not publishing — safe inside a tx (PRD FR-2.2).
- **Zero manual `Begin()`/`Commit()`/`Rollback()`** anywhere in remediated code (PRD acceptance criterion).
- **No behavior change beyond atomicity**: same writes, same order, same happy-path events, same REST responses (PRD FR-2.4).
- Import the lib as: `database "github.com/Chronicle20/atlas/libs/atlas-database"` (repo convention, matches `databasetest/testdb.go`).
- No `*_testhelpers.go` files with test-only constructors; entity setup uses the services' Builder patterns or direct entity `Create` as the existing provider tests do (e.g. `services/atlas-keys/atlas.com/keys/key/provider_test.go`).
- No new libs, no new abstractions — `libs/atlas-database` already owns this concern.
- No `// TODO`, stubs, or 501s in landed commits.
- Committed docs use repo-relative paths only (never `/home/<name>/...`).
- **Sequencing gate:** Tasks 5–13 (remediation) are BLOCKED until Task 4's checkpoint passes (task-114-outbox-adoption and task-116-processor-gen3-unification merged into main, branch rebased). Tasks 1–3 run now.
- Per-module verification: `go test -race ./...`, `go vet ./...`, `go build ./...` from the module root (`services/atlas-<svc>/atlas.com/<name>/` or `libs/atlas-database/`). Final gate adds `docker buildx bake` and `tools/redis-key-guard.sh` (Task 15).
- Line numbers below were verified on this branch pre-rebase (2026-07-02). Task 4 re-verifies them after rebase; match on code shape, not raw line number.

---

### Task 1: Fix `isTransaction` in libs/atlas-database (D0) + regression tests

`ExecuteTransaction` has never opened a transaction: `isTransaction` checks `Statement.ConnPool != nil`, which is true for every `*gorm.DB` (design §2). Fix it with GORM's own `TxCommitter` idiom.

**Files:**
- Modify: `libs/atlas-database/transaction.go`
- Test: `libs/atlas-database/transaction_test.go` (new)

**Interfaces:**
- Consumes: `databasetest.NewInMemoryTenantDB(t, migrations...)`, `databasetest.TenantContext(id)` (both exist in `libs/atlas-database/databasetest/testdb.go`).
- Produces: `database.ExecuteTransaction(db, fn)` with real transaction semantics — root handle opens a tx; a handle already inside a tx joins it. Signature unchanged; every existing caller compiles unchanged.

- [ ] **Step 1: Write the failing regression tests**

Create `libs/atlas-database/transaction_test.go`:

```go
package database_test

import (
	"errors"
	"testing"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type txEntity struct {
	Id       uint32    `gorm:"primaryKey;autoIncrement"`
	TenantId uuid.UUID `gorm:"type:uuid;not null"`
	Name     string    `gorm:"not null"`
}

func (txEntity) TableName() string { return "tx_entities" }

func txMigration(db *gorm.DB) error { return db.AutoMigrate(&txEntity{}) }

func TestExecuteTransaction_RollsBackOnError(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, txMigration)
	handle := db.WithContext(databasetest.TenantContext(uuid.New()))

	err := database.ExecuteTransaction(handle, func(tx *gorm.DB) error {
		if err := tx.Create(&txEntity{Name: "doomed"}).Error; err != nil {
			return err
		}
		return errors.New("boom")
	})
	require.Error(t, err)

	var count int64
	require.NoError(t, handle.Model(&txEntity{}).Count(&count).Error)
	require.Zero(t, count, "write inside failed ExecuteTransaction must roll back")
}

func TestExecuteTransaction_CommitsOnSuccess(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, txMigration)
	handle := db.WithContext(databasetest.TenantContext(uuid.New()))

	require.NoError(t, database.ExecuteTransaction(handle, func(tx *gorm.DB) error {
		return tx.Create(&txEntity{Name: "kept"}).Error
	}))

	var count int64
	require.NoError(t, handle.Model(&txEntity{}).Count(&count).Error)
	require.EqualValues(t, 1, count)
}

func TestExecuteTransaction_NestedJoinsOuterTransaction(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, txMigration)
	handle := db.WithContext(databasetest.TenantContext(uuid.New()))

	err := database.ExecuteTransaction(handle, func(outer *gorm.DB) error {
		innerErr := database.ExecuteTransaction(outer, func(inner *gorm.DB) error {
			return inner.Create(&txEntity{Name: "inner"}).Error
		})
		require.NoError(t, innerErr, "nested call must join, not fail")
		return errors.New("outer fails after inner succeeded")
	})
	require.Error(t, err)

	var count int64
	require.NoError(t, handle.Model(&txEntity{}).Count(&count).Error)
	require.Zero(t, count, "inner write must join the outer tx and roll back with it")
}

func TestExecuteTransaction_TenantCallbacksActiveInsideTransaction(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, txMigration)
	tid := uuid.New()
	handle := db.WithContext(databasetest.TenantContext(tid))

	require.NoError(t, database.ExecuteTransaction(handle, func(tx *gorm.DB) error {
		return tx.Create(&txEntity{Name: "stamped"}).Error
	}))

	var row txEntity
	require.NoError(t, handle.First(&row).Error)
	require.Equal(t, tid, row.TenantId, "tenant create-callback must stamp tenant_id inside the tx")
}
```

(The tenant create-callback stamps `tenant_id` via `injectTenantIdIfZero` — `libs/atlas-database/tenant_scope.go:110-124` — so the last test asserts stamping survives the tx boundary.)

- [ ] **Step 2: Run tests to verify the expected failures**

```bash
cd libs/atlas-database && go test -race -run TestExecuteTransaction ./...
```

Expected: `TestExecuteTransaction_RollsBackOnError` and `TestExecuteTransaction_NestedJoinsOuterTransaction` FAIL (writes survive: count is 1, not 0). `CommitsOnSuccess` and `TenantCallbacksActive` PASS. This is the design §2.2 probe made permanent.

- [ ] **Step 3: Fix `isTransaction`**

In `libs/atlas-database/transaction.go`, replace:

```go
func isTransaction(db *gorm.DB) bool {
	return db.Statement != nil && db.Statement.ConnPool != nil
}
```

with:

```go
// isTransaction reports whether the handle is already inside a transaction.
// Inside a real transaction Statement.ConnPool is a *sql.Tx (implements
// gorm.TxCommitter); on the root pool it is *sql.DB, which does not. This is
// GORM's own idiom for the same check in finisher_api.go.
func isTransaction(db *gorm.DB) bool {
	if db.Statement == nil || db.Statement.ConnPool == nil {
		return false
	}
	committer, ok := db.Statement.ConnPool.(gorm.TxCommitter)
	return ok && committer != nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd libs/atlas-database && go test -race ./... && go vet ./...
```

Expected: all PASS (including the pre-existing `tenant_scope_test.go` and `databasetest` tests).

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-database/transaction.go libs/atlas-database/transaction_test.go
git commit -m "fix(atlas-database): ExecuteTransaction never opened a transaction

isTransaction checked Statement.ConnPool != nil, which is true for every
*gorm.DB gorm.Open produces, so all 53 call sites across 18 services ran
their 'transactions' as plain sequential statements. Detect a live
transaction via gorm.TxCommitter instead (GORM's own idiom)."
```

- [ ] **Step 6: Surface the standalone-PR recommendation**

Report to the user (do not block on an answer; continue with Task 2): the design (§2.4 delivery vehicle) recommends rebase-cutting this commit into an immediate small standalone PR so task-114's outbox atomicity rebases onto real transaction semantics before it merges. Flag the commit SHA and the recommendation; the owner decides at PR time.

---

### Task 2: Add `databasetest.FailWritesOn` fault-injection helper

One shared helper (design §7.2) so 14 services don't hand-roll failure-injection callbacks.

**Files:**
- Create: `libs/atlas-database/databasetest/failwrites.go`
- Test: `libs/atlas-database/databasetest/failwrites_test.go` (new)

**Interfaces:**
- Produces: `databasetest.FailWritesOn(t *testing.T, db *gorm.DB, table string, verbs ...databasetest.WriteVerb)` and verb constants `databasetest.WriteCreate`, `databasetest.WriteUpdate`, `databasetest.WriteDelete`. No verbs = fail all three. Registered callbacks apply to every session/tx derived from `db`. Raw `.Exec(...)` bypasses GORM callbacks and is NOT intercepted (documented on the helper).

- [ ] **Step 1: Write the failing test**

Create `libs/atlas-database/databasetest/failwrites_test.go`:

```go
package databasetest

import (
	"errors"
	"testing"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type fwEntity struct {
	Id       uint32    `gorm:"primaryKey;autoIncrement"`
	TenantId uuid.UUID `gorm:"type:uuid"`
	Name     string
}

func (fwEntity) TableName() string { return "fw_entities" }

type fwOther struct {
	Id   uint32 `gorm:"primaryKey;autoIncrement"`
	Name string
}

func (fwOther) TableName() string { return "fw_others" }

func fwMigration(db *gorm.DB) error { return db.AutoMigrate(&fwEntity{}, &fwOther{}) }

func TestFailWritesOn_FailsNamedVerbOnNamedTableOnly(t *testing.T) {
	db := NewInMemoryTenantDB(t, fwMigration)
	handle := db.WithContext(TenantContext(uuid.New()))

	FailWritesOn(t, db, "fw_entities", WriteCreate)

	require.Error(t, handle.Create(&fwEntity{Name: "blocked"}).Error,
		"create on the named table must fail")
	require.NoError(t, handle.Create(&fwOther{Name: "fine"}).Error,
		"other tables must be unaffected")
	require.NoError(t, handle.Where("1 = 1").Delete(&fwEntity{}).Error,
		"unregistered verbs on the named table must be unaffected")
}

func TestFailWritesOn_DefaultsToAllVerbs(t *testing.T) {
	db := NewInMemoryTenantDB(t, fwMigration)
	handle := db.WithContext(TenantContext(uuid.New()))

	FailWritesOn(t, db, "fw_entities")

	require.Error(t, handle.Create(&fwEntity{Name: "blocked"}).Error)
	require.Error(t, handle.Model(&fwEntity{}).Where("1 = 1").Update("name", "x").Error)
	require.Error(t, handle.Where("1 = 1").Delete(&fwEntity{}).Error)
}

func TestFailWritesOn_DrivesRollbackThroughExecuteTransaction(t *testing.T) {
	db := NewInMemoryTenantDB(t, fwMigration)
	handle := db.WithContext(TenantContext(uuid.New()))
	require.NoError(t, handle.Create(&fwEntity{Name: "original"}).Error)

	// The class-B shape (keys reset): delete-all succeeds, re-create fails,
	// the whole flow must roll back to the pre-flow state.
	FailWritesOn(t, db, "fw_entities", WriteCreate)

	err := database.ExecuteTransaction(handle, func(tx *gorm.DB) error {
		if err := tx.Where("1 = 1").Delete(&fwEntity{}).Error; err != nil {
			return err
		}
		return tx.Create(&fwEntity{Name: "replacement"}).Error
	})
	require.Error(t, err)

	var rows []fwEntity
	require.NoError(t, handle.Find(&rows).Error)
	require.Len(t, rows, 1)
	require.Equal(t, "original", rows[0].Name, "the delete must have rolled back")
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd libs/atlas-database && go test -race -run TestFailWritesOn ./databasetest/
```

Expected: FAIL to compile — `undefined: FailWritesOn`, `undefined: WriteCreate`.

- [ ] **Step 3: Write the helper**

Create `libs/atlas-database/databasetest/failwrites.go`:

```go
package databasetest

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// WriteVerb identifies one GORM write pipeline for FailWritesOn.
type WriteVerb string

const (
	WriteCreate WriteVerb = "create"
	WriteUpdate WriteVerb = "update"
	WriteDelete WriteVerb = "delete"
)

// FailWritesOn registers create/update/delete callbacks that fail any write to
// the named table, for rollback testing (fail a flow's later statement, then
// assert its earlier writes rolled back). With no verbs, all three write verbs
// fail. Callbacks apply to every session and transaction derived from db.
// Raw .Exec(...) statements bypass GORM callbacks and are not intercepted.
func FailWritesOn(t *testing.T, db *gorm.DB, table string, verbs ...WriteVerb) {
	t.Helper()
	if len(verbs) == 0 {
		verbs = []WriteVerb{WriteCreate, WriteUpdate, WriteDelete}
	}
	fail := func(d *gorm.DB) {
		if d.Statement != nil && d.Statement.Table == table {
			_ = d.AddError(fmt.Errorf("databasetest: injected failure writing to %q", table))
		}
	}
	for _, v := range verbs {
		name := fmt.Sprintf("databasetest:fail_%s_%s", v, table)
		switch v {
		case WriteCreate:
			require.NoError(t, db.Callback().Create().Before("gorm:create").Register(name, fail))
		case WriteUpdate:
			require.NoError(t, db.Callback().Update().Before("gorm:update").Register(name, fail))
		case WriteDelete:
			require.NoError(t, db.Callback().Delete().Before("gorm:delete").Register(name, fail))
		}
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd libs/atlas-database && go test -race ./... && go vet ./...
```

Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-database/databasetest/failwrites.go libs/atlas-database/databasetest/failwrites_test.go
git commit -m "feat(atlas-database): databasetest.FailWritesOn fault-injection helper for rollback tests"
```

---

### Task 3: Full write-path audit → audit.md

Full sweep of all 14 services (PRD FR-1; design §5). This is an analysis-and-document task — no code changes. It can run before the Task 4 rebase gate; Task 4 re-verifies citations afterward.

**Files:**
- Create: `docs/tasks/task-119-db-transaction-coverage/audit.md`

**Interfaces:**
- Consumes: design §3 survey table (starting hypotheses), design §5.2 refined taxonomy.
- Produces: one section per service with (a) write-statement inventory table (`file:line` → entry point → class + flags), (b) exclusions list, (c) verdicts with justification for C/D, (d) remediation pointer placeholder `_(commit: pending — Task N)_` for A/B/[T]/[E] rows; plus a closing 14-service matrix. Tasks 5–13 fill the pointers in.

- [ ] **Step 1: Enumerate write statements per service**

For each of the 14 services (`atlas-monster-book`, `atlas-account`, `atlas-ban`, `atlas-families`, `atlas-keys`, `atlas-map-actions`, `atlas-maps`, `atlas-marriages`, `atlas-npc-conversations`, `atlas-party-quests`, `atlas-portal-actions`, `atlas-reactor-actions`, `atlas-storage`, `atlas-saga-orchestrator`):

```bash
grep -rn "\.Create(\|\.Save(\|\.Update(\|\.Updates(\|\.Delete(\|\.Exec(" services/atlas-<svc> --include='*.go' | grep -v _test.go
grep -rn "\.Transaction(\|\.Begin()\|\.Commit()\|\.Rollback()" services/atlas-<svc> --include='*.go' | grep -v _test.go
```

- [ ] **Step 2: Discard and record non-DB hits**

For every hit, resolve the receiver: REST client calls (saga orchestrator's outbound `.Create(...)`), in-memory registry mutations (party-quests `instance/`), and `tenant.Create` are NOT DB writes. Record each exclusion with `file:line` and one-line reason in the service's "Exclusions" subsection — this is what makes the sweep verifiably full (design §3, PRD FR-1.3).

- [ ] **Step 3: Group remaining writes by entry point and classify**

Resolve each write to its triggering entry point (REST handler, Kafka consumer, ticker, seeder) and classify per design §5.2:

- **A** multi-table (≥2 writes, ≥2 tables) → wrap.
- **B** multi-statement single-table (**≥2 write statements**) → wrap. A read-modify-write with exactly ONE write is NOT class B — it is class C with a mandatory **race annotation** (e.g. atlas-account `GetOrCreate` name-uniqueness race: needs a unique constraint, documented not fixed).
- **C** single-statement → no change (+ race annotation where an RMW gap exists).
- **D** intentionally non-atomic → written justification. Known members: saga-orchestrator's optimistic-version store (`saga/store.go:100-186,240,290`); the shared seeder cycle (`libs/atlas-seeder/seed.go:85-120` — per-file error accounting, deliberate continue-on-error, per-(tenant,group) mutex) used by map-actions/portal-actions/reactor-actions/party-quests-definitions; ban/history delete-sweep tickers (single-statement); party-quests in-memory instance machinery (no DB).

Orthogonal flags: **[T]** already-transactional via raw `db.Transaction`/manual `Begin` (npc-conversations ×6, keys ×4, families ×3, marriages ×1, monster-book ×2); **[E]** emit-inside-transaction (monster-book `kafka/consumer/monsterbook/consumer.go:56-57`; marriages `marriage/processor.go:1606` inside `executeInTransaction`).

If any flow sits on a high-frequency path, record its call frequency before proposing a wrap (PRD §8; the design survey found none — storage flows are UI/NPC-driven).

- [ ] **Step 4: Write audit.md**

One section per service using this template, then a closing matrix summarizing all 14 (service → classes found → action → status):

```markdown
## atlas-<svc>

### Write inventory
| file:line | Entry point | Writes | Class | Flags |
|---|---|---|---|---|

### Exclusions (non-DB write-verb hits)
| file:line | Reason |
|---|---|

### Verdicts
- <entry point>: <A/B/C/D> — <justification for C/D; remediation pointer for A/B/[T]/[E]>
```

Expected findings to confirm or refute (from the design §3 survey — the audit is the authority): storage `ExpireAndEmit`/`MergeAndSort`/`GetOrCreateStorageId` are the genuine unwrapped gaps; account/ban/maps are class C (ban↔history pairing to be confirmed one way or the other); npc-conversations/keys/families/marriages/monster-book are [T] (± [E]); the rest are class D.

- [ ] **Step 5: Commit**

```bash
git add docs/tasks/task-119-db-transaction-coverage/audit.md
git commit -m "docs(task-119): full write-path audit of the 14 zero-ExecuteTransaction services"
```

---

### Task 4: CHECKPOINT — task-114/116 merge gate and rebase

**Files:** none (git operations + verification only).

- [ ] **Step 1: Check merge status**

```bash
git fetch origin main
git log origin/main --oneline --grep="task-114" --grep="outbox" | head -5
git log origin/main --oneline --grep="task-116" --grep="gen3" | head -5
```

Both task-114-outbox-adoption and task-116-processor-gen3-unification must be merged into main (PRD §7 sequencing). **If either is unmerged: STOP. Report BLOCKED to the user and do not start Tasks 5–13.** (As of plan-writing, both are still in-flight worktrees.)

- [ ] **Step 2: Rebase**

```bash
git rebase origin/main
```

Resolve conflicts if any (expect them in `docs/` only for Tasks 1–3's commits; the lib fix touches files task-114 may also touch — if task-114 landed an identical/conflicting `isTransaction` fix, reconcile and note it in audit.md).

- [ ] **Step 3: Re-verify remediation targets post-rebase**

For each site cited in Tasks 5–12, confirm the code shape still matches (task-116 rewrites processor files in these services):

```bash
grep -rn "\.Transaction(\|\.Begin()" services/atlas-npc-conversations services/atlas-keys services/atlas-families services/atlas-marriages services/atlas-monster-book --include='*.go' | grep -v _test.go
grep -n "ExpireAndEmit\|MergeAndSort\|GetOrCreateStorageId" services/atlas-storage/atlas.com/storage/storage/processor.go services/atlas-storage/atlas.com/storage/asset/processor.go
```

Update audit.md line citations if they shifted. If task-116 restructured a processor so a diff below no longer applies verbatim, port the change to the new shape — the composition rule (Emit outside, ExecuteTransaction inside, tx threaded) is the invariant, not the exact lines.

- [ ] **Step 4: Re-check the emit convention (design §6.5)**

Check whether task-114's landed FR-3.1 sweep migrated any of the 6 touched services to the outbox:

```bash
grep -rln "outbox" services/atlas-npc-conversations services/atlas-keys services/atlas-families services/atlas-marriages services/atlas-monster-book services/atlas-storage --include='*.go'
```

Default (none migrated, per task-114's PRD §7 scope): use buffer + publish-after-commit exactly as written in Tasks 5–12. If a service WAS migrated: use task-114's outbox `Emit`-shaped wrapper (enqueue-in-tx) for that service instead, and cite the convention + task-114 commit in audit.md. The §6.1 composition makes this a provider swap, not a restructure.

- [ ] **Step 5: Verify clean state and record the gate in audit.md**

```bash
go build ./... 2>/dev/null; git status --short
cd libs/atlas-database && go test -race ./...
```

Append a "Rebase gate" note to audit.md (date, main SHA, task-114/116 merge commits, emit-convention decision per service). Commit:

```bash
git add docs/tasks/task-119-db-transaction-coverage/audit.md
git commit -m "docs(task-119): rebase gate — re-verified citations and emit conventions post task-114/116"
```

---

### Task 5: atlas-keys — standardize 4 raw transactions + rollback test

All four flows are already atomic via raw `db.Transaction` (class B, flag [T]). This converts the wrapper API only — same boundaries, same writes, same order.

**Files:**
- Modify: `services/atlas-keys/atlas.com/keys/key/processor.go:72,91,105,117`
- Test: `services/atlas-keys/atlas.com/keys/key/processor_rollback_test.go` (new)

**Interfaces:**
- Consumes: `database.ExecuteTransaction` (Task 1), `databasetest.FailWritesOn` (Task 2).
- Produces: no signature changes; `Processor` interface untouched.

- [ ] **Step 1: Write the rollback test (green before AND after — the raw tx already rolls back; this locks the behavior across the conversion)**

Create `services/atlas-keys/atlas.com/keys/key/processor_rollback_test.go`:

```go
package key

import (
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
)

// Reset is delete-all + loop-create on the keys table (class B). A failure on
// the re-create must roll the delete back, leaving the prior bindings intact.
func TestReset_RollsBackDeleteWhenCreateFails(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, Migration)
	tid := uuid.New()
	ctx := databasetest.TenantContext(tid)
	require.NoError(t, db.Create(&entity{TenantId: tid, CharacterId: 1001, Key: 10, Type: 1, Action: 100}).Error)

	databasetest.FailWritesOn(t, db, "keys", databasetest.WriteCreate)

	l, _ := test.NewNullLogger()
	err := NewProcessor(l, ctx, db).Reset(uuid.New(), 1001)
	require.Error(t, err)

	var rows []entity
	require.NoError(t, db.Unscoped().Find(&rows).Error)
	require.Len(t, rows, 1, "pre-existing binding must survive: the delete rolled back")
	require.Equal(t, int32(10), rows[0].Key)
}
```

- [ ] **Step 2: Run it — expect PASS (raw tx already atomic)**

```bash
cd services/atlas-keys/atlas.com/keys && go test -race -run TestReset ./key/
```

Expected: PASS.

- [ ] **Step 3: Convert the four sites**

In `services/atlas-keys/atlas.com/keys/key/processor.go`, add to imports:

```go
	database "github.com/Chronicle20/atlas/libs/atlas-database"
```

Then at each of the four sites (`Reset` line 72, `CreateDefault` line 91, `Delete` line 105, `ChangeKey` line 117), change exactly one line:

```go
	return p.db.WithContext(p.ctx).Transaction(func(tx *gorm.DB) error {
```

becomes

```go
	return database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
```

Closure bodies and closing `})` are unchanged.

- [ ] **Step 4: Verify**

```bash
cd services/atlas-keys/atlas.com/keys && go test -race ./... && go vet ./... && go build ./...
grep -n "\.Transaction(" key/processor.go
```

Expected: all green; the grep returns nothing.

- [ ] **Step 5: Commit and record in audit.md**

Fill the atlas-keys remediation pointers in audit.md with this commit, then:

```bash
git add services/atlas-keys docs/tasks/task-119-db-transaction-coverage/audit.md
git commit -m "refactor(atlas-keys): standardize raw db.Transaction onto database.ExecuteTransaction

Four class-B flows (reset, create-default, delete, change-key) keep their
existing boundaries; rollback test locks the reset flow's atomicity."
```

---

### Task 6: atlas-families — standardize 3 raw transactions + rollback test

Three class-B flows (senior+junior member saves), already atomic via raw `db.Transaction`, emit via caller-owned `message.Buffer` (already outside the tx — puts are buffering). Conversion only.

**Files:**
- Modify: `services/atlas-families/atlas.com/family/family/processor.go:173,245,318`
- Test: `services/atlas-families/atlas.com/family/family/processor_rollback_test.go` (new)

**Interfaces:**
- Consumes: `database.ExecuteTransaction`; existing `WithTransaction` (processor.go:73) stays as-is.
- Produces: no signature changes.

- [ ] **Step 1: Write the rollback test**

`AddJunior`'s tx issues two `SaveMember` writes to the same table (`family_members`), so verb-level injection can't split them — use an inline counting callback (design §7.2's sanctioned escape hatch: "the test registers its own counting callback inline").

Create `services/atlas-families/atlas.com/family/family/processor_rollback_test.go`:

```go
package family

import (
	"fmt"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// failNthWriteTo fails the nth (and every later) create/update statement
// against the named table. AddJunior saves senior then junior to the same
// table, so verb-scoped databasetest.FailWritesOn cannot isolate the second
// write — this counting callback can.
func failNthWriteTo(t *testing.T, db *gorm.DB, table string, n int) {
	t.Helper()
	count := 0
	fail := func(d *gorm.DB) {
		if d.Statement != nil && d.Statement.Table == table {
			count++
			if count >= n {
				_ = d.AddError(fmt.Errorf("test: injected failure on write %d to %q", count, table))
			}
		}
	}
	require.NoError(t, db.Callback().Create().Before("gorm:create").Register("test:fail_nth_create", fail))
	require.NoError(t, db.Callback().Update().Before("gorm:update").Register("test:fail_nth_update", fail))
}

// AddJunior updates the senior's junior list and the junior's senior link as
// two writes (class B). Failing the second must roll back the first.
func TestAddJunior_RollsBackSeniorSaveWhenJuniorSaveFails(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, Migration)
	tid := uuid.New()
	ctx := databasetest.TenantContext(tid)
	require.NoError(t, db.Create(&Entity{CharacterId: 1001, TenantId: tid, Level: 100, World: 0}).Error)
	require.NoError(t, db.Create(&Entity{CharacterId: 1002, TenantId: tid, Level: 100, World: 0}).Error)

	failNthWriteTo(t, db, "family_members", 2)

	l, _ := test.NewNullLogger()
	p := NewProcessor(l, ctx, db).(*ProcessorImpl)
	_, err := p.AddJunior(nil)(world.Id(0), 1001, 100, 1002, 100)()
	require.Error(t, err)

	var senior, junior Entity
	require.NoError(t, db.Where("character_id = ?", 1001).First(&senior).Error)
	require.NoError(t, db.Where("character_id = ?", 1002).First(&junior).Error)
	require.Empty(t, senior.JuniorIds, "senior's junior-list update must roll back")
	require.Nil(t, junior.SeniorId, "junior must remain unlinked")
}
```

(If `NewProcessor` returns the `Processor` interface and `AddJunior` is on it, drop the type assertion.)

- [ ] **Step 2: Run it — expect PASS (raw tx already atomic)**

```bash
cd services/atlas-families/atlas.com/family && go test -race -run TestAddJunior_RollsBack ./family/
```

Expected: PASS.

- [ ] **Step 3: Convert the three sites**

In `services/atlas-families/atlas.com/family/family/processor.go`, add to imports:

```go
	database "github.com/Chronicle20/atlas/libs/atlas-database"
```

At each site (`AddJunior` line 173, `RemoveMember` line 245, `BreakLink` line 318), change:

```go
			err = p.db.WithContext(p.ctx).Transaction(func(tx *gorm.DB) error {
```

to

```go
			err = database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
```

- [ ] **Step 4: Verify**

```bash
cd services/atlas-families/atlas.com/family && go test -race ./... && go vet ./... && go build ./...
grep -n "\.Transaction(" family/processor.go
```

Expected: all green; grep empty.

- [ ] **Step 5: Commit and record in audit.md**

```bash
git add services/atlas-families docs/tasks/task-119-db-transaction-coverage/audit.md
git commit -m "refactor(atlas-families): standardize raw db.Transaction onto database.ExecuteTransaction"
```

---

### Task 7: atlas-npc-conversations — standardize 6 raw transactions + rollback test

Six class-A flows (conversation row + derived recipe rows), already atomic. Emit placement: these flows publish nothing (seed/REST-driven CRUD) — verify while editing and note in audit.md.

**Files:**
- Modify: `services/atlas-npc-conversations/atlas.com/npc/conversation/npc/processor.go:130,153,177,200,219,271`
- Test: `services/atlas-npc-conversations/atlas.com/npc/conversation/npc/processor_rollback_test.go` (new)

**Interfaces:**
- Consumes: `database.ExecuteTransaction`; `recipe.NewProcessor(...).RebuildForConversation(tx)`/`ClearForTenant(tx)` keep receiving the tx explicitly (unchanged).
- Produces: no signature changes.

- [ ] **Step 1: Write the rollback test**

`DeleteAllForTenant` clears recipes (hard delete) then hard-deletes conversations — both via GORM `.Delete`, so `FailWritesOn`'s delete callback fires. Seed entities directly (`Entity.Data` is opaque JSON for the delete path).

Create `services/atlas-npc-conversations/atlas.com/npc/conversation/npc/processor_rollback_test.go`:

```go
package npc

import (
	"testing"

	"atlas-npc-conversations/conversation/recipe"

	"github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
)

// DeleteAllForTenant clears recipe rows then conversation rows (class A).
// Failing the conversation delete must restore the cleared recipes.
func TestDeleteAllForTenant_RollsBackRecipeClearWhenConversationDeleteFails(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, MigrateTable, recipe.MigrateTable)
	tid := uuid.New()
	ctx := databasetest.TenantContext(tid)

	convId := uuid.New()
	require.NoError(t, db.Create(&Entity{ID: convId, TenantID: tid, NpcID: 9000000, Data: `{}`}).Error)
	require.NoError(t, db.Create(&recipe.Entity{ID: uuid.New(), TenantID: tid, ConversationID: convId, NpcID: 9000000, StateID: "craft", ItemID: 4000000, Materials: `[]`}).Error)

	databasetest.FailWritesOn(t, db, "conversations", databasetest.WriteDelete)

	l, _ := test.NewNullLogger()
	_, err := NewProcessor(l, ctx, db).DeleteAllForTenant()
	require.Error(t, err)

	var recipes int64
	require.NoError(t, db.Model(&recipe.Entity{}).Count(&recipes).Error)
	require.EqualValues(t, 1, recipes, "recipe clear must roll back with the failed conversation delete")
}
```

(Adjust the module-local import prefix if it differs — check the `module` line of `services/atlas-npc-conversations/atlas.com/npc/go.mod`; sibling packages import as `<module>/conversation/recipe`. If `NewProcessor` doesn't expose `DeleteAllForTenant` on its return type, assert to `*ProcessorImpl`.)

- [ ] **Step 2: Run it — expect PASS**

```bash
cd services/atlas-npc-conversations/atlas.com/npc && go test -race -run TestDeleteAllForTenant_RollsBack ./conversation/npc/
```

Expected: PASS.

- [ ] **Step 3: Convert the six sites**

In `services/atlas-npc-conversations/atlas.com/npc/conversation/npc/processor.go`, add to imports:

```go
	database "github.com/Chronicle20/atlas/libs/atlas-database"
```

At each site — `createWithSkipTracking` (line 130), `Create` (153), `Update` (177), `Delete` (200), `DeleteAllForTenant` (219), `ReindexAllRecipes` (271) — change:

```go
	err := p.db.WithContext(p.ctx).Transaction(func(tx *gorm.DB) error {
```

to

```go
	err := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
```

Also update the two comments that reference the old API: `ReindexAllRecipes`' doc comment "Inside a single db.Transaction so..." → "Inside a single ExecuteTransaction so...", and `recipe/administrator.go:9`'s "(e.g. obtained from db.Transaction)" → "(e.g. obtained from database.ExecuteTransaction)".

- [ ] **Step 4: Verify**

```bash
cd services/atlas-npc-conversations/atlas.com/npc && go test -race ./... && go vet ./... && go build ./...
grep -rn "\.Transaction(" conversation/ --include='*.go' | grep -v _test.go
```

Expected: all green; grep empty.

- [ ] **Step 5: Commit and record in audit.md**

```bash
git add services/atlas-npc-conversations docs/tasks/task-119-db-transaction-coverage/audit.md
git commit -m "refactor(atlas-npc-conversations): standardize raw db.Transaction onto database.ExecuteTransaction"
```

---

### Task 8: atlas-monster-book — invert emit/tx nesting in 2 consumers + rollback test

Both consumers currently nest `message.Emit` INSIDE `db.Transaction` (flag [E]: publish-before-commit — `kafka/consumer/monsterbook/consumer.go:56-57`) or use raw tx without emits (`kafka/consumer/character/consumer.go:49`). Convert to the canonical composition (design §6.1): Emit outside, ExecuteTransaction inside. Processors already expose `WithTransaction` — untouched.

**Files:**
- Modify: `services/atlas-monster-book/atlas.com/monster-book/kafka/consumer/monsterbook/consumer.go:49-77`
- Modify: `services/atlas-monster-book/atlas.com/monster-book/kafka/consumer/character/consumer.go:43-60`
- Test: `services/atlas-monster-book/atlas.com/monster-book/kafka/consumer/monsterbook/consumer_rollback_test.go` (new)

**Interfaces:**
- Consumes: `card.NewProcessor(l, ctx, tx).Add(mb)(eventId, characterId, cardId) (card.UpsertResult, error)`; `collection.NewProcessor(l, ctx, tx).RecomputeAndEmit(mb)(characterId) error`; `card.Migration`, `collection.Migration` (tables `monster_book_cards`, `monster_book_collections`).
- Produces: handler signatures unchanged; happy-path events byte-identical (buffer contents unchanged), publish now strictly after commit.

- [ ] **Step 1: Write the rollback test (the PRD's concrete first instance, §7.1)**

Create `services/atlas-monster-book/atlas.com/monster-book/kafka/consumer/monsterbook/consumer_rollback_test.go`:

```go
package monsterbook

import (
	"testing"

	"atlas-monster-book/card"
	"atlas-monster-book/collection"
	mbmsg "atlas-monster-book/kafka/message/monsterbook"

	"github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
)

// CARD_PICKED_UP upserts a card row then recomputes the collection book level
// (class A: monster_book_cards + monster_book_collections). Failing the
// collection write must roll back the card upsert — the pair moves together.
func TestHandleCardPickedUp_RollsBackCardUpsertWhenCollectionWriteFails(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, card.Migration, collection.Migration)
	ctx := databasetest.TenantContext(uuid.New())

	databasetest.FailWritesOn(t, db, "monster_book_collections")

	l, _ := test.NewNullLogger()
	cmd := mbmsg.Command[mbmsg.CardPickedUpBody]{
		CharacterId: 1001,
		EventId:     uuid.New(),
		Type:        mbmsg.CommandTypeCardPickedUp,
		Body:        mbmsg.CardPickedUpBody{CardId: 2380000},
	}
	handleCardPickedUp(db)(l, ctx, cmd)

	var cards int64
	require.NoError(t, db.Table("monster_book_cards").Count(&cards).Error)
	require.Zero(t, cards, "card upsert must roll back with the failed collection write")
}
```

(The handler swallows the error into a log line by design; the assertion is on DB state. No Kafka is reached: the buffered messages are discarded when the flow errors. If `mbmsg.Command`'s fields differ from `CharacterId/EventId/Type/Body`, read `kafka/message/monsterbook/kafka.go:21-30` and match.)

- [ ] **Step 2: Run it — expect PASS (raw tx already rolls back; this locks behavior across the inversion)**

```bash
cd services/atlas-monster-book/atlas.com/monster-book && go test -race -run TestHandleCardPickedUp ./kafka/consumer/monsterbook/
```

Expected: PASS.

- [ ] **Step 3: Invert `handleCardPickedUp`**

In `kafka/consumer/monsterbook/consumer.go`, add to imports:

```go
	database "github.com/Chronicle20/atlas/libs/atlas-database"
```

Replace the body of `handleCardPickedUp` (lines 49-77) so Emit wraps the transaction instead of the reverse:

```go
func handleCardPickedUp(db *gorm.DB) func(l logrus.FieldLogger, ctx context.Context, cmd mbmsg.Command[mbmsg.CardPickedUpBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, cmd mbmsg.Command[mbmsg.CardPickedUpBody]) {
		if cmd.Type != mbmsg.CommandTypeCardPickedUp {
			return
		}
		characterId := character.Id(cmd.CharacterId)
		cardId := item.Id(cmd.Body.CardId)
		// Emit outside, transaction inside: buffered messages publish only
		// after the transaction commits.
		err := message.Emit(producer.ProviderImpl(l)(ctx))(func(mb *message.Buffer) error {
			return database.ExecuteTransaction(db.WithContext(ctx), func(tx *gorm.DB) error {
				cp := card.NewProcessor(l, ctx, tx)
				colp := collection.NewProcessor(l, ctx, tx)
				res, err := cp.Add(mb)(cmd.EventId, characterId, cardId)
				if err != nil {
					return err
				}
				if res.Duplicate {
					return nil
				}
				if res.Inserted {
					return colp.RecomputeAndEmit(mb)(characterId)
				}
				return nil
			})
		})
		if err != nil {
			l.WithError(err).Errorf("Failed to handle CARD_PICKED_UP for character %d card %d.", cmd.CharacterId, cmd.Body.CardId)
		}
	}
}
```

(Keep the file's existing import aliases — `message`/`kmessage`, `producer` — exactly as they are; only the nesting order and the `Transaction`→`ExecuteTransaction` call change.)

- [ ] **Step 4: Convert `handleStatusEventDeleted`**

In `kafka/consumer/character/consumer.go`, add the same `database` import and change line 49 from:

```go
		if err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
```

to:

```go
		if err := database.ExecuteTransaction(db.WithContext(ctx), func(tx *gorm.DB) error {
```

(No emit in this handler — the two cascading deletes just become helper-wrapped.)

- [ ] **Step 5: Verify**

```bash
cd services/atlas-monster-book/atlas.com/monster-book && go test -race ./... && go vet ./... && go build ./...
grep -rn "\.Transaction(" kafka/ --include='*.go' | grep -v _test.go
```

Expected: all green; grep empty. Existing processor tests confirm happy-path event emission unchanged (FR-2.4).

- [ ] **Step 6: Commit and record in audit.md (note the [E] fix: events no longer publish for rolled-back writes)**

```bash
git add services/atlas-monster-book docs/tasks/task-119-db-transaction-coverage/audit.md
git commit -m "fix(atlas-monster-book): publish events after commit, standardize onto ExecuteTransaction

CARD_PICKED_UP previously nested message.Emit inside db.Transaction, so
events published before commit — a failed tx still announced a card. Emit
now wraps the transaction (canonical guilds composition); happy-path events
are unchanged."
```

---

### Task 9: atlas-marriages — eliminate manual Begin/Commit, move emit outside the tx + rollback test

`executeInTransaction` (marriage/processor.go:1688-1717) is the repo's only manual `Begin`/`Rollback`/`Commit`; its single caller `AcceptProposalWithTransactionAndEmit` (1581) runs `message.EmitWithResult` INSIDE the tx (flag [E]).

**Files:**
- Modify: `services/atlas-marriages/atlas.com/marriages/marriage/processor.go:1578-1718`
- Test: `services/atlas-marriages/atlas.com/marriages/marriage/processor_rollback_test.go` (new)

**Interfaces:**
- Consumes: `GetProposalByIdProvider(db, log)(proposalId)`, `UpdateProposal(db, log)(proposal)`, `CreateMarriage(db, log)(proposerId, targetId, tenantId)`, `UpdateMarriage(db, log)(marriage)` (existing administrators, unchanged); `ProposalAcceptedEventProvider`, `MarriageCreatedEventProvider` (existing).
- Produces: `AcceptProposalWithTransactionAndEmit(transactionId uuid.UUID, proposalId uint32) (Marriage, error)` — signature unchanged (it is on the `Processor` interface, line 47); `executeInTransaction` keeps its signature but delegates to `database.ExecuteTransaction`.

- [ ] **Step 1: Write the rollback test**

Create `services/atlas-marriages/atlas.com/marriages/marriage/processor_rollback_test.go`:

```go
package marriage

import (
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
)

// Proposal acceptance updates the proposal row then creates the marriage row
// (class A: proposals + marriages). Failing the marriage create must roll the
// proposal back to pending.
func TestAcceptProposal_RollsBackProposalUpdateWhenMarriageCreateFails(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, Migration)
	tid := uuid.New()
	ctx := databasetest.TenantContext(tid)

	prop := ProposalEntity{
		ProposerId: 1001,
		TargetId:   1002,
		Status:     ProposalStatusPending,
		ProposedAt: time.Now(),
		ExpiresAt:  time.Now().Add(time.Hour),
		TenantId:   tid,
	}
	require.NoError(t, db.Create(&prop).Error)

	databasetest.FailWritesOn(t, db, "marriages", databasetest.WriteCreate)

	l, _ := test.NewNullLogger()
	_, err := NewProcessor(l, ctx, db).AcceptProposalWithTransactionAndEmit(uuid.New(), prop.ID)
	require.Error(t, err)

	var after ProposalEntity
	require.NoError(t, db.First(&after, prop.ID).Error)
	require.Equal(t, ProposalStatusPending, after.Status, "proposal update must roll back with the failed marriage create")

	var marriages int64
	require.NoError(t, db.Model(&Entity{}).Count(&marriages).Error)
	require.Zero(t, marriages)
}
```

(No Kafka is reached: `EmitWithResult` publishes nothing when the inner function errors, before AND after this change.)

- [ ] **Step 2: Run it — expect PASS (manual tx already rolls back on operation error)**

```bash
cd services/atlas-marriages/atlas.com/marriages && go test -race -run TestAcceptProposal_RollsBack ./marriage/
```

Expected: PASS.

- [ ] **Step 3: Replace `executeInTransaction`**

In `marriage/processor.go`, add to imports:

```go
	database "github.com/Chronicle20/atlas/libs/atlas-database"
```

Replace the whole function (lines 1686-1718) with:

```go
// executeInTransaction runs the operation inside database.ExecuteTransaction,
// handing it a shadow processor bound to the transaction handle. Event
// emission must happen OUTSIDE this wrapper: buffer inside, publish after
// commit.
func (p *ProcessorImpl) executeInTransaction(operation func(*ProcessorImpl) (Marriage, error)) (Marriage, error) {
	var result Marriage
	err := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		txProcessor := &ProcessorImpl{
			log:                p.log,
			ctx:                p.ctx,
			db:                 tx,
			producer:           p.producer,
			characterProcessor: p.characterProcessor,
		}
		var opErr error
		result, opErr = operation(txProcessor)
		return opErr
	})
	if err != nil {
		return Marriage{}, err
	}
	return result, nil
}
```

(Check the current `ProcessorImpl` field list post-task-116 and clone ALL fields, exactly as the old function did.)

- [ ] **Step 4: Invert `AcceptProposalWithTransactionAndEmit`**

Replace the function (lines 1578-1684) so `EmitWithResult` wraps `executeInTransaction` instead of the reverse. Body logic, event content, and ordering are otherwise verbatim from the original:

```go
// AcceptProposalWithTransactionAndEmit accepts a proposal and creates the
// marriage atomically. Events are buffered inside the transaction and publish
// only after it commits.
func (p *ProcessorImpl) AcceptProposalWithTransactionAndEmit(transactionId uuid.UUID, proposalId uint32) (Marriage, error) {
	return message.EmitWithResult[Marriage, uint32](p.producer)(func(buf *message.Buffer) func(uint32) (Marriage, error) {
		return func(proposalId uint32) (Marriage, error) {
			return p.executeInTransaction(func(txProcessor *ProcessorImpl) (Marriage, error) {
				// Get tenant from context
				t := tenant.MustFromContext(p.ctx)

				// Get the proposal
				proposal, err := GetProposalByIdProvider(txProcessor.db, txProcessor.log)(proposalId)()
				if err != nil {
					return Marriage{}, err
				}

				// Check if proposal can be accepted
				if !proposal.CanRespond() {
					return Marriage{}, errors.New("proposal cannot be accepted")
				}

				// Accept the proposal
				acceptedProposal, err := proposal.Accept()
				if err != nil {
					return Marriage{}, err
				}

				// Update the proposal in the database
				if _, err := UpdateProposal(txProcessor.db, txProcessor.log)(acceptedProposal)(); err != nil {
					return Marriage{}, err
				}

				// Create the marriage
				marriageEntity, err := CreateMarriage(txProcessor.db, txProcessor.log)(proposal.ProposerId(), proposal.TargetId(), t.Id())()
				if err != nil {
					return Marriage{}, err
				}

				// Transform entity to domain model
				marriage, err := Make(marriageEntity)
				if err != nil {
					return Marriage{}, err
				}

				// Accept the marriage to set it to engaged status
				engagedMarriage, err := marriage.Accept()
				if err != nil {
					return Marriage{}, err
				}

				// Update the marriage in the database
				updatedEntity, err := UpdateMarriage(txProcessor.db, txProcessor.log)(engagedMarriage)()
				if err != nil {
					return Marriage{}, err
				}

				// Transform entity to domain model
				result, err := Make(updatedEntity)
				if err != nil {
					return Marriage{}, err
				}

				// Buffer ProposalAccepted event (publishes after commit)
				acceptedAt := time.Now()
				if err := buf.Put(marriageMsg.EnvEventTopicStatus, ProposalAcceptedEventProvider(proposalId, result.CharacterId1(), result.CharacterId2(), acceptedAt)); err != nil {
					return Marriage{}, err
				}

				// Buffer MarriageCreated event (publishes after commit)
				marriedAt := time.Now()
				if result.EngagedAt() != nil {
					marriedAt = *result.EngagedAt()
				}
				if err := buf.Put(marriageMsg.EnvEventTopicStatus, MarriageCreatedEventProvider(result.Id(), result.CharacterId1(), result.CharacterId2(), marriedAt)); err != nil {
					return Marriage{}, err
				}

				p.log.WithFields(logrus.Fields{
					"transactionId": transactionId,
					"proposalId":    proposalId,
					"marriageId":    result.Id(),
				}).Info("Proposal accepted and marriage created with full transactional consistency")

				return result, nil
			})
		}
	})(proposalId)
}
```

(Match the file's existing aliases for `message`/`marriageMsg` and the exact `EmitWithResult` generic instantiation style already used in the file.)

- [ ] **Step 5: Verify**

```bash
cd services/atlas-marriages/atlas.com/marriages && go test -race ./... && go vet ./... && go build ./...
grep -n "\.Begin()\|\.Commit()\|\.Rollback()" marriage/processor.go
```

Expected: all green; grep empty (the manual tx is gone).

- [ ] **Step 6: Commit and record in audit.md (note the [E] fix)**

```bash
git add services/atlas-marriages docs/tasks/task-119-db-transaction-coverage/audit.md
git commit -m "fix(atlas-marriages): replace manual Begin/Commit with ExecuteTransaction, publish after commit

executeInTransaction hand-rolled Begin/Rollback/Commit and ran
message.EmitWithResult inside the tx, publishing proposal/marriage events
before commit. Emit now wraps the transaction; happy-path events unchanged."
```

---

### Task 10: atlas-storage — `WithTransaction` + wrap `GetOrCreateStorageId`

First of three storage tasks (the genuine unwrapped gaps, design §6.3). This one adds the tx-threading plumbing both storage processors lack and wraps the asset-side read-then-create so it joins a caller's transaction.

**Files:**
- Modify: `services/atlas-storage/atlas.com/storage/storage/processor.go:28-40` (add method)
- Modify: `services/atlas-storage/atlas.com/storage/asset/processor.go:14-78`
- Test: `services/atlas-storage/atlas.com/storage/asset/processor_rollback_test.go` (new)

**Interfaces:**
- Produces: `(*storage.Processor).WithTransaction(tx *gorm.DB) *Processor` and `(*asset.Processor).WithTransaction(tx *gorm.DB) *Processor` (monster-book shape: clone with `db` swapped); `GetOrCreateStorageId` signature unchanged, now transactional and join-capable. Task 11 consumes `storage.WithTransaction`.

- [ ] **Step 1: Write the join test**

`GetOrCreateStorageId` has exactly one write (class C by the refined taxonomy) — a fail-mid-flow rollback test is undefinable. The design still wraps it so it JOINS a caller's tx (§6.3); the test proves join semantics: an outer failure discards the created storage row.

Create `services/atlas-storage/atlas.com/storage/asset/processor_rollback_test.go`:

```go
package asset

import (
	"errors"
	"testing"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func storagesMigration(db *gorm.DB) error { return db.AutoMigrate(&StorageEntity{}) }

// GetOrCreateStorageId must join an enclosing transaction (re-entrancy), so a
// failing caller discards the storage row it created.
func TestGetOrCreateStorageId_JoinsCallerTransaction(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, storagesMigration)
	ctx := databasetest.TenantContext(uuid.New())
	l, _ := test.NewNullLogger()

	err := database.ExecuteTransaction(db.WithContext(ctx), func(tx *gorm.DB) error {
		id, err := NewProcessor(l, ctx, tx).GetOrCreateStorageId(0, 999)
		require.NoError(t, err)
		require.NotEqual(t, uuid.Nil, id)
		return errors.New("caller fails after storage creation")
	})
	require.Error(t, err)

	var count int64
	require.NoError(t, db.Table("storages").Count(&count).Error)
	require.Zero(t, count, "storage created inside the caller's tx must roll back with it")
}
```

- [ ] **Step 2: Run it — expect FAIL (current code writes through the raw handle; wait — the processor receives `tx` here, so it may pass)**

```bash
cd services/atlas-storage/atlas.com/storage && go test -race -run TestGetOrCreateStorageId ./asset/
```

Expected: PASS even before the change (the test hands the processor the tx handle directly and the fixed lib makes the inner `ExecuteTransaction`-less writes ride that handle). This test is a semantics lock, not a red-green gate — note that in the test run output and proceed. The wrap in Step 3 matters for REST-path callers that construct the processor with the root `db` (`asset/resource.go:34`) and for future composition.

- [ ] **Step 3: Wrap `GetOrCreateStorageId` and add both `WithTransaction` methods**

In `services/atlas-storage/atlas.com/storage/asset/processor.go`: add import `database "github.com/Chronicle20/atlas/libs/atlas-database"`, add after `NewProcessor`:

```go
// WithTransaction returns a clone of the processor bound to the transaction handle.
func (p *Processor) WithTransaction(tx *gorm.DB) *Processor {
	return &Processor{
		l:   p.l,
		ctx: p.ctx,
		db:  tx,
	}
}
```

and replace `GetOrCreateStorageId` (lines 50-78) with:

```go
func (p *Processor) GetOrCreateStorageId(worldId world.Id, accountId uint32) (uuid.UUID, error) {
	t := tenant.MustFromContext(p.ctx)

	var id uuid.UUID
	err := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		var storageEntity StorageEntity
		err := tx.Where("world_id = ? AND account_id = ?", byte(worldId), accountId).
			First(&storageEntity).Error
		if err == nil {
			id = storageEntity.Id
			return nil
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			storageEntity = StorageEntity{
				TenantId:  t.Id(),
				Id:        uuid.New(),
				WorldId:   byte(worldId),
				AccountId: accountId,
				Capacity:  4,
				Mesos:     0,
			}
			if createErr := tx.Create(&storageEntity).Error; createErr != nil {
				return createErr
			}
			id = storageEntity.Id
			return nil
		}
		return err
	})
	if err != nil {
		return uuid.Nil, err
	}
	return id, nil
}
```

In `services/atlas-storage/atlas.com/storage/storage/processor.go`, add after `NewProcessor` (line 40):

```go
// WithTransaction returns a clone of the processor bound to the transaction handle.
func (p *Processor) WithTransaction(tx *gorm.DB) *Processor {
	return &Processor{
		l:   p.l,
		ctx: p.ctx,
		db:  tx,
	}
}
```

- [ ] **Step 4: Verify**

```bash
cd services/atlas-storage/atlas.com/storage && go test -race ./... && go vet ./... && go build ./...
```

Expected: all green.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-storage
git commit -m "feat(atlas-storage): WithTransaction plumbing; GetOrCreateStorageId joins caller transactions"
```

---

### Task 11: atlas-storage — wrap `ExpireAndEmit` (TDD red→green)

The real class-A gap: delete asset → emit mid-flow → create replacement, all unwrapped (`storage/processor.go:720`). A crash after the delete loses the asset AND the replacement; the event fires even if the replacement failed. Wrap delete + replacement-create in one tx; emit strictly after commit.

**Files:**
- Modify: `services/atlas-storage/atlas.com/storage/storage/processor.go:720-765`
- Test: `services/atlas-storage/atlas.com/storage/storage/processor_rollback_test.go` (new)

**Interfaces:**
- Consumes: `storage.WithTransaction` (Task 10); `asset.GetById(db)`, `asset.Delete(l, db)`, `asset.GetByStorageId(db)`, `asset.Create(l, db, tenantId)`, `asset.NewBuilder` (existing).
- Produces: `ExpireAndEmit` signature unchanged. **Failure-mode change (this IS the requirement):** a failed replacement-create now rolls back the delete and returns the error (previously: Warn + keep the delete + event already sent). Happy-path event content unchanged. Record in audit.md.

- [ ] **Step 1: Write the failing rollback test**

Create `services/atlas-storage/atlas.com/storage/storage/processor_rollback_test.go`:

```go
package storage

import (
	"testing"

	"atlas-storage/asset"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
)

// ExpireAndEmit deletes the expired asset then creates its replacement
// (class A-shaped: two writes that must move together). Failing the
// replacement create must restore the deleted asset.
func TestExpireAndEmit_RollsBackDeleteWhenReplacementCreateFails(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, Migration, asset.Migration)
	tid := uuid.New()
	ctx := databasetest.TenantContext(tid)
	l, _ := test.NewNullLogger()

	s, err := Create(l, db.WithContext(ctx), tid)(world.Id(0), 5001)
	require.NoError(t, err)
	a, err := asset.Create(l, db.WithContext(ctx), tid)(asset.NewBuilder(s.Id(), 4000000).SetSlot(0).SetQuantity(1).Build())
	require.NoError(t, err)

	databasetest.FailWritesOn(t, db, "storage_assets", databasetest.WriteCreate)

	p := NewProcessor(l, ctx, db)
	err = p.ExpireAndEmit(uuid.New(), world.Id(0), 5001, a.Id(), false, 4000001, "expired")
	require.Error(t, err, "replacement-create failure must surface as an error")

	var assets int64
	require.NoError(t, db.Table("storage_assets").Count(&assets).Error)
	require.EqualValues(t, 1, assets, "the expired asset's delete must roll back")
}
```

(Adapt the `Create`/`asset.Create` invocation shapes to the exact administrator signatures if they differ — `Create(p.l, p.db.WithContext(p.ctx), t.Id())(worldId, accountId)` and `asset.Create(p.l, ..., t.Id())(model)` are the shapes used inside the processor. `a.Id()` is the created asset's id from the returned model.)

- [ ] **Step 2: Run it — expect FAIL**

```bash
cd services/atlas-storage/atlas.com/storage && go test -race -run TestExpireAndEmit ./storage/
```

Expected: FAIL twice over — current code returns `nil` on replacement failure (Warn-and-continue) and the asset stays deleted (count 0, not 1).

- [ ] **Step 3: Wrap the flow**

In `storage/processor.go`, add import `database "github.com/Chronicle20/atlas/libs/atlas-database"`, then replace `ExpireAndEmit` (lines 720-765) with:

```go
func (p *Processor) ExpireAndEmit(transactionId uuid.UUID, worldId world.Id, accountId uint32, assetId uint32, isCash bool, replaceItemId uint32, replaceMessage string) error {
	t := tenant.MustFromContext(p.ctx)

	err := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		a, err := asset.GetById(tx)(assetId)
		if err != nil {
			p.l.WithError(err).Errorf("Failed to find asset [%d] for expiration.", assetId)
			return err
		}

		if err := asset.Delete(p.l, tx)(assetId); err != nil {
			p.l.WithError(err).Errorf("Failed to delete expired asset [%d].", assetId)
			return err
		}

		if replaceItemId > 0 {
			p.l.Debugf("Creating replacement item [%d] for expired storage item [%d].", replaceItemId, a.TemplateId())

			s, err := p.WithTransaction(tx).GetOrCreateStorage(worldId, accountId)
			if err != nil {
				p.l.WithError(err).Errorf("Failed to get storage for replacement item creation.")
				return err
			}

			assets, err := asset.GetByStorageId(tx)(s.Id())
			if err != nil {
				p.l.WithError(err).Errorf("Failed to get assets for slot calculation.")
				return err
			}
			nextSlot := int16(len(assets))

			replacement := asset.NewBuilder(s.Id(), replaceItemId).
				SetSlot(nextSlot).
				Build()

			if _, err := asset.Create(p.l, tx, t.Id())(replacement); err != nil {
				p.l.WithError(err).Errorf("Failed to create replacement item [%d] for account [%d].", replaceItemId, accountId)
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	// Publish only after the transaction commits: no event for a rolled-back expiry.
	_ = p.emitExpiredEvent(transactionId, worldId, accountId, isCash, replaceItemId, replaceMessage)

	p.l.Debugf("Expired asset [%d] from storage for account [%d].", assetId, accountId)
	return nil
}
```

- [ ] **Step 4: Run tests to verify green**

```bash
cd services/atlas-storage/atlas.com/storage && go test -race ./... && go vet ./... && go build ./...
```

Expected: the new test PASSES; existing tests stay green.

- [ ] **Step 5: Commit and record in audit.md (include the failure-mode change note)**

```bash
git add services/atlas-storage docs/tasks/task-119-db-transaction-coverage/audit.md
git commit -m "fix(atlas-storage): expire+replace is one transaction, event publishes after commit

Previously: delete committed, event published mid-flow, replacement-create
failure was swallowed (Warn) — a crash or failure desynced the pair and
still announced the expiry. Now delete+create commit together and the event
only fires for a committed expiry."
```

---

### Task 12: atlas-storage — wrap `MergeAndSort` (TDD red→green)

The merge/compact/sort sequence (`storage/processor.go:483`) runs many writes (quantity updates, deletes, slot updates) unwrapped. Two structural moves: hoist the `getSlotMaxByTemplateId` lookups (atlas-data REST calls — network I/O) ABOVE the tx (design §8 risk table: no network I/O inside any introduced tx), then wrap all writes including `sortAssets` in one `ExecuteTransaction`.

**Files:**
- Modify: `services/atlas-storage/atlas.com/storage/storage/processor.go:483-600` (`MergeAndSort`, `sortAssets`)
- Test: append to `services/atlas-storage/atlas.com/storage/storage/processor_rollback_test.go`

**Interfaces:**
- Consumes: `asset.UpdateQuantity(l, db)(id, qty)`, `asset.Delete(l, db)(id)`, `asset.UpdateSlot(l, db)(id, slot)` (existing, already take the handle).
- Produces: `MergeAndSort` signature unchanged; `sortAssets` becomes `func (p *Processor) sortAssets(db *gorm.DB, assets []asset.Model) error` (private; sole caller is `MergeAndSort`).

- [ ] **Step 1: Write the failing rollback test**

Append to `storage/processor_rollback_test.go`:

```go
// MergeAndSort is a loop of quantity-updates + deletes + re-slotting
// (class B). Failing a later delete must roll back the earlier quantity
// updates, restoring the pre-merge stacks.
func TestMergeAndSort_RollsBackQuantityUpdatesWhenDeleteFails(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, Migration, asset.Migration)
	tid := uuid.New()
	ctx := databasetest.TenantContext(tid)
	l, _ := test.NewNullLogger()

	s, err := Create(l, db.WithContext(ctx), tid)(world.Id(0), 5002)
	require.NoError(t, err)
	// Two mergeable stacks of the same consumable: 30 + 30 with slotMax 100
	// (the atlas-data lookup fails in tests, falling back to 100) merge into
	// one stack of 60, deleting the second row.
	_, err = asset.Create(l, db.WithContext(ctx), tid)(asset.NewBuilder(s.Id(), 2000000).SetSlot(0).SetQuantity(30).Build())
	require.NoError(t, err)
	_, err = asset.Create(l, db.WithContext(ctx), tid)(asset.NewBuilder(s.Id(), 2000000).SetSlot(1).SetQuantity(30).Build())
	require.NoError(t, err)

	databasetest.FailWritesOn(t, db, "storage_assets", databasetest.WriteDelete)

	p := NewProcessor(l, ctx, db)
	require.Error(t, p.MergeAndSort(world.Id(0), 5002))

	var quantities []uint32
	require.NoError(t, db.Table("storage_assets").Order("slot").Pluck("quantity", &quantities).Error)
	require.Equal(t, []uint32{30, 30}, quantities, "quantity update must roll back with the failed delete")
}
```

- [ ] **Step 2: Run it — expect FAIL**

```bash
cd services/atlas-storage/atlas.com/storage && go test -race -run TestMergeAndSort ./storage/
```

Expected: FAIL — the error IS returned today, but the first stack's quantity was already committed as 60 (`[60 30]`, not `[30 30]`).

- [ ] **Step 3: Restructure `MergeAndSort` and `sortAssets`**

Replace `MergeAndSort` (lines 483-570) with:

```go
func (p *Processor) MergeAndSort(worldId world.Id, accountId uint32) error {
	s, err := GetByWorldAndAccountId(p.l, p.db.WithContext(p.ctx))(worldId, accountId)
	if err != nil {
		return err
	}

	assets, err := asset.GetByStorageId(p.db.WithContext(p.ctx))(s.Id())
	if err != nil {
		return err
	}

	var nonStackables []asset.Model
	stackableGroups := make(map[mergeKey][]asset.Model)

	for _, a := range assets {
		if !a.IsStackable() {
			nonStackables = append(nonStackables, a)
			continue
		}

		// Check if consumable is rechargeable (cannot merge)
		if a.IsConsumable() && a.Rechargeable() > 0 {
			nonStackables = append(nonStackables, a)
			continue
		}

		key := mergeKey{
			templateId: a.TemplateId(),
			ownerId:    a.OwnerId(),
			flag:       a.Flag(),
		}
		stackableGroups[key] = append(stackableGroups[key], a)
	}

	// Prefetch slot maxima before opening the transaction — these are
	// atlas-data lookups (network I/O) and must not run inside the tx.
	slotMaxByTemplate := make(map[uint32]uint32, len(stackableGroups))
	for key := range stackableGroups {
		if _, seen := slotMaxByTemplate[key.templateId]; seen {
			continue
		}
		slotMax, err := p.getSlotMaxByTemplateId(key.templateId)
		if err != nil || slotMax == 0 {
			slotMax = 100
		}
		slotMaxByTemplate[key.templateId] = slotMax
	}

	return database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		if len(stackableGroups) == 0 {
			return p.sortAssets(tx, assets)
		}

		var mergedAssets []asset.Model

		for key, group := range stackableGroups {
			slotMax := slotMaxByTemplate[key.templateId]

			var totalQuantity uint32
			for _, a := range group {
				totalQuantity += a.Quantity()
			}

			numStacks := (totalQuantity + slotMax - 1) / slotMax
			if numStacks == 0 {
				numStacks = 1
			}

			assetsToKeep := min(uint32(len(group)), numStacks)

			sort.Slice(group, func(i, j int) bool {
				return group[i].Slot() < group[j].Slot()
			})

			remainingQuantity := totalQuantity
			for i := uint32(0); i < assetsToKeep; i++ {
				a := group[i]
				newQuantity := min(remainingQuantity, slotMax)
				remainingQuantity -= newQuantity

				if err := asset.UpdateQuantity(p.l, tx)(a.Id(), newQuantity); err != nil {
					return err
				}

				mergedAssets = append(mergedAssets, a)
			}

			for i := int(assetsToKeep); i < len(group); i++ {
				if err := asset.Delete(p.l, tx)(group[i].Id()); err != nil {
					return err
				}
			}
		}

		allAssets := append(nonStackables, mergedAssets...)

		return p.sortAssets(tx, allAssets)
	})
}
```

And change `sortAssets` (lines 572-600) to take the handle:

```go
func (p *Processor) sortAssets(db *gorm.DB, assets []asset.Model) error {
	byInventoryType := make(map[byte][]asset.Model)
	for _, a := range assets {
		invType := inventoryTypeFromTemplateId(a.TemplateId())
		byInventoryType[invType] = append(byInventoryType[invType], a)
	}

	for it := range byInventoryType {
		group := byInventoryType[it]
		sort.Slice(group, func(i, j int) bool {
			return group[i].TemplateId() < group[j].TemplateId()
		})
		byInventoryType[it] = group
	}

	for _, group := range byInventoryType {
		for i, a := range group {
			newSlot := int16(i)
			if a.Slot() != newSlot {
				if err := asset.UpdateSlot(p.l, db)(a.Id(), newSlot); err != nil {
					return err
				}
			}
		}
	}

	return nil
}
```

(If `sortAssets` has callers other than `MergeAndSort` — verify with `grep -n "sortAssets" storage/processor.go` — pass them `p.db.WithContext(p.ctx)`.)

- [ ] **Step 4: Run tests to verify green**

```bash
cd services/atlas-storage/atlas.com/storage && go test -race ./... && go vet ./... && go build ./...
```

Expected: new test PASSES; existing tests green.

- [ ] **Step 5: Commit and record in audit.md (note tx bound: ≤ storage capacity ~100 rows, no network I/O inside — slotMax prefetched)**

```bash
git add services/atlas-storage docs/tasks/task-119-db-transaction-coverage/audit.md
git commit -m "fix(atlas-storage): MergeAndSort merge/compact/sort writes are one transaction

Quantity updates, stack deletes, and re-slotting previously committed
one-by-one — a mid-flow failure left stacks half-merged. Slot-max lookups
(atlas-data REST) are prefetched so no network I/O runs inside the tx."
```

---

### Task 13: Audit-driven wraps for atlas-account / atlas-ban / atlas-maps (conditional)

The design survey (§3) expects all three to be class C (single-statement writes; account's RMWs have exactly one write each; ban's possible ban↔history pairing unconfirmed). The Task 3 audit is the authority.

**Files:**
- Modify: `docs/tasks/task-119-db-transaction-coverage/audit.md` (always)
- Modify: service files ONLY if the audit surfaced a class-A/B flow

**Interfaces:**
- Consumes: Task 3 verdicts; the canonical composition (Task 8's `handleCardPickedUp` is the reference implementation for consumer-driven flows, Task 5's one-line swap for processor-internal flows).

- [ ] **Step 1: Re-read the Task 3 verdicts for these three services post-rebase**

Confirm classifications still hold against post-task-116 code:

```bash
grep -rn "\.Create(\|\.Save(\|\.Update(\|\.Updates(\|\.Delete(\|\.Exec(" services/atlas-account services/atlas-ban services/atlas-maps --include='*.go' | grep -v _test.go
```

Specifically resolve the ban↔history question: does any single entry point write both a `ban` row and a `history` row? Cite `file:line` either way in audit.md.

- [ ] **Step 2 (only if a class-A/B flow exists): wrap it**

Apply the canonical pattern — `database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error { ... })` with all writes on `tx`, any emit buffered outside — plus a rollback test in the same shape as Task 5's (fail the flow's second write with `databasetest.FailWritesOn`, assert the first rolled back, using that service's `Migration` and entity types). If the flow is high-frequency, record its call rate in audit.md BEFORE wrapping (PRD §8).

- [ ] **Step 3 (if class C confirmed): finalize the verdicts**

Write the class-C verdicts with race annotations where an RMW gap exists (e.g. account `GetOrCreate` name uniqueness — annotate that closing it needs a unique constraint, out of scope). No code change.

- [ ] **Step 4: Verify (only if code changed) and commit**

```bash
cd services/atlas-<svc>/atlas.com/<name> && go test -race ./... && go vet ./... && go build ./...
git add docs/tasks/task-119-db-transaction-coverage/audit.md services/atlas-account services/atlas-ban services/atlas-maps
git commit -m "docs(task-119): finalize account/ban/maps verdicts (wraps applied where the audit demanded)"
```

---

### Task 14: Finalize audit.md and annotate DL-4

**Files:**
- Modify: `docs/tasks/task-119-db-transaction-coverage/audit.md`
- Modify: `docs/architectural-improvements.md` (DL-4 entry, ~line 152)

- [ ] **Step 1: Fill every remediation pointer**

Replace each `_(commit: pending — Task N)_` placeholder with the actual commit SHA (`git log --oneline`). Every class-A/B/[T]/[E] row must point at a commit; every class-C/D row must carry its justification. Verify no placeholder remains:

```bash
grep -n "pending" docs/tasks/task-119-db-transaction-coverage/audit.md
```

Expected: no output.

- [ ] **Step 2: Write the closing matrix**

A 14-row table: service → classes found → action taken (standardized / wrapped / no change) → rollback-test file → status. Include the D0 lib fix and its blast radius (53 call sites / 18 services activated) as a preamble row, and the seeder-cycle follow-up candidate (changing `libs/atlas-seeder` semantics is out of scope, affects out-of-scope consumers) as a recorded note, not a task.

- [ ] **Step 3: Annotate DL-4**

Read the DL-4 entry in `docs/architectural-improvements.md` (cited at :152 in the PRD) and append a status line in the document's existing style, e.g.: `Status (task-119, 2026-07-02): audited all 14 services; ExecuteTransaction no-op fixed; multi-statement flows wrapped/standardized with rollback tests — see docs/tasks/task-119-db-transaction-coverage/audit.md.`

- [ ] **Step 4: Commit**

```bash
git add docs/tasks/task-119-db-transaction-coverage/audit.md docs/architectural-improvements.md
git commit -m "docs(task-119): close out audit — remediation pointers, 14-service matrix, DL-4 annotation"
```

---

### Task 15: Fleet verification gate

The lib fix changes runtime behavior of every `ExecuteTransaction` importer (design §2.4): the 18 already-calling services must be tested even though their code is untouched.

**Files:** none (verification only).

- [ ] **Step 1: Test every changed module**

```bash
for m in libs/atlas-database \
         services/atlas-keys/atlas.com/keys \
         services/atlas-families/atlas.com/family \
         services/atlas-npc-conversations/atlas.com/npc \
         services/atlas-monster-book/atlas.com/monster-book \
         services/atlas-marriages/atlas.com/marriages \
         services/atlas-storage/atlas.com/storage; do
  echo "== $m" && (cd "$m" && go test -race ./... && go vet ./... && go build ./...) || echo "FAILED: $m"
done
```

(Add any service Task 13 touched.) Expected: every module green; any `FAILED:` line is a stop-and-fix.

- [ ] **Step 2: Test the 18 activated-semantics modules**

```bash
for s in atlas-buddies atlas-cashshop atlas-character atlas-configurations atlas-data \
         atlas-drop-information atlas-fame atlas-gachapons atlas-guilds atlas-inventory \
         atlas-merchant atlas-mounts atlas-notes atlas-npc-shops atlas-pets atlas-quest \
         atlas-skills atlas-tenants; do
  d=$(ls -d services/$s/atlas.com/*/ | head -1)
  echo "== $s" && (cd "$d" && go test -race ./...) || echo "FAILED: $s"
done
```

Expected: all green. A failure here means a test depended on partial-write survival — that is the D0 fix working as designed; fix the TEST's assumption (or surface the finding to the user if the production code truly relied on non-atomicity), never revert the lib.

- [ ] **Step 3: Bake everything (the lib is COPY'd into every image)**

```bash
docker buildx bake all-go-services
```

Expected: exit 0.

- [ ] **Step 4: Redis key guard**

```bash
tools/redis-key-guard.sh
```

Expected: clean.

- [ ] **Step 5: Acceptance-criteria sweep**

Verify each PRD §10 box against the tree: audit.md complete (no service skipped, every entry classified with citations); zero manual `Begin/Commit`:

```bash
grep -rn "\.Begin()\|\.Commit()\|\.Rollback()" services --include='*.go' | grep -v _test.go | grep -vE "atlas-(buddies|cashshop|character|configurations|data|drop-information|fame|gachapons|guilds|inventory|merchant|mounts|notes|npc-shops|pets|quest|skills|tenants)"
```

Expected: no hits in the 14 audited services. No emit inside any introduced tx (re-read the 6 diffs). Report results to the user with actual command output.

---

### Task 16: Code review, then finish the branch

**Files:** none.

- [ ] **Step 1: Request code review (BEFORE any PR — repo rule)**

Invoke `superpowers:requesting-code-review`. It dispatches `plan-adherence-reviewer` and `backend-guidelines-reviewer` in parallel; findings land in `docs/tasks/task-119-db-transaction-coverage/` review artifacts. Address findings per `superpowers:receiving-code-review` (verify before implementing; push back with evidence where a finding is wrong).

- [ ] **Step 2: Finish the branch**

Invoke `superpowers:finishing-a-development-branch`. Present the merge/PR options to the user, including the still-open design §2.4 question: whether to rebase-cut commit 1 (the lib fix) into a standalone PR ahead of the task PR. PR body cites: the D0 discovery, the audit artifact, and the per-service remediation commits.

---

## Self-Review (completed at plan-writing)

- **Spec coverage:** D0 fix (§2) → Task 1; FailWritesOn (§7.2) → Task 2; audit methodology/taxonomy/format (§5) → Task 3; sequencing gate + §6.5 emit re-check (§8, PRD §7) → Task 4; category-1 standardization (§6.2: npc-conversations, keys, families, monster-book, marriages) → Tasks 5-9; category-2 storage gaps (§6.3) → Tasks 10-12; account/ban/maps audit-decides (§6.3) → Task 13; category-3 class-D justifications (§6.4) → Tasks 3+14; DL-4 closure (PRD §1) → Task 14; fleet verification (§2.4, §7.3) → Task 15; PRD FR-2.3 no-manual-Begin → Tasks 9+15; PRD open questions (§9) resolved per design — §9.1 in Task 4 Step 4, §9.2 moot (maps class C, map-actions no runtime DB writes).
- **Placeholder scan:** one intentional conditional (Task 13 — code exists only if the audit demands it; the deliverable is the verdict). No TBDs.
- **Type consistency:** `FailWritesOn(t, db, table, verbs...)`/`WriteCreate|WriteUpdate|WriteDelete` consistent across Tasks 2, 5, 7, 8, 9, 11, 12; `WithTransaction(tx) *Processor` (storage/asset, Task 10) matches Task 11's `p.WithTransaction(tx).GetOrCreateStorage(...)`; `sortAssets(db, assets)` defined and consumed only in Task 12; `executeInTransaction` signature preserved between Task 9 Steps 3-4.
