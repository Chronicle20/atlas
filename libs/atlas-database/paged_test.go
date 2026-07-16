package database_test

import (
	"testing"
	"time"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type pagedEntity struct {
	Id        uint32    `gorm:"primaryKey;autoIncrement"`
	TenantId  uuid.UUID `gorm:"not null"`
	Grp       string    // deliberately non-unique to exercise the PK tie-break
	CreatedAt time.Time
}

func (pagedEntity) TableName() string { return "paged_entities" }

func migrate(db *gorm.DB) error { return db.AutoMigrate(&pagedEntity{}) }

func seed(t *testing.T, db *gorm.DB, tenantId uuid.UUID, n int, grp string) {
	t.Helper()
	for i := 0; i < n; i++ {
		if err := db.Create(&pagedEntity{TenantId: tenantId, Grp: grp, CreatedAt: time.Unix(int64(1000+i), 0)}).Error; err != nil {
			t.Fatal(err)
		}
	}
}

func TestPagedQueryTenantScopeAgreement(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, migrate)
	t1, t2 := uuid.New(), uuid.New()
	seed(t, db.WithContext(databasetest.TenantContext(t1)), t1, 7, "a")
	seed(t, db.WithContext(databasetest.TenantContext(t2)), t2, 5, "a")

	scoped := db.WithContext(databasetest.TenantContext(t1))
	p, err := database.PagedQuery[pagedEntity](scoped, model.Page{Number: 1, Size: 3})()
	if err != nil {
		t.Fatal(err)
	}
	if p.Total != 7 {
		t.Fatalf("count leaked across tenants: total=%d want 7", p.Total)
	}
	if len(p.Items) != 3 {
		t.Fatalf("items=%d want 3", len(p.Items))
	}
	for _, e := range p.Items {
		if e.TenantId != t1 {
			t.Fatalf("row from wrong tenant: %+v", e)
		}
	}
}

func TestPagedQueryPagesArePartition(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, migrate)
	tid := uuid.New()
	// all rows share Grp so any ORDER BY grp alone is non-total; the PK
	// tie-break must make pages disjoint and exhaustive.
	seed(t, db.WithContext(databasetest.TenantContext(tid)), tid, 10, "same")

	seen := map[uint32]bool{}
	for n := 1; n <= 4; n++ {
		scoped := db.WithContext(databasetest.TenantContext(tid)).Order("grp")
		p, err := database.PagedQuery[pagedEntity](scoped, model.Page{Number: n, Size: 3})()
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range p.Items {
			if seen[e.Id] {
				t.Fatalf("row %d appeared on two pages", e.Id)
			}
			seen[e.Id] = true
		}
	}
	if len(seen) != 10 {
		t.Fatalf("pages missed rows: saw %d want 10", len(seen))
	}
}

func TestPagedQueryCallerOrderPreservedAndCountUnaffected(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, migrate)
	tid := uuid.New()
	seed(t, db.WithContext(databasetest.TenantContext(tid)), tid, 6, "a")

	scoped := db.WithContext(databasetest.TenantContext(tid)).Order("created_at desc")
	p, err := database.PagedQuery[pagedEntity](scoped, model.Page{Number: 1, Size: 6})()
	if err != nil {
		t.Fatal(err)
	}
	if p.Total != 6 {
		t.Fatalf("count with caller ORDER BY: total=%d want 6", p.Total)
	}
	for i := 1; i < len(p.Items); i++ {
		if p.Items[i].CreatedAt.After(p.Items[i-1].CreatedAt) {
			t.Fatalf("caller order not preserved at %d", i)
		}
	}
}

func TestPagedQueryOffsetLimit(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, migrate)
	tid := uuid.New()
	seed(t, db.WithContext(databasetest.TenantContext(tid)), tid, 9, "a")

	scoped := db.WithContext(databasetest.TenantContext(tid))
	p, err := database.PagedQuery[pagedEntity](scoped, model.Page{Number: 2, Size: 3})()
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Items) != 3 {
		t.Fatalf("items=%d want 3", len(p.Items))
	}
	// PK order: page 2 of size 3 = ids 4,5,6
	if p.Items[0].Id != 4 || p.Items[2].Id != 6 {
		t.Fatalf("wrong window: %+v", p.Items)
	}
}

func TestPagedQueryPastEndEmpty(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, migrate)
	tid := uuid.New()
	seed(t, db.WithContext(databasetest.TenantContext(tid)), tid, 2, "a")

	scoped := db.WithContext(databasetest.TenantContext(tid))
	p, err := database.PagedQuery[pagedEntity](scoped, model.Page{Number: 5, Size: 50})()
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Items) != 0 || p.Total != 2 {
		t.Fatalf("past-end: items=%d total=%d", len(p.Items), p.Total)
	}
}

func TestPagedQueryRejectsInvalidPage(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, migrate)
	if _, err := database.PagedQuery[pagedEntity](db, model.Page{Number: 0, Size: 10})(); err == nil {
		t.Fatal("expected error for page.Number=0")
	}
	if _, err := database.PagedQuery[pagedEntity](db, model.Page{Number: 1, Size: 0})(); err == nil {
		t.Fatal("expected error for page.Size=0")
	}
}

// noPKEntity deliberately has no primary key: no Id/ID field, no
// gorm:"primaryKey" tag, and no field GORM auto-promotes to a PK. It exists
// only to exercise the PrioritizedPrimaryField == nil branch of PagedQuery.
type noPKEntity struct {
	TenantId uuid.UUID `gorm:"not null"`
	Label    string
}

func (noPKEntity) TableName() string { return "no_pk_entities" }

func migrateNoPK(db *gorm.DB) error { return db.AutoMigrate(&noPKEntity{}) }

func TestPagedQueryRejectsEntityWithoutPrimaryKey(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, migrateNoPK)

	_, err := database.PagedQuery[noPKEntity](db, model.Page{Number: 1, Size: 10})()
	if err == nil {
		t.Fatal("expected error for entity with no primary key")
	}
}
