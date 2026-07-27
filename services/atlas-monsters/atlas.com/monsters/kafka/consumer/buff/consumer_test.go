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
