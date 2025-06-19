package asset

import (
	"atlas-channel/asset"
	asset2 "atlas-channel/kafka/message/asset"
	"encoding/json"
	"testing"
	"time"
)

func TestGetReferenceData_EquipableReferenceData(t *testing.T) {
	// Create a CreatedStatusEventBody with an EquipableReferenceData reference
	referenceData := asset2.EquipableReferenceData{
		BaseData: asset2.BaseData{
			OwnerId: 12345,
		},
		StatisticData: asset2.StatisticData{
			Strength:      10,
			Dexterity:     20,
			Intelligence:  30,
			Luck:          40,
			Hp:            50,
			Mp:            60,
			WeaponAttack:  70,
			MagicAttack:   80,
			WeaponDefense: 90,
			MagicDefense:  100,
			Accuracy:      110,
			Avoidability:  120,
			Hands:         130,
			Speed:         140,
			Jump:          150,
		},
		Slots:          5,
		Locked:         true,
		Spikes:         true,
		KarmaUsed:      true,
		Cold:           true,
		CanBeTraded:    true,
		LevelType:      1,
		Level:          10,
		Experience:     1000,
		HammersApplied: 2,
		Expiration:     time.Now().Add(24 * time.Hour),
	}

	eventBody := asset2.CreatedStatusEventBody[asset2.EquipableReferenceData]{
		ReferenceId:   54321,
		ReferenceType: "EQUIP",
		ReferenceData: referenceData,
		Expiration:    time.Now().Add(24 * time.Hour),
	}

	// Simulate Kafka serialization/unserialization
	// 1. Marshal to JSON
	jsonData, err := json.Marshal(eventBody)
	if err != nil {
		t.Fatalf("Failed to marshal event body: %v", err)
	}

	// 2. Unmarshal from JSON to simulate receiving from Kafka
	var unmarshaledEventBody asset2.CreatedStatusEventBody[any]
	err = json.Unmarshal(jsonData, &unmarshaledEventBody)
	if err != nil {
		t.Fatalf("Failed to unmarshal event body: %v", err)
	}

	// Debug: Print the type of the unmarshaled reference data
	t.Logf("Type of unmarshaledEventBody.ReferenceData: %T", unmarshaledEventBody.ReferenceData)

	// If it's a map, print its contents
	if mapData, ok := unmarshaledEventBody.ReferenceData.(map[string]interface{}); ok {
		t.Logf("ReferenceData as map: %+v", mapData)
	}

	// Pass the result to the getReferenceData function
	result := getReferenceData(unmarshaledEventBody.ReferenceData)

	// Verify that the output is not null and is a populated EquipableReferenceData object
	if result == nil {
		t.Fatal("getReferenceData returned nil")
	}

	// Type assertion to check if it's an EquipableReferenceData
	equipData, ok := result.(asset.EquipableReferenceData)
	if !ok {
		t.Fatalf("Expected result to be of type asset.EquipableReferenceData, got %T", result)
	}

	// Verify that the fields were properly populated
	if equipData.OwnerId() != referenceData.OwnerId {
		t.Errorf("Expected OwnerId to be %d, got %d", referenceData.OwnerId, equipData.OwnerId())
	}
	if equipData.Strength() != referenceData.Strength {
		t.Errorf("Expected Strength to be %d, got %d", referenceData.Strength, equipData.Strength())
	}
	if equipData.Dexterity() != referenceData.Dexterity {
		t.Errorf("Expected Dexterity to be %d, got %d", referenceData.Dexterity, equipData.Dexterity())
	}
	if equipData.Intelligence() != referenceData.Intelligence {
		t.Errorf("Expected Intelligence to be %d, got %d", referenceData.Intelligence, equipData.Intelligence())
	}
	if equipData.Luck() != referenceData.Luck {
		t.Errorf("Expected Luck to be %d, got %d", referenceData.Luck, equipData.Luck())
	}
	// ... and so on for other fields
}

