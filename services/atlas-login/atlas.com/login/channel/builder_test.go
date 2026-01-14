package channel_test

import (
	"atlas-login/channel"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestBuilder_Build(t *testing.T) {
	id := uuid.New()
	createdAt := time.Now()

	m := channel.NewBuilder().
		SetId(id).
		SetWorldId(1).
		SetChannelId(0).
		SetIpAddress("192.168.1.1").
		SetPort(8585).
		SetCurrentCapacity(100).
		SetMaxCapacity(1000).
		SetCreatedAt(createdAt).
		Build()

	if m.Id() != id {
		t.Errorf("Id() = %v, want %v", m.Id(), id)
	}
	if m.WorldId() != 1 {
		t.Errorf("WorldId() = %d, want 1", m.WorldId())
	}
	if m.ChannelId() != 0 {
		t.Errorf("ChannelId() = %d, want 0", m.ChannelId())
	}
	if m.IpAddress() != "192.168.1.1" {
		t.Errorf("IpAddress() = %s, want '192.168.1.1'", m.IpAddress())
	}
	if m.Port() != 8585 {
		t.Errorf("Port() = %d, want 8585", m.Port())
	}
	if m.CurrentCapacity() != 100 {
		t.Errorf("CurrentCapacity() = %d, want 100", m.CurrentCapacity())
	}
	if m.MaxCapacity() != 1000 {
		t.Errorf("MaxCapacity() = %d, want 1000", m.MaxCapacity())
	}
}

func TestModel_ToBuilder(t *testing.T) {
	id := uuid.New()

	original := channel.NewBuilder().
		SetId(id).
		SetWorldId(1).
		SetChannelId(0).
		SetIpAddress("192.168.1.1").
		SetPort(8585).
		SetCurrentCapacity(100).
		SetMaxCapacity(1000).
		Build()

	// Clone and modify current capacity
	cloned := original.ToBuilder().
		SetCurrentCapacity(200).
		Build()

	// Original should be unchanged
	if original.CurrentCapacity() != 100 {
		t.Errorf("Original CurrentCapacity() = %d, want 100", original.CurrentCapacity())
	}

	// Cloned should have new capacity
	if cloned.CurrentCapacity() != 200 {
		t.Errorf("Cloned CurrentCapacity() = %d, want 200", cloned.CurrentCapacity())
	}

	// Other fields should be preserved
	if cloned.Id() != id {
		t.Errorf("Cloned Id() = %v, want %v", cloned.Id(), id)
	}
	if cloned.WorldId() != 1 {
		t.Errorf("Cloned WorldId() = %d, want 1", cloned.WorldId())
	}
	if cloned.IpAddress() != "192.168.1.1" {
		t.Errorf("Cloned IpAddress() = %s, want '192.168.1.1'", cloned.IpAddress())
	}
}

func TestNewBuilder_DefaultValues(t *testing.T) {
	m := channel.NewBuilder().Build()

	if m.Id() != uuid.Nil {
		t.Errorf("Default Id() = %v, want nil UUID", m.Id())
	}
	if m.WorldId() != 0 {
		t.Errorf("Default WorldId() = %d, want 0", m.WorldId())
	}
	if m.ChannelId() != 0 {
		t.Errorf("Default ChannelId() = %d, want 0", m.ChannelId())
	}
	if m.IpAddress() != "" {
		t.Errorf("Default IpAddress() = %s, want ''", m.IpAddress())
	}
	if m.Port() != 0 {
		t.Errorf("Default Port() = %d, want 0", m.Port())
	}
}
