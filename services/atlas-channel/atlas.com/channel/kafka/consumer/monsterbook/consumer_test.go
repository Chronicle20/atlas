package monsterbook

import (
	mbmsg "atlas-channel/kafka/message/monsterbook"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"testing"

	channelconst "github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	mbcb "github.com/Chronicle20/atlas/libs/atlas-packet/character/clientbound/monsterbook"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// withRecordingSeams swaps every package-level seam for a recorder so
// tests can assert exactly which fan-out paths fired without standing
// up the session registry, the map processor, or a real socket writer.
// Returns counters/captures and a restore func.
type recorder struct {
	sessionLookups       int
	sessionPresent       bool
	setCardCalls         int
	lastSetCard          mbcb.SetCard
	cardGetEffectCalls   int
	foreignBroadcasts    int
	setCoverCalls        int
	lastSetCover         mbcb.SetCover
}

func withRecordingSeams(t *testing.T, sessionPresent bool) (*recorder, func()) {
	t.Helper()
	r := &recorder{sessionPresent: sessionPresent}
	origSession := sessionForCharacter
	origSetCard := announceSetCard
	origCardGet := announceCardGetEffect
	origForeign := broadcastCardGetEffectForeign
	origSetCover := announceSetCover

	sessionForCharacter = func(_ logrus.FieldLogger, _ context.Context, _ server.Model, _ uint32, f model.Operator[session.Model]) {
		r.sessionLookups++
		if !r.sessionPresent {
			return
		}
		_ = f(session.Model{})
	}
	announceSetCard = func(_ logrus.FieldLogger, _ context.Context, _ writer.Producer, _ session.Model, body mbcb.SetCard) error {
		r.setCardCalls++
		r.lastSetCard = body
		return nil
	}
	announceCardGetEffect = func(_ logrus.FieldLogger, _ context.Context, _ writer.Producer, _ session.Model) error {
		r.cardGetEffectCalls++
		return nil
	}
	broadcastCardGetEffectForeign = func(_ logrus.FieldLogger, _ context.Context, _ writer.Producer, _ session.Model) {
		r.foreignBroadcasts++
	}
	announceSetCover = func(_ logrus.FieldLogger, _ context.Context, _ writer.Producer, _ session.Model, body mbcb.SetCover) error {
		r.setCoverCalls++
		r.lastSetCover = body
		return nil
	}

	return r, func() {
		sessionForCharacter = origSession
		announceSetCard = origSetCard
		announceCardGetEffect = origCardGet
		broadcastCardGetEffectForeign = origForeign
		announceSetCover = origSetCover
	}
}

func newTestTenant(t *testing.T) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	return tm
}

func newTestServer(t *testing.T, tm tenant.Model) server.Model {
	t.Helper()
	ch := channelconst.NewModel(0, 1)
	return server.Register(tm, ch, "127.0.0.1", 8484)
}

// TestCardAdded_NotFull_FansOutSetCardPlusEffects exercises the most
// common path: the player draws a new card that is not yet at max
// level, so the consumer must emit the SetCard inventory mutation,
// the local CardGet effect, and the foreign broadcast to other
// players in the map.
func TestCardAdded_NotFull_FansOutSetCardPlusEffects(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)

	r, restore := withRecordingSeams(t, true)
	defer restore()

	h := handleCardAdded(sc, nil)
	h(logrus.New(), ctx, mbmsg.StatusEvent[mbmsg.CardAddedBody]{
		TenantId:    tm.Id(),
		CharacterId: 12345,
		EventId:     uuid.New(),
		Type:        mbmsg.StatusEventTypeCardAdded,
		Body: mbmsg.CardAddedBody{
			CardId:   2380000,
			NewLevel: 3,
			Full:     false,
		},
	})

	if r.setCardCalls != 1 {
		t.Errorf("SetCard calls = %d, want 1", r.setCardCalls)
	}
	if r.lastSetCard.CardId != 2380000 || r.lastSetCard.Level != 3 || !r.lastSetCard.Added {
		t.Errorf("SetCard body = %+v, want {2380000 3 true}", r.lastSetCard)
	}
	if r.cardGetEffectCalls != 1 {
		t.Errorf("CardGetEffect calls = %d, want 1", r.cardGetEffectCalls)
	}
	if r.foreignBroadcasts != 1 {
		t.Errorf("foreign broadcasts = %d, want 1", r.foreignBroadcasts)
	}
}

