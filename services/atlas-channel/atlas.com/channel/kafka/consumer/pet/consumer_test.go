package pet

import (
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	testlog "github.com/sirupsen/logrus/hooks/test"

	pet2 "atlas-channel/kafka/message/pet"

	channelconst "github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	socketwriter "github.com/Chronicle20/atlas/libs/atlas-socket/writer"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func nullLogger() *logrus.Logger {
	l, _ := testlog.NewNullLogger()
	return l
}

func newTestTenant(t *testing.T) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	return tm
}

// newZeroFieldTestServer registers a server whose world/channel (0, 0) match
// session.NewSession's un-set default field. session.Processor exposes no
// public setter for a session's worldId/channelId, so a directly-registered
// test session only survives the world/channel filters in
// GetByCharacterId / IfPresentByCharacterId against this server.
func newZeroFieldTestServer(t *testing.T, tm tenant.Model) server.Model {
	t.Helper()
	ch := channelconst.NewModel(0, 0)
	return server.NewProcessor(nullLogger(), context.Background()).Register(tm, ch, "127.0.0.1", 8484)
}

// refreshCall captures one invocation of the petAssetRefresher seam.
type refreshCall struct {
	petId   uint32
	ownerId uint32
}

// withRecordingRefresher swaps the package-level petAssetRefresher seam for a
// recorder. The real seam reaches for the character REST service (to re-fetch
// the cash compartment with PetAssetEnrichmentDecorator), which a unit test has
// no backend for; the assertion here is that the handler CALLS it, which is
// exactly the thing that was missing.
func withRecordingRefresher(t *testing.T) *[]refreshCall {
	t.Helper()
	var recorded []refreshCall
	orig := petAssetRefresher
	t.Cleanup(func() { petAssetRefresher = orig })
	petAssetRefresher = func(_ logrus.FieldLogger, _ context.Context, _ writer.Producer, _ server.Model, petId uint32, ownerId uint32) {
		recorded = append(recorded, refreshCall{petId: petId, ownerId: ownerId})
	}
	return &recorded
}

// noWriterProducer resolves no writer at all. handleNameChanged's map-wide
// PetNameChanged broadcast is a separate concern from the inventory refresh
// under test, and a nil Producer nil-derefs inside session.Announce's
// background goroutine; this short-circuits that leg cleanly instead.
func noWriterProducer() writer.Producer {
	return func(_ string) (socketwriter.BodyFunc, error) {
		return nil, errors.New("writer not found")
	}
}

// registerSession puts a character-bound session in the registry so the
// handlers' session lookups resolve.
func registerSession(t *testing.T, tm tenant.Model, ctx context.Context, characterId uint32) {
	t.Helper()
	sessionId := uuid.New()
	s := session.NewSession(sessionId, tm, 0, nil)
	session.AddSessionToRegistry(tm.Id(), s)
	t.Cleanup(func() { session.ClearRegistryForTenant(tm.Id()) })
	_ = session.NewProcessor(nullLogger(), ctx).SetCharacterId(sessionId, characterId)
}

// TestHandleNameChanged_RefreshesInventoryAsset is a regression guard for a
// half-applied rename. handleNameChanged originally broadcast only
// PetNameChanged, which repaints the name tag over the SPAWNED pet. The
// inventory slot renders from a different client-side record —
// GW_ItemSlotPet.sPetName — that no pet packet writes, so the item kept the
// OLD name until an unrelated full inventory re-send (entering or leaving the
// cash shop) happened to correct it. Every sibling status handler
// (closeness/fullness/level/flag) already re-announces the cash asset; name
// change was the one that did not.
func TestHandleNameChanged_RefreshesInventoryAsset(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newZeroFieldTestServer(t, tm)

	const ownerId = uint32(6001)
	const petId = uint32(77)
	registerSession(t, tm, ctx, ownerId)
	calls := withRecordingRefresher(t)

	h := handleNameChanged(sc, noWriterProducer())
	h(nullLogger(), ctx, pet2.StatusEvent[pet2.NameChangedStatusEventBody]{
		PetId:   petId,
		OwnerId: ownerId,
		Type:    pet2.StatusEventTypeNameChanged,
		Body: pet2.NameChangedStatusEventBody{
			Slot:          0,
			Name:          "Renamed",
			PreviousName:  "Original",
			TransactionId: uuid.New(),
		},
	})

	if len(*calls) != 1 {
		t.Fatalf("inventory asset refresh: got %d calls, want 1 — the renamed pet item stays stale in the inventory without it", len(*calls))
	}
	if (*calls)[0].petId != petId || (*calls)[0].ownerId != ownerId {
		t.Fatalf("refresh targeted pet %d owner %d, want pet %d owner %d", (*calls)[0].petId, (*calls)[0].ownerId, petId, ownerId)
	}
}

// A status event of the wrong type shares the topic with NAME_CHANGED and must
// not trigger a refresh.
func TestHandleNameChanged_WrongType_NoRefresh(t *testing.T) {
	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newZeroFieldTestServer(t, tm)

	registerSession(t, tm, ctx, 6002)
	calls := withRecordingRefresher(t)

	h := handleNameChanged(sc, noWriterProducer())
	h(nullLogger(), ctx, pet2.StatusEvent[pet2.NameChangedStatusEventBody]{
		PetId:   77,
		OwnerId: 6002,
		Type:    pet2.StatusEventTypeSlotChanged,
	})

	if len(*calls) != 0 {
		t.Fatalf("wrong event type: want no refresh, got %d", len(*calls))
	}
}

// A NAME_CHANGED for another tenant must not reach this channel's sessions.
func TestHandleNameChanged_ForeignTenant_NoRefresh(t *testing.T) {
	tm := newTestTenant(t)
	other := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), other)
	sc := newZeroFieldTestServer(t, tm)

	registerSession(t, tm, tenant.WithContext(context.Background(), tm), 6003)
	calls := withRecordingRefresher(t)

	h := handleNameChanged(sc, noWriterProducer())
	h(nullLogger(), ctx, pet2.StatusEvent[pet2.NameChangedStatusEventBody]{
		PetId:   77,
		OwnerId: 6003,
		Type:    pet2.StatusEventTypeNameChanged,
		Body:    pet2.NameChangedStatusEventBody{Name: "Renamed"},
	})

	if len(*calls) != 0 {
		t.Fatalf("foreign tenant: want no refresh, got %d", len(*calls))
	}
}
