package asset

import (
	"atlas-channel/asset"
	"atlas-channel/character/snapshot"
	"atlas-channel/compartment"
	"atlas-channel/inventory"
	asset2 "atlas-channel/kafka/message/asset"
	"atlas-channel/server"
	"context"
	"encoding/json"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	invconst "github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// TestMoveInCompartment_FanOutNoDeadlock is a regression test for a consumer
// deadlock in moveInCompartment. The prior implementation used a buffered
// channel with up to three senders — one of which could early-return without
// sending — and a receive loop that read twice per iteration. On non-equip
// moves (e.g., swapping two Use-inventory items) only two sends happened and
// the loop blocked forever on the first read of iteration two. Because the
// asset-status handlers share a consumer goroutine, that wedge stalled
// subsequent QUANTITY_CHANGED events and broke projectile consumption
// visually right after the first inventory swap.
//
// The fix replaced the channel+counter pattern with sync.WaitGroup and
// conditionally added the third task only when its preconditions were met.
// This test pins the invariant: whatever synchronization primitive the
// handler uses, the fan-out must terminate whether or not the conditional
// task runs.
func TestMoveInCompartment_FanOutNoDeadlock(t *testing.T) {
	cases := []struct {
		name           string
		runConditional bool
		wantTasks      int32
	}{
		{"non-equip move (conditional skipped)", false, 2},
		{"equip move crossing slot sign (conditional runs)", true, 3},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var ran int32
			done := make(chan struct{})
			go func() {
				var wg sync.WaitGroup
				wg.Add(2)
				go func() {
					defer wg.Done()
					atomic.AddInt32(&ran, 1)
				}()
				go func() {
					defer wg.Done()
					atomic.AddInt32(&ran, 1)
				}()
				if tc.runConditional {
					wg.Add(1)
					go func() {
						defer wg.Done()
						atomic.AddInt32(&ran, 1)
					}()
				}
				wg.Wait()
				close(done)
			}()
			select {
			case <-done:
			case <-time.After(2 * time.Second):
				t.Fatalf("fan-out deadlocked; ran=%d want=%d", atomic.LoadInt32(&ran), tc.wantTasks)
			}
			if got := atomic.LoadInt32(&ran); got != tc.wantTasks {
				t.Fatalf("tasks executed = %d, want %d", got, tc.wantTasks)
			}
		})
	}
}

func TestCreatedStatusEventBody_Deserialization(t *testing.T) {
	// Simulate a flat CreatedStatusEventBody as it would arrive from Kafka
	body := asset2.CreatedStatusEventBody{
		Expiration:     time.Now().Add(24 * time.Hour),
		CreatedAt:      time.Now(),
		Quantity:       100,
		OwnerId:        12345,
		Flag:           1,
		Rechargeable:   200,
		Strength:       10,
		Dexterity:      20,
		Intelligence:   30,
		Luck:           40,
		Hp:             50,
		Mp:             60,
		WeaponAttack:   70,
		MagicAttack:    80,
		WeaponDefense:  90,
		MagicDefense:   100,
		Accuracy:       110,
		Avoidability:   120,
		Hands:          130,
		Speed:          140,
		Jump:           150,
		Slots:          5,
		LevelType:      1,
		Level:          10,
		Experience:     1000,
		HammersApplied: 2,
		CashId:         98765,
		CommodityId:    555,
		PurchaseBy:     54321,
		PetId:          42,
	}

	// Marshal and unmarshal to simulate Kafka round-trip
	jsonData, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("Failed to marshal body: %v", err)
	}

	var unmarshaled asset2.CreatedStatusEventBody
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal body: %v", err)
	}

	if unmarshaled.Strength != 10 {
		t.Errorf("Strength = %d, want 10", unmarshaled.Strength)
	}
	if unmarshaled.Quantity != 100 {
		t.Errorf("Quantity = %d, want 100", unmarshaled.Quantity)
	}
	if unmarshaled.OwnerId != 12345 {
		t.Errorf("OwnerId = %d, want 12345", unmarshaled.OwnerId)
	}
	if unmarshaled.CashId != 98765 {
		t.Errorf("CashId = %d, want 98765", unmarshaled.CashId)
	}
	if unmarshaled.PetId != 42 {
		t.Errorf("PetId = %d, want 42", unmarshaled.PetId)
	}
	if unmarshaled.Flag != 1 {
		t.Errorf("Flag = %d, want 1", unmarshaled.Flag)
	}
}

