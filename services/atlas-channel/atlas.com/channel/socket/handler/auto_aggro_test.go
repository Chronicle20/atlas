package handler

import (
	"atlas-channel/monster"
	"atlas-channel/session"
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	testlog "github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	monstersb "github.com/Chronicle20/atlas/libs/atlas-packet/monster/serverbound"
	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// TestAutoAggroDecode proves the handler's decode step recovers every field
// from the wire (mirrors TestMonsterCatchItemUseDecode).
func TestAutoAggroDecode(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	encoded := monstersb.NewAutoAggro(0x07654321, 39).Encode(nil, ctx)(nil)

	req := request.Request(encoded)
	reader := request.NewRequestReader(&req, 0)

	var p monstersb.AutoAggro
	p.Decode(nil, ctx)(&reader, nil)

	if p.MobId() != 0x07654321 {
		t.Fatalf("MobId() = %#x, want %#x", p.MobId(), 0x07654321)
	}
	if p.Distance() != 39 {
		t.Fatalf("Distance() = %d, want 39", p.Distance())
	}
	if p.Operation() != monstersb.AutoAggroHandle {
		t.Fatalf("Operation() = %q, want %q", p.Operation(), monstersb.AutoAggroHandle)
	}
}

// newAutoAggroTestSession builds a session.Model for characterId in
// world 1 / channel 2 / map 100000000, plus the tenant it was created under
// (idiom: newTeleportRockUseTestSession in teleport_rock_use_test.go).
func newAutoAggroTestSession(t *testing.T, characterId uint32) (session.Model, tenant.Model, context.Context, func()) {
	t.Helper()
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	ctx := tenant.WithContext(context.Background(), ten)

	sessionId := uuid.New()
	s := session.NewSession(sessionId, ten, 0, nil)
	session.AddSessionToRegistry(ten.Id(), s)

	sp := session.NewProcessor(logrus.New(), ctx)
	sp.SetCharacterId(sessionId, characterId)
	f := field.NewBuilder(world.Id(1), channel.Id(2), _map.Id(100000000)).Build()
	updated := sp.SetField(sessionId, f)

	return updated, ten, ctx, func() { session.ClearRegistryForTenant(ten.Id()) }
}

// autoAggroEmitCall records a single autoAggroEmitFn invocation.
type autoAggroEmitCall struct {
	monsterId   uint32
	characterId uint32
}

// installAutoAggroSeams swaps autoAggroMirrorLookupFn and autoAggroEmitFn for
// the test and returns the recorded emit calls plus a restore func.
func installAutoAggroSeams(entry monster.LiveEntry, present bool) (*[]autoAggroEmitCall, func()) {
	origLookup := autoAggroMirrorLookupFn
	origEmit := autoAggroEmitFn
	calls := &[]autoAggroEmitCall{}

	autoAggroMirrorLookupFn = func(_ tenant.Model, _ uint32) (monster.LiveEntry, bool) {
		return entry, present
	}
	autoAggroEmitFn = func(_ logrus.FieldLogger, _ context.Context, _ field.Model, monsterId uint32, characterId uint32) error {
		*calls = append(*calls, autoAggroEmitCall{monsterId: monsterId, characterId: characterId})
		return nil
	}

	return calls, func() {
		autoAggroMirrorLookupFn = origLookup
		autoAggroEmitFn = origEmit
	}
}

func TestAutoAggroAdmission(t *testing.T) {
	tests := []struct {
		name       string
		distance   uint32
		present    bool
		sameField  bool
		expectEmit bool
	}{
		{
			name:       "valid claim forwards",
			distance:   39,
			present:    true,
			sameField:  true,
			expectEmit: true,
		},
		{
			name:       "score above threshold is dropped",
			distance:   41,
			present:    true,
			sameField:  true,
			expectEmit: false,
		},
		{
			name:       "score at threshold forwards",
			distance:   40,
			present:    true,
			sameField:  true,
			expectEmit: true,
		},
		{
			name:       "mob absent from mirror is dropped",
			distance:   39,
			present:    false,
			sameField:  true,
			expectEmit: false,
		},
		{
			name:       "mob in another field is dropped",
			distance:   39,
			present:    true,
			sameField:  false,
			expectEmit: false,
		},
	}

	l, _ := testlog.NewNullLogger()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s, ten, ctx, cleanup := newAutoAggroTestSession(t, 7)
			defer cleanup()
			defer monster.GetAutoAggroGate().EvictTenant(ten.Id())

			entryField := s.Field()
			if !tc.sameField {
				entryField = field.NewBuilder(s.Field().WorldId(), s.Field().ChannelId(), _map.Id(104000000)).Build()
			}
			entry := monster.LiveEntry{Field: entryField, ControllerHasAggro: false}
			calls, restore := installAutoAggroSeams(entry, tc.present)
			defer restore()

			encoded := monstersb.NewAutoAggro(0x07654321, tc.distance).Encode(nil, ctx)(nil)
			req := request.Request(encoded)
			reader := request.NewRequestReader(&req, 0)

			fn := AutoAggroHandleFunc(l, ctx, nil)
			fn(s, &reader, nil)

			if tc.expectEmit {
				if len(*calls) != 1 {
					t.Fatalf("want 1 emit, got %d: %+v", len(*calls), *calls)
				}
				got := (*calls)[0]
				if got.monsterId != 0x07654321 || got.characterId != 7 {
					t.Fatalf("emit = %+v, want monsterId 0x07654321, characterId 7", got)
				}
			} else {
				if len(*calls) != 0 {
					t.Fatalf("want 0 emits, got %d: %+v", len(*calls), *calls)
				}
			}
		})
	}
}

// TestAutoAggroAdmissionRateGateClosed proves the rate gate blocks a second
// back-to-back claim for the same character/mob: exactly one emit across two
// handler invocations run with the same session and packet.
func TestAutoAggroAdmissionRateGateClosed(t *testing.T) {
	s, ten, ctx, cleanup := newAutoAggroTestSession(t, 7)
	defer cleanup()
	defer monster.GetAutoAggroGate().EvictTenant(ten.Id())

	entry := monster.LiveEntry{Field: s.Field(), ControllerHasAggro: false}
	calls, restore := installAutoAggroSeams(entry, true)
	defer restore()

	l, _ := testlog.NewNullLogger()
	encoded := monstersb.NewAutoAggro(0x07654321, 39).Encode(nil, ctx)(nil)

	fn := AutoAggroHandleFunc(l, ctx, nil)

	req1 := request.Request(encoded)
	reader1 := request.NewRequestReader(&req1, 0)
	fn(s, &reader1, nil)

	req2 := request.Request(encoded)
	reader2 := request.NewRequestReader(&req2, 0)
	fn(s, &reader2, nil)

	if len(*calls) != 1 {
		t.Fatalf("want exactly 1 emit across both calls, got %d: %+v", len(*calls), *calls)
	}
}
