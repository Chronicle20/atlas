# Monster Book Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship the Monster Book feature: a new `atlas-monster-book` microservice that owns per-character card collections and cover, plus the cross-service plumbing (consume-on-pickup wiring, channel packets, quest condition, UI widget) that makes it usable end-to-end.

**Architecture:** New Go service modeled on `atlas-keys` (immutable models, GORM, JSON:API, Kafka via `message.Buffer`/`message.Emit`). Drop pickup → `atlas-inventory` (skip-insert when `consumeOnPickup=true`) → emit generic `ITEM.CONSUMED_ON_PICKUP` → `atlas-consumables` routes card items to `MONSTER_BOOK.CARD_PICKED_UP` → `atlas-monster-book` upserts card + collection (idempotent via `last_event_id` UUID) → emits `CARD_ADDED`/`STATS_CHANGED`/`EXPERIENCE_DISTRIBUTION` → `atlas-channel` translates to v83 packets (`0x53`/`0x54`/effects). Cover read at login via decorator on `character.Processor`. Quest `monsterBookCount` requirement evaluated in `atlas-query-aggregator` via REST to monster-book.

**Tech Stack:** Go 1.25.5, GORM, segmentio/kafka-go, api2go/jsonapi, gorilla/mux. Frontend: React 19 + TanStack Query + Tailwind in `atlas-ui`.

**Reference docs:** PRD `docs/tasks/task-056-monster-book/prd.md` · Design `docs/tasks/task-056-monster-book/design.md` · Context `docs/tasks/task-056-monster-book/context.md`.

**Phase plan:**

| Phase | Tasks | What it delivers |
|---|---|---|
| A | 1–17 | `atlas-monster-book` greenfield service (DB, processors, REST, Kafka, lifecycle, Docker). |
| B | 18–20 | `atlas-inventory` consume-on-pickup branch. |
| C | 21–23 | `atlas-consumables` `ITEM.CONSUMED_ON_PICKUP` consumer. |
| D | 24–28 | `libs/atlas-packet` writers + `atlas-channel` `0x39` recv handler. |
| E | 29–32 | `atlas-channel` outbound consumers + cover decorator. |
| F | 33–36 | `libs/atlas-saga` constant + `atlas-query-aggregator` + `atlas-quest` quest condition. |
| G | 37–40 | `atlas-ui` Monster Book widget. |

The phases are deploy-ordered (design §12). Phase A is the prerequisite; B–G can run in parallel by different subagents only if A is fully merged first.

---

## Phase A — atlas-monster-book service

### Task 1: Service skeleton (go.mod, logger, empty main)

**Files:**
- Create: `services/atlas-monster-book/atlas.com/monster-book/go.mod`
- Create: `services/atlas-monster-book/atlas.com/monster-book/logger/logger.go`
- Create: `services/atlas-monster-book/atlas.com/monster-book/main.go`

Reference template: `services/atlas-keys/atlas.com/keys/{go.mod,logger,main.go}`. Copy them and rename `keys` → `monster-book`, module `atlas-keys` → `atlas-monster-book`.

- [ ] **Step 1: Create the service directory structure**

```bash
mkdir -p services/atlas-monster-book/atlas.com/monster-book/{logger,collection,card,character,kafka/consumer/character,kafka/consumer/monsterbook,kafka/message/character,kafka/message/monsterbook,kafka/producer,rest}
```

- [ ] **Step 2: Write `go.mod` (mirror atlas-keys)**

```
module atlas-monster-book

go 1.25.5

require (
	github.com/Chronicle20/atlas/libs/atlas-constants v0.0.0
	github.com/Chronicle20/atlas/libs/atlas-database v0.0.0-00010101000000-000000000000
	github.com/Chronicle20/atlas/libs/atlas-kafka v0.0.0
	github.com/Chronicle20/atlas/libs/atlas-model v0.0.0
	github.com/Chronicle20/atlas/libs/atlas-rest v0.0.0
	github.com/Chronicle20/atlas/libs/atlas-service v0.0.0-00010101000000-000000000000
	github.com/Chronicle20/atlas/libs/atlas-tenant v0.0.0
	github.com/Chronicle20/atlas/libs/atlas-tracing v0.0.0
	github.com/google/uuid v1.6.0
	github.com/gorilla/mux v1.8.1
	github.com/jtumidanski/api2go v1.0.4
	github.com/segmentio/kafka-go v0.4.51
	github.com/sirupsen/logrus v1.9.4
	gorm.io/gorm v1.31.1
)
```

