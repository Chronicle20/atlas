package teleportrock

import (
	"atlas-channel/saga"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"errors"
	"testing"

	chartrock "atlas-channel/character/teleportrock"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	testlog "github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	trpkt "github.com/Chronicle20/atlas/libs/atlas-packet/teleportrock"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

var errNotFound = errors.New("not found")

// Fixture (installed by installFixture, restored via t.Cleanup): character 42.
// Regular list contains 102000000 and 103000000; VIP list contains 220000000
// (different continent, 2xx). fieldLimit: 103000000 -> 0x40 (rock-banned),
// else 0. 103000000 is in-list so the "target field barred" case reaches
// step 4 (the fieldLimit bar) instead of failing earlier at step 2a
// (list-membership).
// characterByNameFunc: "Buddy" -> id 77 (session on map 102000000), else error.

func TestUseRockRejections(t *testing.T) {
	l, _ := testlog.NewNullLogger()

	type tc struct {
		name     string
		itemId   item.Id
		srcMap   _map.Id
		target   trpkt.Target
		wantMode string
	}
	cases := []tc{
		{"source field barred", 2320000, 103000000, trpkt.NewTargetByMap(102000000), trpkt.MapTransferModeCannotGo},
		{"target not in list", 2320000, 100000000, trpkt.NewTargetByMap(105000000), trpkt.MapTransferModeCannotGo},
		{"target is current map", 2320000, 102000000, trpkt.NewTargetByMap(102000000), trpkt.MapTransferModeCurrentMap},
		{"target field barred", 2320000, 100000000, trpkt.NewTargetByMap(103000000), trpkt.MapTransferModeCannotGo},
		{"continent mismatch regular", 5040000, 100000000, trpkt.NewTargetByMap(220000000), trpkt.MapTransferModeCannotGoContinent},
		{"player not found", 2320000, 100000000, trpkt.NewTargetByName("Ghost"), trpkt.MapTransferModeUnableToLocate},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var announced string
			var sagaCreated *saga.Saga
			var enabled int
			installFixture(t, c.srcMap, &announced, &sagaCreated, &enabled)

			UseRock(l, context.Background(), nil)(testSession(t, 42, c.srcMap), c.itemId, c.target)

			if announced != c.wantMode {
				t.Errorf("announced mode: got %q want %q", announced, c.wantMode)
			}
			if sagaCreated != nil {
				t.Errorf("failed validation must not create a saga (FR-1)")
			}
			// Every rejection must re-enable the client's exclusive-action lock,
			// or the client freezes after the error (the observed bug).
			if enabled != 1 {
				t.Errorf("rejection must re-enable actions once, got %d", enabled)
			}
		})
	}
}

func TestUseRockSuccessRegularConsumes(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	var announced string
	var sagaCreated *saga.Saga
	var enabled int
	installFixture(t, 100000000, &announced, &sagaCreated, &enabled)

	UseRock(l, context.Background(), nil)(testSession(t, 42, 100000000), 2320000, trpkt.NewTargetByMap(102000000))

	if announced != "" {
		t.Fatalf("success must not announce an error, got %q", announced)
	}
	if enabled != 0 {
		t.Errorf("success must not re-enable actions (the warp/map change unfreezes the client), got %d", enabled)
	}
	if sagaCreated == nil {
		t.Fatalf("expected a saga")
	}
	if sagaCreated.SagaType != saga.TeleportRockUse {
		t.Errorf("saga type: %v", sagaCreated.SagaType)
	}
	if len(sagaCreated.Steps) != 2 {
		t.Fatalf("regular rock: warp + destroy, got %d steps", len(sagaCreated.Steps))
	}
	if sagaCreated.Steps[0].Action != saga.WarpToRandomPortal || sagaCreated.Steps[1].Action != saga.DestroyAsset {
		t.Errorf("step order must be warp-then-destroy (FR-2): %v, %v", sagaCreated.Steps[0].Action, sagaCreated.Steps[1].Action)
	}
}

func TestUseRockSuccessCashDoesNotConsume(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	var announced string
	var sagaCreated *saga.Saga
	var enabled int
	installFixture(t, 100000000, &announced, &sagaCreated, &enabled)

	// 5041000 uses the VIP list and skips the continent check.
	UseRock(l, context.Background(), nil)(testSession(t, 42, 100000000), 5041000, trpkt.NewTargetByMap(220000000))

	if sagaCreated == nil {
		t.Fatalf("expected a saga")
	}
	if len(sagaCreated.Steps) != 1 {
		t.Fatalf("cash rock: warp only, got %d steps", len(sagaCreated.Steps))
	}
}