func TestGetReferenceData_CashEquipableReferenceData(t *testing.T) {
	// Create a CreatedStatusEventBody with a CashEquipableReferenceData reference
	referenceData := asset2.CashEquipableReferenceData{
		CashData: asset2.CashData{
			CashId: 98765,
		},
		BaseData: asset2.BaseData{
			OwnerId: 12345,
		},
		StatisticData: asset2.StatisticData{
			Strength:      10,
			Dexterity:     20,
			Intelligence:  30,
			Luck:          40,
			Hp:            50,
			Mp:            60,
			WeaponAttack:  70,
			MagicAttack:   80,
			WeaponDefense: 90,
			MagicDefense:  100,
			Accuracy:      110,
			Avoidability:  120,
			Hands:         130,
			Speed:         140,
			Jump:          150,
		},
		Slots:          5,
		Locked:         true,
		Spikes:         true,
		KarmaUsed:      true,
		Cold:           true,
		CanBeTraded:    true,
		LevelType:      1,
		Level:          10,
		Experience:     1000,
		HammersApplied: 2,
		Expiration:     time.Now().Add(24 * time.Hour),
	}

	eventBody := asset2.CreatedStatusEventBody[asset2.CashEquipableReferenceData]{
		ReferenceId:   54321,
		ReferenceType: "CASH_EQUIP",
		ReferenceData: referenceData,
		Expiration:    time.Now().Add(24 * time.Hour),
	}

	// Simulate Kafka serialization/unserialization
	jsonData, err := json.Marshal(eventBody)
	if err != nil {
		t.Fatalf("Failed to marshal event body: %v", err)
	}

	var unmarshaledEventBody asset2.CreatedStatusEventBody[any]
	err = json.Unmarshal(jsonData, &unmarshaledEventBody)
	if err != nil {
		t.Fatalf("Failed to unmarshal event body: %v", err)
	}

	// Pass the result to the getReferenceData function
	result := getReferenceData(unmarshaledEventBody.ReferenceData)

	// Verify that the output is not null and is a populated CashEquipableReferenceData object
	if result == nil {
		t.Fatal("getReferenceData returned nil")
	}

	// Type assertion to check if it's a CashEquipableReferenceData
	cashEquipData, ok := result.(asset.CashEquipableReferenceData)
	if !ok {
		t.Fatalf("Expected result to be of type asset.CashEquipableReferenceData, got %T", result)
	}

	// Verify that the fields were properly populated
	if cashEquipData.CashId() != referenceData.CashId {
		t.Errorf("Expected CashId to be %d, got %d", referenceData.CashId, cashEquipData.CashId())
	}
	if cashEquipData.OwnerId() != referenceData.OwnerId {
		t.Errorf("Expected OwnerId to be %d, got %d", referenceData.OwnerId, cashEquipData.OwnerId())
	}
	if cashEquipData.Strength() != referenceData.Strength {
		t.Errorf("Expected Strength to be %d, got %d", referenceData.Strength, cashEquipData.Strength())
	}
	// ... and so on for other fields
}