(The remainder — `replace` directives, indirect deps — must match atlas-keys's `go.mod` line-for-line for the module names listed. Easiest path: copy the file from `services/atlas-keys/atlas.com/keys/go.mod`, replace `module atlas-keys` with `module atlas-monster-book`.)

- [ ] **Step 3: Copy `logger/logger.go` from atlas-keys verbatim**

Path: `services/atlas-monster-book/atlas.com/monster-book/logger/logger.go`. No edits needed — it only depends on the service name passed in at call site.

- [ ] **Step 4: Write minimal `main.go` that compiles**

```go
package main

import (
	"atlas-monster-book/logger"
	"github.com/Chronicle20/atlas/libs/atlas-service"
)

const serviceName = "atlas-monster-book"

func main() {
	l := logger.CreateLogger(serviceName)
	l.Infoln("Starting main service.")
	tdm := service.GetTeardownManager()
	tdm.Wait()
	l.Infoln("Service shutdown.")
}
```

- [ ] **Step 5: Add a `go.work` entry locally and confirm `go build` succeeds**

Run from repo root:

```bash
go work use ./services/atlas-monster-book/atlas.com/monster-book
cd services/atlas-monster-book/atlas.com/monster-book && go build ./...
```

Expected: build succeeds. (Module replaces resolve via root `go.work`.)

- [ ] **Step 6: Commit**

```bash
git add services/atlas-monster-book/atlas.com/monster-book/{go.mod,go.sum,logger,main.go} go.work
git commit -m "feat(monster-book): service skeleton"
```

---

### Task 2: `collection` entity + migration

**Files:**
- Create: `services/atlas-monster-book/atlas.com/monster-book/collection/entity.go`
- Test: `services/atlas-monster-book/atlas.com/monster-book/collection/entity_test.go`

- [ ] **Step 1: Write the entity test (table name + migration)**

```go
package collection

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestEntityTableName(t *testing.T) {
	var e entity
	if got := e.TableName(); got != "monster_book_collections" {
		t.Fatalf("expected monster_book_collections, got %q", got)
	}
}

func TestMigrationCreatesTable(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := Migration(db); err != nil {
		t.Fatalf("migration: %v", err)
	}
	if !db.Migrator().HasTable(&entity{}) {
		t.Fatal("expected monster_book_collections to exist after migration")
	}
}
```

- [ ] **Step 2: Run test (expected to fail: package doesn't compile yet)**

```bash
cd services/atlas-monster-book/atlas.com/monster-book && go test ./collection/...
```

Expected: build error / undefined `entity`, `Migration`.

- [ ] **Step 3: Write `entity.go`**

```go
package collection

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&entity{})
}

type entity struct {
	TenantId         uuid.UUID  `gorm:"primaryKey;autoIncrement:false;not null"`
	CharacterId      uint32     `gorm:"primaryKey;autoIncrement:false;not null"`
	CoverCardId      uint32     `gorm:"not null;default:0"`
	BookLevel        uint16     `gorm:"not null;default:1"`
	NormalCount      uint16     `gorm:"not null;default:0"`
	SpecialCount     uint16     `gorm:"not null;default:0"`
	ExpBonusPercent  uint16     `gorm:"not null;default:0"`
	LastCoverEventId *uuid.UUID `gorm:""`
	CreatedAt        time.Time  `gorm:"autoCreateTime"`
	UpdatedAt        time.Time  `gorm:"autoUpdateTime"`
}

func (entity) TableName() string { return "monster_book_collections" }
```

Add the sqlite driver to go.mod test-only:

```bash
go get gorm.io/driver/sqlite
```

- [ ] **Step 4: Run test (expected to pass)**

```bash
go test ./collection/...
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-monster-book/atlas.com/monster-book/collection/{entity.go,entity_test.go} services/atlas-monster-book/atlas.com/monster-book/{go.mod,go.sum}
git commit -m "feat(monster-book): collection entity + migration"
```

---

### Task 3: `collection` model + builder

**Files:**
- Create: `services/atlas-monster-book/atlas.com/monster-book/collection/model.go`
- Create: `services/atlas-monster-book/atlas.com/monster-book/collection/builder.go`
- Test: `services/atlas-monster-book/atlas.com/monster-book/collection/builder_test.go`

The builder is the only file with logic worth testing (validation). Pattern matches `services/atlas-keys/atlas.com/keys/key/builder.go`.

- [ ] **Step 1: Write builder test**

```go
package collection

import (
	"testing"

	"github.com/google/uuid"
)

func TestBuilderRequiresIdentity(t *testing.T) {
	_, err := NewModelBuilder().Build()
	if err == nil {
		t.Fatal("expected error when characterId is zero")
	}
}

func TestBuilderRoundtrip(t *testing.T) {
	tid := uuid.New()
	m, err := NewModelBuilder().
		SetTenantId(tid).
		SetCharacterId(42).
		SetCoverCardId(2380000).
		SetBookLevel(3).
		SetNormalCount(7).
		SetSpecialCount(2).
		SetExpBonusPercent(3).
		Build()
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if m.CharacterId() != 42 || m.CoverCardId() != 2380000 ||
		m.BookLevel() != 3 || m.NormalCount() != 7 || m.SpecialCount() != 2 ||
		m.ExpBonusPercent() != 3 || m.TenantId() != tid {
		t.Fatalf("roundtrip mismatch: %+v", m)
	}
	if total := m.TotalUniqueCards(); total != 9 {
		t.Fatalf("expected total 9, got %d", total)
	}
}
```

- [ ] **Step 2: Run test (expected to fail: undefined symbols)**

```bash
go test ./collection/...
```

- [ ] **Step 3: Write `model.go`**

```go
package collection

import (
	"time"

	"github.com/google/uuid"
)

type Model struct {
	tenantId         uuid.UUID
	characterId      uint32
	coverCardId      uint32
	bookLevel        uint16
	normalCount      uint16
	specialCount     uint16
	expBonusPercent  uint16
	lastCoverEventId *uuid.UUID
	createdAt        time.Time
	updatedAt        time.Time
}

func (m Model) TenantId() uuid.UUID            { return m.tenantId }
func (m Model) CharacterId() uint32            { return m.characterId }
func (m Model) CoverCardId() uint32            { return m.coverCardId }
func (m Model) BookLevel() uint16              { return m.bookLevel }
func (m Model) NormalCount() uint16            { return m.normalCount }
func (m Model) SpecialCount() uint16           { return m.specialCount }
func (m Model) ExpBonusPercent() uint16        { return m.expBonusPercent }
func (m Model) LastCoverEventId() *uuid.UUID   { return m.lastCoverEventId }
func (m Model) CreatedAt() time.Time           { return m.createdAt }
func (m Model) UpdatedAt() time.Time           { return m.updatedAt }
func (m Model) TotalUniqueCards() uint16       { return m.normalCount + m.specialCount }
```

- [ ] **Step 4: Write `builder.go`**

```go
package collection

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type ModelBuilder struct {
	tenantId         uuid.UUID
	characterId      uint32
	coverCardId      uint32
	bookLevel        uint16
	normalCount      uint16
	specialCount     uint16
	expBonusPercent  uint16
	lastCoverEventId *uuid.UUID
	createdAt        time.Time
	updatedAt        time.Time
}

func NewModelBuilder() *ModelBuilder { return &ModelBuilder{} }

func CloneModelBuilder(m Model) *ModelBuilder {
	return &ModelBuilder{
		tenantId:         m.tenantId,
		characterId:      m.characterId,
		coverCardId:      m.coverCardId,
		bookLevel:        m.bookLevel,
		normalCount:      m.normalCount,
		specialCount:     m.specialCount,
		expBonusPercent:  m.expBonusPercent,
		lastCoverEventId: m.lastCoverEventId,
		createdAt:        m.createdAt,
		updatedAt:        m.updatedAt,
	}
}

func (b *ModelBuilder) SetTenantId(v uuid.UUID) *ModelBuilder         { b.tenantId = v; return b }
func (b *ModelBuilder) SetCharacterId(v uint32) *ModelBuilder         { b.characterId = v; return b }
func (b *ModelBuilder) SetCoverCardId(v uint32) *ModelBuilder         { b.coverCardId = v; return b }
func (b *ModelBuilder) SetBookLevel(v uint16) *ModelBuilder           { b.bookLevel = v; return b }
func (b *ModelBuilder) SetNormalCount(v uint16) *ModelBuilder         { b.normalCount = v; return b }
func (b *ModelBuilder) SetSpecialCount(v uint16) *ModelBuilder        { b.specialCount = v; return b }
func (b *ModelBuilder) SetExpBonusPercent(v uint16) *ModelBuilder     { b.expBonusPercent = v; return b }
func (b *ModelBuilder) SetLastCoverEventId(v *uuid.UUID) *ModelBuilder { b.lastCoverEventId = v; return b }
func (b *ModelBuilder) SetCreatedAt(v time.Time) *ModelBuilder        { b.createdAt = v; return b }
func (b *ModelBuilder) SetUpdatedAt(v time.Time) *ModelBuilder        { b.updatedAt = v; return b }

func (b *ModelBuilder) Build() (Model, error) {
	if b.characterId == 0 {
		return Model{}, errors.New("characterId is required")
	}
	return Model{
		tenantId:         b.tenantId,
		characterId:      b.characterId,
		coverCardId:      b.coverCardId,
		bookLevel:        b.bookLevel,
		normalCount:      b.normalCount,
		specialCount:     b.specialCount,
		expBonusPercent:  b.expBonusPercent,
		lastCoverEventId: b.lastCoverEventId,
		createdAt:        b.createdAt,
		updatedAt:        b.updatedAt,
	}, nil
}

func (b *ModelBuilder) MustBuild() Model {
	m, err := b.Build()
	if err != nil {
		panic("MustBuild: " + err.Error())
	}
	return m
}

// Make is the entity → Model adapter used by EntityProvider.
func Make(e entity) (Model, error) {
	return NewModelBuilder().
		SetTenantId(e.TenantId).
		SetCharacterId(e.CharacterId).
		SetCoverCardId(e.CoverCardId).
		SetBookLevel(e.BookLevel).
		SetNormalCount(e.NormalCount).
		SetSpecialCount(e.SpecialCount).
		SetExpBonusPercent(e.ExpBonusPercent).
		SetLastCoverEventId(e.LastCoverEventId).
		SetCreatedAt(e.CreatedAt).
		SetUpdatedAt(e.UpdatedAt).
		Build()
}
```

- [ ] **Step 5: Run tests (expected PASS)**

```bash
go test ./collection/...
```

- [ ] **Step 6: Commit**

```bash
git add services/atlas-monster-book/atlas.com/monster-book/collection/{model.go,builder.go,builder_test.go}
git commit -m "feat(monster-book): collection model + builder"
```

---

### Task 4: `collection` administrator + provider

**Files:**
- Create: `services/atlas-monster-book/atlas.com/monster-book/collection/administrator.go`
- Create: `services/atlas-monster-book/atlas.com/monster-book/collection/provider.go`
- Test: `services/atlas-monster-book/atlas.com/monster-book/collection/administrator_test.go`

Provides raw GORM helpers used by the processor: idempotent upsert (cover only, since counts are recomputed elsewhere), get-by-character, delete-by-character.

- [ ] **Step 1: Write administrator test**

```go
package collection

import (
	"testing"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := Migration(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestUpsertCreatesAndUpdates(t *testing.T) {
	db := newDB(t)
	tid := uuid.New()
	if _, err := upsertStats(db, tid, 7, statsUpdate{NormalCount: 1, SpecialCount: 0, BookLevel: 1, ExpBonusPercent: 1}); err != nil {
		t.Fatalf("upsert create: %v", err)
	}
	if _, err := upsertStats(db, tid, 7, statsUpdate{NormalCount: 2, SpecialCount: 1, BookLevel: 1, ExpBonusPercent: 1}); err != nil {
		t.Fatalf("upsert update: %v", err)
	}
	got, err := getByCharacter(db, tid, 7)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.NormalCount != 2 || got.SpecialCount != 1 {
		t.Fatalf("expected (2,1) got (%d,%d)", got.NormalCount, got.SpecialCount)
	}
}

func TestSetCoverIdempotent(t *testing.T) {
	db := newDB(t)
	tid := uuid.New()
	if _, err := upsertStats(db, tid, 7, statsUpdate{NormalCount: 1, SpecialCount: 0, BookLevel: 1, ExpBonusPercent: 1}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	eid := uuid.New()
	changed, err := setCover(db, tid, 7, 2380000, eid)
	if err != nil || !changed {
		t.Fatalf("first set: changed=%v err=%v", changed, err)
	}
	changed, err = setCover(db, tid, 7, 2380001, eid) // same eventId, should be no-op
	if err != nil || changed {
		t.Fatalf("dup eventId: changed=%v err=%v", changed, err)
	}
	got, _ := getByCharacter(db, tid, 7)
	if got.CoverCardId != 2380000 {
		t.Fatalf("cover should still be 2380000, got %d", got.CoverCardId)
	}
}
```

- [ ] **Step 2: Run test (expected: undefined symbols)**

```bash
go test ./collection/...
```

- [ ] **Step 3: Write `administrator.go`**

```go
package collection

import (
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type statsUpdate struct {
	NormalCount     uint16
	SpecialCount    uint16
	BookLevel       uint16
	ExpBonusPercent uint16
}

// upsertStats inserts or updates the per-character collection row.
// Returns true if the row was inserted (vs updated).
func upsertStats(db *gorm.DB, tenantId uuid.UUID, characterId uint32, s statsUpdate) (bool, error) {
	e := entity{
		TenantId:        tenantId,
		CharacterId:     characterId,
		NormalCount:     s.NormalCount,
		SpecialCount:    s.SpecialCount,
		BookLevel:       s.BookLevel,
		ExpBonusPercent: s.ExpBonusPercent,
	}
	res := db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "tenant_id"}, {Name: "character_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"normal_count", "special_count", "book_level", "exp_bonus_percent", "updated_at",
		}),
	}).Create(&e)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected == 1, nil
}

// setCover updates the cover card guarded by lastCoverEventId.
// Returns true if the row was modified, false if duplicate eventId.
func setCover(db *gorm.DB, tenantId uuid.UUID, characterId uint32, coverCardId uint32, eventId uuid.UUID) (bool, error) {
	res := db.Model(&entity{}).
		Where("tenant_id = ? AND character_id = ?", tenantId, characterId).
		Where("last_cover_event_id IS NULL OR last_cover_event_id <> ?", eventId).
		Updates(map[string]interface{}{
			"cover_card_id":       coverCardId,
			"last_cover_event_id": eventId,
		})
	if res.Error != nil {
		return false, res.Error
	}
	if res.RowsAffected == 0 {
		// Either the row doesn't exist, or this eventId was already applied.
		// Distinguish by checking existence.
		var count int64
		if err := db.Model(&entity{}).
			Where("tenant_id = ? AND character_id = ?", tenantId, characterId).
			Count(&count).Error; err != nil {
			return false, err
		}
		if count == 0 {
			return false, errors.New("collection row does not exist; cover requires owned card")
		}
		return false, nil
	}
	return true, nil
}

func getByCharacter(db *gorm.DB, tenantId uuid.UUID, characterId uint32) (entity, error) {
	var e entity
	err := db.Where("tenant_id = ? AND character_id = ?", tenantId, characterId).First(&e).Error
	return e, err
}

func deleteByCharacter(db *gorm.DB, tenantId uuid.UUID, characterId uint32) error {
	return db.Where("tenant_id = ? AND character_id = ?", tenantId, characterId).Delete(&entity{}).Error
}
```

- [ ] **Step 4: Write `provider.go`**

```go
package collection

import (
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func byCharacterIdEntityProvider(tenantId uuid.UUID, characterId uint32) database.EntityProvider[entity] {
	return func(db *gorm.DB) model.Provider[entity] {
		return database.Query[entity](db.Where("tenant_id = ? AND character_id = ?", tenantId, characterId), &entity{})
	}
}
```

- [ ] **Step 5: Run tests (expected PASS)**

```bash
go test ./collection/...
```

- [ ] **Step 6: Commit**

```bash
git add services/atlas-monster-book/atlas.com/monster-book/collection/{administrator.go,provider.go,administrator_test.go}
git commit -m "feat(monster-book): collection administrator + provider"
```

---

### Task 5: `card` entity + migration

**Files:**
- Create: `services/atlas-monster-book/atlas.com/monster-book/card/entity.go`
- Test: `services/atlas-monster-book/atlas.com/monster-book/card/entity_test.go`

- [ ] **Step 1: Write entity test**

```go
package card

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestEntityTableName(t *testing.T) {
	if (entity{}).TableName() != "monster_book_cards" {
		t.Fatal("table name mismatch")
	}
}

func TestMigration(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := Migration(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if !db.Migrator().HasTable(&entity{}) {
		t.Fatal("expected monster_book_cards")
	}
}
```

- [ ] **Step 2: Run test (FAIL — undefined)**

```bash
go test ./card/...
```

- [ ] **Step 3: Write `entity.go`**

```go
package card

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&entity{})
}

type entity struct {
	TenantId        uuid.UUID  `gorm:"primaryKey;autoIncrement:false;not null"`
	CharacterId     uint32     `gorm:"primaryKey;autoIncrement:false;not null"`
	CardId          uint32     `gorm:"primaryKey;autoIncrement:false;not null"`
	Level           uint8      `gorm:"not null"`
	IsSpecial       bool       `gorm:"not null;default:false;index"`
	LastEventId     *uuid.UUID `gorm:""`
	FirstAcquiredAt time.Time  `gorm:"autoCreateTime"`
	UpdatedAt       time.Time  `gorm:"autoUpdateTime"`
}

func (entity) TableName() string { return "monster_book_cards" }
```

- [ ] **Step 4: Run test (PASS)**

```bash
go test ./card/...
```

- [ ] **Step 5: Commit**

```bash
git add services/atlas-monster-book/atlas.com/monster-book/card/{entity.go,entity_test.go}
git commit -m "feat(monster-book): card entity + migration"
```

---

### Task 6: `card` model + builder

**Files:**
- Create: `services/atlas-monster-book/atlas.com/monster-book/card/model.go`
- Create: `services/atlas-monster-book/atlas.com/monster-book/card/builder.go`
- Test: `services/atlas-monster-book/atlas.com/monster-book/card/builder_test.go`

- [ ] **Step 1: Write builder test (validates card-id range + is_special derivation)**

```go
package card

import (
	"testing"

	"github.com/google/uuid"
)

func TestBuilderRejectsZeroCharacter(t *testing.T) {
	_, err := NewModelBuilder().SetCardId(2380000).SetLevel(1).Build()
	if err == nil {
		t.Fatal("expected error: characterId required")
	}
}

func TestBuilderRejectsOutOfRangeCardId(t *testing.T) {
	for _, badId := range []uint32{0, 2370000, 2390000, 2389999 + 1} {
		if _, err := NewModelBuilder().SetCharacterId(1).SetCardId(badId).SetLevel(1).Build(); err == nil {
			t.Fatalf("expected reject for cardId %d", badId)
		}
	}
}

func TestBuilderRejectsLevelOutOfRange(t *testing.T) {
	for _, l := range []uint8{0, 6, 255} {
		if _, err := NewModelBuilder().SetCharacterId(1).SetCardId(2380000).SetLevel(l).Build(); err == nil {
			t.Fatalf("expected reject for level %d", l)
		}
	}
}

func TestIsSpecialDerivation(t *testing.T) {
	cases := map[uint32]bool{
		2380000: false,
		2387999: false,
		2388000: true,
		2389999: true,
	}
	for cid, want := range cases {
		m, err := NewModelBuilder().
			SetTenantId(uuid.New()).SetCharacterId(1).SetCardId(cid).SetLevel(1).Build()
		if err != nil {
			t.Fatalf("build cid %d: %v", cid, err)
		}
		if m.IsSpecial() != want {
			t.Fatalf("cardId %d: want isSpecial=%v got %v", cid, want, m.IsSpecial())
		}
	}
}
```

- [ ] **Step 2: Run test (FAIL)**

- [ ] **Step 3: Write `model.go`**

```go
package card

import (
	"time"

	"github.com/google/uuid"
)

const (
	MinCardId       uint32 = 2380000
	MaxCardId       uint32 = 2389999
	SpecialCardBase uint32 = 2388000 // cardId/1000 >= 2388
	MaxLevel        uint8  = 5
)

func IsCardId(itemId uint32) bool {
	return itemId >= MinCardId && itemId <= MaxCardId
}

func IsSpecialCard(cardId uint32) bool {
	return cardId/1000 >= 2388
}

type Model struct {
	tenantId        uuid.UUID
	characterId     uint32
	cardId          uint32
	level           uint8
	isSpecial       bool
	lastEventId     *uuid.UUID
	firstAcquiredAt time.Time
	updatedAt       time.Time
}

func (m Model) TenantId() uuid.UUID          { return m.tenantId }
func (m Model) CharacterId() uint32          { return m.characterId }
func (m Model) CardId() uint32               { return m.cardId }
func (m Model) Level() uint8                 { return m.level }
func (m Model) IsSpecial() bool              { return m.isSpecial }
func (m Model) LastEventId() *uuid.UUID      { return m.lastEventId }
func (m Model) FirstAcquiredAt() time.Time   { return m.firstAcquiredAt }
func (m Model) UpdatedAt() time.Time         { return m.updatedAt }
```

- [ ] **Step 4: Write `builder.go`**

```go
package card

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type ModelBuilder struct {
	tenantId        uuid.UUID
	characterId     uint32
	cardId          uint32
	level           uint8
	lastEventId     *uuid.UUID
	firstAcquiredAt time.Time
	updatedAt       time.Time
}

func NewModelBuilder() *ModelBuilder { return &ModelBuilder{} }

func (b *ModelBuilder) SetTenantId(v uuid.UUID) *ModelBuilder        { b.tenantId = v; return b }
func (b *ModelBuilder) SetCharacterId(v uint32) *ModelBuilder        { b.characterId = v; return b }
func (b *ModelBuilder) SetCardId(v uint32) *ModelBuilder             { b.cardId = v; return b }
func (b *ModelBuilder) SetLevel(v uint8) *ModelBuilder               { b.level = v; return b }
func (b *ModelBuilder) SetLastEventId(v *uuid.UUID) *ModelBuilder    { b.lastEventId = v; return b }
func (b *ModelBuilder) SetFirstAcquiredAt(v time.Time) *ModelBuilder { b.firstAcquiredAt = v; return b }
func (b *ModelBuilder) SetUpdatedAt(v time.Time) *ModelBuilder       { b.updatedAt = v; return b }

func (b *ModelBuilder) Build() (Model, error) {
	if b.characterId == 0 {
		return Model{}, errors.New("characterId is required")
	}
	if !IsCardId(b.cardId) {
		return Model{}, fmt.Errorf("cardId %d out of range [%d, %d]", b.cardId, MinCardId, MaxCardId)
	}
	if b.level < 1 || b.level > MaxLevel {
		return Model{}, fmt.Errorf("level %d out of range [1, %d]", b.level, MaxLevel)
	}
	return Model{
		tenantId:        b.tenantId,
		characterId:     b.characterId,
		cardId:          b.cardId,
		level:           b.level,
		isSpecial:       IsSpecialCard(b.cardId),
		lastEventId:     b.lastEventId,
		firstAcquiredAt: b.firstAcquiredAt,
		updatedAt:       b.updatedAt,
	}, nil
}

func (b *ModelBuilder) MustBuild() Model {
	m, err := b.Build()
	if err != nil {
		panic("MustBuild: " + err.Error())
	}
	return m
}

func Make(e entity) (Model, error) {
	return NewModelBuilder().
		SetTenantId(e.TenantId).
		SetCharacterId(e.CharacterId).
		SetCardId(e.CardId).
		SetLevel(e.Level).
		SetLastEventId(e.LastEventId).
		SetFirstAcquiredAt(e.FirstAcquiredAt).
		SetUpdatedAt(e.UpdatedAt).
		Build()
}
```

- [ ] **Step 5: Run tests (PASS)**

- [ ] **Step 6: Commit**

```bash
git add services/atlas-monster-book/atlas.com/monster-book/card/{model.go,builder.go,builder_test.go}
git commit -m "feat(monster-book): card model + builder"
```

---

### Task 7: `card` administrator (idempotent upsert)

**Files:**
- Create: `services/atlas-monster-book/atlas.com/monster-book/card/administrator.go`
- Create: `services/atlas-monster-book/atlas.com/monster-book/card/provider.go`
- Test: `services/atlas-monster-book/atlas.com/monster-book/card/administrator_test.go`

This is the **idempotency-critical** path (design §7).

- [ ] **Step 1: Write administrator test**

```go
package card

import (
	"testing"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := Migration(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestUpsertFirstInsertion(t *testing.T) {
	db, tid, eid := newDB(t), uuid.New(), uuid.New()
	res, err := upsertCard(db, tid, 1, 2380000, eid)
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if !res.Inserted || res.NewLevel != 1 || res.Duplicate {
		t.Fatalf("got %+v", res)
	}
}

func TestUpsertLevelsUpToFive(t *testing.T) {
	db, tid := newDB(t), uuid.New()
	for i := 1; i <= 7; i++ {
		eid := uuid.New()
		res, err := upsertCard(db, tid, 1, 2380000, eid)
		if err != nil {
			t.Fatalf("step %d: %v", i, err)
		}
		expectedLevel := uint8(i)
		if expectedLevel > MaxLevel {
			expectedLevel = MaxLevel
		}
		if res.NewLevel != expectedLevel {
			t.Fatalf("step %d: want level %d, got %d", i, expectedLevel, res.NewLevel)
		}
		if res.Inserted != (i == 1) {
			t.Fatalf("step %d: inserted=%v", i, res.Inserted)
		}
		if res.Full != (i >= int(MaxLevel)) {
			t.Fatalf("step %d: full=%v", i, res.Full)
		}
		if res.Duplicate {
			t.Fatalf("step %d: unexpected duplicate", i)
		}
	}
}

func TestUpsertDuplicateEventIdNoOp(t *testing.T) {
	db, tid, eid := newDB(t), uuid.New(), uuid.New()
	if _, err := upsertCard(db, tid, 1, 2380000, eid); err != nil {
		t.Fatalf("first: %v", err)
	}
	res, err := upsertCard(db, tid, 1, 2380000, eid)
	if err != nil {
		t.Fatalf("dup: %v", err)
	}
	if !res.Duplicate || res.NewLevel != 1 {
		t.Fatalf("got %+v", res)
	}
}
```

- [ ] **Step 2: Run test (FAIL)**

- [ ] **Step 3: Write `administrator.go`**

```go
package card

import (
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UpsertResult struct {
	Inserted  bool
	NewLevel  uint8
	Full      bool
	Duplicate bool
}

// upsertCard inserts at level 1 or increments level (cap MaxLevel) for an existing row,
// guarded by lastEventId for idempotency. Runs in the caller's transaction.
func upsertCard(db *gorm.DB, tenantId uuid.UUID, characterId uint32, cardId uint32, eventId uuid.UUID) (UpsertResult, error) {
	if !IsCardId(cardId) {
		return UpsertResult{}, errors.New("cardId out of range")
	}

	// Try to load existing row.
	var existing entity
	err := db.Where("tenant_id = ? AND character_id = ? AND card_id = ?", tenantId, characterId, cardId).
		First(&existing).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		e := entity{
			TenantId:    tenantId,
			CharacterId: characterId,
			CardId:      cardId,
			Level:       1,
			IsSpecial:   IsSpecialCard(cardId),
			LastEventId: &eventId,
		}
		if err := db.Create(&e).Error; err != nil {
			return UpsertResult{}, err
		}
		return UpsertResult{Inserted: true, NewLevel: 1, Full: MaxLevel == 1, Duplicate: false}, nil
	}
	if err != nil {
		return UpsertResult{}, err
	}

	// Idempotency guard: same eventId -> no-op.
	if existing.LastEventId != nil && *existing.LastEventId == eventId {
		return UpsertResult{Inserted: false, NewLevel: existing.Level, Full: existing.Level >= MaxLevel, Duplicate: true}, nil
	}

	if existing.Level >= MaxLevel {
		// Persist the eventId so future replays of *this* eventId no-op,
		// but level stays capped.
		if err := db.Model(&entity{}).
			Where("tenant_id = ? AND character_id = ? AND card_id = ?", tenantId, characterId, cardId).
			Update("last_event_id", eventId).Error; err != nil {
			return UpsertResult{}, err
		}
		return UpsertResult{Inserted: false, NewLevel: MaxLevel, Full: true, Duplicate: false}, nil
	}

	newLevel := existing.Level + 1
	if err := db.Model(&entity{}).
		Where("tenant_id = ? AND character_id = ? AND card_id = ?", tenantId, characterId, cardId).
		Updates(map[string]interface{}{"level": newLevel, "last_event_id": eventId}).Error; err != nil {
		return UpsertResult{}, err
	}
	return UpsertResult{Inserted: false, NewLevel: newLevel, Full: newLevel >= MaxLevel, Duplicate: false}, nil
}

func deleteByCharacter(db *gorm.DB, tenantId uuid.UUID, characterId uint32) error {
	return db.Where("tenant_id = ? AND character_id = ?", tenantId, characterId).Delete(&entity{}).Error
}
```

- [ ] **Step 4: Write `provider.go`**

```go
package card

import (
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func byCharacterIdEntityProvider(tenantId uuid.UUID, characterId uint32) database.EntityProvider[[]entity] {
	return func(db *gorm.DB) model.Provider[[]entity] {
		return database.SliceQuery[entity](db.Where("tenant_id = ? AND character_id = ?", tenantId, characterId), &entity{})
	}
}

func byCharacterIdAndCardIdEntityProvider(tenantId uuid.UUID, characterId uint32, cardId uint32) database.EntityProvider[entity] {
	return func(db *gorm.DB) model.Provider[entity] {
		return database.Query[entity](db.Where("tenant_id = ? AND character_id = ? AND card_id = ?", tenantId, characterId, cardId), &entity{})
	}
}

func bySpecialEntityProvider(tenantId uuid.UUID, characterId uint32, isSpecial bool) database.EntityProvider[[]entity] {
	return func(db *gorm.DB) model.Provider[[]entity] {
		return database.SliceQuery[entity](db.Where("tenant_id = ? AND character_id = ? AND is_special = ?", tenantId, characterId, isSpecial), &entity{})
	}
}
```

- [ ] **Step 5: Run tests (PASS)**

- [ ] **Step 6: Commit**

```bash
git add services/atlas-monster-book/atlas.com/monster-book/card/{administrator.go,provider.go,administrator_test.go}
git commit -m "feat(monster-book): card administrator + provider (idempotent upsert)"
```

---

### Task 8: Kafka producer + message buffer scaffold

**Files:**
- Create: `services/atlas-monster-book/atlas.com/monster-book/kafka/producer/producer.go`
- Create: `services/atlas-monster-book/atlas.com/monster-book/kafka/message/message.go`
- Create: `services/atlas-monster-book/atlas.com/monster-book/kafka/consumer/consumer.go`

These three files are pure copies of the atlas-keys equivalents with `atlas-keys` swapped to `atlas-monster-book`.

- [ ] **Step 1: Copy `producer/producer.go` from atlas-keys**

Source: `services/atlas-keys/atlas.com/keys/kafka/producer/producer.go`. No content changes — only the package import path on consumers' side will differ later.

- [ ] **Step 2: Copy `message/message.go` from atlas-keys**

Source: `services/atlas-keys/atlas.com/keys/kafka/message/message.go`. Update the import line:

```go
import (
	"atlas-monster-book/kafka/producer"
	// rest unchanged
)
```

- [ ] **Step 3: Copy `consumer/consumer.go` from atlas-keys**

Source: `services/atlas-keys/atlas.com/keys/kafka/consumer/consumer.go`. No edits needed.

- [ ] **Step 4: Verify the package compiles**

```bash
go build ./kafka/...
```

Expected: build succeeds.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-monster-book/atlas.com/monster-book/kafka/{producer,message,consumer}
git commit -m "feat(monster-book): kafka scaffold (producer, buffer, consumer config)"
```

---

### Task 9: Kafka message types — character + monsterbook

**Files:**
- Create: `services/atlas-monster-book/atlas.com/monster-book/kafka/message/character/kafka.go`
- Create: `services/atlas-monster-book/atlas.com/monster-book/kafka/message/monsterbook/kafka.go`

The character file mirrors atlas-keys verbatim. The monsterbook file is new.

- [ ] **Step 1: Copy `kafka/message/character/kafka.go` from atlas-keys**

Source: `services/atlas-keys/atlas.com/keys/kafka/message/character/kafka.go`. No edits.

- [ ] **Step 2: Write `kafka/message/monsterbook/kafka.go`**

```go
package monsterbook

import "github.com/google/uuid"

const (
	// Inbound (commands)
	EnvCommandTopic = "COMMAND_TOPIC_MONSTER_BOOK"

	CommandTypeCardPickedUp = "CARD_PICKED_UP"
	CommandTypeSetCover     = "SET_COVER"

	// Outbound (statuses)
	EnvEventTopicStatus = "EVENT_TOPIC_MONSTER_BOOK_STATUS"

	StatusEventTypeCardAdded     = "CARD_ADDED"
	StatusEventTypeCoverChanged  = "COVER_CHANGED"
	StatusEventTypeStatsChanged  = "STATS_CHANGED"
)

// Command<T> is the inbound command envelope.
type Command[B any] struct {
	TenantId    uuid.UUID `json:"tenantId"`
	CharacterId uint32    `json:"characterId"`
	EventId     uuid.UUID `json:"eventId"`
	Type        string    `json:"type"`
	Body        B         `json:"body"`
}

type CardPickedUpBody struct {
	CardId uint32 `json:"cardId"`
	Source string `json:"source"`
}

type SetCoverBody struct {
	CoverCardId uint32 `json:"coverCardId"`
}

// StatusEvent<T> is the outbound status envelope.
type StatusEvent[B any] struct {
	TenantId    uuid.UUID `json:"tenantId"`
	CharacterId uint32    `json:"characterId"`
	EventId     uuid.UUID `json:"eventId"`
	Type        string    `json:"type"`
	Body        B         `json:"body"`
}

type CardAddedBody struct {
	CardId   uint32 `json:"cardId"`
	NewLevel uint8  `json:"newLevel"`
	Full     bool   `json:"full"`
}

type CoverChangedBody struct {
	CoverCardId uint32 `json:"coverCardId"`
}

type StatsChangedBody struct {
	BookLevel        uint16 `json:"bookLevel"`
	NormalCount      uint16 `json:"normalCount"`
	SpecialCount     uint16 `json:"specialCount"`
	TotalUniqueCards uint16 `json:"totalUniqueCards"`
	ExpBonusPercent  uint16 `json:"expBonusPercent"`
}
```

- [ ] **Step 3: Build**

```bash
go build ./...
```

- [ ] **Step 4: Commit**

```bash
git add services/atlas-monster-book/atlas.com/monster-book/kafka/message/{character,monsterbook}
git commit -m "feat(monster-book): kafka message types"
```

---

### Task 10: `card.Processor` — `AddAndEmit`

**Files:**
- Create: `services/atlas-monster-book/atlas.com/monster-book/card/processor.go`
- Test: `services/atlas-monster-book/atlas.com/monster-book/card/processor_test.go`

- [ ] **Step 1: Write processor test**

```go
package card

import (
	"context"
	"testing"

	"atlas-monster-book/kafka/message"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

func TestProcessorAddInsertsAtLevel1(t *testing.T) {
	db := newDB(t)
	tid := uuid.New()
	ctx := tenantCtx(tid)
	p := NewProcessor(logrus.New(), ctx, db)
	mb := message.NewBuffer()
	res, err := p.Add(mb)(uuid.New(), 1, 2380000)
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if !res.Inserted || res.NewLevel != 1 || res.Duplicate {
		t.Fatalf("got %+v", res)
	}
}

func TestProcessorGetByCharacter(t *testing.T) {
	db := newDB(t)
	tid := uuid.New()
	ctx := tenantCtx(tid)
	p := NewProcessor(logrus.New(), ctx, db)
	mb := message.NewBuffer()
	if _, err := p.Add(mb)(uuid.New(), 1, 2380000); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Add(mb)(uuid.New(), 1, 2380001); err != nil {
		t.Fatal(err)
	}
	got, err := p.GetByCharacterId(1)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 cards, got %d", len(got))
	}
}

// tenantCtx attaches a tenant.Model to ctx for tests.
func tenantCtx(t uuid.UUID) context.Context {
	// Implementation copied from atlas-keys's existing test helper convention.
	// Use tenant.WithTenant or equivalent — see services/atlas-keys/atlas.com/keys/key/processor_test.go for reference.
	// (Stub below — replace with the project's actual helper when wiring.)
	panic("use the project's tenant test helper; see services/atlas-keys for reference")
}
```

> **Engineer note:** atlas-keys uses `tenant.WithTenant(context.Background(), tenant.Model{...})` (or the equivalent helper from `libs/atlas-tenant`) — replace the `panic` with the actual call exercised in atlas-keys's processor tests if any exist; otherwise construct a tenant.Model via `tenant.MustFromContext` after attaching with `context.WithValue`. Keep the test self-contained.

- [ ] **Step 2: Run test (FAIL)**

- [ ] **Step 3: Write `processor.go`**

```go
package card

import (
	"context"
	"errors"

	"atlas-monster-book/kafka/message"
	"atlas-monster-book/kafka/message/monsterbook"
	"atlas-monster-book/kafka/producer"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

var entityModelMapper = model.Map(Make)
var entitySliceMapper = model.SliceMap(Make)

type Processor interface {
	GetByCharacterId(characterId uint32) ([]Model, error)
	GetByCharacterIdAndCardId(characterId uint32, cardId uint32) (Model, error)
	GetByCharacterIdAndIsSpecial(characterId uint32, isSpecial bool) ([]Model, error)
	Add(mb *message.Buffer) func(eventId uuid.UUID, characterId uint32, cardId uint32) (UpsertResult, error)
	AddAndEmit(eventId uuid.UUID, characterId uint32, cardId uint32) (UpsertResult, error)
	WithTransaction(tx *gorm.DB) Processor
	DeleteByCharacterId(characterId uint32) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	db  *gorm.DB
	t   tenant.Model
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	return &ProcessorImpl{l: l, ctx: ctx, db: db, t: tenant.MustFromContext(ctx)}
}

func (p *ProcessorImpl) WithTransaction(tx *gorm.DB) Processor {
	return &ProcessorImpl{l: p.l, ctx: p.ctx, db: tx, t: p.t}
}

func (p *ProcessorImpl) GetByCharacterId(characterId uint32) ([]Model, error) {
	return entitySliceMapper(byCharacterIdEntityProvider(p.t.Id(), characterId)(p.db.WithContext(p.ctx)))()
}

func (p *ProcessorImpl) GetByCharacterIdAndCardId(characterId uint32, cardId uint32) (Model, error) {
	return entityModelMapper(byCharacterIdAndCardIdEntityProvider(p.t.Id(), characterId, cardId)(p.db.WithContext(p.ctx)))()
}

func (p *ProcessorImpl) GetByCharacterIdAndIsSpecial(characterId uint32, isSpecial bool) ([]Model, error) {
	return entitySliceMapper(bySpecialEntityProvider(p.t.Id(), characterId, isSpecial)(p.db.WithContext(p.ctx)))()
}

func (p *ProcessorImpl) DeleteByCharacterId(characterId uint32) error {
	return deleteByCharacter(p.db.WithContext(p.ctx), p.t.Id(), characterId)
}

func (p *ProcessorImpl) Add(mb *message.Buffer) func(eventId uuid.UUID, characterId uint32, cardId uint32) (UpsertResult, error) {
	return func(eventId uuid.UUID, characterId uint32, cardId uint32) (UpsertResult, error) {
		if !IsCardId(cardId) {
			return UpsertResult{}, errors.New("cardId out of range")
		}
		res, err := upsertCard(p.db, p.t.Id(), characterId, cardId, eventId)
		if err != nil {
			return UpsertResult{}, err
		}
		if res.Duplicate {
			return res, nil
		}
		// Buffer the CARD_ADDED status event. STATS_CHANGED + EXP_DISTRIBUTION are
		// emitted by the collection processor when bookLevel actually changes.
		ev := monsterbook.StatusEvent[monsterbook.CardAddedBody]{
			TenantId:    p.t.Id(),
			CharacterId: characterId,
			EventId:     eventId,
			Type:        monsterbook.StatusEventTypeCardAdded,
			Body: monsterbook.CardAddedBody{
				CardId:   cardId,
				NewLevel: res.NewLevel,
				Full:     res.Full,
			},
		}
		if err := mb.Put(monsterbook.EnvEventTopicStatus, providerOf(ev)); err != nil {
			return UpsertResult{}, err
		}
		return res, nil
	}
}

func (p *ProcessorImpl) AddAndEmit(eventId uuid.UUID, characterId uint32, cardId uint32) (UpsertResult, error) {
	var out UpsertResult
	err := message.Emit(producer.ProviderImpl(p.l)(p.ctx))(func(buf *message.Buffer) error {
		var err error
		out, err = p.Add(buf)(eventId, characterId, cardId)
		return err
	})
	return out, err
}

// providerOf wraps a single status event as a model.Provider[[]kafka.Message].
// Mirrors the helper used in atlas-keys/kafka/message internals.
func providerOf[B any](ev monsterbook.StatusEvent[B]) model.Provider[[]kafka.Message] {
	return func() ([]kafka.Message, error) {
		body, err := jsonMarshal(ev)
		if err != nil {
			return nil, err
		}
		return []kafka.Message{{Key: keyOf(ev.CharacterId), Value: body}}, nil
	}
}
```

- [ ] **Step 4: Write the small JSON/key helpers**

Append to `card/processor.go`:

```go
import (
	"encoding/binary"
	"encoding/json"
)

func jsonMarshal(v interface{}) ([]byte, error) { return json.Marshal(v) }

func keyOf(characterId uint32) []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, characterId)
	return b
}
```

> **Note:** atlas-keys does not use a per-message provider helper because it uses the producer.ManagerWriterProvider plumbing. Look at `services/atlas-keys/atlas.com/keys/character/producer.go` (if present in your branch) or any existing `kafka/message/<topic>/producer.go` in another service for the canonical wrapper. If a shared helper exists, replace `providerOf` with it. Otherwise the inline form above is acceptable — it produces the same wire format.

- [ ] **Step 5: Run tests (PASS)**

```bash
go test ./card/...
```

- [ ] **Step 6: Commit**

```bash
git add services/atlas-monster-book/atlas.com/monster-book/card/processor.go services/atlas-monster-book/atlas.com/monster-book/card/processor_test.go
git commit -m "feat(monster-book): card.Processor (AddAndEmit + getters)"
```

---

### Task 11: `collection.Processor` — book-level recompute, EXP bonus, cover

**Files:**
- Create: `services/atlas-monster-book/atlas.com/monster-book/collection/processor.go`
- Test: `services/atlas-monster-book/atlas.com/monster-book/collection/processor_test.go`

This processor owns: book-level formula, EXP bonus formula, `STATS_CHANGED` and `EXPERIENCE_DISTRIBUTION` emission, cover validation+upsert.

- [ ] **Step 1: Write tests for book-level formula and EXP bonus**

```go
package collection

import "testing"

func TestComputeBookLevelMatchesCosmicFormula(t *testing.T) {
	cases := map[uint16]uint16{
		0: 1, // 0 unique cards → level 1 (per Cosmic init)
		1: 1,
		2: 2,
		// At total=2, level=2 because expToNext after level 1 is 1+1*10 = 11; loop exits when total >= expToNext
		// Actually Cosmic loop: level=0; expToNext=1; do { level++; expToNext += level*10 } while (total >= expToNext)
		// step 1: level=1, expToNext=11; 0>=11? no -> exit -> level 1
		// step 1: level=1, expToNext=11; 12>=11? yes -> step 2: level=2, expToNext=11+20=31; 12>=31? no -> level 2
		12: 2,
		31: 3,
	}
	for total, want := range cases {
		if got := computeBookLevel(total); got != want {
			t.Errorf("total %d: want level %d got %d", total, want, got)
		}
	}
}

func TestExpBonusEqualsBookLevel(t *testing.T) {
	if got := computeExpBonusPercent(7); got != 7 {
		t.Errorf("want 7, got %d", got)
	}
}
```

- [ ] **Step 2: Run test (FAIL)**

- [ ] **Step 3: Write `processor.go`**

```go
package collection

import (
	"context"
	"errors"

	"atlas-monster-book/card"
	"atlas-monster-book/kafka/message"
	"atlas-monster-book/kafka/message/monsterbook"
	"atlas-monster-book/kafka/producer"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// computeBookLevel implements the Cosmic Monster Book level formula (PRD §4.4).
func computeBookLevel(totalUniqueCards uint16) uint16 {
	var level uint16 = 0
	var expToNext uint16 = 1
	for {
		level++
		expToNext += level * 10
		if totalUniqueCards < expToNext {
			return level
		}
	}
}

// computeExpBonusPercent: v1 formula = bookLevel (design §6.4).
func computeExpBonusPercent(bookLevel uint16) uint16 { return bookLevel }

type Processor interface {
	GetByCharacterId(characterId uint32) (Model, error)
	SetCoverAndEmit(eventId uuid.UUID, characterId uint32, cardId uint32) error
	RecomputeAndEmit(mb *message.Buffer) func(characterId uint32) error
	WithTransaction(tx *gorm.DB) Processor
	DeleteByCharacterId(characterId uint32) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	db  *gorm.DB
	t   tenant.Model
	cp  card.Processor
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	return &ProcessorImpl{
		l: l, ctx: ctx, db: db,
		t:  tenant.MustFromContext(ctx),
		cp: card.NewProcessor(l, ctx, db),
	}
}

func (p *ProcessorImpl) WithTransaction(tx *gorm.DB) Processor {
	return &ProcessorImpl{l: p.l, ctx: p.ctx, db: tx, t: p.t, cp: p.cp.WithTransaction(tx)}
}

func (p *ProcessorImpl) GetByCharacterId(characterId uint32) (Model, error) {
	mp := model.Map(Make)(byCharacterIdEntityProvider(p.t.Id(), characterId)(p.db.WithContext(p.ctx)))
	m, err := mp()
	if err != nil {
		// On not-found, return defaults per PRD §5.1.
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return NewModelBuilder().SetTenantId(p.t.Id()).SetCharacterId(characterId).SetBookLevel(1).Build()
		}
		return Model{}, err
	}
	return m, nil
}

// RecomputeAndEmit recomputes denormalised stats from the per-card rows and persists.
// Emits STATS_CHANGED + EXPERIENCE_DISTRIBUTION when values changed.
func (p *ProcessorImpl) RecomputeAndEmit(mb *message.Buffer) func(characterId uint32) error {
	return func(characterId uint32) error {
		normals, err := p.cp.GetByCharacterIdAndIsSpecial(characterId, false)
		if err != nil {
			return err
		}
		specials, err := p.cp.GetByCharacterIdAndIsSpecial(characterId, true)
		if err != nil {
			return err
		}
		normalCount := uint16(len(normals))
		specialCount := uint16(len(specials))
		total := normalCount + specialCount
		bookLevel := computeBookLevel(total)
		expBonus := computeExpBonusPercent(bookLevel)

		// Read existing for change detection.
		prior, _ := p.GetByCharacterId(characterId)
		changed := prior.NormalCount() != normalCount ||
			prior.SpecialCount() != specialCount ||
			prior.BookLevel() != bookLevel ||
			prior.ExpBonusPercent() != expBonus

		if _, err := upsertStats(p.db.WithContext(p.ctx), p.t.Id(), characterId, statsUpdate{
			NormalCount:     normalCount,
			SpecialCount:    specialCount,
			BookLevel:       bookLevel,
			ExpBonusPercent: expBonus,
		}); err != nil {
			return err
		}

		if !changed {
			return nil
		}

		// STATS_CHANGED status event
		stats := monsterbook.StatusEvent[monsterbook.StatsChangedBody]{
			TenantId:    p.t.Id(),
			CharacterId: characterId,
			EventId:     uuid.New(),
			Type:        monsterbook.StatusEventTypeStatsChanged,
			Body: monsterbook.StatsChangedBody{
				BookLevel:        bookLevel,
				NormalCount:      normalCount,
				SpecialCount:     specialCount,
				TotalUniqueCards: total,
				ExpBonusPercent:  expBonus,
			},
		}
		if err := mb.Put(monsterbook.EnvEventTopicStatus, providerOf(stats)); err != nil {
			return err
		}

		// EXPERIENCE_DISTRIBUTION on the existing topic atlas-channel already consumes.
		if err := mb.Put(envExperienceDistributionTopic, expDistributionProvider(characterId, expBonus)); err != nil {
			return err
		}
		return nil
	}
}

func (p *ProcessorImpl) SetCoverAndEmit(eventId uuid.UUID, characterId uint32, cardId uint32) error {
	// Validate: 0 (clear) is allowed; otherwise must be in card range AND owned at level >= 1.
	if cardId != 0 {
		if !card.IsCardId(cardId) {
			return errors.New("cardId out of range")
		}
		owned, err := p.cp.GetByCharacterIdAndCardId(characterId, cardId)
		if err != nil || owned.Level() < 1 {
			return errors.New("cover must be an owned card")
		}
	}
	return message.Emit(producer.ProviderImpl(p.l)(p.ctx))(func(mb *message.Buffer) error {
		changed, err := setCover(p.db.WithContext(p.ctx), p.t.Id(), characterId, cardId, eventId)
		if err != nil {
			return err
		}
		if !changed {
			return nil
		}
		ev := monsterbook.StatusEvent[monsterbook.CoverChangedBody]{
			TenantId:    p.t.Id(),
			CharacterId: characterId,
			EventId:     eventId,
			Type:        monsterbook.StatusEventTypeCoverChanged,
			Body:        monsterbook.CoverChangedBody{CoverCardId: cardId},
		}
		return mb.Put(monsterbook.EnvEventTopicStatus, providerOf(ev))
	})
}

func (p *ProcessorImpl) DeleteByCharacterId(characterId uint32) error {
	return deleteByCharacter(p.db.WithContext(p.ctx), p.t.Id(), characterId)
}

// ---- helpers (mirror card.processor.go) ----

const envExperienceDistributionTopic = "EVENT_TOPIC_CHARACTER_EXPERIENCE_DISTRIBUTION"

// expDistributionProvider builds the kafka message that atlas-channel's
// kafka/consumer/character/consumer.go already handles
// (ExperienceDistributionTypeMonsterBook → experience_status.MonsterBookBonus).
// Shape mirrors the canonical character.ExperienceDistributions struct.
func expDistributionProvider(characterId uint32, percent uint16) model.Provider[[]kafka.Message] {
	type distribution struct {
		ExperienceType string `json:"experienceType"`
		Amount         int32  `json:"amount"`
	}
	type body struct {
		Distributions []distribution `json:"distributions"`
	}
	type envelope struct {
		CharacterId uint32 `json:"characterId"`
		Body        body   `json:"body"`
	}
	env := envelope{
		CharacterId: characterId,
		Body: body{
			Distributions: []distribution{{
				ExperienceType: "MONSTER_BOOK",
				Amount:         int32(percent),
			}},
		},
	}
	return providerOf(env)
}

func providerOf[T any](ev T) model.Provider[[]kafka.Message] {
	return func() ([]kafka.Message, error) {
		b, err := jsonMarshal(ev)
		if err != nil {
			return nil, err
		}
		return []kafka.Message{{Value: b}}, nil
	}
}
```

> **Engineer note (envelope shape):** the `EXPERIENCE_DISTRIBUTION` envelope must match what `atlas-channel/kafka/consumer/character/consumer.go` (around lines 252–270) already deserializes. Open that file and the matching message struct in `atlas-channel/kafka/message/character/*.go` to copy the exact JSON shape (look for `ExperienceDistributions` struct and the per-character "AwardExperience" event). The illustrative shape above will likely need a small field-name fix — confirm before committing. The topic env var name is the one defined alongside that struct.

- [ ] **Step 4: Add `jsonMarshal` import to processor**

(Already in card/processor.go — but if `collection/processor.go` doesn't import it, add `"encoding/json"` and define a local `func jsonMarshal(v interface{}) ([]byte, error) { return json.Marshal(v) }` at file bottom, OR move the helper to a small `collection/marshal.go` file.)

- [ ] **Step 5: Write processor tests for SetCover + Recompute**

```go
package collection

import (
	"context"
	"testing"

	"atlas-monster-book/card"
	"atlas-monster-book/kafka/message"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

func TestRecomputeAfterFirstAcquisition(t *testing.T) {
	db := newDB(t)
	if err := card.Migration(db); err != nil {
		t.Fatal(err)
	}
	tid := uuid.New()
	ctx := tenantCtx(tid)
	cp := card.NewProcessor(logrus.New(), ctx, db)
	mb := message.NewBuffer()
	if _, err := cp.Add(mb)(uuid.New(), 1, 2380000); err != nil {
		t.Fatal(err)
	}
	p := NewProcessor(logrus.New(), ctx, db)
	if err := p.RecomputeAndEmit(mb)(1); err != nil {
		t.Fatal(err)
	}
	got, err := p.GetByCharacterId(1)
	if err != nil {
		t.Fatal(err)
	}
	if got.NormalCount() != 1 || got.BookLevel() != 1 || got.ExpBonusPercent() != 1 {
		t.Fatalf("got %+v", got)
	}
}

func TestSetCoverRequiresOwnedCard(t *testing.T) {
	db := newDB(t)
	if err := card.Migration(db); err != nil {
		t.Fatal(err)
	}
	tid := uuid.New()
	ctx := tenantCtx(tid)
	mb := message.NewBuffer()
	cp := card.NewProcessor(logrus.New(), ctx, db)
	if _, err := cp.Add(mb)(uuid.New(), 1, 2380000); err != nil {
		t.Fatal(err)
	}
	p := NewProcessor(logrus.New(), ctx, db)
	// First seed a collection row so setCover can update.
	if err := p.RecomputeAndEmit(mb)(1); err != nil {
		t.Fatal(err)
	}
	// Owned card → ok
	if err := p.SetCoverAndEmit(uuid.New(), 1, 2380000); err != nil {
		t.Fatalf("owned: %v", err)
	}
	// Unowned → error
	if err := p.SetCoverAndEmit(uuid.New(), 1, 2380001); err == nil {
		t.Fatal("expected error for unowned card")
	}
	// Clear → ok
	if err := p.SetCoverAndEmit(uuid.New(), 1, 0); err != nil {
		t.Fatalf("clear: %v", err)
	}
}

func tenantCtx(t uuid.UUID) context.Context {
	// Use the same helper as the card package's processor_test.go.
	panic("share the project tenant test helper")
}
```

- [ ] **Step 6: Run tests (PASS)**

```bash
go test ./collection/...
```

- [ ] **Step 7: Commit**

```bash
git add services/atlas-monster-book/atlas.com/monster-book/collection/processor.go services/atlas-monster-book/atlas.com/monster-book/collection/processor_test.go
git commit -m "feat(monster-book): collection.Processor (recompute + cover)"
```

---

### Task 12: Inbound consumer — `MONSTER_BOOK.CARD_PICKED_UP`

**Files:**
- Create: `services/atlas-monster-book/atlas.com/monster-book/kafka/consumer/monsterbook/consumer.go`
- Test: `services/atlas-monster-book/atlas.com/monster-book/kafka/consumer/monsterbook/consumer_test.go`

The handler runs in a single transaction: card upsert → collection recompute → buffer drain.

- [ ] **Step 1: Write handler test (using sqlite + in-memory dispatch)**

```go
package monsterbook

import (
	"context"
	"testing"

	"atlas-monster-book/card"
	"atlas-monster-book/collection"
	"atlas-monster-book/kafka/message/monsterbook"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestHandleCardPickedUpInsertsAndRecomputes(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err := card.Migration(db); err != nil {
		t.Fatal(err)
	}
	if err := collection.Migration(db); err != nil {
		t.Fatal(err)
	}
	tid := uuid.New()
	ctx := tenantCtx(tid)
	handleCardPickedUp(db)(logrus.New(), ctx, monsterbook.Command[monsterbook.CardPickedUpBody]{
		TenantId:    tid,
		CharacterId: 1,
		EventId:     uuid.New(),
		Type:        monsterbook.CommandTypeCardPickedUp,
		Body:        monsterbook.CardPickedUpBody{CardId: 2380000},
	})
	cp := card.NewProcessor(logrus.New(), ctx, db)
	cards, _ := cp.GetByCharacterId(1)
	if len(cards) != 1 {
		t.Fatalf("expected 1 card, got %d", len(cards))
	}
	colp := collection.NewProcessor(logrus.New(), ctx, db)
	col, _ := colp.GetByCharacterId(1)
	if col.NormalCount() != 1 || col.BookLevel() != 1 {
		t.Fatalf("collection wrong: %+v", col)
	}
}

func tenantCtx(t uuid.UUID) context.Context { panic("use project helper") }
```

- [ ] **Step 2: Run (FAIL — symbols undefined)**

- [ ] **Step 3: Write `consumer.go`**

```go
package monsterbook

import (
	"context"

	"atlas-monster-book/card"
	"atlas-monster-book/collection"
	consumer2 "atlas-monster-book/kafka/consumer"
	"atlas-monster-book/kafka/message"
	mbmsg "atlas-monster-book/kafka/message/monsterbook"
	"atlas-monster-book/kafka/producer"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	kmessage "github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func InitConsumers(l logrus.FieldLogger) func(rf func(consumer.Config, ...model.Decorator[consumer.Config])) func(string) {
	return func(rf func(consumer.Config, ...model.Decorator[consumer.Config])) func(string) {
		return func(groupId string) {
			rf(consumer2.NewConfig(l)("monster_book_command")(mbmsg.EnvCommandTopic)(groupId),
				consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(*gorm.DB) func(rf func(string, handler.Handler) (string, error)) error {
	return func(db *gorm.DB) func(rf func(string, handler.Handler) (string, error)) error {
		return func(rf func(string, handler.Handler) (string, error)) error {
			t, _ := topic.EnvProvider(l)(mbmsg.EnvCommandTopic)()
			if _, err := rf(t, kmessage.AdaptHandler(kmessage.PersistentConfig(handleCardPickedUp(db)))); err != nil {
				return err
			}
			if _, err := rf(t, kmessage.AdaptHandler(kmessage.PersistentConfig(handleSetCover(db)))); err != nil {
				return err
			}
			return nil
		}
	}
}

func handleCardPickedUp(db *gorm.DB) func(logrus.FieldLogger, context.Context, mbmsg.Command[mbmsg.CardPickedUpBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, cmd mbmsg.Command[mbmsg.CardPickedUpBody]) {
		if cmd.Type != mbmsg.CommandTypeCardPickedUp {
			return
		}
		err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			return message.Emit(producer.ProviderImpl(l)(ctx))(func(mb *message.Buffer) error {
				cp := card.NewProcessor(l, ctx, tx)
				colp := collection.NewProcessor(l, ctx, tx)
				res, err := cp.Add(mb)(cmd.EventId, cmd.CharacterId, cmd.Body.CardId)
				if err != nil {
					return err
				}
				if res.Duplicate {
					return nil
				}
				// Only recompute on first acquisition (level 1).
				if res.Inserted {
					return colp.RecomputeAndEmit(mb)(cmd.CharacterId)
				}
				return nil
			})
		})
		if err != nil {
			l.WithError(err).Errorf("Failed to handle CARD_PICKED_UP for character %d card %d.", cmd.CharacterId, cmd.Body.CardId)
		}
	}
}

func handleSetCover(db *gorm.DB) func(logrus.FieldLogger, context.Context, mbmsg.Command[mbmsg.SetCoverBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, cmd mbmsg.Command[mbmsg.SetCoverBody]) {
		if cmd.Type != mbmsg.CommandTypeSetCover {
			return
		}
		colp := collection.NewProcessor(l, ctx, db)
		if err := colp.SetCoverAndEmit(cmd.EventId, cmd.CharacterId, cmd.Body.CoverCardId); err != nil {
			l.WithError(err).Warnf("SetCover rejected for character %d cover %d.", cmd.CharacterId, cmd.Body.CoverCardId)
		}
	}
}
```

- [ ] **Step 4: Run tests (PASS)**

```bash
go test ./kafka/consumer/monsterbook/...
```

- [ ] **Step 5: Commit**

```bash
git add services/atlas-monster-book/atlas.com/monster-book/kafka/consumer/monsterbook
git commit -m "feat(monster-book): inbound CARD_PICKED_UP + SET_COVER consumers"
```

---

### Task 13: Character lifecycle consumer (cascade delete)

**Files:**
- Create: `services/atlas-monster-book/atlas.com/monster-book/kafka/consumer/character/consumer.go`
- Test: `services/atlas-monster-book/atlas.com/monster-book/kafka/consumer/character/consumer_test.go`

Mirrors atlas-keys's exact pattern (`services/atlas-keys/atlas.com/keys/kafka/consumer/character/consumer.go`).

- [ ] **Step 1: Write deletion test**

```go
package character

import (
	"context"
	"testing"

	"atlas-monster-book/card"
	"atlas-monster-book/collection"
	characterMsg "atlas-monster-book/kafka/message/character"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestHandleDeletedCascades(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	_ = card.Migration(db)
	_ = collection.Migration(db)
	tid := uuid.New()
	ctx := tenantCtx(tid)
	cp := card.NewProcessor(logrus.New(), ctx, db)
	colp := collection.NewProcessor(logrus.New(), ctx, db)
	// Seed a character with one card and a collection row.
	cp.AddAndEmit(uuid.New(), 99, 2380000)
	colp.RecomputeAndEmit(/* engineer: see test helper for buffer */)(99)
	handleStatusEventDeleted(db)(logrus.New(), ctx, characterMsg.StatusEvent[characterMsg.DeletedStatusEventBody]{
		CharacterId: 99,
		Type:        characterMsg.StatusEventTypeDeleted,
	})
	cards, _ := cp.GetByCharacterId(99)
	if len(cards) != 0 {
		t.Fatalf("expected cards deleted, got %d", len(cards))
	}
	col, _ := colp.GetByCharacterId(99)
	if col.NormalCount() != 0 {
		t.Fatalf("expected collection cleared, got %+v", col)
	}
}

