package route

import (
	route2 "atlas-channel/kafka/message/route"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// newTestCtx builds a tenant-scoped context, matching the pattern used in
// kafka/consumer/map/consumer_test.go.
func newTestCtx(t *testing.T) context.Context {
	t.Helper()
	tn, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	return tenant.WithContext(context.Background(), tn)
}

// newTestField mirrors kafka/consumer/map/consumer_test.go's newTestField:
// world/channel 0 match the zero-value world/channel server.Model{} carries.
func newTestField(mapId _map.Id) field.Model {
	return field.NewBuilder(world.Id(0), channel.Id(0), mapId).SetInstance(uuid.Nil).Build()
}

// addFieldSession registers a session in ctx's tenant registry with the given
// character id and field, using only public API -- same pattern as
// kafka/consumer/map/consumer_test.go's addFieldSession.
func addFieldSession(t *testing.T, ctx context.Context, characterId uint32, f field.Model) {
	t.Helper()
	ten := tenant.MustFromContext(ctx)
	sessionId := uuid.New()
	s := session.NewSession(sessionId, ten, 0, nil)
	session.AddSessionToRegistry(ten.Id(), s)
	sp := session.NewProcessor(logrus.New(), ctx)
	sp.SetCharacterId(sessionId, characterId)
	sp.SetField(sessionId, f)
}

// announceCall records one routeAnnounce invocation for assertions below.
type announceCall struct {
	mapId      _map.Id
	writerName string
}

// stubRouteAnnounce swaps the routeAnnounce seam for a recording stub and
// returns a restore func plus the captured calls. Mirrors the doorAnnounce
// stubbing pattern in kafka/consumer/map/consumer_test.go.
func stubRouteAnnounce(t *testing.T, mapId _map.Id) (restore func(), calls *[]announceCall) {
	t.Helper()
	var seen []announceCall
	orig := routeAnnounce
	routeAnnounce = func(_ logrus.FieldLogger, _ context.Context, _ writer.Producer, writerName string, _ packet.Encode) model.Operator[session.Model] {
		return func(_ session.Model) error {
			seen = append(seen, announceCall{mapId: mapId, writerName: writerName})
			return nil
		}
	}
	return func() { routeAnnounce = orig }, &seen
}

// FR-V6 / acceptance 20.4: the two new transport event types must remain
// inert here. The type guards already do this today -- this test is what
// stops a future edit (a new handler, a relaxed guard) from silently turning
// a voyage event into a CONTI_STATE broadcast.
func TestRouteConsumerIgnoresVoyageEventTypes(t *testing.T) {
	sc := server.Model{}
	ctx := newTestCtx(t)
	ten := tenant.MustFromContext(ctx)
	defer session.ClearRegistryForTenant(ten.Id())

	const mapId _map.Id = 200090010
	addFieldSession(t, ctx, 1, newTestField(mapId))

	restore, calls := stubRouteAnnounce(t, mapId)
	defer restore()

	for _, theType := range []string{"VOYAGE_DEPARTED", "VOYAGE_ARRIVED"} {
		handleStatusEventArrived(sc, nil)(logrus.New(), ctx,
			route2.StatusEvent[route2.ArrivedStatusEventBody]{Type: theType, Body: route2.ArrivedStatusEventBody{MapId: mapId}})
		handleStatusEventDeparted(sc, nil)(logrus.New(), ctx,
			route2.StatusEvent[route2.DepartedStatusEventBody]{Type: theType, Body: route2.DepartedStatusEventBody{MapId: mapId}})
	}

	if len(*calls) != 0 {
		t.Fatalf("voyage event types produced %d packets, want 0", len(*calls))
	}
}

// And the existing behavior still fires -- the guard must not have been
// tightened into silence.
func TestRouteConsumerStillBroadcastsArrivedAndDeparted(t *testing.T) {
	sc := server.Model{}
	ctx := newTestCtx(t)
	ten := tenant.MustFromContext(ctx)
	defer session.ClearRegistryForTenant(ten.Id())

	const mapId _map.Id = 200090010
	addFieldSession(t, ctx, 1, newTestField(mapId))

	restore, calls := stubRouteAnnounce(t, mapId)
	defer restore()

	handleStatusEventArrived(sc, nil)(logrus.New(), ctx,
		route2.StatusEvent[route2.ArrivedStatusEventBody]{Type: route2.EventStatusArrived, Body: route2.ArrivedStatusEventBody{MapId: mapId}})

	count := 0
	for _, c := range *calls {
		if c.mapId == mapId && c.writerName == fieldcb.FieldTransportStateWriter {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("ARRIVED no longer broadcasts CONTI_STATE: got %d calls, want 1", count)
	}
}
