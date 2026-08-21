package monster

import (
	monster2 "atlas-channel/kafka/message/monster"
	"atlas-channel/monster"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	channelconst "github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	monsterpkt "github.com/Chronicle20/atlas/libs/atlas-packet/monster/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// registerTestSession registers a session in the given tenant's registry
// (world 0 / channel 0, matching both this file's newTestServer0 and a
// session's zero-value field) and assigns it the given character id, using
// only the session package's public/test-helper API. No net.Conn is ever
// exercised: registration goes straight through AddSessionToRegistry, so no
// hello packet is written and no live connection is required.
func registerTestSession(t *testing.T, tm tenant.Model, characterId uint32) {
	t.Helper()
	sessionId := uuid.New()
	s := session.NewSession(sessionId, tm, 0, nil)
	session.AddSessionToRegistry(tm.Id(), s)
	t.Cleanup(func() { session.ClearRegistryForTenant(tm.Id()) })

	ctx := tenant.WithContext(context.Background(), tm)
	session.NewProcessor(logrus.New(), ctx).SetCharacterId(sessionId, characterId)
}

// newTestServer0 registers a server on world 0 / channel 0, matching the
// zero-value world/channel a session.Model carries until explicitly bound to
// a channel (which normally happens via Create, which needs a live
// net.Conn). Kept separate from newTestServer (world 0 / channel 1, used
// throughout consumer_test.go) so the aggro/start-control tests below can
// drive real sessions through the registry without standing up a
// connection.
func newTestServer0(t *testing.T, tm tenant.Model) server.Model {
	t.Helper()
	ch := channelconst.NewModel(0, 0)
	return server.NewProcessor(logrus.New(), context.Background()).Register(tm, ch, "127.0.0.1", 8484)
}

// TestAggroChangedReissuesControlWithAggroSet pins FR-6.1: an AGGRO_CHANGED
// event re-issues control state with the aggro flag threaded through, in
// both directions (grant and revoke), all the way to the packet body handed
// to the announce seam.
//
// handleStatusEventAggroChanged now routes its monster fetch and announce
// through the package's monsterGetByIdFn / announceFn seams (same
// convention as handleStatusEventStartControl), so this test seeds a
// deterministic monster.Model via monsterGetByIdFn, registers a real session
// for the controller character id, and lets the handler run its real
// session.NewProcessor(...).IfPresentByCharacterId(...) / announceFn path.
// The announceFn spy executes the captured packet.Encode body and compares
// the resulting bytes against writer.StartControlMonsterBody(m, aggro)
// computed directly - so a regression that hard-codes aggro at the announce
// site (e.g. always false) makes the byte comparison fail, not just the
// writer name.
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
			sc := newTestServer0(t, tm)
			l := logrus.New()

			f := field.NewBuilder(0, 0, 100000000).Build()
			uniqueId := uint32(8001)
			if tt.controllerHas {
				uniqueId = uint32(8002)
			}
			monster.GetLiveMirror().Put(tm, uniqueId, monster.LiveEntry{Field: f, MonsterId: 100100, ControllerHasAggro: !tt.controllerHas})

			m := monster.NewModelBuilder(uniqueId, f, 100100).SetControlCharacterId(7).MustBuild()
			prevGet := monsterGetByIdFn
			monsterGetByIdFn = func(_ logrus.FieldLogger, _ context.Context, _ uint32) (monster.Model, error) {
				return m, nil
			}
			defer func() { monsterGetByIdFn = prevGet }()

			var capturedWriters []string
			var capturedControlBody []byte
			prevAnnounce := announceFn
			announceFn = func(al logrus.FieldLogger, actx context.Context, _ writer.Producer, writerName string, body packet.Encode, _ session.Model) error {
				capturedWriters = append(capturedWriters, writerName)
				if writerName == monsterpkt.MonsterControlWriter {
					capturedControlBody = body(al, actx)(nil)
				}
				return nil
			}
			defer func() { announceFn = prevAnnounce }()

			registerTestSession(t, tm, 7)

			e := monster2.StatusEvent[monster2.StatusEventAggroChangedBody]{
				WorldId: 0, ChannelId: 0, MapId: 100000000, UniqueId: uniqueId,
				MonsterId: 100100, Type: monster2.EventStatusAggroChanged,
				Body: monster2.StatusEventAggroChangedBody{ControllerCharacterId: 7, ControllerHasAggro: tt.controllerHas},
			}
			handleStatusEventAggroChanged(sc, nil)(l, ctx, e)

			got, ok := monster.GetLiveMirror().Lookup(tm, uniqueId)
			if !ok {
				t.Fatalf("expected mirror entry for monster [%d]", uniqueId)
			}
			if got.ControllerHasAggro != tt.controllerHas {
				t.Fatalf("ControllerHasAggro: want %v, got %v", tt.controllerHas, got.ControllerHasAggro)
			}

			if len(capturedWriters) != 1 || capturedWriters[0] != monsterpkt.MonsterControlWriter {
				t.Fatalf("want exactly one %s announce; got %v", monsterpkt.MonsterControlWriter, capturedWriters)
			}

			wantBody := writer.StartControlMonsterBody(m, tt.controllerHas)(l, ctx)(nil)
			otherBody := writer.StartControlMonsterBody(m, !tt.controllerHas)(l, ctx)(nil)
			if string(wantBody) == string(otherBody) {
				t.Fatalf("test fixture cannot distinguish aggro encoding; StartControlMonsterBody produced identical bytes for both aggro states")
			}
			if string(capturedControlBody) != string(wantBody) {
				t.Fatalf("StartControlMonsterBody aggro mismatch: want the aggro=%v encoding, got bytes matching aggro=%v", tt.controllerHas, !tt.controllerHas)
			}
		})
	}
}