func tenantCtx(t uuid.UUID) context.Context { panic("use project helper") }
```

- [ ] **Step 2: Run (FAIL)**

- [ ] **Step 3: Write `consumer.go`** (copy atlas-keys verbatim, swap package and processor wiring)

```go
package character

import (
	"context"

	"atlas-monster-book/card"
	"atlas-monster-book/collection"
	consumer2 "atlas-monster-book/kafka/consumer"
	characterMsg "atlas-monster-book/kafka/message/character"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func InitConsumers(l logrus.FieldLogger) func(func(consumer.Config, ...model.Decorator[consumer.Config])) func(string) {
	return func(rf func(consumer.Config, ...model.Decorator[consumer.Config])) func(string) {
		return func(groupId string) {
			rf(consumer2.NewConfig(l)("character_status")(characterMsg.EnvEventTopicStatus)(groupId),
				consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(*gorm.DB) func(rf func(string, handler.Handler) (string, error)) error {
	return func(db *gorm.DB) func(rf func(string, handler.Handler) (string, error)) error {
		return func(rf func(string, handler.Handler) (string, error)) error {
			t, _ := topic.EnvProvider(l)(characterMsg.EnvEventTopicStatus)()
			if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventDeleted(db)))); err != nil {
				return err
			}
			return nil
		}
	}
}

func handleStatusEventDeleted(db *gorm.DB) message.Handler[characterMsg.StatusEvent[characterMsg.DeletedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e characterMsg.StatusEvent[characterMsg.DeletedStatusEventBody]) {
		if e.Type != characterMsg.StatusEventTypeDeleted {
			return
		}
		if err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			cp := card.NewProcessor(l, ctx, tx)
			colp := collection.NewProcessor(l, ctx, tx)
			if err := cp.DeleteByCharacterId(e.CharacterId); err != nil {
				return err
			}
			return colp.DeleteByCharacterId(e.CharacterId)
		}); err != nil {
			l.WithError(err).Errorf("Cascading monster-book delete failed for character %d.", e.CharacterId)
		}
	}
}
```

- [ ] **Step 4: Run tests (PASS)**

- [ ] **Step 5: Commit**

```bash
git add services/atlas-monster-book/atlas.com/monster-book/kafka/consumer/character
git commit -m "feat(monster-book): cascade-delete on character lifecycle"
```

---

### Task 14: REST resource + handlers (GET, PATCH, paginated cards)

**Files:**
- Create: `services/atlas-monster-book/atlas.com/monster-book/rest/handler.go`
- Create: `services/atlas-monster-book/atlas.com/monster-book/collection/rest.go` (RestModel + Transform)
- Create: `services/atlas-monster-book/atlas.com/monster-book/card/rest.go` (RestModel + Transform)
- Create: `services/atlas-monster-book/atlas.com/monster-book/character/resource.go` (route registration)

Mirrors `services/atlas-keys/atlas.com/keys/{rest/handler.go,key/rest.go,character/resource.go}`.

- [ ] **Step 1: Write `rest/handler.go`** (copy atlas-keys + add ParseCardId)

```go
package rest

