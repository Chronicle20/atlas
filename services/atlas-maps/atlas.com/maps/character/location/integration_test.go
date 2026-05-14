package location

import (
	"testing"

	"atlas-maps/data/map/info"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// These tests cover the integration scenarios enumerated in design.md §8 that
// can run against the in-memory infrastructure (sqlite + stub info.Processor).
// They drive Resolve + Set end-to-end, asserting the persisted character_locations
// row reflects the resolver's decision.
//
// Scenarios I2 (transit map), I7 (concurrent disconnect-during-channel-change),
// and I8 (atlas-maps unreachable) require multi-service or live infrastructure
// and live in integration_live_test.go behind `//go:build integration`.

// scenarioI1DisconnectOnKPQRoom — design.md §8 row I1.
// "Disconnect on KPQ room (103000800, forcedReturn=103000890) → Login lands at
// 103000890 instance=Nil. No PQ membership."
//
// In-memory translation: simulate the disconnect by Resolve()ing the character's
// current field (KPQ room with an instance), then Set()ing the result. Login
// reads the persisted row and lands wherever Set wrote.
func TestI1_DisconnectOnKPQRoom_RelocatesToForcedReturn(t *testing.T) {
	ctx := newCtxTenant(t)
	db := newTestDB(t)
	stub := &stubInfoProcessor{out: info.NewBuilder().
		SetId(_map.Id(103000800)).
		SetForcedReturnMapId(_map.Id(103000890)).
		Build()}
	p := newProcessorWithInfo(logrus.New(), ctx, db, stub)

	cur := field.NewBuilder(0, 1, _map.Id(103000800)).SetInstance(uuid.New()).Build()

	resolved, reason, err := p.Resolve(cur)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if reason != ReasonForcedReturn {
		t.Fatalf("reason = %s, want forced_return", reason)
	}
	if _, err := p.Set(uint32(101), resolved); err != nil {
		t.Fatalf("Set: %v", err)
	}

	got, err := p.GetById(uint32(101))
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	if got.MapId() != _map.Id(103000890) {
		t.Fatalf("MapId = %d, want 103000890", got.MapId())
	}
	if got.Instance() != uuid.Nil {
		t.Fatalf("Instance = %s, want Nil after forced relocation", got.Instance())
	}
	if got.ChannelId() != 1 {
		t.Fatalf("ChannelId = %d, want 1 (channel preserved on disconnect)", got.ChannelId())
	}
}

// scenarioI3DisconnectOnTimeLimitedMap — design.md §8 row I3.
// "Disconnect on time-limited map → Login lands at the WZ forcedReturn target.
// Timer cancelled."
//
// The resolver branch is identical to I1 (any non-sentinel forcedReturn relocates
// with instance=Nil). What differentiates I3 is the source map carries a
// non-zero timeLimit. We assert the resolver is unaffected by timeLimit and
// relocates to the target.
func TestI3_DisconnectOnTimeLimitedMap_RelocatesToForcedReturn(t *testing.T) {
	ctx := newCtxTenant(t)
	db := newTestDB(t)
	// Example time-limited map (e.g. an event map) with a forced-return target.
	stub := &stubInfoProcessor{out: info.NewBuilder().
		SetId(_map.Id(910000000)).
		SetTimeLimit(600).
		SetForcedReturnMapId(_map.Id(100000000)).
		Build()}
	p := newProcessorWithInfo(logrus.New(), ctx, db, stub)

	cur := field.NewBuilder(0, 1, _map.Id(910000000)).SetInstance(uuid.New()).Build()

	resolved, reason, err := p.Resolve(cur)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if reason != ReasonForcedReturn {
		t.Fatalf("reason = %s, want forced_return", reason)
	}
	if _, err := p.Set(uint32(103), resolved); err != nil {
		t.Fatalf("Set: %v", err)
	}

	got, err := p.GetById(uint32(103))
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	if got.MapId() != _map.Id(100000000) {
		t.Fatalf("MapId = %d, want 100000000", got.MapId())
	}
	if got.Instance() != uuid.Nil {
		t.Fatalf("Instance = %s, want Nil after forced relocation", got.Instance())
	}
}

// scenarioI4DisconnectOnHenesysHuntingGround — design.md §8 row I4.
// "Disconnect on Henesys Hunting Ground 1 (100020000, sentinel forcedReturn)
//  → Login lands at 100020000, same instance as before logout."
func TestI4_DisconnectOnHenesysHuntingGround_StaysPut(t *testing.T) {
	ctx := newCtxTenant(t)
	db := newTestDB(t)
	stub := &stubInfoProcessor{out: info.NewBuilder().
		SetId(_map.Id(100020000)).
		SetForcedReturnMapId(_map.EmptyMapId).
		Build()}
	p := newProcessorWithInfo(logrus.New(), ctx, db, stub)

	inst := uuid.New()
	cur := field.NewBuilder(0, 1, _map.Id(100020000)).SetInstance(inst).Build()

	resolved, reason, err := p.Resolve(cur)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if reason != ReasonStayPut {
		t.Fatalf("reason = %s, want stay_put", reason)
	}
	if _, err := p.Set(uint32(104), resolved); err != nil {
		t.Fatalf("Set: %v", err)
	}

	got, err := p.GetById(uint32(104))
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	if got.MapId() != _map.Id(100020000) {
		t.Fatalf("MapId = %d, want 100020000", got.MapId())
	}
	if got.Instance() != inst {
		t.Fatalf("Instance = %s, want %s (stay-put preserves instance)", got.Instance(), inst)
	}
}

// scenarioI5ChannelChangeOnKPQRoom — design.md §8 row I5.
// "Channel-change on KPQ room → Lands on new channel at 103000890 instance=Nil."
//
// In-memory translation: the player issues a channel-change while standing in
// the KPQ room on channel 1, target channel 2. Resolve runs on the *current*
// field (channel 1, instance=X), the resolver builds the relocated field
// preserving cur.ChannelId. After Resolve, the channel-change pipeline rebases
// the channel to the target before Set. We model that explicitly here.
func TestI5_ChannelChangeOnKPQRoom_RelocatesAndSwitchesChannel(t *testing.T) {
	ctx := newCtxTenant(t)
	db := newTestDB(t)
	stub := &stubInfoProcessor{out: info.NewBuilder().
		SetId(_map.Id(103000800)).
		SetForcedReturnMapId(_map.Id(103000890)).
		Build()}
	p := newProcessorWithInfo(logrus.New(), ctx, db, stub)

	cur := field.NewBuilder(0, 1, _map.Id(103000800)).SetInstance(uuid.New()).Build()

	resolved, reason, err := p.Resolve(cur)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if reason != ReasonForcedReturn {
		t.Fatalf("reason = %s, want forced_return", reason)
	}

	// Channel-change rebases the resolved field onto the target channel.
	const targetChannel channel.Id = 2
	rebased := field.NewBuilder(resolved.WorldId(), targetChannel, resolved.MapId()).
		SetInstance(resolved.Instance()).
		Build()

	if _, err := p.Set(uint32(105), rebased); err != nil {
		t.Fatalf("Set: %v", err)
	}

	got, err := p.GetById(uint32(105))
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	if got.MapId() != _map.Id(103000890) {
		t.Fatalf("MapId = %d, want 103000890", got.MapId())
	}
	if got.ChannelId() != targetChannel {
		t.Fatalf("ChannelId = %d, want %d", got.ChannelId(), targetChannel)
	}
	if got.Instance() != uuid.Nil {
		t.Fatalf("Instance = %s, want Nil after forced relocation", got.Instance())
	}
}

// scenarioI6ChannelChangeOnRegularMap — design.md §8 row I6.
// "Channel-change on regular map → Lands on new channel at same map, same instance."
func TestI6_ChannelChangeOnRegularMap_PreservesMapAndInstance(t *testing.T) {
	ctx := newCtxTenant(t)
	db := newTestDB(t)
	stub := &stubInfoProcessor{out: info.NewBuilder().
		SetId(_map.Id(100000000)).
		SetForcedReturnMapId(_map.EmptyMapId).
		Build()}
	p := newProcessorWithInfo(logrus.New(), ctx, db, stub)

	inst := uuid.New()
	cur := field.NewBuilder(0, 1, _map.Id(100000000)).SetInstance(inst).Build()

	resolved, reason, err := p.Resolve(cur)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if reason != ReasonStayPut {
		t.Fatalf("reason = %s, want stay_put", reason)
	}

	// Channel-change rebases onto the target channel.
	const targetChannel channel.Id = 3
	rebased := field.NewBuilder(resolved.WorldId(), targetChannel, resolved.MapId()).
		SetInstance(resolved.Instance()).
		Build()

	if _, err := p.Set(uint32(106), rebased); err != nil {
		t.Fatalf("Set: %v", err)
	}

	got, err := p.GetById(uint32(106))
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	if got.MapId() != _map.Id(100000000) {
		t.Fatalf("MapId = %d, want 100000000 (same map after channel-change)", got.MapId())
	}
	if got.ChannelId() != targetChannel {
		t.Fatalf("ChannelId = %d, want %d", got.ChannelId(), targetChannel)
	}
	if got.Instance() != inst {
		t.Fatalf("Instance = %s, want %s (instance preserved across channel-change on stay-put map)", got.Instance(), inst)
	}
}