// TestStartControlCarriesAggroThroughHandover pins FR-6.2: a controller
// handover (START_CONTROL) carries the mob's aggro state through
// controlGrantFn truthfully rather than resetting it, all the way to the
// packet body handed to the announce seam.
//
// This drives the real (non-mocked) controlGrantFn -> spawnThenControlOperator
// -> announceFn chain against a registered session, and compares the
// captured Control body's bytes against writer.StartControlMonsterBody(m,
// aggro) computed directly - so a regression that drops or hard-codes the
// aggro bool anywhere along that chain makes the byte comparison fail.
func TestStartControlCarriesAggroThroughHandover(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer0(t, tm)
	l := logrus.New()

	prevGet := monsterGetByIdFn
	monsterGetByIdFn = func(_ logrus.FieldLogger, _ context.Context, _ uint32) (monster.Model, error) {
		return monster.Model{}, errors.New("boom")
	}
	defer func() { monsterGetByIdFn = prevGet }()

	var capturedWriters []string
	var capturedControlBody []byte
	prevAnnounce := announceFn
	announceFn = func(al logrus.FieldLogger, actx context.Context, _ writer.Producer, writerName string, body packet.Encode, _ session.Model) error {
		capturedWriters = append(capturedWriters, writerName)
		if writerName == monsterpkt.MonsterControlWriter {
			capturedControlBody = body(al, actx)(nil)
		}
		return nil
	}
	defer func() { announceFn = prevAnnounce }()

	registerTestSession(t, tm, 9)

	e := monster2.StatusEvent[monster2.StatusEventStartControlBody]{
		WorldId: 0, ChannelId: 0, MapId: 100000000, UniqueId: 8010,
		MonsterId: 100100, Type: monster2.EventStatusStartControl,
		Body: monster2.StatusEventStartControlBody{ActorId: 9, ControllerHasAggro: true},
	}
	handleStatusEventStartControl(sc, nil)(l, ctx, e)

	want := []string{monsterpkt.MonsterSpawnWriter, monsterpkt.MonsterControlWriter}
	if len(capturedWriters) != len(want) {
		t.Fatalf("want %d packets %v; got %d %v", len(want), want, len(capturedWriters), capturedWriters)
	}
	for i := range want {
		if capturedWriters[i] != want[i] {
			t.Fatalf("packet order mismatch: want %v, got %v", want, capturedWriters)
		}
	}

	// Reconstruct the same envelope fallback model handleStatusEventStartControl
	// builds when the REST fetch fails (consumer.go's fallback branch), so the
	// reference encoding is computed from the exact same monster.Model.
	f := field.NewBuilder(e.WorldId, e.ChannelId, e.MapId).SetInstance(e.Instance).Build()
	fallback := monster.NewModelBuilder(e.UniqueId, f, e.MonsterId).
		SetControlCharacterId(e.Body.ActorId).
		SetX(e.Body.X).SetY(e.Body.Y).
		SetStance(e.Body.Stance).
		SetFH(e.Body.FH).
		SetTeam(e.Body.Team).
		MustBuild()

	wantBody := writer.StartControlMonsterBody(fallback, true)(l, ctx)(nil)
	otherBody := writer.StartControlMonsterBody(fallback, false)(l, ctx)(nil)
	if string(wantBody) == string(otherBody) {
		t.Fatalf("test fixture cannot distinguish aggro encoding; StartControlMonsterBody produced identical bytes for both aggro states")
	}
	if string(capturedControlBody) != string(wantBody) {
		t.Fatalf("StartControlMonsterBody aggro mismatch: a mob that is aggro'd must stay aggro'd through a controller handover; got bytes matching aggro=false")
	}
}