import (
	"net/http"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
)

type HandlerDependency = server.HandlerDependency
type HandlerContext = server.HandlerContext
type GetHandler = server.GetHandler
type InputHandler[M any] = server.InputHandler[M]

func ParseInput[M any](d *HandlerDependency, c *HandlerContext, next InputHandler[M]) http.HandlerFunc {
	return server.ParseInput[M](d, c, next)
}

var RegisterHandler = server.RegisterHandler

func RegisterInputHandler[M any](l logrus.FieldLogger) func(si jsonapi.ServerInformation) func(string, InputHandler[M]) http.HandlerFunc {
	return server.RegisterInputHandler[M](l)
}

func ParseCharacterId(l logrus.FieldLogger, next func(uint32) http.HandlerFunc) http.HandlerFunc {
	return server.ParseIntId[uint32](l, "characterId", next)
}

func ParseCardId(l logrus.FieldLogger, next func(uint32) http.HandlerFunc) http.HandlerFunc {
	return server.ParseIntId[uint32](l, "cardId", next)
}
```

- [ ] **Step 2: Write `collection/rest.go`**

```go
package collection

import "strconv"

type RestModel struct {
	Id               uint32 `json:"-"`
	BookLevel        uint16 `json:"bookLevel"`
	NormalCount      uint16 `json:"normalCount"`
	SpecialCount     uint16 `json:"specialCount"`
	TotalUniqueCards uint16 `json:"totalUniqueCards"`
	CoverCardId      uint32 `json:"coverCardId"`
	ExpBonusPercent  uint16 `json:"expBonusPercent"`
}

