package drop

import (
	"testing"
	"time"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
)

func TestNewModelBuilder_DefaultValues(t *testing.T) {
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	f := field.NewBuilder(world.Id(1), channel.Id(2), _map.Id(100000000)).Build()
	mb := NewModelBuilder(ten, f)

	mbTenant := mb.Tenant()
	if mbTenant.Id() != ten.Id() {
		t.Fatal("Expected tenant to be set")
	}
	if mb.WorldId() != 1 {
		t.Fatalf("Expected worldId 1, got %d", mb.WorldId())
	}
	if mb.ChannelId() != 2 {
		t.Fatalf("Expected channelId 2, got %d", mb.ChannelId())
	}
	if mb.MapId() != 100000000 {
		t.Fatalf("Expected mapId 100000000, got %d", mb.MapId())
	}
	if mb.TransactionId() == uuid.Nil {
		t.Fatal("Expected transactionId to be generated")
	}

	m, err := mb.Build()
	if err != nil {
		t.Fatalf("Build() failed: %v", err)
	}
	if m.PetSlot() != -1 {
		t.Fatalf("Expected default petSlot -1, got %d", m.PetSlot())
	}
}

func TestModelBuilder_FluentSetters(t *testing.T) {
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	f := field.NewBuilder(world.Id(1), channel.Id(1), _map.Id(100000000)).Build()
	mb := NewModelBuilder(ten, f)

	result := mb.SetId(123)
	if result != mb {
		t.Fatal("SetId should return builder for chaining")
	}

	result = mb.SetItem(1000000, 50)
	if result != mb {
		t.Fatal("SetItem should return builder for chaining")
	}

	result = mb.SetMeso(5000)
	if result != mb {
		t.Fatal("SetMeso should return builder for chaining")
	}

	result = mb.SetType(1)
	if result != mb {
		t.Fatal("SetType should return builder for chaining")
	}

	result = mb.SetEquipmentId(99999)
	if result != mb {
		t.Fatal("SetEquipmentId should return builder for chaining")
	}

	result = mb.SetPosition(100, 200)
	if result != mb {
		t.Fatal("SetPosition should return builder for chaining")
	}

	result = mb.SetOwner(12345, 67890)
	if result != mb {
		t.Fatal("SetOwner should return builder for chaining")
	}

	result = mb.SetDropper(11111, 50, 75)
	if result != mb {
		t.Fatal("SetDropper should return builder for chaining")
	}

	result = mb.SetPlayerDrop(true)
	if result != mb {
		t.Fatal("SetPlayerDrop should return builder for chaining")
	}

	result = mb.SetStatus(StatusReserved)
	if result != mb {
		t.Fatal("SetStatus should return builder for chaining")
	}

	result = mb.SetPetSlot(2)
	if result != mb {
		t.Fatal("SetPetSlot should return builder for chaining")
	}

	txId := uuid.New()
	result = mb.SetTransactionId(txId)
	if result != mb {
		t.Fatal("SetTransactionId should return builder for chaining")
	}
}