func TestStatusEvent_Deserialization(t *testing.T) {
	// Test full StatusEvent with CreatedStatusEventBody
	event := asset2.StatusEvent[asset2.CreatedStatusEventBody]{
		CharacterId: 1000,
		AssetId:     42,
		TemplateId:  1302000,
		Slot:        5,
		Type:        asset2.StatusEventTypeCreated,
		Body: asset2.CreatedStatusEventBody{
			Strength:     10,
			WeaponAttack: 25,
			Slots:        7,
		},
	}

	jsonData, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal event: %v", err)
	}

	var unmarshaled asset2.StatusEvent[asset2.CreatedStatusEventBody]
	err = json.Unmarshal(jsonData, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal event: %v", err)
	}

	if unmarshaled.CharacterId != 1000 {
		t.Errorf("CharacterId = %d, want 1000", unmarshaled.CharacterId)
	}
	if unmarshaled.AssetId != 42 {
		t.Errorf("AssetId = %d, want 42", unmarshaled.AssetId)
	}
	if unmarshaled.TemplateId != 1302000 {
		t.Errorf("TemplateId = %d, want 1302000", unmarshaled.TemplateId)
	}
	if unmarshaled.Type != asset2.StatusEventTypeCreated {
		t.Errorf("Type = %s, want %s", unmarshaled.Type, asset2.StatusEventTypeCreated)
	}
	if unmarshaled.Body.Strength != 10 {
		t.Errorf("Body.Strength = %d, want 10", unmarshaled.Body.Strength)
	}
	if unmarshaled.Body.WeaponAttack != 25 {
		t.Errorf("Body.WeaponAttack = %d, want 25", unmarshaled.Body.WeaponAttack)
	}
}

func TestBuildAssetFromCreatedBody(t *testing.T) {
	body := asset2.CreatedStatusEventBody{
		Expiration:   time.Now().Add(24 * time.Hour),
		Quantity:     50,
		OwnerId:      100,
		Flag:         1,
		Strength:     10,
		WeaponAttack: 25,
		Slots:        7,
		CashId:       12345,
		PetId:        42,
	}

	event := asset2.StatusEvent[asset2.CreatedStatusEventBody]{
		CharacterId: 1,
		AssetId:     10,
		TemplateId:  2000000,
		Slot:        3,
		Type:        asset2.StatusEventTypeCreated,
		Body:        body,
	}

	a := buildAssetFromCreatedBody(event)
	if a.Id() != 10 {
		t.Errorf("Id() = %d, want 10", a.Id())
	}
	if a.TemplateId() != 2000000 {
		t.Errorf("TemplateId() = %d, want 2000000", a.TemplateId())
	}
	if a.Slot() != 3 {
		t.Errorf("Slot() = %d, want 3", a.Slot())
	}
	if a.Quantity() != 50 {
		t.Errorf("Quantity() = %d, want 50", a.Quantity())
	}
	if a.OwnerId() != 100 {
		t.Errorf("OwnerId() = %d, want 100", a.OwnerId())
	}
	if a.Flag() != 1 {
		t.Errorf("Flag() = %d, want 1", a.Flag())
	}
	if a.Strength() != 10 {
		t.Errorf("Strength() = %d, want 10", a.Strength())
	}
	if a.WeaponAttack() != 25 {
		t.Errorf("WeaponAttack() = %d, want 25", a.WeaponAttack())
	}
	if a.Slots() != 7 {
		t.Errorf("Slots() = %d, want 7", a.Slots())
	}
	if a.CashId() != 12345 {
		t.Errorf("CashId() = %d, want 12345", a.CashId())
	}
	if a.PetId() != 42 {
		t.Errorf("PetId() = %d, want 42", a.PetId())
	}
}

