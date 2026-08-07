package buff

import (
	"atlas-monsters/character/hidden"
	buff2 "atlas-monsters/kafka/message/buff"
	"context"
	"os"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// TestMain provisions a single miniredis-backed hidden.Registry for the whole
// package's test binary. hidden.InitRegistry is sync.Once-guarded (shared
// production singleton contract), so per-test Init calls would silently
// no-op after the first test and leave later tests bound to an already-
// closed miniredis connection — route through one shared instance instead.
func TestMain(m *testing.M) {
	mr, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	defer mr.Close()
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	hidden.InitRegistry(rc)
	os.Exit(m.Run())
}

// TestAppliedIgnoresNonSuperGmHideSources proves the APPLIED handler's
// SourceId filter: only SuperGmHide (9101004) mutates the hidden set. Dark
// Sight (RogueDarkSightId) and the absent-from-v83 GmHideId (9001004) must
// both pass through untouched — this pins the acceptance criterion that Dark
// Sight is unaffected by the GM-hide relinquish/re-elect feature.
func TestAppliedIgnoresNonSuperGmHideSources(t *testing.T) {
	t.Cleanup(func() { hidden.GetRegistry().Clear(context.Background()) })

	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	ctx := tenant.WithContext(context.Background(), ten)
	l, _ := test.NewNullLogger()

	for _, sourceId := range []int32{int32(skill.RogueDarkSightId), 9001004} {
		handleStatusEventApplied(l, ctx, buff2.StatusEvent[buff2.AppliedStatusEventBody]{
			WorldId: 0, CharacterId: 5, Type: buff2.EventStatusTypeBuffApplied,
			Body: buff2.AppliedStatusEventBody{SourceId: sourceId},
		})
	}

	ms, err := hidden.GetRegistry().MemberSet(context.Background(), ten)
	if err != nil {
		t.Fatalf("MemberSet: %v", err)
	}
	if len(ms) != 0 {
		t.Fatalf("non-SuperGmHide sources must not mutate the hidden set, got %v", ms)
	}
}

// TestIsSuperGmHideSource_VersionAware pins the task-187 correctness
// property directly on the resolver helper: at v0.48 SuperGmHide's wire id
// is 5101004 (NOT the v83-canonical 9101004), so a raw `==
// int32(skill.SuperGmHideId)` compare would silently never match a v0.48
// hide buff. Both v48 (wire 5101004) and v83 (wire 9101004) must resolve to
// the SuperGmHide identity, and an unrelated skill (Rogue Dark Sight) must
// not, under either tenant.
func TestIsSuperGmHideSource_VersionAware(t *testing.T) {
	ten83, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	ten48, err := tenant.Create(uuid.New(), "GMS", 48, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}

	if !isSuperGmHideSource(ten83, int32(skill.SuperGmHideId)) {
		t.Errorf("isSuperGmHideSource(v83, %d) = false, want true", skill.SuperGmHideId)
	}
	if !isSuperGmHideSource(ten48, 5101004) {
		t.Errorf("isSuperGmHideSource(v48, 5101004) = false, want true")
	}
	// The v83-canonical wire value means something else at v48 (or is
	// absent) and must not be misread as hide.
	if isSuperGmHideSource(ten48, int32(skill.SuperGmHideId)) {
		t.Errorf("isSuperGmHideSource(v48, %d) = true, want false", skill.SuperGmHideId)
	}
	if isSuperGmHideSource(ten83, int32(skill.RogueDarkSightId)) {
		t.Errorf("isSuperGmHideSource(v83, RogueDarkSightId) = true, want false")
	}
}

// TestAppliedV48WireMarksHidden proves the end-to-end bug this task fixes:
// under a v0.48 tenant, an APPLIED event sourced from wire 5101004 (v48's
// SuperGmHide wire id, NOT the v83-canonical 9101004) must still mark the
// character hidden. Before task-187, the handler's raw
// `== int32(skill.SuperGmHideId)` compare would silently never match this
// wire value at v48, leaving the hiding character's monsters uncontrolled.
func TestAppliedV48WireMarksHidden(t *testing.T) {
	t.Cleanup(func() { hidden.GetRegistry().Clear(context.Background()) })

	ten, err := tenant.Create(uuid.New(), "GMS", 48, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	ctx := tenant.WithContext(context.Background(), ten)
	l, _ := test.NewNullLogger()

	handleStatusEventApplied(l, ctx, buff2.StatusEvent[buff2.AppliedStatusEventBody]{
		WorldId: 0, CharacterId: 7, Type: buff2.EventStatusTypeBuffApplied,
		Body: buff2.AppliedStatusEventBody{SourceId: 5101004},
	})

	ms, err := hidden.GetRegistry().MemberSet(context.Background(), ten)
	if err != nil {
		t.Fatalf("MemberSet: %v", err)
	}
	if _, ok := ms[7]; !ok {
		t.Fatalf("v48 wire 5101004 APPLIED event must mark character 7 hidden, got %v", ms)
	}
}

// TestAppliedNonAppliedTypeIsIgnored verifies the Type guard: an event
// carrying the SuperGmHide SourceId but the wrong Type (defensive; APPLIED
// handler should only be invoked for APPLIED-type messages by the topic's
// producer, but the guard exists to keep the handler correct in isolation).
func TestAppliedNonAppliedTypeIsIgnored(t *testing.T) {
	t.Cleanup(func() { hidden.GetRegistry().Clear(context.Background()) })

	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	ctx := tenant.WithContext(context.Background(), ten)
	l, _ := test.NewNullLogger()

	handleStatusEventApplied(l, ctx, buff2.StatusEvent[buff2.AppliedStatusEventBody]{
		WorldId: 0, CharacterId: 5, Type: buff2.EventStatusTypeBuffExpired,
		Body: buff2.AppliedStatusEventBody{SourceId: int32(skill.SuperGmHideId)},
	})

	ms, err := hidden.GetRegistry().MemberSet(context.Background(), ten)
	if err != nil {
		t.Fatalf("MemberSet: %v", err)
	}
	if len(ms) != 0 {
		t.Fatalf("mismatched Type must not mutate the hidden set, got %v", ms)
	}
}
