package report

import (
	"context"
	"testing"

	report2 "atlas-ban/kafka/message/report"
	report3 "atlas-ban/report"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func setupDb(t *testing.T) *gorm.DB {
	l, _ := test.NewNullLogger()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("db: %v", err)
	}
	database.RegisterTenantCallbacks(l, db)
	if err := db.AutoMigrate(&report3.Entity{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestHandleCreateCommandIgnoresOtherTypes(t *testing.T) {
	db := setupDb(t)
	l, _ := test.NewNullLogger()
	tm, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), tm)

	h := handleCreateReportCommand(db)
	h(l, ctx, report2.Command[report2.CreateCommandBody]{
		Type: "DELETE",
		Body: report2.CreateCommandBody{Kind: report2.KindSue, ReporterId: 1, AccusedId: 2},
	})

	var count int64
	db.WithContext(ctx).Model(&report3.Entity{}).Count(&count)
	if count != 0 {
		t.Fatalf("expected no rows for ignored command type, got %d", count)
	}
}
