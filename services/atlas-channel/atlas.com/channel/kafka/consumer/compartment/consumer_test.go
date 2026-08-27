package compartment

import (
	"atlas-channel/asset"
	"atlas-channel/character/snapshot"
	"atlas-channel/compartment"
	"atlas-channel/inventory"
	compartment2 "atlas-channel/kafka/message/compartment"
	"atlas-channel/server"
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	invconst "github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func newSnapshotTestTenant(t *testing.T) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	return tm
}

func newSnapshotTestServer(t *testing.T, tm tenant.Model) server.Model {
	t.Helper()
	return server.NewProcessor(logrus.New(), context.Background()).Register(tm, channel.NewModel(0, 1), "127.0.0.1", 8484)
}

// seedTestInventory backfills a one-consumable-asset inventory for characterId.
// Same shape as the asset package's seedInventory; the two packages cannot
// share unexported test helpers.
func seedTestInventory(t *testing.T, tm tenant.Model, characterId uint32) uuid.UUID {
	t.Helper()
	compId := uuid.New()
	a := asset.NewModelBuilder(9001, compId, 2060000).SetSlot(2).SetQuantity(400).MustBuild()
	comp := compartment.NewBuilder(compId, characterId, invconst.TypeValueUse, 96).AddAsset(a).MustBuild()
	inv := inventory.NewBuilder(characterId).
		SetEquipable(compartment.NewBuilder(uuid.New(), characterId, invconst.TypeValueEquip, 96).MustBuild()).
		SetConsumable(comp).
		SetSetup(compartment.NewBuilder(uuid.New(), characterId, invconst.TypeValueSetup, 96).MustBuild()).
		SetEtc(compartment.NewBuilder(uuid.New(), characterId, invconst.TypeValueETC, 96).MustBuild()).
		SetCash(compartment.NewBuilder(uuid.New(), characterId, invconst.TypeValueCash, 96).MustBuild()).
		MustBuild()
	v := snapshot.GetRegistry().View(tm, characterId)
	if !snapshot.GetRegistry().BackfillInventory(tm, characterId, inv, v.InvGen) {
		t.Fatalf("seed backfill rejected")
	}
	return compId
}

func TestHandleSnapshotCompartmentEvents_Invalidate(t *testing.T) {
	tm := newSnapshotTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newSnapshotTestServer(t, tm)

	cases := []struct {
		name string
		fire func()
	}{
		{"capacity changed", func() {
			e := compartment2.StatusEvent[compartment2.CapacityChangedEventBody]{
				CharacterId: 71, Type: compartment2.StatusEventTypeCapacityChanged,
				Body: compartment2.CapacityChangedEventBody{Type: 2, Capacity: 128},
			}
			handleSnapshotCompartmentCapacityChanged(sc, nil)(logrus.New(), ctx, e)
		}},
		{"deleted", func() {
			e := compartment2.StatusEvent[compartment2.DeletedStatusEventBody]{
				CharacterId: 71, Type: compartment2.StatusEventTypeDeleted,
			}
			handleSnapshotCompartmentDeleted(sc, nil)(logrus.New(), ctx, e)
		}},
		{"created", func() {
			e := compartment2.StatusEvent[compartment2.CreatedStatusEventBody]{
				CharacterId: 71, Type: compartment2.StatusEventTypeCreated,
				Body: compartment2.CreatedStatusEventBody{Type: 2, Capacity: 96},
			}
			handleSnapshotCompartmentCreated(sc, nil)(logrus.New(), ctx, e)
		}},
		{"merge complete", func() {
			e := compartment2.StatusEvent[compartment2.MergeCompleteEventBody]{
				CharacterId: 71, Type: compartment2.StatusEventTypeMergeComplete,
				Body: compartment2.MergeCompleteEventBody{Type: 2},
			}
			handleSnapshotMergeComplete(sc, nil)(logrus.New(), ctx, e)
		}},
		{"sort complete", func() {
			e := compartment2.StatusEvent[compartment2.SortCompleteEventBody]{
				CharacterId: 71, Type: compartment2.StatusEventTypeSortComplete,
				Body: compartment2.SortCompleteEventBody{Type: 2},
			}
			handleSnapshotSortComplete(sc, nil)(logrus.New(), ctx, e)
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			seedTestInventory(t, tm, 71)
			tc.fire()
			if v := snapshot.GetRegistry().View(tm, 71); v.InvValid {
				t.Fatalf("%s must invalidate the inventory component", tc.name)
			}
		})
	}
}