func (r RestModel) GetName() string { return "monster-book" }
func (r RestModel) GetID() string   { return strconv.FormatUint(uint64(r.Id), 10) }
func (r *RestModel) SetID(id string) error {
	v, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		return err
	}
	r.Id = uint32(v)
	return nil
}

func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:               m.CharacterId(),
		BookLevel:        m.BookLevel(),
		NormalCount:      m.NormalCount(),
		SpecialCount:     m.SpecialCount(),
		TotalUniqueCards: m.TotalUniqueCards(),
		CoverCardId:      m.CoverCardId(),
		ExpBonusPercent:  m.ExpBonusPercent(),
	}, nil
}

// Extract pulls coverCardId from a PATCH input. Other fields are server-owned.
type PatchInput struct {
	Id          uint32 `json:"-"`
	CoverCardId uint32 `json:"coverCardId"`
}

func (p PatchInput) GetName() string { return "monster-book" }
func (p PatchInput) GetID() string   { return strconv.FormatUint(uint64(p.Id), 10) }
func (p *PatchInput) SetID(id string) error {
	v, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		return err
	}
	p.Id = uint32(v)
	return nil
}
```

- [ ] **Step 3: Write `card/rest.go`**

```go
package card

import (
	"strconv"
	"time"
)

type RestModel struct {
	CardId          uint32    `json:"-"`
	Level           uint8     `json:"level"`
	IsSpecial       bool      `json:"isSpecial"`
	FirstAcquiredAt time.Time `json:"firstAcquiredAt"`
}

func (r RestModel) GetName() string { return "monster-book-card" }
func (r RestModel) GetID() string   { return strconv.FormatUint(uint64(r.CardId), 10) }
func (r *RestModel) SetID(id string) error {
	v, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		return err
	}
	r.CardId = uint32(v)
	return nil
}

func Transform(m Model) (RestModel, error) {
	return RestModel{
		CardId:          m.CardId(),
		Level:           m.Level(),
		IsSpecial:       m.IsSpecial(),
		FirstAcquiredAt: m.FirstAcquiredAt(),
	}, nil
}
```

- [ ] **Step 4: Write `character/resource.go`**

```go
package character

import (
	"net/http"
	"strconv"

	"atlas-monster-book/card"
	"atlas-monster-book/collection"
	"atlas-monster-book/rest"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

const (
	GetMonsterBook   = "get_monster_book"
	PatchMonsterBook = "patch_monster_book"
	GetCards         = "get_monster_book_cards"
	GetCard          = "get_monster_book_card"
)

func InitResource(si jsonapi.ServerInformation) func(*gorm.DB) server.RouteInitializer {
	return func(db *gorm.DB) server.RouteInitializer {
		return func(router *mux.Router, l logrus.FieldLogger) {
			get := rest.RegisterHandler(l)(si)
			r := router.PathPrefix("/characters").Subrouter()
			r.HandleFunc("/{characterId}/monster-book", get(GetMonsterBook, handleGet(db))).Methods(http.MethodGet)
			r.HandleFunc("/{characterId}/monster-book", rest.RegisterInputHandler[collection.PatchInput](l)(si)(PatchMonsterBook, handlePatch(db))).Methods(http.MethodPatch)
			r.HandleFunc("/{characterId}/monster-book/cards", get(GetCards, handleListCards(db))).Methods(http.MethodGet)
			r.HandleFunc("/{characterId}/monster-book/cards/{cardId}", get(GetCard, handleGetCard(db))).Methods(http.MethodGet)
		}
	}
}

func handleGet(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				p := collection.NewProcessor(d.Logger(), d.Context(), db)
				m, err := p.GetByCharacterId(characterId)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				rm, _ := collection.Transform(m)
				server.MarshalResponse[collection.RestModel](d.Logger())(w)(c.ServerInformation())(r.URL.Query())(rm)
			}
		})
	}
}

func handlePatch(db *gorm.DB) rest.InputHandler[collection.PatchInput] {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext, in collection.PatchInput) http.HandlerFunc {
		return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				p := collection.NewProcessor(d.Logger(), d.Context(), db)
				if err := p.SetCoverAndEmit(uuid.New(), characterId, in.CoverCardId); err != nil {
					w.WriteHeader(http.StatusUnprocessableEntity)
					return
				}
				m, err := p.GetByCharacterId(characterId)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				rm, _ := collection.Transform(m)
				server.MarshalResponse[collection.RestModel](d.Logger())(w)(c.ServerInformation())(r.URL.Query())(rm)
			}
		})
	}
}

func handleListCards(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				cp := card.NewProcessor(d.Logger(), d.Context(), db)
				ms, err := cp.GetByCharacterId(characterId)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				// Optional filter[isSpecial]=true|false
				if v := r.URL.Query().Get("filter[isSpecial]"); v != "" {
					want, perr := strconv.ParseBool(v)
					if perr == nil {
						filtered := ms[:0]
						for _, m := range ms {
							if m.IsSpecial() == want {
								filtered = append(filtered, m)
							}
						}
						ms = filtered
					}
				}
				// Pagination — page[offset]/page[limit] (default 100, max 200).
				offset := parseUintQ(r.URL.Query().Get("page[offset]"), 0)
				limit := parseUintQ(r.URL.Query().Get("page[limit]"), 100)
				if limit > 200 {
					limit = 200
				}
				if int(offset) >= len(ms) {
					ms = nil
				} else {
					end := int(offset) + int(limit)
					if end > len(ms) {
						end = len(ms)
					}
					ms = ms[offset:end]
				}
				res, err := model.SliceMap(card.Transform)(model.FixedProvider(ms))()()
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				server.MarshalResponse[[]card.RestModel](d.Logger())(w)(c.ServerInformation())(r.URL.Query())(res)
			}
		})
	}
}

func handleGetCard(db *gorm.DB) rest.GetHandler {
	return func(d *rest.HandlerDependency, c *rest.HandlerContext) http.HandlerFunc {
		return rest.ParseCharacterId(d.Logger(), func(characterId uint32) http.HandlerFunc {
			return rest.ParseCardId(d.Logger(), func(cardId uint32) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					cp := card.NewProcessor(d.Logger(), d.Context(), db)
					m, err := cp.GetByCharacterIdAndCardId(characterId, cardId)
					if err != nil {
						w.WriteHeader(http.StatusNotFound)
						return
					}
					rm, _ := card.Transform(m)
					server.MarshalResponse[card.RestModel](d.Logger())(w)(c.ServerInformation())(r.URL.Query())(rm)
				}
			})
		})
	}
}

func parseUintQ(s string, def uint32) uint32 {
	if s == "" {
		return def
	}
	v, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return def
	}
	return uint32(v)
}
```

- [ ] **Step 5: Quick smoke test (build only)**

```bash
go build ./...
```

Expected: success.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-monster-book/atlas.com/monster-book/{rest,collection/rest.go,card/rest.go,character/resource.go}
git commit -m "feat(monster-book): REST handlers (GET, PATCH cover, paginated cards)"
```

---

### Task 15: Wire `main.go`

**Files:**
- Modify: `services/atlas-monster-book/atlas.com/monster-book/main.go`

- [ ] **Step 1: Replace `main.go` with the full bootstrap (mirror atlas-keys)**

```go
package main

import (
	"os"

	"atlas-monster-book/card"
	character2 "atlas-monster-book/kafka/consumer/character"
	mbconsumer "atlas-monster-book/kafka/consumer/monsterbook"
	"atlas-monster-book/character"
	"atlas-monster-book/collection"
	"atlas-monster-book/logger"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"
	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/Chronicle20/atlas/libs/atlas-service"
	tracing "github.com/Chronicle20/atlas/libs/atlas-tracing"
)

const serviceName = "atlas-monster-book"
const consumerGroupId = "Monster Book Service"

type Server struct{ baseUrl, prefix string }

func (s Server) GetBaseURL() string { return s.baseUrl }
func (s Server) GetPrefix() string  { return s.prefix }
func GetServer() Server             { return Server{baseUrl: "", prefix: "/api/"} }

func main() {
	l := logger.CreateLogger(serviceName)
	l.Infoln("Starting main service.")

	tdm := service.GetTeardownManager()
	tc, err := tracing.InitTracer(serviceName)
	if err != nil {
		l.WithError(err).Fatal("Unable to initialize tracer.")
	}

	db := database.Connect(l, database.SetMigrations(collection.Migration, card.Migration))

	cmf := consumer.GetManager().AddConsumer(l, tdm.Context(), tdm.WaitGroup())
	character2.InitConsumers(l)(cmf)(consumerGroupId)
	mbconsumer.InitConsumers(l)(cmf)(consumerGroupId)
	if err := character2.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register character lifecycle handlers.")
	}
	if err := mbconsumer.InitHandlers(l)(db)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register monster-book command handlers.")
	}

	tdm.TeardownFunc(func() { _ = producer.GetManager().Close(l) })

	server.New(l).
		WithContext(tdm.Context()).
		WithWaitGroup(tdm.WaitGroup()).
		SetBasePath(GetServer().GetPrefix()).
		AddRouteInitializer(character.InitResource(GetServer())(db)).
		SetPort(os.Getenv("REST_PORT")).
		AddRouteInitializer(server.MountHandler("/debug/consumers", consumer.GetManager().DebugHandler())).
		Run()

	tdm.TeardownFunc(tracing.Teardown(l)(tc))
	tdm.Wait()
	l.Infoln("Service shutdown.")
}
```

- [ ] **Step 2: Build**

```bash
go build ./...
```

- [ ] **Step 3: Commit**

```bash
git add services/atlas-monster-book/atlas.com/monster-book/main.go
git commit -m "feat(monster-book): main.go wiring (db, consumers, REST)"
```

---

### Task 16: Dockerfile

**Files:**
- Create: `services/atlas-monster-book/Dockerfile`

- [ ] **Step 1: Copy `services/atlas-keys/Dockerfile` to `services/atlas-monster-book/Dockerfile`**

- [ ] **Step 2: Replace every occurrence of `atlas-keys` with `atlas-monster-book`** (paths, `keys` directory inside the service path becomes `monster-book`).

Use:

```bash
sed -i 's|atlas-keys/atlas.com/keys|atlas-monster-book/atlas.com/monster-book|g; s|atlas-keys|atlas-monster-book|g' services/atlas-monster-book/Dockerfile
```

- [ ] **Step 3: Build the image to verify**

```bash
docker build -f services/atlas-monster-book/Dockerfile -t atlas-monster-book:plan-test .
```

Expected: image builds. Discard the resulting image.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-monster-book/Dockerfile
git commit -m "build(monster-book): Dockerfile"
```

---

### Task 17: docker-compose entry

**Files:**
- Modify: root `docker-compose.yml` (or whichever compose file exists in this repo)

- [ ] **Step 1: Locate the compose file**

```bash
ls docker-compose*.yml docker-compose*.yaml 2>/dev/null
```

Open the file and find the `atlas-keys` service entry as a template.

- [ ] **Step 2: Add a parallel `atlas-monster-book` block**

Use the atlas-keys block as a 1:1 template (same env vars, same dependency on Postgres + Kafka, same port pattern). Replace `keys` → `monster-book` in service name, image name, container name, volume mounts, and `REST_PORT`. Pick the next free host port for the REST mapping (look at the highest existing service port and +1).

- [ ] **Step 3: Bring up the service alone to confirm it starts**

```bash
docker compose up atlas-monster-book
```

Expected: service starts, logs `Starting main service.`, REST port listens. `Ctrl-C` to stop.

- [ ] **Step 4: Commit**

```bash
git add docker-compose.yml  # or whichever file you edited
git commit -m "build(monster-book): add to docker-compose"
```

---

## Phase B — atlas-inventory: consume-on-pickup branch

### Task 18: New `ITEM.CONSUMED_ON_PICKUP` message + producer plumbing

**Files:**
- Create: `services/atlas-inventory/atlas.com/inventory/kafka/message/pickup/kafka.go`

- [ ] **Step 1: Locate the existing Kafka message pattern in atlas-inventory**

```bash
ls services/atlas-inventory/atlas.com/inventory/kafka/message/
```

Identify a peer file (e.g. `compartment/kafka.go`) and use its envelope shape as a template.

- [ ] **Step 2: Write the new message types**

```go
package pickup

import "github.com/google/uuid"

const (
	EnvCommandTopic = "COMMAND_TOPIC_ITEM_CONSUMED_ON_PICKUP"
	CommandType     = "ITEM_CONSUMED_ON_PICKUP"
)

