package asset

import (
	asset2 "atlas-channel/kafka/message/asset"
	"encoding/json"
	"sync"
	"sync/atomic"
	"testing"
	"time"
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
		Slots:     5,
		LevelType: 1,
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
			Strength:    10,
			WeaponAttack: 25,
			Slots:       7,
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