// TestCardAdded_Full_OnlySetCard guards the "card already at max" branch:
// the inventory mutation still fires, but the visual effects must not.
func TestCardAdded_Full_OnlySetCard(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)

	r, restore := withRecordingSeams(t, true)
	defer restore()

	h := handleCardAdded(sc, nil)
	h(logrus.New(), ctx, mbmsg.StatusEvent[mbmsg.CardAddedBody]{
		TenantId:    tm.Id(),
		CharacterId: 12345,
		Type:        mbmsg.StatusEventTypeCardAdded,
		Body: mbmsg.CardAddedBody{
			CardId:   2380000,
			NewLevel: 5,
			Full:     true,
		},
	})

	if r.setCardCalls != 1 {
		t.Errorf("SetCard calls = %d, want 1", r.setCardCalls)
	}
	if r.cardGetEffectCalls != 0 {
		t.Errorf("CardGetEffect calls = %d, want 0 when Full=true", r.cardGetEffectCalls)
	}
	if r.foreignBroadcasts != 0 {
		t.Errorf("foreign broadcasts = %d, want 0 when Full=true", r.foreignBroadcasts)
	}
}

// TestCardAdded_NoSession_NoWrites guards against firing packets at a
// character who is not on this channel. The session-lookup seam reports
// "not present" and the handler must not emit anything.
func TestCardAdded_NoSession_NoWrites(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)

	r, restore := withRecordingSeams(t, false)
	defer restore()

	h := handleCardAdded(sc, nil)
	h(logrus.New(), ctx, mbmsg.StatusEvent[mbmsg.CardAddedBody]{
		TenantId:    tm.Id(),
		CharacterId: 12345,
		Type:        mbmsg.StatusEventTypeCardAdded,
		Body:        mbmsg.CardAddedBody{CardId: 2380000, NewLevel: 1, Full: false},
	})

	if r.sessionLookups != 1 {
		t.Errorf("session lookups = %d, want 1", r.sessionLookups)
	}
	if r.setCardCalls != 0 || r.cardGetEffectCalls != 0 || r.foreignBroadcasts != 0 {
		t.Errorf("expected no writes when session absent: setCard=%d cardGet=%d foreign=%d",
			r.setCardCalls, r.cardGetEffectCalls, r.foreignBroadcasts)
	}
}

// TestCardAdded_WrongType_NoOp guards against firing for unrelated event
// types delivered on the same topic (COVER_CHANGED, STATS_CHANGED).
func TestCardAdded_WrongType_NoOp(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)

	r, restore := withRecordingSeams(t, true)
	defer restore()

	h := handleCardAdded(sc, nil)
	h(logrus.New(), ctx, mbmsg.StatusEvent[mbmsg.CardAddedBody]{
		TenantId:    tm.Id(),
		CharacterId: 12345,
		Type:        mbmsg.StatusEventTypeCoverChanged, // wrong type for this handler
		Body:        mbmsg.CardAddedBody{CardId: 2380000},
	})

	if r.sessionLookups != 0 {
		t.Errorf("session lookups = %d, want 0 on wrong type", r.sessionLookups)
	}
	if r.setCardCalls != 0 || r.cardGetEffectCalls != 0 || r.foreignBroadcasts != 0 {
		t.Errorf("expected no writes on wrong type: setCard=%d cardGet=%d foreign=%d",
			r.setCardCalls, r.cardGetEffectCalls, r.foreignBroadcasts)
	}
}

// TestCoverChanged_SendsSetCoverToOwner asserts the COVER_CHANGED handler
// emits a SetCover packet carrying the new cover card id to the owner's
// session.
func TestCoverChanged_SendsSetCoverToOwner(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)

	r, restore := withRecordingSeams(t, true)
	defer restore()

	h := handleCoverChanged(sc, nil)
	h(logrus.New(), ctx, mbmsg.StatusEvent[mbmsg.CoverChangedBody]{
		TenantId:    tm.Id(),
		CharacterId: 12345,
		Type:        mbmsg.StatusEventTypeCoverChanged,
		Body:        mbmsg.CoverChangedBody{CoverCardId: 2380000},
	})

	if r.setCoverCalls != 1 {
		t.Errorf("SetCover calls = %d, want 1", r.setCoverCalls)
	}
	if r.lastSetCover.CardId != 2380000 {
		t.Errorf("SetCover.CardId = %d, want 2380000", r.lastSetCover.CardId)
	}
}

// TestCoverChanged_WrongType_NoOp guards the COVER handler against
// firing for non-COVER_CHANGED events on the same topic.
func TestCoverChanged_WrongType_NoOp(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newTestServer(t, tm)

	r, restore := withRecordingSeams(t, true)
	defer restore()

	h := handleCoverChanged(sc, nil)
	h(logrus.New(), ctx, mbmsg.StatusEvent[mbmsg.CoverChangedBody]{
		TenantId:    tm.Id(),
		CharacterId: 12345,
		Type:        mbmsg.StatusEventTypeCardAdded, // wrong type for this handler
		Body:        mbmsg.CoverChangedBody{CoverCardId: 2380000},
	})

	if r.setCoverCalls != 0 {
		t.Errorf("SetCover calls = %d, want 0 on wrong type", r.setCoverCalls)
	}
}