type Command struct {
	TenantId      uuid.UUID `json:"tenantId"`
	CharacterId   uint32    `json:"characterId"`
	ItemId        uint32    `json:"itemId"`
	TransactionId uuid.UUID `json:"transactionId"`
	Type          string    `json:"type"`
}
```

- [ ] **Step 3: Build**

```bash
cd services/atlas-inventory/atlas.com/inventory && go build ./...
```

- [ ] **Step 4: Commit**

```bash
git add services/atlas-inventory/atlas.com/inventory/kafka/message/pickup/kafka.go
git commit -m "feat(inventory): ITEM.CONSUMED_ON_PICKUP message type"
```

---

### Task 19: Branch in `compartment.Processor.AttemptItemPickUp`

**Files:**
- Modify: `services/atlas-inventory/atlas.com/inventory/compartment/processor.go` (around line 1148)

- [ ] **Step 1: Write a test for the new branch**

Open `services/atlas-inventory/atlas.com/inventory/compartment/processor_test.go` and add:

```go
func TestAttemptItemPickUpSkipsInsertWhenConsumeOnPickup(t *testing.T) {
	// Arrange a card item (templateId in 2380000..2389999) and a fake consumable
	// data layer that returns ConsumeOnPickup=true. Use the existing test scaffolding
	// pattern in this file (assets, mock drop processor, in-memory db).
	// Assert: no asset row exists for the templateId after the call,
	//         the drop reservation was released (RequestPickUp called),
	//         and the message buffer contains a Command on EnvCommandTopic.
}
```

(The full test body depends on the existing scaffolds in `processor_test.go` — read those first and adapt.)

- [ ] **Step 2: Run test (FAIL)**

- [ ] **Step 3: Add the branch in `AttemptItemPickUp`**

In `services/atlas-inventory/atlas.com/inventory/compartment/processor.go`, modify the function starting around line 1148. Insert the consume-on-pickup branch *before* the existing inventoryType lookup (at line 1150). Pseudocode skeleton — fill in the actual import path for the consumable data accessor that already exists in this package:

```go
// At the top of AttemptItemPickUp (after closure, before LockRegistry().Get).
inventoryType, ok := inventory.TypeFromItemId(item.Id(templateId))
if !ok {
    return errors.New("invalid inventory item")
}

// Consume-on-pickup branch: only relevant for use-type items.
if inventoryType == inventory.TypeValueUse {
    cm, cerr := consumable.NewProcessor(p.l, p.ctx).GetById(templateId)
    if cerr == nil && cm.ConsumeOnPickup() {
        // Skip insert. Emit the generic Kafka command. Still release the drop reservation.
        if err := mb.Put(pickup.EnvCommandTopic, pickup.NewCommandProvider(p.t.Id(), characterId, templateId, transactionId)); err != nil {
            return err
        }
        return p.dropProcessor.RequestPickUp(mb)(f, dropId, characterId)
    }
}
```

- [ ] **Step 4: Add a `NewCommandProvider` helper in `pickup` package**

In `services/atlas-inventory/atlas.com/inventory/kafka/message/pickup/kafka.go`:

```go
import (
	"encoding/json"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/segmentio/kafka-go"
)

func NewCommandProvider(tenantId uuid.UUID, characterId uint32, itemId uint32, transactionId uuid.UUID) model.Provider[[]kafka.Message] {
	cmd := Command{
		TenantId:      tenantId,
		CharacterId:   characterId,
		ItemId:        itemId,
		TransactionId: transactionId,
		Type:          CommandType,
	}
	return func() ([]kafka.Message, error) {
		body, err := json.Marshal(cmd)
		if err != nil {
			return nil, err
		}
		return []kafka.Message{{Value: body}}, nil
	}
}
```

- [ ] **Step 5: Run test (PASS)**

```bash
cd services/atlas-inventory/atlas.com/inventory && go test ./compartment/...
```

- [ ] **Step 6: Commit**

```bash
git add services/atlas-inventory/atlas.com/inventory/{compartment/processor.go,compartment/processor_test.go,kafka/message/pickup/kafka.go}
git commit -m "feat(inventory): consume-on-pickup branch in AttemptItemPickUp"
```

---

### Task 20: Verify atlas-inventory build + integration

**Files:** none

- [ ] **Step 1: Build atlas-inventory**

```bash
cd services/atlas-inventory/atlas.com/inventory && go build ./...
```

- [ ] **Step 2: Run all atlas-inventory tests**

```bash
go test ./...
```

Expected: all green.

- [ ] **Step 3: Build the Docker image**

```bash
docker build -f services/atlas-inventory/Dockerfile -t atlas-inventory:plan-test .
```

Expected: success. Discard the image.

- [ ] **Step 4: Commit (only if any small fixups were needed)**

If build/tests passed cleanly, no commit. If you had to fix imports or similar, commit those:

```bash
git add -p && git commit -m "fix(inventory): build fixups for consume-on-pickup branch"
```

---

## Phase C — atlas-consumables: pickup consumer

### Task 21: New consumer for `ITEM.CONSUMED_ON_PICKUP`

**Files:**
- Create: `services/atlas-consumables/atlas.com/consumables/kafka/message/pickup/kafka.go` (mirror of atlas-inventory's `Command` struct)
- Create: `services/atlas-consumables/atlas.com/consumables/kafka/message/monsterbook/kafka.go`
- Create: `services/atlas-consumables/atlas.com/consumables/kafka/consumer/pickup/consumer.go`
- Test: `services/atlas-consumables/atlas.com/consumables/kafka/consumer/pickup/consumer_test.go`

- [ ] **Step 1: Mirror the pickup `Command` struct**

```go
package pickup

import "github.com/google/uuid"

const (
	EnvCommandTopic = "COMMAND_TOPIC_ITEM_CONSUMED_ON_PICKUP"
	CommandType     = "ITEM_CONSUMED_ON_PICKUP"
)

type Command struct {
	TenantId      uuid.UUID `json:"tenantId"`
	CharacterId   uint32    `json:"characterId"`
	ItemId        uint32    `json:"itemId"`
	TransactionId uuid.UUID `json:"transactionId"`
	Type          string    `json:"type"`
}
```

- [ ] **Step 2: Mirror the monster-book outbound command struct**

```go
package monsterbook

import "github.com/google/uuid"

const (
	EnvCommandTopic         = "COMMAND_TOPIC_MONSTER_BOOK"
	CommandTypeCardPickedUp = "CARD_PICKED_UP"
)

type Command[B any] struct {
	TenantId    uuid.UUID `json:"tenantId"`
	CharacterId uint32    `json:"characterId"`
	EventId     uuid.UUID `json:"eventId"`
	Type        string    `json:"type"`
	Body        B         `json:"body"`
}

type CardPickedUpBody struct {
	CardId uint32 `json:"cardId"`
	Source string `json:"source"`
}
```

- [ ] **Step 3: Write consumer test**

```go
package pickup

