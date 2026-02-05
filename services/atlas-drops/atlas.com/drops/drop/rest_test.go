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

func TestRestModel_GetName(t *testing.T) {
	rm := RestModel{}

	name := rm.GetName()
	if name != "drops" {
		t.Fatalf("Expected GetName() to return 'drops', got '%s'", name)
	}
}

func TestRestModel_GetID(t *testing.T) {
	rm := RestModel{Id: 12345}

	id := rm.GetID()
	if id != "12345" {
		t.Fatalf("Expected GetID() to return '12345', got '%s'", id)
	}
}

func TestRestModel_GetID_Zero(t *testing.T) {
	rm := RestModel{Id: 0}

	id := rm.GetID()
	if id != "0" {
		t.Fatalf("Expected GetID() to return '0', got '%s'", id)
	}
}

func TestRestModel_GetID_LargeNumber(t *testing.T) {
	rm := RestModel{Id: 4294967295}

	id := rm.GetID()
	if id != "4294967295" {
		t.Fatalf("Expected GetID() to return '4294967295', got '%s'", id)
	}
}

func TestRestModel_SetID_ValidNumber(t *testing.T) {
	rm := RestModel{}

	err := rm.SetID("12345")
	if err != nil {
		t.Fatalf("SetID failed: %v", err)
	}
	if rm.Id != 12345 {
		t.Fatalf("Expected Id to be 12345, got %d", rm.Id)
	}
}

func TestRestModel_SetID_Zero(t *testing.T) {
	rm := RestModel{}

	err := rm.SetID("0")
	if err != nil {
		t.Fatalf("SetID failed: %v", err)
	}
	if rm.Id != 0 {
		t.Fatalf("Expected Id to be 0, got %d", rm.Id)
	}
}

func TestRestModel_SetID_InvalidNumber(t *testing.T) {
	rm := RestModel{}

	err := rm.SetID("not-a-number")
	if err == nil {
		t.Fatal("Expected SetID to fail for non-numeric string")
	}
}

func TestRestModel_SetID_NegativeNumber(t *testing.T) {
	rm := RestModel{}

	err := rm.SetID("-1")
	if err == nil {
		t.Fatal("Expected SetID to fail for negative number")
	}
}

func TestRestModel_SetID_EmptyString(t *testing.T) {
	rm := RestModel{}

	err := rm.SetID("")
	if err == nil {
		t.Fatal("Expected SetID to fail for empty string")
	}
}

