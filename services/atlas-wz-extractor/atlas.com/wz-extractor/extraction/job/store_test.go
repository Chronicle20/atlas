package job

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
)

func newTestClient(t *testing.T) *goredis.Client {
	t.Helper()
	mr := miniredis.RunT(t)
	return goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
}

func TestStore_CreateGetDelete(t *testing.T) {
	ctx := context.Background()
	c := newTestClient(t)
	s := NewStore(c)

	now := time.Now().UTC().Truncate(time.Second)
	j := NewJobBuilder().
		SetId("job-1").
		SetTenantId("tenant-1").
		SetRegion("GMS").
		SetMajorVersion(83).SetMinorVersion(1).
		SetStatus(JobPending).
		SetUnitsTotal(2).
		SetXmlOnly(false).SetImagesOnly(false).
		SetCreatedAt(now).SetUpdatedAt(now).
		Build()
	units := []Unit{
		NewUnitBuilder().SetWzFile("Map.wz").SetStatus(UnitPending).Build(),
		NewUnitBuilder().SetWzFile("Mob.wz").SetStatus(UnitPending).Build(),
	}

	if err := s.Create(ctx, j, units, 3600); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, gotUnits, err := s.Get(ctx, "job-1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Id() != "job-1" || got.UnitsTotal() != 2 || got.Status() != JobPending {
		t.Fatalf("Get returned: %+v", got)
	}
	if len(gotUnits) != 2 {
		t.Fatalf("expected 2 units, got %d", len(gotUnits))
	}

	if err := s.Delete(ctx, "job-1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, _, err := s.Get(ctx, "job-1"); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound after Delete, got %v", err)
	}
}

func TestStore_MarkJobRunning(t *testing.T) {
	ctx := context.Background()
	c := newTestClient(t)
	s := NewStore(c)

	now := time.Now().UTC().Truncate(time.Second)
	j := NewJobBuilder().SetId("j2").SetStatus(JobPending).
		SetUnitsTotal(1).SetCreatedAt(now).SetUpdatedAt(now).Build()
	if err := s.Create(ctx, j, []Unit{NewUnitBuilder().SetWzFile("Map.wz").SetStatus(UnitPending).Build()}, 3600); err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := s.MarkJobRunning(ctx, "j2"); err != nil {
		t.Fatalf("MarkJobRunning: %v", err)
	}

	got, _, err := s.Get(ctx, "j2")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Status() != JobRunning {
		t.Fatalf("status: got %s", got.Status())
	}
}