import (
	"context"
	"testing"

	"atlas-consumables/kafka/message/pickup"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

func TestHandleCardItemEmitsMonsterBookCommand(t *testing.T) {
	// Arrange a fake message buffer that records Put calls.
	// Call handlePickup with a Command for itemId=2380000.
	// Assert: buffer received one MONSTER_BOOK Command with CardPickedUpBody{CardId:2380000}.
	t.Skip("integrate with project's existing test scaffold for kafka consumers")
}

func TestHandleNonCardItemLogsAndSkips(t *testing.T) {
	t.Skip("same scaffold; verify no MONSTER_BOOK emission for itemId=2000000")
}
```

(Replace the skips with the project's existing in-process consumer test pattern — search for an existing `kafka/consumer/*_test.go` in atlas-consumables for the canonical scaffold.)

- [ ] **Step 4: Write `consumer.go`**

```go
package pickup

import (
	"context"

	consumer2 "atlas-consumables/kafka/consumer"
	"atlas-consumables/kafka/message"
	mbmsg "atlas-consumables/kafka/message/monsterbook"
	pickupmsg "atlas-consumables/kafka/message/pickup"
	"atlas-consumables/kafka/producer"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	kmessage "github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(rf func(consumer.Config, ...model.Decorator[consumer.Config])) func(string) {
	return func(rf func(consumer.Config, ...model.Decorator[consumer.Config])) func(string) {
		return func(groupId string) {
			rf(consumer2.NewConfig(l)("item_consumed_on_pickup")(pickupmsg.EnvCommandTopic)(groupId),
				consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(string, handler.Handler) (string, error)) error {
	return func(rf func(string, handler.Handler) (string, error)) error {
		t, _ := topic.EnvProvider(l)(pickupmsg.EnvCommandTopic)()
		_, err := rf(t, kmessage.AdaptHandler(kmessage.PersistentConfig(handlePickup())))
		return err
	}
}

const (
	cardItemPrefix = 238 // itemId / 10000 == 238
)

func handlePickup() func(logrus.FieldLogger, context.Context, pickupmsg.Command) {
	return func(l logrus.FieldLogger, ctx context.Context, cmd pickupmsg.Command) {
		if cmd.Type != pickupmsg.CommandType {
			return
		}
		// Only the card branch is implemented in v1 (design §4.1 step 4).
		if cmd.ItemId/10000 != cardItemPrefix {
			l.Warnf("ITEM.CONSUMED_ON_PICKUP for non-card item %d — no handler yet, skipping.", cmd.ItemId)
			return
		}
		err := message.Emit(producer.ProviderImpl(l)(ctx))(func(mb *message.Buffer) error {
			out := mbmsg.Command[mbmsg.CardPickedUpBody]{
				TenantId:    cmd.TenantId,
				CharacterId: cmd.CharacterId,
				EventId:     cmd.TransactionId, // transactionId IS the eventId per design §4.1
				Type:        mbmsg.CommandTypeCardPickedUp,
				Body:        mbmsg.CardPickedUpBody{CardId: cmd.ItemId, Source: "drop_pickup"},
			}
			return mb.Put(mbmsg.EnvCommandTopic, providerOf(out))
		})
		if err != nil {
			l.WithError(err).Errorf("Failed to emit MONSTER_BOOK.CARD_PICKED_UP for character %d card %d.", cmd.CharacterId, cmd.ItemId)
		}
		_ = uuid.Nil // silence unused
	}
}

func providerOf[T any](v T) model.Provider[[]kafka.Message] {
	return func() ([]kafka.Message, error) {
		b, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		return []kafka.Message{{Value: b}}, nil
	}
}
```

(Add the appropriate `encoding/json` and `kafka-go` imports.)

- [ ] **Step 5: Run tests**

```bash
cd services/atlas-consumables/atlas.com/consumables && go test ./kafka/consumer/pickup/...
```

- [ ] **Step 6: Commit**

```bash
git add services/atlas-consumables/atlas.com/consumables/kafka/{message/pickup,message/monsterbook,consumer/pickup}
git commit -m "feat(consumables): ITEM.CONSUMED_ON_PICKUP consumer (card branch)"
```

---

### Task 22: Wire consumer into `atlas-consumables/main.go`

**Files:**
- Modify: `services/atlas-consumables/atlas.com/consumables/main.go`

- [ ] **Step 1: Add imports + register consumer** alongside existing `cmf` and `RegisterHandler` calls. Pattern is identical to other consumer registrations already in this main.go.

```go
import (
    pickupconsumer "atlas-consumables/kafka/consumer/pickup"
)

// ... after existing InitConsumers/InitHandlers calls:
pickupconsumer.InitConsumers(l)(cmf)(consumerGroupId)
if err := pickupconsumer.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
    l.WithError(err).Fatal("Unable to register pickup handlers.")
}
```

- [ ] **Step 2: Build**

```bash
cd services/atlas-consumables/atlas.com/consumables && go build ./...
```

- [ ] **Step 3: Commit**

```bash
git add services/atlas-consumables/atlas.com/consumables/main.go
git commit -m "feat(consumables): register ITEM.CONSUMED_ON_PICKUP consumer in main"
```

---

### Task 23: Integration build + Docker

**Files:** none

- [ ] **Step 1: `go test ./...` for atlas-consumables**

- [ ] **Step 2: `docker build -f services/atlas-consumables/Dockerfile .`**

- [ ] **Step 3: No commit unless fixups were required.**

---

## Phase D — atlas-channel: packet writers + 0x39 handler

### Task 24: `MonsterBookSetCard` writer (`0x53`)

**Files:**
- Create: `libs/atlas-packet/character/clientbound/monsterbook/set_card.go`
- Test: `libs/atlas-packet/character/clientbound/monsterbook/set_card_test.go`

- [ ] **Step 1: Read `libs/atlas-packet/character/clientbound/effect.go`** for the `Operation()`/`Encode()` writer pattern. Note the constant naming convention (`CharacterEffectWriter = "CharacterEffect"`).

- [ ] **Step 2: Write the test**

```go
package monsterbook

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestSetCardEncodeShape(t *testing.T) {
	body := SetCard{CardId: 2380000, Level: 3, Added: true}
	out := body.Encode(logrus.New(), context.Background())(map[string]interface{}{})
	// Expected: 1 byte flag (1=added, 0=full) + 4 bytes cardId + 4 bytes level
	if len(out) != 9 {
		t.Fatalf("expected 9-byte body, got %d", len(out))
	}
	if out[0] != 1 {
		t.Fatalf("expected flag byte=1, got %d", out[0])
	}
}
```

- [ ] **Step 3: Run test (FAIL)**

- [ ] **Step 4: Write `set_card.go`**

```go
package monsterbook

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-packet/response"
	"github.com/sirupsen/logrus"
)

const MonsterBookSetCardWriter = "MonsterBookSetCard"

type SetCard struct {
	CardId uint32
	Level  uint8
	Added  bool // true = added/levelled (flag 1), false = already full (flag 0)
}

func (s SetCard) Operation() string { return MonsterBookSetCardWriter }

func (s SetCard) Encode(l logrus.FieldLogger, ctx context.Context) func(map[string]interface{}) []byte {
	return func(_ map[string]interface{}) []byte {
		w := response.NewWriter(l)
		var flag byte
		if s.Added {
			flag = 1
		}
		w.WriteByte(flag)
		w.WriteInt(int32(s.CardId))
		w.WriteInt(int32(s.Level))
		return w.Bytes()
	}
}
```

> **Engineer note (writer API):** the actual API on `response.Writer` may use names like `WriteUint32` / `WriteUint8` rather than `WriteInt` / `WriteByte`. Open `libs/atlas-packet/character/clientbound/effect.go` and copy the call patterns there exactly.

- [ ] **Step 5: Run test (PASS)**

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-packet/character/clientbound/monsterbook/set_card.go libs/atlas-packet/character/clientbound/monsterbook/set_card_test.go
git commit -m "feat(packet): MonsterBookSetCard writer (0x53)"
```

---

### Task 25: `MonsterBookSetCover` writer (`0x54`)

**Files:**
- Create: `libs/atlas-packet/character/clientbound/monsterbook/set_cover.go`
- Test: `libs/atlas-packet/character/clientbound/monsterbook/set_cover_test.go`

- [ ] **Step 1: Write test**

```go
package monsterbook

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestSetCoverEncodeShape(t *testing.T) {
	body := SetCover{CardId: 2380000}
	out := body.Encode(logrus.New(), context.Background())(map[string]interface{}{})
	if len(out) != 4 {
		t.Fatalf("expected 4 bytes (int cardId), got %d", len(out))
	}
}
```

- [ ] **Step 2: Run test (FAIL)**

- [ ] **Step 3: Write `set_cover.go`**

```go
package monsterbook

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-packet/response"
	"github.com/sirupsen/logrus"
)

const MonsterBookSetCoverWriter = "MonsterBookSetCover"

type SetCover struct {
	CardId uint32 // 0 to clear
}

func (s SetCover) Operation() string { return MonsterBookSetCoverWriter }

func (s SetCover) Encode(l logrus.FieldLogger, ctx context.Context) func(map[string]interface{}) []byte {
	return func(_ map[string]interface{}) []byte {
		w := response.NewWriter(l)
		w.WriteInt(int32(s.CardId))
		return w.Bytes()
	}
}
```

- [ ] **Step 4: Run test (PASS)**

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-packet/character/clientbound/monsterbook/set_cover.go libs/atlas-packet/character/clientbound/monsterbook/set_cover_test.go
git commit -m "feat(packet): MonsterBookSetCover writer (0x54)"
```

---

### Task 26: Recv handler — `MonsterBookCover` (`0x39`)

**Files:**
- Create: `libs/atlas-packet/character/serverbound/monsterbook/cover.go` (decode struct)
- Create: `services/atlas-channel/atlas.com/channel/socket/handler/monster_book_cover.go`

- [ ] **Step 1: Read `libs/atlas-packet/character/serverbound/`** for the standard `Decode(l, ctx)(r, opts)` pattern. Pick a similar handler (e.g. one that takes a single Int) and use it as a template.

- [ ] **Step 2: Write the inbound packet decoder**

```go
package monsterbook

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

const MonsterBookCoverHandler = "MonsterBookCover"

type Cover struct {
	cardId uint32
}

func (c *Cover) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, opts map[string]interface{}) {
	return func(r *request.Reader, _ map[string]interface{}) {
		c.cardId = uint32(r.ReadInt())
	}
}

func (c Cover) CardId() uint32     { return c.cardId }
func (c Cover) Operation() string  { return MonsterBookCoverHandler }
func (c Cover) String() string     { return fmt.Sprintf("MonsterBookCover{cardId=%d}", c.cardId) }
```

- [ ] **Step 3: Write atlas-channel handler**

```go
package handler

import (
	"atlas-channel/monsterbook"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	mbsb "github.com/Chronicle20/atlas/libs/atlas-packet/character/serverbound/monsterbook"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

func MonsterBookCoverHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := mbsb.Cover{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())
		if err := monsterbook.NewProcessor(l, ctx).RequestSetCover(s.CharacterId(), p.CardId()); err != nil {
			l.WithError(err).Errorf("Failed to emit MONSTER_BOOK.SET_COVER for character %d.", s.CharacterId())
		}
	}
}
```

- [ ] **Step 4: Stub `monsterbook.NewProcessor(l, ctx).RequestSetCover`**

The atlas-channel `monsterbook` package doesn't exist yet. Create `services/atlas-channel/atlas.com/channel/monsterbook/processor.go`:

```go
package monsterbook

import (
	"context"

	"atlas-channel/kafka/message"
	mbmsg "atlas-channel/kafka/message/monsterbook"
	"atlas-channel/kafka/producer"

	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	RequestSetCover(characterId uint32, coverCardId uint32) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	t   tenant.Model
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx, t: tenant.MustFromContext(ctx)}
}

func (p *ProcessorImpl) RequestSetCover(characterId uint32, coverCardId uint32) error {
	return message.Emit(producer.ProviderImpl(p.l)(p.ctx))(func(mb *message.Buffer) error {
		cmd := mbmsg.Command[mbmsg.SetCoverBody]{
			TenantId:    p.t.Id(),
			CharacterId: characterId,
			EventId:     uuid.New(),
			Type:        mbmsg.CommandTypeSetCover,
			Body:        mbmsg.SetCoverBody{CoverCardId: coverCardId},
		}
		return mb.Put(mbmsg.EnvCommandTopic, providerOf(cmd))
	})
}

// providerOf — same pattern as in atlas-monster-book; or import from a shared helper if one exists.
```

- [ ] **Step 5: Add `kafka/message/monsterbook/kafka.go` to atlas-channel**

This is the *channel-side* mirror of the message types defined in atlas-monster-book Task 9. Copy the file from atlas-monster-book and adjust imports:

```go
// services/atlas-channel/atlas.com/channel/kafka/message/monsterbook/kafka.go
// Same content as services/atlas-monster-book/atlas.com/monster-book/kafka/message/monsterbook/kafka.go
```

- [ ] **Step 6: Build**

```bash
cd services/atlas-channel/atlas.com/channel && go build ./...
```

- [ ] **Step 7: Commit**

```bash
git add libs/atlas-packet/character/serverbound/monsterbook/ services/atlas-channel/atlas.com/channel/{socket/handler/monster_book_cover.go,monsterbook/processor.go,kafka/message/monsterbook}
git commit -m "feat(channel): MonsterBookCover (0x39) recv handler + producer"
```

---

### Task 27: Register the handler + writers in atlas-channel `main.go`

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/main.go`

- [ ] **Step 1: Find the writer/handler registration section**

```bash
grep -n "WriterRegistry\|HandlerRegistry\|RegisterWriter\|RegisterHandler" services/atlas-channel/atlas.com/channel/main.go | head
```

- [ ] **Step 2: Add `MonsterBookSetCardWriter` and `MonsterBookSetCoverWriter` to the writer name list (mirror existing `CharacterEffectWriter` registration).** Add `MonsterBookCoverHandler` to the handler name list (mirror an existing handler registration).

The exact lines depend on this main.go's structure. Pattern: find an existing `*Writer = "..."` constant being registered and add the new ones in the same sequence.

- [ ] **Step 3: Build**

```bash
go build ./...
```

- [ ] **Step 4: Update opcode tenant config notes**

This is **not a code change** — leave a note in `docs/tasks/task-056-monster-book/audit.md` (or in this plan's "ops handoff" section) that tenant operators must add the following opcode mappings via atlas-tenants (no code task; one-time runtime config):

| Opcode | Direction | Name |
|---|---|---|
| `0x39` | Recv | `MonsterBookCover` |
| `0x53` | Send | `MonsterBookSetCard` |
| `0x54` | Send | `MonsterBookSetCover` |

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/main.go
git commit -m "feat(channel): register MonsterBook writers + recv handler"
```

---

### Task 28: Decoder/handler unit tests

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/socket/handler/monster_book_cover_test.go`

- [ ] **Step 1: Write a decode-and-emit test**

Open an existing handler test in this directory (e.g. `character_chair_interaction_test.go` or similar) for the in-process scaffold. Mirror it:

```go
// Test that decoding a 4-byte body produces the right cardId,
// and that calling MonsterBookCoverHandleFunc emits exactly one
// MONSTER_BOOK.SET_COVER on the buffer.
```

- [ ] **Step 2: Run test (PASS once dependencies wired)**

```bash
go test ./socket/handler/...
```

- [ ] **Step 3: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/handler/monster_book_cover_test.go
git commit -m "test(channel): MonsterBookCover handler decode + emit"
```

---

## Phase E — atlas-channel: outbound consumers + cover decorator

### Task 29: REST client to atlas-monster-book

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/monsterbook/rest.go`
- Create: `services/atlas-channel/atlas.com/channel/monsterbook/requests.go`

Mirror an existing per-service REST client in `atlas-channel/<peer>/{rest.go,requests.go}` (e.g. `atlas-channel/quest/`).

- [ ] **Step 1: Write `rest.go` (RestModel of the response)**

```go
package monsterbook

type CollectionRestModel struct {
	Id               uint32 `json:"-"`
	BookLevel        uint16 `json:"bookLevel"`
	NormalCount      uint16 `json:"normalCount"`
	SpecialCount     uint16 `json:"specialCount"`
	TotalUniqueCards uint16 `json:"totalUniqueCards"`
	CoverCardId      uint32 `json:"coverCardId"`
	ExpBonusPercent  uint16 `json:"expBonusPercent"`
}

func (r CollectionRestModel) GetName() string { return "monster-book" }
func (r CollectionRestModel) GetID() string   { return "" }
func (r *CollectionRestModel) SetID(_ string) error { return nil }
```

- [ ] **Step 2: Write `requests.go` (`requestByCharacterId` + `Extract`)**

Use the existing pattern in `services/atlas-channel/atlas.com/channel/quest/requests.go` (or similar) for `requests.Provider[RestModel, Model]` shape.

- [ ] **Step 3: Add `Get` method on the processor** (extending Task 26's `monsterbook.Processor`):

```go
func (p *ProcessorImpl) GetByCharacterId(characterId uint32) (Collection, error) {
    mp := requests.Provider[CollectionRestModel, Collection](p.l, p.ctx)(requestByCharacterId(characterId), Extract)
    return mp()
}

type Collection struct {
    bookLevel       uint16
    normalCount     uint16
    specialCount    uint16
    coverCardId     uint32
    expBonusPercent uint16
}

// + getters and Extract function
```

- [ ] **Step 4: Build**

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/monsterbook/{rest.go,requests.go,processor.go}
git commit -m "feat(channel): monster-book REST client"
```

---

### Task 30: Outbound consumer — `MONSTER_BOOK.CARD_ADDED`

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/kafka/consumer/monsterbook/consumer.go`
- Test: `services/atlas-channel/atlas.com/channel/kafka/consumer/monsterbook/consumer_test.go`

Translates `CARD_ADDED` → `MonsterBookSetCard` (always) + `EffectSimple{MonsterBookCardGet}` to owner (only when `!Full`) + `EffectSimpleForeign{MonsterBookCardGet}` map broadcast (only when `!Full`).

- [ ] **Step 1: Find the writer plumbing**

Read an existing consumer that fans out into multiple session writes — `atlas-channel/kafka/consumer/buff/` is a good template. Note the `session.Announce(l)(ctx)(wp)(writerName)(body.Encode)(s)` pattern.

- [ ] **Step 2: Write tests for the three-packet fan-out (added) and one-packet fan-out (full)**

(Follow the existing consumer test scaffold.)

- [ ] **Step 3: Write `consumer.go`**

```go
package monsterbook

import (
	"context"

	"atlas-channel/session"
	"atlas-channel/socket/writer"
	consumer2 "atlas-channel/kafka/consumer"
	mbmsg "atlas-channel/kafka/message/monsterbook"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	kmessage "github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	mbcb "github.com/Chronicle20/atlas/libs/atlas-packet/character/clientbound/monsterbook"
	charpkt "github.com/Chronicle20/atlas/libs/atlas-packet/character/clientbound"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(rf func(consumer.Config, ...model.Decorator[consumer.Config])) func(string) {
	return func(rf func(consumer.Config, ...model.Decorator[consumer.Config])) func(string) {
		return func(groupId string) {
			rf(consumer2.NewConfig(l)("monster_book_status")(mbmsg.EnvEventTopicStatus)(groupId),
				consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc session.Cache) func(wp writer.Producer) func(rf func(string, handler.Handler) (string, error)) error {
	return func(sc session.Cache) func(wp writer.Producer) func(rf func(string, handler.Handler) (string, error)) error {
		return func(wp writer.Producer) func(rf func(string, handler.Handler) (string, error)) error {
			return func(rf func(string, handler.Handler) (string, error)) error {
				t, _ := topic.EnvProvider(l)(mbmsg.EnvEventTopicStatus)()
				if _, err := rf(t, kmessage.AdaptHandler(kmessage.PersistentConfig(handleCardAdded(sc, wp)))); err != nil {
					return err
				}
				if _, err := rf(t, kmessage.AdaptHandler(kmessage.PersistentConfig(handleCoverChanged(sc, wp)))); err != nil {
					return err
				}
				return nil
			}
		}
	}
}

func handleCardAdded(sc session.Cache, wp writer.Producer) func(logrus.FieldLogger, context.Context, mbmsg.StatusEvent[mbmsg.CardAddedBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, ev mbmsg.StatusEvent[mbmsg.CardAddedBody]) {
		if ev.Type != mbmsg.StatusEventTypeCardAdded {
			return
		}
		s, ok := sc.GetByCharacterId(ev.CharacterId)
		if !ok {
			return
		}
		// Always: SetCard packet to owner.
		set := mbcb.SetCard{CardId: ev.Body.CardId, Level: ev.Body.NewLevel, Added: !ev.Body.Full}
		_ = session.Announce(l)(ctx)(wp)(mbcb.MonsterBookSetCardWriter)(set.Encode)(s)
		if ev.Body.Full {
			return
		}
		// Owner effect.
		owner := charpkt.EffectSimple{Mode: charpkt.CharacterEffectMonsterBookCardGet}
		_ = session.Announce(l)(ctx)(wp)(charpkt.CharacterEffectWriter)(owner.Encode)(s)
		// Map broadcast (foreign).
		foreign := charpkt.EffectSimpleForeign{CharacterId: ev.CharacterId, Mode: charpkt.CharacterEffectMonsterBookCardGet}
		_ = session.AnnounceMap(l)(ctx)(wp)(charpkt.CharacterEffectForeignWriter)(foreign.Encode)(s) // confirm exact AnnounceMap signature in session pkg
	}
}

func handleCoverChanged(sc session.Cache, wp writer.Producer) func(logrus.FieldLogger, context.Context, mbmsg.StatusEvent[mbmsg.CoverChangedBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, ev mbmsg.StatusEvent[mbmsg.CoverChangedBody]) {
		if ev.Type != mbmsg.StatusEventTypeCoverChanged {
			return
		}
		s, ok := sc.GetByCharacterId(ev.CharacterId)
		if !ok {
			return
		}
		body := mbcb.SetCover{CardId: ev.Body.CoverCardId}
		_ = session.Announce(l)(ctx)(wp)(mbcb.MonsterBookSetCoverWriter)(body.Encode)(s)
	}
}
```

> **Engineer notes:**
> - The exact `session.Announce` / `session.AnnounceMap` signature differs across consumers — search `services/atlas-channel/atlas.com/channel/kafka/consumer/buff/` for the canonical map-broadcast pattern.
> - `EffectSimple` / `EffectSimpleForeign` field names (`Mode` / `CharacterId`) need to match the actual struct in `libs/atlas-packet/character/clientbound/effect.go`. Open that file to confirm.

- [ ] **Step 4: Run tests (PASS)**

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/kafka/consumer/monsterbook
git commit -m "feat(channel): outbound MONSTER_BOOK status consumers"
```

---

### Task 31: Wire outbound consumer in `atlas-channel/main.go`

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/main.go`

- [ ] **Step 1: Add `mbconsumer.InitConsumers` + `InitHandlers` calls** alongside the existing `account2.InitHandlers` / `asset.InitHandlers` block (around line 241–270 in current main.go).

```go
import (
    mbconsumer "atlas-channel/kafka/consumer/monsterbook"
)

// In the InitHandlers block:
mbconsumer.InitConsumers(fl)(cmf)(consumerGroupId)
if err = mbconsumer.InitHandlers(fl)(sc)(wp)(consumer.GetManager().RegisterHandler); err != nil {
    fl.WithError(err).Fatal("Unable to register monster-book status handlers.")
}
```

- [ ] **Step 2: Build**

- [ ] **Step 3: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/main.go
git commit -m "feat(channel): register monster-book status consumers"
```

---

### Task 32: `MonsterBookCoverDecorator` on `character.Processor`

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/character/processor.go` (add method ~line 156, register in interface ~line 31)
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_info_request.go` (line 29)
- Modify: `services/atlas-channel/atlas.com/channel/character/model.go` (add `coverCardId` field + `SetCoverCardId` + `CoverCardId()`)

- [ ] **Step 1: Read `character/model.go`** for the immutable model + builder pattern. Add a `coverCardId uint32` field via the existing builder, with the corresponding `SetCoverCardId(uint32)` builder method and `CoverCardId() uint32` getter.

- [ ] **Step 2: Add `SetCoverCardId(uint32) Model` on `character.Model`** (mirror `SetSkills`, `SetQuests`).

- [ ] **Step 3: Add `MonsterBookCoverDecorator(m Model) Model` to the `Processor` interface and implementation** (mirror `SkillModelDecorator`):

```go
// In interface (line ~31):
MonsterBookCoverDecorator(m Model) Model

// In impl (after QuestModelDecorator, ~line 156):
func (p *ProcessorImpl) MonsterBookCoverDecorator(m Model) Model {
    col, err := monsterbook.NewProcessor(p.l, p.ctx).GetByCharacterId(m.Id())
    if err != nil {
        return m
    }
    return m.SetCoverCardId(col.CoverCardId())
}
```

Add the import: `"atlas-channel/monsterbook"`.

- [ ] **Step 4: Append the decorator in `character_info_request.go` line 27-31**

```go
decorators := make([]model.Decorator[character.Model], 0)
decorators = append(decorators, cp.MonsterBookCoverDecorator) // unconditional
if p.PetInfo() {
    decorators = append(decorators, cp.PetAssetEnrichmentDecorator)
}
```

- [ ] **Step 5: Decide whether the cover ships in the character info packet**

Per design §4.6: "if any are wired in v1". For v1, the cover field is exposed *only* through REST (UI consumption). The character info packet itself does not need to embed the cover for v83 client compatibility. Therefore the decorator's job in v1 is to **populate the model** so other in-channel features can read it without an extra fetch — not to mutate the wire packet.

If a follow-up confirms the v83 packet expects the cover at a specific offset, add the encode hook to `libs/atlas-packet/character/data.go:676` (`encodeMonsterBook`) at that point. **Out of scope for this task.**

- [ ] **Step 6: Build**

```bash
cd services/atlas-channel/atlas.com/channel && go build ./...
```

- [ ] **Step 7: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/character/{processor.go,model.go,builder.go} services/atlas-channel/atlas.com/channel/socket/handler/character_info_request.go
git commit -m "feat(channel): MonsterBookCoverDecorator on character info"
```

---

## Phase F — Quest condition

### Task 33: Add constant to `libs/atlas-saga`

**Files:**
- Modify: `libs/atlas-saga/validation.go` (after line 44, before the closing `)`)

- [ ] **Step 1: Add the constant**

```go
PqCustomDataCondition           = "pqCustomData"
MonsterBookCountCondition       = "monsterBookCount"
)
```

- [ ] **Step 2: Build**

```bash
cd libs/atlas-saga && go build ./...
```

- [ ] **Step 3: Commit**

```bash
git add libs/atlas-saga/validation.go
git commit -m "feat(atlas-saga): MonsterBookCountCondition constant"
```

---

### Task 34: atlas-query-aggregator — accept `monsterBookCount` and evaluate

**Files:**
- Modify: `services/atlas-query-aggregator/atlas.com/query-aggregator/validation/model.go` (lines 20–56, 124, 380+)
- Create: `services/atlas-query-aggregator/atlas.com/query-aggregator/monsterbook/{rest.go,requests.go,processor.go}` (REST client to atlas-monster-book)

- [ ] **Step 1: Add `MonsterBookCountCondition` constant in `validation/model.go` (line 56 area)**

```go
PqCustomDataCondition           ConditionType = ConditionType(sharedsaga.PqCustomDataCondition)
MonsterBookCountCondition       ConditionType = ConditionType(sharedsaga.MonsterBookCountCondition)
)
```

- [ ] **Step 2: Add `MonsterBookCountCondition` to the `SetType` accept list (line 124)**

```go
case JobCondition, MesoCondition, ..., PqCustomDataCondition, MonsterBookCountCondition:
    b.conditionType = ConditionType(condType)
```

- [ ] **Step 3: Create the REST client package** (mirror an existing per-service client like `services/atlas-query-aggregator/atlas.com/query-aggregator/quest/`)

```go
// monsterbook/rest.go
package monsterbook

type CollectionRestModel struct {
    TotalUniqueCards uint16 `json:"totalUniqueCards"`
}

func (r CollectionRestModel) GetName() string { return "monster-book" }
func (r CollectionRestModel) GetID() string   { return "" }
func (r *CollectionRestModel) SetID(_ string) error { return nil }
```

```go
// monsterbook/requests.go — adapt from quest/requests.go pattern
```

```go
// monsterbook/processor.go
package monsterbook

type Processor interface {
    GetTotalUniqueCards(characterId uint32) (uint16, error)
}
// + impl wired to the REST client
```

- [ ] **Step 4: Add `MonsterBookCountCondition` case in `Condition.Evaluate` (line 380+)**

The standard `Evaluate` reads `actualValue` from the character model. Monster-book data is *not* on the character model — it requires a REST call. Mirror the `QuestStatusCondition` / `QuestProgressCondition` "requires ValidationContext" pattern.

Open `validation/model.go` and search for where `ValidationContext` is declared. There is an evaluator pathway that takes the context (look around lines 600–800) where the `QuestStatusCondition` *is* evaluated against a `quest.Processor`. Add a parallel `MonsterBookCountCondition` case there, reading `totalUniqueCards` from the `monsterbook.Processor` injected into the validation context.

- [ ] **Step 5: Wire `monsterbook.Processor` into the validation context constructor**

Find the place where `quest.Processor` and friends are injected into the validation context (likely `validation/processor.go` or wherever the dispatcher entrypoint lives) and add the new processor next to it.

- [ ] **Step 6: Add accept-pattern to `rest.go` line 291** (`case LevelCondition, RebornsCondition, …`) so REST-side condition descriptors include `MonsterBookCountCondition`.

- [ ] **Step 7: Add tests for the new condition** (mirror `TestQuestStatusCondition…` patterns).

- [ ] **Step 8: Build + test**

```bash
cd services/atlas-query-aggregator/atlas.com/query-aggregator && go build ./... && go test ./...
```

- [ ] **Step 9: Commit**

```bash
git add services/atlas-query-aggregator/atlas.com/query-aggregator/{validation,monsterbook}
git commit -m "feat(query-aggregator): monsterBookCount condition (REST to atlas-monster-book)"
```

---

### Task 35: atlas-quest — emit the condition from quest definitions

**Files:**
- Modify: `services/atlas-quest/atlas.com/quest/data/validation/model.go`
- Modify: `services/atlas-quest/atlas.com/quest/data/validation/processor.go` (lines 55, 208 per design)

- [ ] **Step 1: Add the constant in `validation/model.go`**

```go
const (
    LevelCondition       = "level"
    JobCondition         = "jobId"
    FameCondition        = "fame"
    MesoCondition        = "meso"
    ItemCondition        = "item"
    QuestStatusCondition = "questStatus"
    SkillCondition       = "skillLevel"
    MonsterBookCountCondition = "monsterBookCount"
)
```

- [ ] **Step 2: Add a builder branch in `processor.go`'s `buildStartConditions` and `ValidateEndRequirements`**

These two functions translate quest WZ-derived requirement records into `ConditionInput`. Find the `case` (or equivalent dispatch) for an existing requirement type like `FameCondition` and add a parallel `MonsterBookCountCondition` branch:

```go
case "monster_book_count": // or whatever the WZ requirement key is in this codebase
    out = append(out, ConditionInput{
        Type:     MonsterBookCountCondition,
        Operator: ">=",
        Value:    requiredCards,
    })
```

> **Engineer note:** the actual key string in WZ-derived data depends on the quest data parser. Search for how `"fame"` is matched to `FameCondition` in this file and use the same lookup mechanism.

- [ ] **Step 3: Add tests** in `processor_test.go` exercising a quest definition with a `monster_book_count` requirement. Confirm the generated `ConditionInput` matches expectations.

- [ ] **Step 4: Build + test**

```bash
cd services/atlas-quest/atlas.com/quest && go build ./... && go test ./...
```

- [ ] **Step 5: Commit**

```bash
git add services/atlas-quest/atlas.com/quest/data/validation
git commit -m "feat(quest): emit monsterBookCount condition from requirements"
```

---

### Task 36: End-to-end build verification across services

**Files:** none

- [ ] **Step 1: Build atlas-monster-book, atlas-inventory, atlas-consumables, atlas-channel, atlas-quest, atlas-query-aggregator**

```bash
for svc in atlas-monster-book atlas-inventory atlas-consumables atlas-channel atlas-quest atlas-query-aggregator; do
  echo "=== $svc ==="
  (cd services/$svc/atlas.com/* && go build ./...) || exit 1
done
```

Expected: all green.

- [ ] **Step 2: Run all tests for the same set**

```bash
for svc in atlas-monster-book atlas-inventory atlas-consumables atlas-channel atlas-quest atlas-query-aggregator; do
  echo "=== $svc ==="
  (cd services/$svc/atlas.com/* && go test ./...) || exit 1
done
```

Expected: all green.

- [ ] **Step 3: No commit (verification only). Report any breakages and loop back.**

---

## Phase G — atlas-ui Monster Book widget

### Task 37: API client service

**Files:**
- Create: `services/atlas-ui/src/services/api/monster-book.service.ts`

Mirror an existing service like `services/atlas-ui/src/services/api/quest-status.service.ts` for shape and JSON:API parsing.

- [ ] **Step 1: Read the existing service template** to confirm tenant header propagation, base URL config, and JSON:API deserialization conventions.

- [ ] **Step 2: Write the service**

```ts
import { apiClient } from '@/lib/api-client';
import type { MonsterBookCollection, MonsterBookCard } from '@/types/monster-book';

const monsterBookServiceFactory = () => ({
  getCollection: async (characterId: number): Promise<MonsterBookCollection> => {
    const res = await apiClient.get(`/api/monster-book/characters/${characterId}/monster-book`);
    return parseCollection(res.data);
  },
  listCards: async (characterId: number, opts?: { offset?: number; limit?: number; isSpecial?: boolean }): Promise<MonsterBookCard[]> => {
    const params = new URLSearchParams();
    if (opts?.offset != null) params.set('page[offset]', String(opts.offset));
    if (opts?.limit != null) params.set('page[limit]', String(opts.limit));
    if (opts?.isSpecial != null) params.set('filter[isSpecial]', String(opts.isSpecial));
    const res = await apiClient.get(`/api/monster-book/characters/${characterId}/monster-book/cards?${params}`);
    return parseCards(res.data);
  },
});
export const monsterBookService = monsterBookServiceFactory();

function parseCollection(payload: any): MonsterBookCollection { /* JSON:API → flat */ }
function parseCards(payload: any): MonsterBookCard[] { /* JSON:API → flat */ }
```

> **Engineer note:** the `apiClient` import path and JSON:API parser helper differ across this repo's services — open a peer service file in the same directory for the canonical idiom. Use the project's existing JSON:API parser (often a shared util) rather than hand-rolling.

- [ ] **Step 3: Add the type definitions**

Create `services/atlas-ui/src/types/monster-book.ts`:

```ts
export interface MonsterBookCollection {
  characterId: number;
  bookLevel: number;
  normalCount: number;
  specialCount: number;
  totalUniqueCards: number;
  coverCardId: number;
  expBonusPercent: number;
}