func TestTransform_AllFields(t *testing.T) {
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	dropTime := time.Now()

	m := Model{
		tenant:       ten,
		id:           123,
		field:        field.NewBuilder(1, 2, 100000000).Build(),
		itemId:       1000000,
		equipmentId:  99999,
		quantity:     50,
		meso:         5000,
		dropType:     1,
		x:            100,
		y:            200,
		ownerId:      12345,
		ownerPartyId: 67890,
		dropTime:     dropTime,
		dropperId:    11111,
		dropperX:     50,
		dropperY:     75,
		playerDrop:   true,
	}

	rm, err := Transform(m)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	if rm.Id != m.Id() {
		t.Fatalf("Expected Id %d, got %d", m.Id(), rm.Id)
	}
	if rm.WorldId != m.WorldId() {
		t.Fatalf("Expected WorldId %d, got %d", m.WorldId(), rm.WorldId)
	}
	if rm.ChannelId != m.ChannelId() {
		t.Fatalf("Expected ChannelId %d, got %d", m.ChannelId(), rm.ChannelId)
	}
	if rm.MapId != m.MapId() {
		t.Fatalf("Expected MapId %d, got %d", m.MapId(), rm.MapId)
	}
	if rm.ItemId != m.ItemId() {
		t.Fatalf("Expected ItemId %d, got %d", m.ItemId(), rm.ItemId)
	}
	if rm.EquipmentId != m.EquipmentId() {
		t.Fatalf("Expected EquipmentId %d, got %d", m.EquipmentId(), rm.EquipmentId)
	}
	if rm.Quantity != m.Quantity() {
		t.Fatalf("Expected Quantity %d, got %d", m.Quantity(), rm.Quantity)
	}
	if rm.Meso != m.Meso() {
		t.Fatalf("Expected Meso %d, got %d", m.Meso(), rm.Meso)
	}
	if rm.Type != m.Type() {
		t.Fatalf("Expected Type %d, got %d", m.Type(), rm.Type)
	}
	if rm.X != m.X() {
		t.Fatalf("Expected X %d, got %d", m.X(), rm.X)
	}
	if rm.Y != m.Y() {
		t.Fatalf("Expected Y %d, got %d", m.Y(), rm.Y)
	}
	if rm.OwnerId != m.OwnerId() {
		t.Fatalf("Expected OwnerId %d, got %d", m.OwnerId(), rm.OwnerId)
	}
	if rm.OwnerPartyId != m.OwnerPartyId() {
		t.Fatalf("Expected OwnerPartyId %d, got %d", m.OwnerPartyId(), rm.OwnerPartyId)
	}
	if !rm.DropTime.Equal(m.DropTime()) {
		t.Fatal("Expected DropTime to match")
	}
	if rm.DropperId != m.DropperId() {
		t.Fatalf("Expected DropperId %d, got %d", m.DropperId(), rm.DropperId)
	}
	if rm.DropperX != m.DropperX() {
		t.Fatalf("Expected DropperX %d, got %d", m.DropperX(), rm.DropperX)
	}
	if rm.DropperY != m.DropperY() {
		t.Fatalf("Expected DropperY %d, got %d", m.DropperY(), rm.DropperY)
	}
	if rm.CharacterDrop != m.CharacterDrop() {
		t.Fatalf("Expected CharacterDrop %v, got %v", m.CharacterDrop(), rm.CharacterDrop)
	}
}

func TestTransform_ZeroValues(t *testing.T) {
	m := Model{}

	rm, err := Transform(m)
	if err != nil {
		t.Fatalf("Transform failed for zero model: %v", err)
	}

	if rm.Id != 0 {
		t.Fatalf("Expected Id 0, got %d", rm.Id)
	}
	if rm.WorldId != 0 {
		t.Fatalf("Expected WorldId 0, got %d", rm.WorldId)
	}
	if rm.ItemId != 0 {
		t.Fatalf("Expected ItemId 0, got %d", rm.ItemId)
	}
	if rm.CharacterDrop != false {
		t.Fatal("Expected CharacterDrop false")
	}
}

func TestTransform_MesoDrop(t *testing.T) {
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)

	f := field.NewBuilder(world.Id(1), channel.Id(1), _map.Id(100000000)).Build()
	m, err := NewModelBuilder(ten, f).
		SetId(456).
		SetMeso(10000).
		SetType(3).
		Build()
	if err != nil {
		t.Fatalf("Build() failed: %v", err)
	}

	rm, err := Transform(m)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}

	if rm.Meso != 10000 {
		t.Fatalf("Expected Meso 10000, got %d", rm.Meso)
	}
	if rm.ItemId != 0 {
		t.Fatalf("Expected ItemId 0 for meso drop, got %d", rm.ItemId)
	}
}

func TestRestModel_JSONTags(t *testing.T) {
	rm := RestModel{
		Id:            123,
		WorldId:       1,
		ChannelId:     2,
		MapId:         100000000,
		ItemId:        1000000,
		EquipmentId:   99999,
		Quantity:      50,
		Meso:          5000,
		Type:          1,
		X:             100,
		Y:             200,
		OwnerId:       12345,
		OwnerPartyId:  67890,
		DropTime:      time.Now(),
		DropperId:     11111,
		DropperX:      50,
		DropperY:      75,
		CharacterDrop: true,
		Mod:           0,
	}

	if rm.GetName() != "drops" {
		t.Fatal("GetName should return 'drops' for JSON:API resource name")
	}

	if rm.GetID() != "123" {
		t.Fatal("GetID should return string representation of Id")
	}
}