func TestModelBuilder_Build_CreatesCorrectModel(t *testing.T) {
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	txId := uuid.New()

	f := field.NewBuilder(world.Id(1), channel.Id(2), _map.Id(100000000)).Build()
	mb := NewModelBuilder(ten, f).
		SetId(123).
		SetTransactionId(txId).
		SetItem(1000000, 50).
		SetMeso(5000).
		SetType(1).
		SetEquipmentId(99999).
		SetPosition(100, 200).
		SetOwner(12345, 67890).
		SetDropper(11111, 50, 75).
		SetPlayerDrop(true).
		SetStatus(StatusAvailable).
		SetPetSlot(2)

	m, err := mb.Build()
	if err != nil {
		t.Fatalf("Build() failed: %v", err)
	}

	if m.Id() != 123 {
		t.Fatalf("Expected Id 123, got %d", m.Id())
	}
	if m.TransactionId() != txId {
		t.Fatal("Expected transactionId to match")
	}
	if m.WorldId() != 1 {
		t.Fatalf("Expected WorldId 1, got %d", m.WorldId())
	}
	if m.ChannelId() != 2 {
		t.Fatalf("Expected ChannelId 2, got %d", m.ChannelId())
	}
	if m.MapId() != 100000000 {
		t.Fatalf("Expected MapId 100000000, got %d", m.MapId())
	}
	if m.ItemId() != 1000000 {
		t.Fatalf("Expected ItemId 1000000, got %d", m.ItemId())
	}
	if m.Quantity() != 50 {
		t.Fatalf("Expected Quantity 50, got %d", m.Quantity())
	}
	if m.Meso() != 5000 {
		t.Fatalf("Expected Meso 5000, got %d", m.Meso())
	}
	if m.Type() != 1 {
		t.Fatalf("Expected Type 1, got %d", m.Type())
	}
	if m.EquipmentId() != 99999 {
		t.Fatalf("Expected EquipmentId 99999, got %d", m.EquipmentId())
	}
	if m.X() != 100 || m.Y() != 200 {
		t.Fatalf("Expected position (100, 200), got (%d, %d)", m.X(), m.Y())
	}
	if m.OwnerId() != 12345 {
		t.Fatalf("Expected OwnerId 12345, got %d", m.OwnerId())
	}
	if m.OwnerPartyId() != 67890 {
		t.Fatalf("Expected OwnerPartyId 67890, got %d", m.OwnerPartyId())
	}
	if m.DropperId() != 11111 {
		t.Fatalf("Expected DropperId 11111, got %d", m.DropperId())
	}
	if m.DropperX() != 50 || m.DropperY() != 75 {
		t.Fatalf("Expected dropper position (50, 75), got (%d, %d)", m.DropperX(), m.DropperY())
	}
	if !m.PlayerDrop() {
		t.Fatal("Expected PlayerDrop true")
	}
	if m.Status() != StatusAvailable {
		t.Fatalf("Expected Status %s, got %s", StatusAvailable, m.Status())
	}
	if m.PetSlot() != 2 {
		t.Fatalf("Expected PetSlot 2, got %d", m.PetSlot())
	}
	mTenant := m.Tenant()
	if mTenant.Id() != ten.Id() {
		t.Fatal("Expected tenant to match")
	}
}

func TestCloneModelBuilder_CopiesAllFields(t *testing.T) {
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	txId := uuid.New()
	dropTime := time.Now().Add(-time.Hour)

	f := field.NewBuilder(world.Id(1), channel.Id(2), _map.Id(100000000)).Build()
	original, err := NewModelBuilder(ten, f).
		SetId(123).
		SetTransactionId(txId).
		SetItem(1000000, 50).
		SetMeso(5000).
		SetType(1).
		SetEquipmentId(99999).
		SetPosition(100, 200).
		SetOwner(12345, 67890).
		SetDropper(11111, 50, 75).
		SetPlayerDrop(true).
		SetStatus(StatusReserved).
		SetPetSlot(2).
		Build()
	if err != nil {
		t.Fatalf("Build() failed for original: %v", err)
	}

	original.dropTime = dropTime

	cloned, err := CloneModelBuilder(original).Build()
	if err != nil {
		t.Fatalf("Build() failed for cloned: %v", err)
	}

	if cloned.Id() != original.Id() {
		t.Fatalf("Expected Id %d, got %d", original.Id(), cloned.Id())
	}
	if cloned.TransactionId() != original.TransactionId() {
		t.Fatal("Expected transactionId to match")
	}
	if cloned.WorldId() != original.WorldId() {
		t.Fatalf("Expected WorldId %d, got %d", original.WorldId(), cloned.WorldId())
	}
	if cloned.ChannelId() != original.ChannelId() {
		t.Fatalf("Expected ChannelId %d, got %d", original.ChannelId(), cloned.ChannelId())
	}
	if cloned.MapId() != original.MapId() {
		t.Fatalf("Expected MapId %d, got %d", original.MapId(), cloned.MapId())
	}
	if cloned.ItemId() != original.ItemId() {
		t.Fatalf("Expected ItemId %d, got %d", original.ItemId(), cloned.ItemId())
	}
	if cloned.Quantity() != original.Quantity() {
		t.Fatalf("Expected Quantity %d, got %d", original.Quantity(), cloned.Quantity())
	}
	if cloned.Meso() != original.Meso() {
		t.Fatalf("Expected Meso %d, got %d", original.Meso(), cloned.Meso())
	}
	if cloned.Type() != original.Type() {
		t.Fatalf("Expected Type %d, got %d", original.Type(), cloned.Type())
	}
	if cloned.EquipmentId() != original.EquipmentId() {
		t.Fatalf("Expected EquipmentId %d, got %d", original.EquipmentId(), cloned.EquipmentId())
	}
	if cloned.X() != original.X() || cloned.Y() != original.Y() {
		t.Fatalf("Expected position (%d, %d), got (%d, %d)", original.X(), original.Y(), cloned.X(), cloned.Y())
	}
	if cloned.OwnerId() != original.OwnerId() {
		t.Fatalf("Expected OwnerId %d, got %d", original.OwnerId(), cloned.OwnerId())
	}
	if cloned.OwnerPartyId() != original.OwnerPartyId() {
		t.Fatalf("Expected OwnerPartyId %d, got %d", original.OwnerPartyId(), cloned.OwnerPartyId())
	}
	if cloned.DropperId() != original.DropperId() {
		t.Fatalf("Expected DropperId %d, got %d", original.DropperId(), cloned.DropperId())
	}
	if cloned.DropperX() != original.DropperX() || cloned.DropperY() != original.DropperY() {
		t.Fatalf("Expected dropper position (%d, %d), got (%d, %d)", original.DropperX(), original.DropperY(), cloned.DropperX(), cloned.DropperY())
	}
	if cloned.PlayerDrop() != original.PlayerDrop() {
		t.Fatalf("Expected PlayerDrop %v, got %v", original.PlayerDrop(), cloned.PlayerDrop())
	}
	if cloned.Status() != original.Status() {
		t.Fatalf("Expected Status %s, got %s", original.Status(), cloned.Status())
	}
	if cloned.PetSlot() != original.PetSlot() {
		t.Fatalf("Expected PetSlot %d, got %d", original.PetSlot(), cloned.PetSlot())
	}
	clonedTenant := cloned.Tenant()
	originalTenant := original.Tenant()
	if clonedTenant.Id() != originalTenant.Id() {
		t.Fatal("Expected tenant to match")
	}
	if !cloned.DropTime().Equal(original.DropTime()) {
		t.Fatal("Expected dropTime to match")
	}
}