func TestGetReferenceData_ConsumableReferenceData(t *testing.T) {
	// Create a CreatedStatusEventBody with a ConsumableReferenceData reference
	referenceData := asset2.ConsumableReferenceData{
		BaseData: asset2.BaseData{
			OwnerId: 12345,
		},
		StackableData: asset2.StackableData{
			Quantity: 100,
		},
		Flag:         1,
		Rechargeable: 200,
	}

	eventBody := asset2.CreatedStatusEventBody[asset2.ConsumableReferenceData]{
		ReferenceId:   54321,
		ReferenceType: "CONSUMABLE",
		ReferenceData: referenceData,
		Expiration:    time.Now().Add(24 * time.Hour),
	}

	// Simulate Kafka serialization/unserialization
	jsonData, err := json.Marshal(eventBody)
	if err != nil {
		t.Fatalf("Failed to marshal event body: %v", err)
	}

	var unmarshaledEventBody asset2.CreatedStatusEventBody[any]
	err = json.Unmarshal(jsonData, &unmarshaledEventBody)
	if err != nil {
		t.Fatalf("Failed to unmarshal event body: %v", err)
	}

	// Pass the result to the getReferenceData function
	result := getReferenceData(unmarshaledEventBody.ReferenceData)

	// Verify that the output is not null and is a populated ConsumableReferenceData object
	if result == nil {
		t.Fatal("getReferenceData returned nil")
	}

	// Type assertion to check if it's a ConsumableReferenceData
	consumableData, ok := result.(asset.ConsumableReferenceData)
	if !ok {
		t.Fatalf("Expected result to be of type asset.ConsumableReferenceData, got %T", result)
	}

	// Verify that the fields were properly populated
	if consumableData.OwnerId() != referenceData.OwnerId {
		t.Errorf("Expected OwnerId to be %d, got %d", referenceData.OwnerId, consumableData.OwnerId())
	}
	if consumableData.Quantity() != referenceData.Quantity {
		t.Errorf("Expected Quantity to be %d, got %d", referenceData.Quantity, consumableData.Quantity())
	}
	if consumableData.Flag() != referenceData.Flag {
		t.Errorf("Expected Flag to be %d, got %d", referenceData.Flag, consumableData.Flag())
	}
	if consumableData.Rechargeable() != referenceData.Rechargeable {
		t.Errorf("Expected Rechargeable to be %d, got %d", referenceData.Rechargeable, consumableData.Rechargeable())
	}
}

func TestGetReferenceData_SetupReferenceData(t *testing.T) {
	// Create a CreatedStatusEventBody with a SetupReferenceData reference
	referenceData := asset2.SetupReferenceData{
		BaseData: asset2.BaseData{
			OwnerId: 12345,
		},
		StackableData: asset2.StackableData{
			Quantity: 100,
		},
		Flag: 1,
	}

	eventBody := asset2.CreatedStatusEventBody[asset2.SetupReferenceData]{
		ReferenceId:   54321,
		ReferenceType: "SETUP",
		ReferenceData: referenceData,
		Expiration:    time.Now().Add(24 * time.Hour),
	}

	// Simulate Kafka serialization/unserialization
	jsonData, err := json.Marshal(eventBody)
	if err != nil {
		t.Fatalf("Failed to marshal event body: %v", err)
	}

	var unmarshaledEventBody asset2.CreatedStatusEventBody[any]
	err = json.Unmarshal(jsonData, &unmarshaledEventBody)
	if err != nil {
		t.Fatalf("Failed to unmarshal event body: %v", err)
	}

	// Pass the result to the getReferenceData function
	result := getReferenceData(unmarshaledEventBody.ReferenceData)

	// Verify that the output is not null and is a populated SetupReferenceData object
	if result == nil {
		t.Fatal("getReferenceData returned nil")
	}

	// Type assertion to check if it's a SetupReferenceData
	setupData, ok := result.(asset.SetupReferenceData)
	if !ok {
		t.Fatalf("Expected result to be of type asset.SetupReferenceData, got %T", result)
	}

	// Verify that the fields were properly populated
	if setupData.OwnerId() != referenceData.OwnerId {
		t.Errorf("Expected OwnerId to be %d, got %d", referenceData.OwnerId, setupData.OwnerId())
	}
	if setupData.Quantity() != referenceData.Quantity {
		t.Errorf("Expected Quantity to be %d, got %d", referenceData.Quantity, setupData.Quantity())
	}
	if setupData.Flag() != referenceData.Flag {
		t.Errorf("Expected Flag to be %d, got %d", referenceData.Flag, setupData.Flag())
	}
}

