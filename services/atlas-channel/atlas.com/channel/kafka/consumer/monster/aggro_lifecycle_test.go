package monster

import (
	monster2 "atlas-channel/kafka/message/monster"
	"atlas-channel/monster"
	"atlas-channel/server"
	"atlas-channel/socket/writer"
	"context"
	"errors"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// TestAggroChangedReissuesControlWithAggroSet pins FR-6.1: an AGGRO_CHANGED
// event re-issues control state with the aggro flag threaded through, in
// both directions (grant and revoke).
//
// handleStatusEventAggroChanged does not go through the monsterGetByIdFn /
// announceFn seams — it calls monster.NewProcessor(l, ctx).GetById directly
// and announces via session.Announce directly (consumer.go:397-402), unlike
// handleStatusEventStartControl and spawnThenControlOperator. There is no
// REST fake standing up a monster.Model in this package's tests (see
// TestHandleStatusEventStartStopAggro_UpdateMirrorAggro above, which already
// exercises this handler the same way), so GetById fails fast in this
// environment and the handler returns after its first action — the mirror
// update — without reaching the announce. This test therefore uses the
// brief's documented fallback: assert the mirror side effect, which is
// unconditional and precedes the fetch.
func TestAggroChangedReissuesControlWithAggroSet(t *testing.T) {
	tests := []struct {
		name          string
		controllerHas bool
	}{
		{name: "aggro on", controllerHas: true},
		{name: "aggro off", controllerHas: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tm := newTestTenant(t)
			ctx := tenant.WithContext(context.Background(), tm)
			sc := newTestServer(t, tm)

			f := field.NewBuilder(0, 1, 100000000).Build()
			uniqueId := uint32(8001)
			if tt.controllerHas {
				uniqueId = uint32(8002)
			}
			monster.GetLiveMirror().Put(tm, uniqueId, monster.LiveEntry{Field: f, MonsterId: 100100, ControllerHasAggro: !tt.controllerHas})

			e := monster2.StatusEvent[monster2.StatusEventAggroChangedBody]{
				WorldId: 0, ChannelId: 1, MapId: 100000000, UniqueId: uniqueId,
				MonsterId: 100100, Type: monster2.EventStatusAggroChanged,
				Body: monster2.StatusEventAggroChangedBody{ControllerCharacterId: 7, ControllerHasAggro: tt.controllerHas},
			}
			handleStatusEventAggroChanged(sc, nil)(logrus.New(), ctx, e)

			got, ok := monster.GetLiveMirror().Lookup(tm, uniqueId)
			if !ok {
				t.Fatalf("expected mirror entry for monster [%d]", uniqueId)
			}
			if got.ControllerHasAggro != tt.controllerHas {
				t.Fatalf("ControllerHasAggro: want %v, got %v", tt.controllerHas, got.ControllerHasAggro)
			}
		})
	}
}

// TestStartControlCarriesAggroThroughHandover pins FR-6.2: a controller
// handover (START_CONTROL) carries the mob's aggro state through
// controlGrantFn truthfully rather than resetting it. A mob that is
// aggro'd stays aggro'd through a controller change.
func TestStartControlCarriesAggroThroughHandover(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)

	prevGet := monsterGetByIdFn
	monsterGetByIdFn = func(_ logrus.FieldLogger, _ context.Context, _ uint32) (monster.Model, error) {
		return monster.Model{}, errors.New("boom")
	}
	defer func() { monsterGetByIdFn = prevGet }()

	var recordedAggro bool
	var recordedCharacterId uint32
	var calls int
	prevGrant := controlGrantFn
	controlGrantFn = func(_ logrus.FieldLogger, _ context.Context, _ server.Model, _ writer.Producer, _ monster.Model, aggro bool, characterId uint32) error {
		calls++
		recordedAggro = aggro
		recordedCharacterId = characterId
		return nil
	}
	defer func() { controlGrantFn = prevGrant }()

	e := monster2.StatusEvent[monster2.StatusEventStartControlBody]{
		WorldId: 0, ChannelId: 1, MapId: 100000000, UniqueId: 8010,
		MonsterId: 100100, Type: monster2.EventStatusStartControl,
		Body: monster2.StatusEventStartControlBody{ActorId: 9, ControllerHasAggro: true},
	}
	handleStatusEventStartControl(sc, nil)(logrus.New(), ctx, e)

	if calls != 1 {
		t.Fatalf("expected exactly one control grant; got %d", calls)
	}
	if !recordedAggro {
		t.Fatalf("a mob that is aggro'd must stay aggro'd through a controller handover; got aggro=%v", recordedAggro)
	}
	if recordedCharacterId != 9 {
		t.Fatalf("characterId: want 9, got %d", recordedCharacterId)
	}
}