func TestModel_Reserve_ReturnsNewInstance(t *testing.T) {
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)

	f := field.NewBuilder(world.Id(1), channel.Id(1), _map.Id(100000000)).Build()
	original, err := NewModelBuilder(ten, f).
		SetId(123).
		SetStatus(StatusAvailable).
		SetPetSlot(-1).
		Build()
	if err != nil {
		t.Fatalf("Build() failed: %v", err)
	}

	reserved := original.Reserve(2)

	if original.Status() != StatusAvailable {
		t.Fatal("Original model should remain AVAILABLE")
	}
	if original.PetSlot() != -1 {
		t.Fatal("Original model petSlot should remain -1")
	}

	if reserved.Status() != StatusReserved {
		t.Fatalf("Reserved model should have status %s, got %s", StatusReserved, reserved.Status())
	}
	if reserved.PetSlot() != 2 {
		t.Fatalf("Reserved model should have petSlot 2, got %d", reserved.PetSlot())
	}

	if reserved.Id() != original.Id() {
		t.Fatal("Reserved model should preserve other fields")
	}
	if reserved.WorldId() != original.WorldId() {
		t.Fatal("Reserved model should preserve other fields")
	}
}

func TestModel_CancelReservation_ReturnsNewInstance(t *testing.T) {
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)

	f := field.NewBuilder(world.Id(1), channel.Id(1), _map.Id(100000000)).Build()
	original, err := NewModelBuilder(ten, f).
		SetId(123).
		SetStatus(StatusReserved).
		SetPetSlot(2).
		Build()
	if err != nil {
		t.Fatalf("Build() failed: %v", err)
	}

	cancelled := original.CancelReservation()

	if original.Status() != StatusReserved {
		t.Fatal("Original model should remain RESERVED")
	}
	if original.PetSlot() != 2 {
		t.Fatal("Original model petSlot should remain 2")
	}

	if cancelled.Status() != StatusAvailable {
		t.Fatalf("Cancelled model should have status %s, got %s", StatusAvailable, cancelled.Status())
	}
	if cancelled.PetSlot() != -1 {
		t.Fatalf("Cancelled model should have petSlot -1, got %d", cancelled.PetSlot())
	}

	if cancelled.Id() != original.Id() {
		t.Fatal("Cancelled model should preserve other fields")
	}
	if cancelled.WorldId() != original.WorldId() {
		t.Fatal("Cancelled model should preserve other fields")
	}
}

func TestModel_CharacterDrop_AliasForPlayerDrop(t *testing.T) {
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)

	f := field.NewBuilder(world.Id(1), channel.Id(1), _map.Id(100000000)).Build()
	m, err := NewModelBuilder(ten, f).
		SetPlayerDrop(true).
		Build()
	if err != nil {
		t.Fatalf("Build() failed: %v", err)
	}

	if m.CharacterDrop() != m.PlayerDrop() {
		t.Fatal("CharacterDrop() should return same value as PlayerDrop()")
	}
}

func TestModelBuilder_ItemId_Getter(t *testing.T) {
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	f := field.NewBuilder(world.Id(1), channel.Id(1), _map.Id(100000000)).Build()
	mb := NewModelBuilder(ten, f).SetItem(1234567, 10)

	if mb.ItemId() != 1234567 {
		t.Fatalf("Expected ItemId() 1234567, got %d", mb.ItemId())
	}
}