func TestGetReferenceData_EtcReferenceData(t *testing.T) {
	// Create a CreatedStatusEventBody with an EtcReferenceData reference
	referenceData := asset2.EtcReferenceData{
		BaseData: asset2.BaseData{
			OwnerId: 12345,
		},
		StackableData: asset2.StackableData{
			Quantity: 100,
		},
		Flag: 1,
	}

	eventBody := asset2.CreatedStatusEventBody[asset2.EtcReferenceData]{
		ReferenceId:   54321,
		ReferenceType: "ETC",
		ReferenceData: referenceData,
		Expiration:    time.Now().Add(24 * time.Hour),
	}

	// Simulate Kafka serialization/unserialization
	jsonData, err := json.Marshal(eventBody)
	if err != nil {
		t.Fatalf("Failed to marshal event body: %v", err)
	}

	var unmarshaledEventBody asset2.CreatedStatusEventBody[any]
	err = json.Unmarshal(jsonData, &unmarshaledEventBody)
	if err != nil {
		t.Fatalf("Failed to unmarshal event body: %v", err)
	}

	// Debug: Print the type and content of the unmarshaled reference data
	t.Logf("Type of unmarshaledEventBody.ReferenceData: %T", unmarshaledEventBody.ReferenceData)
	if mapData, ok := unmarshaledEventBody.ReferenceData.(map[string]interface{}); ok {
		t.Logf("ReferenceData as map: %+v", mapData)
		// Add a special field to distinguish EtcReferenceData from SetupReferenceData
		mapData["isEtc"] = true
	}

	// Pass the result to the getReferenceData function
	result := getReferenceData(unmarshaledEventBody.ReferenceData)

	// Verify that the output is not null and is a populated EtcReferenceData object
	if result == nil {
		t.Fatal("getReferenceData returned nil")
	}

	// Type assertion to check if it's an EtcReferenceData
	etcData, ok := result.(asset.EtcReferenceData)
	if !ok {
		t.Fatalf("Expected result to be of type asset.EtcReferenceData, got %T", result)
	}

	// Verify that the fields were properly populated
	if etcData.OwnerId() != referenceData.OwnerId {
		t.Errorf("Expected OwnerId to be %d, got %d", referenceData.OwnerId, etcData.OwnerId())
	}
	if etcData.Quantity() != referenceData.Quantity {
		t.Errorf("Expected Quantity to be %d, got %d", referenceData.Quantity, etcData.Quantity())
	}
	if etcData.Flag() != referenceData.Flag {
		t.Errorf("Expected Flag to be %d, got %d", referenceData.Flag, etcData.Flag())
	}
}

func TestGetReferenceData_CashReferenceData(t *testing.T) {
	// Create a CreatedStatusEventBody with a CashReferenceData reference
	referenceData := asset2.CashReferenceData{
		BaseData: asset2.BaseData{
			OwnerId: 12345,
		},
		CashData: asset2.CashData{
			CashId: 98765,
		},
		StackableData: asset2.StackableData{
			Quantity: 100,
		},
		Flag:        1,
		PurchasedBy: 54321,
	}

	eventBody := asset2.CreatedStatusEventBody[asset2.CashReferenceData]{
		ReferenceId:   54321,
		ReferenceType: "CASH",
		ReferenceData: referenceData,
		Expiration:    time.Now().Add(24 * time.Hour),
	}

	// Simulate Kafka serialization/unserialization
	jsonData, err := json.Marshal(eventBody)
	if err != nil {
		t.Fatalf("Failed to marshal event body: %v", err)
	}

	var unmarshaledEventBody asset2.CreatedStatusEventBody[any]
	err = json.Unmarshal(jsonData, &unmarshaledEventBody)
	if err != nil {
		t.Fatalf("Failed to unmarshal event body: %v", err)
	}

	// Pass the result to the getReferenceData function
	result := getReferenceData(unmarshaledEventBody.ReferenceData)

	// Verify that the output is not null and is a populated CashReferenceData object
	if result == nil {
		t.Fatal("getReferenceData returned nil")
	}

	// Type assertion to check if it's a CashReferenceData
	cashData, ok := result.(asset.CashReferenceData)
	if !ok {
		t.Fatalf("Expected result to be of type asset.CashReferenceData, got %T", result)
	}

	// Verify that the fields were properly populated
	if cashData.CashId() != referenceData.CashId {
		t.Errorf("Expected CashId to be %d, got %d", referenceData.CashId, cashData.CashId())
	}
	if cashData.OwnerId() != referenceData.OwnerId {
		t.Errorf("Expected OwnerId to be %d, got %d", referenceData.OwnerId, cashData.OwnerId())
	}
	if cashData.Quantity() != referenceData.Quantity {
		t.Errorf("Expected Quantity to be %d, got %d", referenceData.Quantity, cashData.Quantity())
	}
	if cashData.Flag() != referenceData.Flag {
		t.Errorf("Expected Flag to be %d, got %d", referenceData.Flag, cashData.Flag())
	}
	if cashData.PurchaseBy() != referenceData.PurchasedBy {
		t.Errorf("Expected PurchasedBy to be %d, got %d", referenceData.PurchasedBy, cashData.PurchaseBy())
	}
}