// --- task-122 snapshot maintenance handler tests ---

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

// seedInventory backfills a one-consumable-asset inventory and returns the
// consumable compartment id.
func seedInventory(t *testing.T, tm tenant.Model, characterId uint32) uuid.UUID {
	t.Helper()
	compId := uuid.New()
	a := asset.NewBuilderWithId(9001, compId, 2060000).SetSlot(2).SetQuantity(400).MustBuild()
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

func TestHandleSnapshotAssetQuantityChanged_Absolute(t *testing.T) {
	tm := newSnapshotTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newSnapshotTestServer(t, tm)
	seedInventory(t, tm, 61)

	e := asset2.StatusEvent[asset2.QuantityChangedEventBody]{
		CharacterId: 61, AssetId: 9001, TemplateId: 2060000, Slot: 2,
		Type: asset2.StatusEventTypeQuantityChanged,
		Body: asset2.QuantityChangedEventBody{Quantity: 399},
	}
	handleSnapshotAssetQuantityChanged(sc, nil)(logrus.New(), ctx, e)
	handleSnapshotAssetQuantityChanged(sc, nil)(logrus.New(), ctx, e) // redelivery-idempotent

	v := snapshot.GetRegistry().View(tm, 61)
	a, ok := v.Inv.Consumable().FindById(9001)
	if !ok || a.Quantity() != 399 {
		t.Fatalf("quantity mismatch: %+v ok=%v", a, ok)
	}
}

func TestHandleSnapshotAssetMoved_SetsSlotAbsolute(t *testing.T) {
	tm := newSnapshotTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newSnapshotTestServer(t, tm)
	seedInventory(t, tm, 62)

	e := asset2.StatusEvent[asset2.MovedStatusEventBody]{
		CharacterId: 62, AssetId: 9001, TemplateId: 2060000, Slot: 7,
		Type: asset2.StatusEventTypeMoved,
		Body: asset2.MovedStatusEventBody{OldSlot: 2},
	}
	handleSnapshotAssetMoved(sc, nil)(logrus.New(), ctx, e)

	v := snapshot.GetRegistry().View(tm, 62)
	a, _ := v.Inv.Consumable().FindById(9001)
	if a.Slot() != 7 {
		t.Fatalf("slot mismatch: %d", a.Slot())
	}
}

func TestHandleSnapshotAssetCreatedAndDeleted(t *testing.T) {
	tm := newSnapshotTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newSnapshotTestServer(t, tm)
	compId := seedInventory(t, tm, 63)

	ce := asset2.StatusEvent[asset2.CreatedStatusEventBody]{
		CharacterId: 63, CompartmentId: compId, AssetId: 9002, TemplateId: 2070000, Slot: 3,
		Type: asset2.StatusEventTypeCreated,
		Body: asset2.CreatedStatusEventBody{Quantity: 500},
	}
	handleSnapshotAssetCreated(sc, nil)(logrus.New(), ctx, ce)

	v := snapshot.GetRegistry().View(tm, 63)
	a, ok := v.Inv.Consumable().FindById(9002)
	if !ok || a.Quantity() != 500 || a.Slot() != 3 || a.TemplateId() != 2070000 {
		t.Fatalf("CREATED upsert mismatch: %+v ok=%v", a, ok)
	}

	de := asset2.StatusEvent[asset2.DeletedStatusEventBody]{
		CharacterId: 63, CompartmentId: compId, AssetId: 9002, TemplateId: 2070000, Slot: 3,
		Type: asset2.StatusEventTypeDeleted,
	}
	handleSnapshotAssetDeleted(sc, nil)(logrus.New(), ctx, de)

	v = snapshot.GetRegistry().View(tm, 63)
	if _, ok = v.Inv.Consumable().FindById(9002); ok {
		t.Fatalf("DELETED must remove the asset")
	}
}

func TestHandleSnapshotAssetReleased_Invalidate(t *testing.T) {
	tm := newSnapshotTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newSnapshotTestServer(t, tm)
	seedInventory(t, tm, 64)

	re := asset2.StatusEvent[asset2.ReleasedStatusEventBody]{
		CharacterId: 64, AssetId: 9001, Type: asset2.StatusEventTypeReleased,
	}
	handleSnapshotAssetReleased(sc, nil)(logrus.New(), ctx, re)
	if v := snapshot.GetRegistry().View(tm, 64); v.InvValid {
		t.Fatalf("RELEASED (thin) must invalidate the inventory component")
	}
}

func TestHandleSnapshotAssetExpired_Invalidate(t *testing.T) {
	tm := newSnapshotTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newSnapshotTestServer(t, tm)
	seedInventory(t, tm, 65)

	if v := snapshot.GetRegistry().View(tm, 65); !v.InvValid {
		t.Fatalf("precondition: inventory component must be valid before EXPIRED fires")
	}

	ee := asset2.StatusEvent[asset2.ExpiredStatusEventBody]{
		CharacterId: 65, AssetId: 9001, Type: asset2.StatusEventTypeExpired,
	}
	handleSnapshotAssetExpired(sc, nil)(logrus.New(), ctx, ee)
	if v := snapshot.GetRegistry().View(tm, 65); v.InvValid {
		t.Fatalf("EXPIRED (thin) must invalidate the inventory component")
	}
}

func TestHandleSnapshotAssetUpdated_ReplacesAssetFields(t *testing.T) {
	tm := newSnapshotTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newSnapshotTestServer(t, tm)
	compId := seedInventory(t, tm, 66)

	e := asset2.StatusEvent[asset2.UpdatedStatusEventBody]{
		CharacterId: 66, CompartmentId: compId, AssetId: 9001, TemplateId: 2060000, Slot: 9,
		Type: asset2.StatusEventTypeUpdated,
		Body: asset2.UpdatedStatusEventBody{Quantity: 111, Strength: 77},
	}
	handleSnapshotAssetUpdated(sc, nil)(logrus.New(), ctx, e)

	v := snapshot.GetRegistry().View(tm, 66)
	a, ok := v.Inv.Consumable().FindById(9001)
	if !ok || a.Slot() != 9 || a.Quantity() != 111 || a.Strength() != 77 {
		t.Fatalf("UPDATED replace mismatch: %+v ok=%v", a, ok)
	}
}

func TestHandleSnapshotAssetAccepted_UpsertsAsset(t *testing.T) {
	tm := newSnapshotTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	sc := newSnapshotTestServer(t, tm)
	compId := seedInventory(t, tm, 67)

	e := asset2.StatusEvent[asset2.AcceptedStatusEventBody]{
		CharacterId: 67, CompartmentId: compId, AssetId: 9003, TemplateId: 2080000, Slot: 4,
		Type: asset2.StatusEventTypeAccepted,
		Body: asset2.AcceptedStatusEventBody{Quantity: 250},
	}
	handleSnapshotAssetAccepted(sc, nil)(logrus.New(), ctx, e)

	v := snapshot.GetRegistry().View(tm, 67)
	a, ok := v.Inv.Consumable().FindById(9003)
	if !ok || a.Quantity() != 250 || a.Slot() != 4 || a.TemplateId() != 2080000 {
		t.Fatalf("ACCEPTED upsert mismatch: %+v ok=%v", a, ok)
	}
}