func TestModel_AllGetters(t *testing.T) {
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	txId := uuid.New()
	dropTime := time.Now()

	m := Model{
		tenant:        ten,
		id:            123,
		transactionId: txId,
		worldId:       1,
		channelId:     2,
		mapId:         100000000,
		itemId:        1000000,
		equipmentId:   99999,
		quantity:      50,
		meso:          5000,
		dropType:      1,
		x:             100,
		y:             200,
		ownerId:       12345,
		ownerPartyId:  67890,
		dropTime:      dropTime,
		dropperId:     11111,
		dropperX:      50,
		dropperY:      75,
		playerDrop:    true,
		status:        StatusAvailable,
		petSlot:       2,
	}

	mTen := m.Tenant()
	if mTen.Id() != ten.Id() {
		t.Fatal("Tenant() failed")
	}
	if m.Id() != 123 {
		t.Fatal("Id() failed")
	}
	if m.TransactionId() != txId {
		t.Fatal("TransactionId() failed")
	}
	if m.WorldId() != 1 {
		t.Fatal("WorldId() failed")
	}
	if m.ChannelId() != 2 {
		t.Fatal("ChannelId() failed")
	}
	if m.MapId() != 100000000 {
		t.Fatal("MapId() failed")
	}
	if m.ItemId() != 1000000 {
		t.Fatal("ItemId() failed")
	}
	if m.EquipmentId() != 99999 {
		t.Fatal("EquipmentId() failed")
	}
	if m.Quantity() != 50 {
		t.Fatal("Quantity() failed")
	}
	if m.Meso() != 5000 {
		t.Fatal("Meso() failed")
	}
	if m.Type() != 1 {
		t.Fatal("Type() failed")
	}
	if m.X() != 100 {
		t.Fatal("X() failed")
	}
	if m.Y() != 200 {
		t.Fatal("Y() failed")
	}
	if m.OwnerId() != 12345 {
		t.Fatal("OwnerId() failed")
	}
	if m.OwnerPartyId() != 67890 {
		t.Fatal("OwnerPartyId() failed")
	}
	if !m.DropTime().Equal(dropTime) {
		t.Fatal("DropTime() failed")
	}
	if m.DropperId() != 11111 {
		t.Fatal("DropperId() failed")
	}
	if m.DropperX() != 50 {
		t.Fatal("DropperX() failed")
	}
	if m.DropperY() != 75 {
		t.Fatal("DropperY() failed")
	}
	if !m.PlayerDrop() {
		t.Fatal("PlayerDrop() failed")
	}
	if m.Status() != StatusAvailable {
		t.Fatal("Status() failed")
	}
	if m.PetSlot() != 2 {
		t.Fatal("PetSlot() failed")
	}
}

func TestModelBuilder_Build_ValidationErrors(t *testing.T) {
	tests := []struct {
		name        string
		buildFunc   func() (Model, error)
		expectError bool
		errorMsg    string
	}{
		{
			name: "missing tenant",
			buildFunc: func() (Model, error) {
				mb := &ModelBuilder{}
				mb.transactionId = uuid.New()
				return mb.Build()
			},
			expectError: true,
			errorMsg:    "tenant is required",
		},
		{
			name: "missing transactionId",
			buildFunc: func() (Model, error) {
				ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
				mb := &ModelBuilder{}
				mb.tenant = ten
				return mb.Build()
			},
			expectError: true,
			errorMsg:    "transactionId is required",
		},
		{
			name: "valid builder",
			buildFunc: func() (Model, error) {
				ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
				f := field.NewBuilder(world.Id(1), channel.Id(1), _map.Id(100000000)).Build()
				return NewModelBuilder(ten, f).Build()
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.buildFunc()
			if tt.expectError {
				if err == nil {
					t.Fatalf("Expected error containing '%s', got nil", tt.errorMsg)
				}
				if err.Error() != tt.errorMsg {
					t.Fatalf("Expected error '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

func TestModelBuilder_MustBuild_Panics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("Expected MustBuild to panic on invalid builder")
		}
	}()

	mb := &ModelBuilder{}
	mb.MustBuild()
}

func TestModelBuilder_MustBuild_Success(t *testing.T) {
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	f := field.NewBuilder(world.Id(1), channel.Id(1), _map.Id(100000000)).Build()
	mb := NewModelBuilder(ten, f)

	// Should not panic
	m := mb.MustBuild()

	if m.WorldId() != 1 {
		t.Fatal("Expected WorldId 1")
	}
}