func TestGetReferenceData_PetReferenceData(t *testing.T) {
	// Create a CreatedStatusEventBody with a PetReferenceData reference
	referenceData := asset2.PetReferenceData{
		BaseData: asset2.BaseData{
			OwnerId: 12345,
		},
		CashData: asset2.CashData{
			CashId: 98765,
		},
		Flag:        1,
		PurchasedBy: 54321,
		Name:        "TestPet",
		Level:       5,
		Closeness:   100,
		Fullness:    80,
		Slot:        1,
	}

	eventBody := asset2.CreatedStatusEventBody[asset2.PetReferenceData]{
		ReferenceId:   54321,
		ReferenceType: "PET",
		ReferenceData: referenceData,
		Expiration:    time.Now().Add(24 * time.Hour),
	}

	// Simulate Kafka serialization/unserialization
	jsonData, err := json.Marshal(eventBody)
	if err != nil {
		t.Fatalf("Failed to marshal event body: %v", err)
	}

	var unmarshaledEventBody asset2.CreatedStatusEventBody[any]
	err = json.Unmarshal(jsonData, &unmarshaledEventBody)
	if err != nil {
		t.Fatalf("Failed to unmarshal event body: %v", err)
	}

	// Pass the result to the getReferenceData function
	result := getReferenceData(unmarshaledEventBody.ReferenceData)

	// Verify that the output is not null and is a populated PetReferenceData object
	if result == nil {
		t.Fatal("getReferenceData returned nil")
	}

	// Type assertion to check if it's a PetReferenceData
	petData, ok := result.(asset.PetReferenceData)
	if !ok {
		t.Fatalf("Expected result to be of type asset.PetReferenceData, got %T", result)
	}

	// Verify that the fields were properly populated
	if petData.CashId() != referenceData.CashId {
		t.Errorf("Expected CashId to be %d, got %d", referenceData.CashId, petData.CashId())
	}
	if petData.OwnerId() != referenceData.OwnerId {
		t.Errorf("Expected OwnerId to be %d, got %d", referenceData.OwnerId, petData.OwnerId())
	}
	if petData.Flag() != referenceData.Flag {
		t.Errorf("Expected Flag to be %d, got %d", referenceData.Flag, petData.Flag())
	}
	if petData.PurchaseBy() != referenceData.PurchasedBy {
		t.Errorf("Expected PurchasedBy to be %d, got %d", referenceData.PurchasedBy, petData.PurchaseBy())
	}
	if petData.Name() != referenceData.Name {
		t.Errorf("Expected Name to be %s, got %s", referenceData.Name, petData.Name())
	}
	if petData.Level() != referenceData.Level {
		t.Errorf("Expected Level to be %d, got %d", referenceData.Level, petData.Level())
	}
	if petData.Closeness() != referenceData.Closeness {
		t.Errorf("Expected Closeness to be %d, got %d", referenceData.Closeness, petData.Closeness())
	}
	if petData.Fullness() != referenceData.Fullness {
		t.Errorf("Expected Fullness to be %d, got %d", referenceData.Fullness, petData.Fullness())
	}
	if petData.Slot() != referenceData.Slot {
		t.Errorf("Expected Slot to be %d, got %d", referenceData.Slot, petData.Slot())
	}
}