export interface MonsterBookCard {
  cardId: number;
  level: number;
  isSpecial: boolean;
  firstAcquiredAt: string;
}
```

- [ ] **Step 4: Commit**

```bash
git add services/atlas-ui/src/services/api/monster-book.service.ts services/atlas-ui/src/types/monster-book.ts
git commit -m "feat(ui): monster-book API service"
```

---

### Task 38: Widget component

**Files:**
- Create: `services/atlas-ui/src/components/features/characters/MonsterBookWidget.tsx`
- Test: `services/atlas-ui/src/components/features/characters/__tests__/MonsterBookWidget.test.tsx`

- [ ] **Step 1: Write a Vitest test for the widget** (mirror an existing test like `SkillsSection.test.tsx`)

```tsx
import { render, screen } from '@testing-library/react';
import { MonsterBookWidget } from '../MonsterBookWidget';
// ... wrap with QueryClientProvider, mock services
test('renders cover card and book level', async () => {
  // ...
});
test('renders empty state when no collection exists', async () => {
  // ...
});
```

- [ ] **Step 2: Run test (FAIL)**

```bash
cd services/atlas-ui && npm test -- MonsterBookWidget
```

- [ ] **Step 3: Write the widget**

```tsx
import { useQuery, useInfiniteQuery } from '@tanstack/react-query';
import { monsterBookService } from '@/services/api/monster-book.service';
import { dataConsumablesService, dataMonstersService } from '@/services/api'; // adapt to actual exports

export function MonsterBookWidget({ characterId }: { characterId: number }) {
  const { data: collection } = useQuery({
    queryKey: ['monster-book', characterId],
    queryFn: () => monsterBookService.getCollection(characterId),
  });
  const cards = useInfiniteQuery({
    queryKey: ['monster-book', characterId, 'cards'],
    queryFn: ({ pageParam = 0 }) =>
      monsterBookService.listCards(characterId, { offset: pageParam, limit: 100 }),
    getNextPageParam: (lastPage, allPages) =>
      lastPage.length < 100 ? undefined : allPages.flat().length,
    initialPageParam: 0,
  });
  // Render: cover card image + name (resolve via consumable→monster), book level, counts, scroll list of cards.
  // For mob name resolution per card, batch the lookups by cardId via consumables service.
  return (
    <div className="rounded-lg border p-4">
      {/* header: cover card or placeholder */}
      {/* stats row: bookLevel | totalUniqueCards | normalCount | specialCount */}
      {/* paginated list */}
    </div>
  );
}
```

> **Engineer notes:**
> - Use Tailwind tokens consistent with existing character-detail widgets (`SkillWidget`, `EquipmentPanel`).
> - Mob name resolution: `GET /api/data/consumables/{cardId}` returns `monsterId`; `GET /api/data/monsters/{monsterId}` returns name. Use a per-card `useQuery({ enabled: !!cardId })` or batch via `useQueries`.
> - Empty state: when `collection.totalUniqueCards === 0`, show a "No cards collected yet" placeholder.

- [ ] **Step 4: Run test (PASS)**

- [ ] **Step 5: Commit**

```bash
git add services/atlas-ui/src/components/features/characters/MonsterBookWidget.tsx services/atlas-ui/src/components/features/characters/__tests__/MonsterBookWidget.test.tsx
git commit -m "feat(ui): MonsterBookWidget component"
```

---

### Task 39: Mount widget on character detail page

**Files:**
- Modify: the character detail page (find via `grep -nR "CharacterPageHeader" services/atlas-ui/src/pages/`)

- [ ] **Step 1: Locate the character detail page**

```bash
grep -lR "CharacterPageHeader\|InventoryGrid\|EquipmentPanel" services/atlas-ui/src/pages/ services/atlas-ui/src/components/
```

- [ ] **Step 2: Import and render the widget** in the same column / section as `SkillWidget`/`EquipmentPanel`. Use the `characterId` from the page's params.

```tsx
import { MonsterBookWidget } from '@/components/features/characters/MonsterBookWidget';

// inside the page component:
<MonsterBookWidget characterId={characterId} />
```

- [ ] **Step 3: Smoke-test in dev**

```bash
cd services/atlas-ui && npm run dev
```

Open the character detail page in a browser, confirm the widget renders for a character with at least one card and for a character with none.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-ui/src/pages/<character-detail-page-path>
git commit -m "feat(ui): mount MonsterBookWidget on character detail page"
```

---

### Task 40: Final verification + audit handoff

**Files:** none

- [ ] **Step 1: Build all six Go services + atlas-ui**

```bash
for svc in atlas-monster-book atlas-inventory atlas-consumables atlas-channel atlas-quest atlas-query-aggregator; do
  (cd services/$svc/atlas.com/* && go build ./...) || exit 1
done
(cd services/atlas-ui && npm run build) || exit 1
```

- [ ] **Step 2: Run all tests**

```bash
for svc in atlas-monster-book atlas-inventory atlas-consumables atlas-channel atlas-quest atlas-query-aggregator; do
  (cd services/$svc/atlas.com/* && go test ./...) || exit 1
done
(cd services/atlas-ui && npm test) || exit 1
```

- [ ] **Step 3: Build all relevant Docker images**

```bash
for svc in atlas-monster-book atlas-inventory atlas-consumables atlas-channel atlas-quest atlas-query-aggregator; do
  docker build -f services/$svc/Dockerfile -t $svc:plan-test . || exit 1
done
```

- [ ] **Step 4: Run code review**

Invoke `superpowers:requesting-code-review`. It dispatches the relevant subset of reviewer agents (plan-adherence, backend-guidelines, frontend-guidelines) and writes findings to `docs/tasks/task-056-monster-book/audit.md`.

- [ ] **Step 5: Address review feedback** (subagent-driven follow-up tasks per audit findings).

- [ ] **Step 6: Manual end-to-end smoke test (PRD acceptance criteria)**

Walk through the PRD §10 acceptance checklist by hand against a running stack:

1. Pick up a card item → row in `monster_book_cards` at level 1, `monster_book_collections` row, packets observed (`MonsterBookSetCard`, `EffectSimple`, `EffectSimpleForeign`), no inventory row.
2. Pick up the same card again → level increments to 2; level 5 → flag=0 path.
3. Send `0x39` with valid owned cardId → cover updates; with `0` → cleared; with unowned id → rejected.
4. Quest with `monsterBookCount: N` → gates correctly.
5. Delete character → all monster-book rows removed.
6. Tenant isolation → same-id character in tenant B is not visible from tenant A.
7. atlas-ui character detail page shows widget correctly.

- [ ] **Step 7: PR-ready summary commit**

If any cleanup is needed:

```bash
git add -p
git commit -m "chore(monster-book): final cleanups before PR"
```

---

## Operations handoff

After merge, tenant operators must add three opcode mappings to atlas-tenants per tenant:

| Opcode | Direction | Name |
|---|---|---|
| `0x39` | Recv | `MonsterBookCover` |
| `0x53` | Send | `MonsterBookSetCard` |
| `0x54` | Send | `MonsterBookSetCover` |

Until configured, the channel will log warnings about unmapped writers but will not crash. The Monster Book service still works at the REST/Kafka level; only client packets are unwired.

WZ data: Cards (item ids `2380000`–`2389999`) must have `consumeOnPickup=true` set in their consumable WZ entries. Confirm during deployment that no other item type accidentally has the flag set (per design §12 risk note).

---

## Self-review notes

- **Spec coverage:** every PRD §10 acceptance criterion maps to a task — service exists (Tasks 1, 15–16); card pickup creates row + packets (Tasks 7, 10, 12, 19, 21–22, 24, 30); levelling cap (Task 7); cover ops via `0x39` (Tasks 11, 12, 26); book level / EXP bonus formula (Task 11); EXP-distribution wiring is zero-change in atlas-channel (covered by reuse of existing topic in Task 11); quest condition (Tasks 33–35); character delete cascade (Task 13); tenant isolation is enforced by `tenant.MustFromContext(ctx)` in every processor; UI widget (Tasks 37–39); DOM-/FE- checklists (Task 40).
- **Type consistency:** `card.Model.CardId() uint32` matches `card.RestModel.CardId uint32` matches `monsterbook.CardPickedUpBody.CardId uint32`. `collection.Model.CoverCardId() uint32` matches `monsterbook.CoverChangedBody.CoverCardId uint32` matches `mbcb.SetCover.CardId uint32`. `bookLevel` and `expBonusPercent` are `uint16` everywhere (Model getters, RestModel fields, `StatsChangedBody`). The `EXPERIENCE_DISTRIBUTION` `Amount` field is `int32` per the existing atlas-channel struct — Task 11 casts `uint16 → int32` explicitly.
- **Placeholder scan:** none of the steps say "TODO" / "fill in details". Engineer notes call out specific verification points (writer API names, JSON envelope shape) where the existing peer file is the canonical source — these are explicit lookups, not deferred work.
- **Bite-sized:** every task ends in a commit and most have 4–7 steps. Tasks 11, 14, 30, 34, 38 are the largest single tasks (multi-file emissions); each is still scoped to one logical unit and can be picked up by a fresh subagent.