func TestUseRockByNameSuccess(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	var announced string
	var sagaCreated *saga.Saga
	var enabled int
	installFixture(t, 100000000, &announced, &sagaCreated, &enabled)

	// "Buddy" resolves to a session on map 102000000 (fixture).
	UseRock(l, context.Background(), nil)(testSession(t, 42, 100000000), 2320000, trpkt.NewTargetByName("Buddy"))

	if announced != "" || sagaCreated == nil {
		t.Fatalf("expected warp saga, announced=%q", announced)
	}
}

// installFixture overrides the package-var injection seams so UseRock's
// dependency calls are deterministic in a table test. Restored via
// t.Cleanup (precedent: doorsByOwnerFunc in
// socket/handler/mystic_door_enter.go / mystic_door_enter_test.go). srcMap
// is accepted for call-site symmetry with testSession but the fixture
// itself does not need to vary by it.
func installFixture(t *testing.T, _ _map.Id, announced *string, sagaCreated **saga.Saga, enabled *int) {
	t.Helper()

	origLists := listsFunc
	origMapLimit := mapLimitFunc
	origCharacterByName := characterByNameFunc
	origSessionByCharacterId := sessionByCharacterIdFunc
	origCreateSaga := createSagaFunc
	origAnnounceError := announceErrorFunc
	origEnableActions := enableActionsFunc

	// The regular list also carries 220000000 (in addition to 102000000) so
	// the "continent mismatch" case can reach the continent check (design
	// §1 Q3/Q5, §4.3 row 2a-then-5: list membership is checked BEFORE
	// continent — a target absent from the rock's list fails CANNOT_GO at
	// 2a, never reaching the continent step). A player can register the
	// same map in both their regular and VIP lists.
	//
	// 103000000 is also in the regular list so the "target field barred"
	// case passes step 2a (list membership) and step 3 (not the current
	// map) and actually reaches step 4 (target fieldLimit bar) — the
	// server-only policy half of design §1 Q2 that this task uniquely adds.
	listsFunc = func(_ logrus.FieldLogger, _ context.Context, _ uint32) (chartrock.Model, error) {
		return chartrock.NewModel([]_map.Id{102000000, 220000000, 103000000}, []_map.Id{220000000}), nil
	}

	mapLimitFunc = func(_ logrus.FieldLogger, _ context.Context, mapId _map.Id) (uint32, error) {
		if mapId == 103000000 {
			return 0x40, nil
		}
		return 0, nil
	}

	characterByNameFunc = func(_ logrus.FieldLogger, _ context.Context, name string) (uint32, error) {
		if name == "Buddy" {
			return 77, nil
		}
		return 0, errNotFound
	}

	sessionByCharacterIdFunc = func(_ logrus.FieldLogger, _ context.Context, _ session.Model, characterId uint32) (field.Model, error) {
		if characterId == 77 {
			return field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(102000000)).Build(), nil
		}
		return field.Model{}, errNotFound
	}

	createSagaFunc = func(_ logrus.FieldLogger, _ context.Context, s saga.Saga) error {
		sc := s
		*sagaCreated = &sc
		return nil
	}

	announceErrorFunc = func(_ logrus.FieldLogger, _ context.Context, _ writer.Producer, _ session.Model, key string, _ bool) {
		*announced = key
	}

	enableActionsFunc = func(_ logrus.FieldLogger, _ context.Context, _ writer.Producer, _ session.Model) {
		*enabled++
	}

	t.Cleanup(func() {
		listsFunc = origLists
		mapLimitFunc = origMapLimit
		characterByNameFunc = origCharacterByName
		sessionByCharacterIdFunc = origSessionByCharacterId
		createSagaFunc = origCreateSaga
		announceErrorFunc = origAnnounceError
		enableActionsFunc = origEnableActions
	})
}

// testSession builds a session.Model for world 0 / channel 0 / the given map
// (idiom: newTeleportRockTestSession in
// socket/handler/teleport_rock_add_map_test.go).
func testSession(t *testing.T, characterId uint32, mapId _map.Id) session.Model {
	t.Helper()

	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	ctx := tenant.WithContext(context.Background(), ten)

	sessionId := uuid.New()
	s := session.NewSession(sessionId, ten, 0, nil)
	session.AddSessionToRegistry(ten.Id(), s)
	t.Cleanup(func() { session.ClearRegistryForTenant(ten.Id()) })

	sp := session.NewProcessor(logrus.New(), ctx)
	sp.SetCharacterId(sessionId, characterId)
	f := field.NewBuilder(world.Id(0), channel.Id(0), mapId).Build()
	return sp.SetField(sessionId, f)
}
