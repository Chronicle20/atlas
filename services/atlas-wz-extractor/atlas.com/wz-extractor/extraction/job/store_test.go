package job

import (
	"context"
	"errors"
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

func TestStore_MarkUnitRunning_FirstTime(t *testing.T) {
	ctx := context.Background()
	c := newTestClient(t)
	s := NewStore(c)
	seedJob(t, ctx, s, "j3", []string{"Map.wz"})

	claimed, err := s.MarkUnitRunning(ctx, "j3", "Map.wz")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !claimed {
		t.Fatalf("expected claimed=true on first transition")
	}
}

func TestStore_MarkUnitRunning_AlreadyTerminal(t *testing.T) {
	ctx := context.Background()
	c := newTestClient(t)
	s := NewStore(c)
	seedJob(t, ctx, s, "j4", []string{"Map.wz"})

	// Manually set the unit to terminal.
	raw, _ := unitToJSON(NewUnitBuilder().SetWzFile("Map.wz").SetStatus(UnitSucceeded).Build())
	if err := c.HSet(ctx, unitsKey("j4"), "Map.wz", raw).Err(); err != nil {
		t.Fatalf("seed terminal: %v", err)
	}

	claimed, err := s.MarkUnitRunning(ctx, "j4", "Map.wz")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if claimed {
		t.Fatalf("expected claimed=false on already-terminal unit (redelivery)")
	}
}

// helper used by store tests
func seedJob(t *testing.T, ctx context.Context, s Store, id string, files []string) {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Second)
	j := NewJobBuilder().SetId(id).SetTenantId("t").SetRegion("GMS").
		SetMajorVersion(83).SetMinorVersion(1).
		SetStatus(JobRunning).
		SetUnitsTotal(len(files)).
		SetCreatedAt(now).SetUpdatedAt(now).Build()
	units := make([]Unit, 0, len(files))
	for _, f := range files {
		units = append(units, NewUnitBuilder().SetWzFile(f).SetStatus(UnitPending).Build())
	}
	if err := s.Create(ctx, j, units, 3600); err != nil {
		t.Fatalf("seed Create: %v", err)
	}
}

func TestStore_FinalizeUnit_Succeeded(t *testing.T) {
	ctx := context.Background()
	c := newTestClient(t)
	s := NewStore(c)
	seedJob(t, ctx, s, "j5", []string{"Map.wz", "Mob.wz"})
	if _, err := s.MarkUnitRunning(ctx, "j5", "Map.wz"); err != nil {
		t.Fatalf("MarkUnitRunning: %v", err)
	}

	cnt, err := s.FinalizeUnit(ctx, "j5", "Map.wz", UnitSucceeded, nil)
	if err != nil {
		t.Fatalf("FinalizeUnit: %v", err)
	}
	if cnt.UnitsCompleted != 1 || cnt.UnitsFailed != 0 || cnt.UnitsTotal != 2 || cnt.AllDone {
		t.Fatalf("counters: %+v", cnt)
	}

	got, units, _ := s.Get(ctx, "j5")
	if got.UnitsCompleted() != 1 {
		t.Fatalf("Get unitsCompleted: %d", got.UnitsCompleted())
	}
	for _, u := range units {
		if u.WzFile() == "Map.wz" && u.Status() != UnitSucceeded {
			t.Fatalf("unit not succeeded: %v", u.Status())
		}
	}
}

func TestStore_FinalizeUnit_Failed(t *testing.T) {
	ctx := context.Background()
	c := newTestClient(t)
	s := NewStore(c)
	seedJob(t, ctx, s, "j6", []string{"Map.wz"})
	if _, err := s.MarkUnitRunning(ctx, "j6", "Map.wz"); err != nil {
		t.Fatalf("MarkUnitRunning: %v", err)
	}

	cnt, err := s.FinalizeUnit(ctx, "j6", "Map.wz", UnitFailed, errors.New("open failed"))
	if err != nil {
		t.Fatalf("FinalizeUnit: %v", err)
	}
	if cnt.UnitsFailed != 1 || cnt.UnitsCompleted != 0 || !cnt.AllDone {
		t.Fatalf("counters: %+v", cnt)
	}
}

func TestStore_FinalizeUnit_RedeliveryNoOp(t *testing.T) {
	ctx := context.Background()
	c := newTestClient(t)
	s := NewStore(c)
	seedJob(t, ctx, s, "j7", []string{"Map.wz"})
	if _, err := s.MarkUnitRunning(ctx, "j7", "Map.wz"); err != nil {
		t.Fatalf("MarkUnitRunning: %v", err)
	}
	if _, err := s.FinalizeUnit(ctx, "j7", "Map.wz", UnitSucceeded, nil); err != nil {
		t.Fatalf("first finalize: %v", err)
	}

	// Redelivery: a second FinalizeUnit with the unit already terminal must
	// not double-increment counters.
	cnt, err := s.FinalizeUnit(ctx, "j7", "Map.wz", UnitSucceeded, nil)
	if err != nil {
		t.Fatalf("redelivery finalize: %v", err)
	}
	if cnt.UnitsCompleted != 1 {
		t.Fatalf("redelivery double-counted: %+v", cnt)
	}
}
