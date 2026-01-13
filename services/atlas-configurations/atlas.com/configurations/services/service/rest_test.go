package service

import (
	"atlas-configurations/services/task"
	"encoding/json"
	"testing"
)

func TestGenericRestModel_GetName(t *testing.T) {
	rm := GenericRestModel{}
	expected := "services"
	if rm.GetName() != expected {
		t.Errorf("expected GetName() to return '%s', got '%s'", expected, rm.GetName())
	}
}

func TestGenericRestModel_GetID(t *testing.T) {
	testID := "test-uuid-123"
	rm := GenericRestModel{Id: testID}

	if rm.GetID() != testID {
		t.Errorf("expected GetID() to return '%s', got '%s'", testID, rm.GetID())
	}
}

func TestGenericRestModel_SetID(t *testing.T) {
	rm := GenericRestModel{}
	testID := "new-test-id"

	err := rm.SetID(testID)
	if err != nil {
		t.Fatalf("SetID returned error: %v", err)
	}

	if rm.Id != testID {
		t.Errorf("expected Id to be '%s', got '%s'", testID, rm.Id)
	}
}

func TestGenericRestModel_JSONMarshal(t *testing.T) {
	rm := GenericRestModel{
		Id: "test-id",
		Tasks: []task.RestModel{
			{Type: "cleanup", Interval: 60000, Duration: 0},
		},
	}

	data, err := json.Marshal(rm)
	if err != nil {
		t.Fatalf("JSON marshal failed: %v", err)
	}

	var decoded GenericRestModel
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	// Id should not be marshaled (json:"-")
	if decoded.Id != "" {
		t.Errorf("expected Id to be empty after unmarshal, got '%s'", decoded.Id)
	}

	if len(decoded.Tasks) != 1 {
		t.Errorf("expected 1 task, got %d", len(decoded.Tasks))
	}
}

func TestLoginRestModel_GetName(t *testing.T) {
	rm := LoginRestModel{}
	expected := "services"
	if rm.GetName() != expected {
		t.Errorf("expected GetName() to return '%s', got '%s'", expected, rm.GetName())
	}
}

func TestLoginRestModel_GetID(t *testing.T) {
	testID := "test-uuid-123"
	rm := LoginRestModel{Id: testID}

	if rm.GetID() != testID {
		t.Errorf("expected GetID() to return '%s', got '%s'", testID, rm.GetID())
	}
}

func TestLoginRestModel_SetID(t *testing.T) {
	rm := LoginRestModel{}
	testID := "new-test-id"

	err := rm.SetID(testID)
	if err != nil {
		t.Fatalf("SetID returned error: %v", err)
	}

	if rm.Id != testID {
		t.Errorf("expected Id to be '%s', got '%s'", testID, rm.Id)
	}
}

func TestLoginRestModel_JSONMarshal(t *testing.T) {
	rm := LoginRestModel{
		Id: "test-id",
		Tasks: []task.RestModel{
			{Type: "heartbeat", Interval: 10000},
		},
		Tenants: []LoginTenantRestModel{
			{Id: "tenant-1", Port: 8484},
		},
	}

	data, err := json.Marshal(rm)
	if err != nil {
		t.Fatalf("JSON marshal failed: %v", err)
	}

	var decoded LoginRestModel
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	if len(decoded.Tasks) != 1 {
		t.Errorf("expected 1 task, got %d", len(decoded.Tasks))
	}
	if len(decoded.Tenants) != 1 {
		t.Errorf("expected 1 tenant, got %d", len(decoded.Tenants))
	}
	if decoded.Tenants[0].Port != 8484 {
		t.Errorf("expected port 8484, got %d", decoded.Tenants[0].Port)
	}
}

func TestChannelRestModel_GetName(t *testing.T) {
	rm := ChannelRestModel{}
	expected := "services"
	if rm.GetName() != expected {
		t.Errorf("expected GetName() to return '%s', got '%s'", expected, rm.GetName())
	}
}

func TestChannelRestModel_GetID(t *testing.T) {
	testID := "test-uuid-123"
	rm := ChannelRestModel{Id: testID}

	if rm.GetID() != testID {
		t.Errorf("expected GetID() to return '%s', got '%s'", testID, rm.GetID())
	}
}

func TestChannelRestModel_SetID(t *testing.T) {
	rm := ChannelRestModel{}
	testID := "new-test-id"

	err := rm.SetID(testID)
	if err != nil {
		t.Fatalf("SetID returned error: %v", err)
	}

	if rm.Id != testID {
		t.Errorf("expected Id to be '%s', got '%s'", testID, rm.Id)
	}
}

func TestChannelRestModel_JSONMarshal(t *testing.T) {
	rm := ChannelRestModel{
		Id: "test-id",
		Tasks: []task.RestModel{
			{Type: "respawn", Interval: 5000},
		},
		Tenants: []ChannelTenantRestModel{
			{
				Id:        "tenant-1",
				IPAddress: "127.0.0.1",
				Worlds: []ChannelWorldRestModel{
					{
						Id: 0,
						Channels: []ChannelChannelRestModel{
							{Id: 0, Port: 7575},
							{Id: 1, Port: 7576},
						},
					},
				},
			},
		},
	}

	data, err := json.Marshal(rm)
	if err != nil {
		t.Fatalf("JSON marshal failed: %v", err)
	}

	var decoded ChannelRestModel
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	if len(decoded.Tasks) != 1 {
		t.Errorf("expected 1 task, got %d", len(decoded.Tasks))
	}
	if len(decoded.Tenants) != 1 {
		t.Errorf("expected 1 tenant, got %d", len(decoded.Tenants))
	}
	if decoded.Tenants[0].IPAddress != "127.0.0.1" {
		t.Errorf("expected IPAddress '127.0.0.1', got '%s'", decoded.Tenants[0].IPAddress)
	}
	if len(decoded.Tenants[0].Worlds) != 1 {
		t.Errorf("expected 1 world, got %d", len(decoded.Tenants[0].Worlds))
	}
	if len(decoded.Tenants[0].Worlds[0].Channels) != 2 {
		t.Errorf("expected 2 channels, got %d", len(decoded.Tenants[0].Worlds[0].Channels))
	}
}

func TestLoginTenantRestModel_JSONMarshal(t *testing.T) {
	rm := LoginTenantRestModel{
		Id:   "tenant-1",
		Port: 8484,
	}

	data, err := json.Marshal(rm)
	if err != nil {
		t.Fatalf("JSON marshal failed: %v", err)
	}

	var decoded LoginTenantRestModel
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	if decoded.Id != rm.Id {
		t.Errorf("expected Id '%s', got '%s'", rm.Id, decoded.Id)
	}
	if decoded.Port != rm.Port {
		t.Errorf("expected Port %d, got %d", rm.Port, decoded.Port)
	}
}

func TestChannelWorldRestModel_JSONMarshal(t *testing.T) {
	rm := ChannelWorldRestModel{
		Id: 0,
		Channels: []ChannelChannelRestModel{
			{Id: 0, Port: 7575},
		},
	}

	data, err := json.Marshal(rm)
	if err != nil {
		t.Fatalf("JSON marshal failed: %v", err)
	}

	var decoded ChannelWorldRestModel
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	if decoded.Id != rm.Id {
		t.Errorf("expected Id %d, got %d", rm.Id, decoded.Id)
	}
	if len(decoded.Channels) != 1 {
		t.Errorf("expected 1 channel, got %d", len(decoded.Channels))
	}
}
