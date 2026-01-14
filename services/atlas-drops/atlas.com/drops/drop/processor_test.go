package drop

import (
	"atlas-drops/kafka/message"
	"context"
	"testing"

	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
)

func createTestContext(t *testing.T) (context.Context, tenant.Model) {
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("Failed to create test tenant: %v", err)
	}
	ctx := tenant.WithContext(context.Background(), ten)
	return ctx, ten
}

func createTestLogger() logrus.FieldLogger {
	logger, _ := test.NewNullLogger()
	return logger
}

func TestNewProcessor_CreatesProperly(t *testing.T) {
	ctx, _ := createTestContext(t)
	l := createTestLogger()

	p := NewProcessor(l, ctx)

	if p == nil {
		t.Fatal("Expected processor to be created")
	}
}

func TestProcessor_SpawnForCharacter_CreatesDropAndBuffersMessage(t *testing.T) {
	resetRegistry()
	ctx, ten := createTestContext(t)
	l := createTestLogger()

	p := NewProcessor(l, ctx)
	buf := message.NewBuffer()

	mb := NewModelBuilder(ten, 1, 1, 100000000).
		SetItem(1000000, 10).
		SetPosition(100, 200).
		SetOwner(12345, 0).
		SetDropper(99999, 50, 150)

	m, err := p.SpawnForCharacter(buf)(mb)
	if err != nil {
		t.Fatalf("Failed to spawn drop: %v", err)
	}

	if m.Id() == 0 {
		t.Fatal("Expected drop to have non-zero ID")
	}
	if m.Status() != StatusAvailable {
		t.Fatalf("Expected status %s, got %s", StatusAvailable, m.Status())
	}
	if m.ItemId() != 1000000 {
		t.Fatalf("Expected itemId 1000000, got %d", m.ItemId())
	}

	messages := buf.GetAll()
	if len(messages) == 0 {
		t.Fatal("Expected message to be buffered")
	}

	registryDrop, err := GetRegistry().GetDrop(m.Id())
	if err != nil {
		t.Fatalf("Failed to get drop from registry: %v", err)
	}
	if registryDrop.Id() != m.Id() {
		t.Fatal("Drop should be in registry")
	}
}

func TestProcessor_Reserve_SuccessfulReservation(t *testing.T) {
	resetRegistry()
	ctx, ten := createTestContext(t)
	l := createTestLogger()

	p := NewProcessor(l, ctx)
	spawnBuf := message.NewBuffer()

	mb := NewModelBuilder(ten, 1, 1, 100000000).
		SetItem(1000000, 10).
		SetPosition(100, 200)

	drop, _ := p.SpawnForCharacter(spawnBuf)(mb)

	reserveBuf := message.NewBuffer()
	txId := uuid.New()
	characterId := uint32(12345)
	petSlot := int8(-1)

	reserved, err := p.Reserve(reserveBuf)(txId, 1, 1, 100000000, drop.Id(), characterId, petSlot)
	if err != nil {
		t.Fatalf("Failed to reserve drop: %v", err)
	}

	if reserved.Status() != StatusReserved {
		t.Fatalf("Expected status %s, got %s", StatusReserved, reserved.Status())
	}

	messages := reserveBuf.GetAll()
	if len(messages) == 0 {
		t.Fatal("Expected reservation message to be buffered")
	}
}

func TestProcessor_Reserve_FailedReservation_BuffersFailureMessage(t *testing.T) {
	resetRegistry()
	ctx, ten := createTestContext(t)
	l := createTestLogger()

	p := NewProcessor(l, ctx)
	spawnBuf := message.NewBuffer()

	mb := NewModelBuilder(ten, 1, 1, 100000000).
		SetItem(1000000, 10)

	drop, _ := p.SpawnForCharacter(spawnBuf)(mb)

	reserveBuf1 := message.NewBuffer()
	txId := uuid.New()
	_, _ = p.Reserve(reserveBuf1)(txId, 1, 1, 100000000, drop.Id(), uint32(11111), -1)

	reserveBuf2 := message.NewBuffer()
	_, err := p.Reserve(reserveBuf2)(txId, 1, 1, 100000000, drop.Id(), uint32(22222), -1)
	if err == nil {
		t.Fatal("Expected error when reserving already reserved drop")
	}

	messages := reserveBuf2.GetAll()
	if len(messages) == 0 {
		t.Fatal("Expected failure message to be buffered")
	}
}

func TestProcessor_CancelReservation_BuffersMessage(t *testing.T) {
	resetRegistry()
	ctx, ten := createTestContext(t)
	l := createTestLogger()

	p := NewProcessor(l, ctx)
	spawnBuf := message.NewBuffer()

	mb := NewModelBuilder(ten, 1, 1, 100000000).
		SetItem(1000000, 10)

	drop, _ := p.SpawnForCharacter(spawnBuf)(mb)

	reserveBuf := message.NewBuffer()
	txId := uuid.New()
	characterId := uint32(12345)
	_, _ = p.Reserve(reserveBuf)(txId, 1, 1, 100000000, drop.Id(), characterId, -1)

	cancelBuf := message.NewBuffer()
	err := p.CancelReservation(cancelBuf)(txId, 1, 1, 100000000, drop.Id(), characterId)
	if err != nil {
		t.Fatalf("Failed to cancel reservation: %v", err)
	}

	messages := cancelBuf.GetAll()
	if len(messages) == 0 {
		t.Fatal("Expected cancellation message to be buffered")
	}

	updated, _ := GetRegistry().GetDrop(drop.Id())
	if updated.Status() != StatusAvailable {
		t.Fatalf("Expected status %s after cancellation, got %s", StatusAvailable, updated.Status())
	}
}

func TestProcessor_Gather_RemovesDropAndBuffersMessage(t *testing.T) {
	resetRegistry()
	ctx, ten := createTestContext(t)
	l := createTestLogger()

	p := NewProcessor(l, ctx)
	spawnBuf := message.NewBuffer()

	mb := NewModelBuilder(ten, 1, 1, 100000000).
		SetItem(1000000, 10)

	drop, _ := p.SpawnForCharacter(spawnBuf)(mb)

	gatherBuf := message.NewBuffer()
	txId := uuid.New()
	characterId := uint32(12345)

	gathered, err := p.Gather(gatherBuf)(txId, 1, 1, 100000000, drop.Id(), characterId)
	if err != nil {
		t.Fatalf("Failed to gather drop: %v", err)
	}

	if gathered.Id() != drop.Id() {
		t.Fatal("Expected gathered drop to match original")
	}

	messages := gatherBuf.GetAll()
	if len(messages) == 0 {
		t.Fatal("Expected gather message to be buffered")
	}

	_, err = GetRegistry().GetDrop(drop.Id())
	if err == nil {
		t.Fatal("Expected drop to be removed from registry")
	}
}

func TestProcessor_Expire_RemovesDropAndBuffersMessage(t *testing.T) {
	resetRegistry()
	ctx, ten := createTestContext(t)
	l := createTestLogger()

	p := NewProcessor(l, ctx)
	spawnBuf := message.NewBuffer()

	mb := NewModelBuilder(ten, 1, 1, 100000000).
		SetItem(1000000, 10)

	drop, _ := p.SpawnForCharacter(spawnBuf)(mb)

	expireBuf := message.NewBuffer()
	err := p.Expire(expireBuf)(drop)
	if err != nil {
		t.Fatalf("Failed to expire drop: %v", err)
	}

	messages := expireBuf.GetAll()
	if len(messages) == 0 {
		t.Fatal("Expected expire message to be buffered")
	}

	_, err = GetRegistry().GetDrop(drop.Id())
	if err == nil {
		t.Fatal("Expected drop to be removed from registry")
	}
}

func TestProcessor_GetById_ReturnsCorrectDrop(t *testing.T) {
	resetRegistry()
	ctx, ten := createTestContext(t)
	l := createTestLogger()

	p := NewProcessor(l, ctx)
	buf := message.NewBuffer()

	mb := NewModelBuilder(ten, 1, 1, 100000000).
		SetItem(1000000, 10)

	created, _ := p.SpawnForCharacter(buf)(mb)

	found, err := p.GetById(created.Id())
	if err != nil {
		t.Fatalf("Failed to get drop by ID: %v", err)
	}

	if found.Id() != created.Id() {
		t.Fatalf("Expected ID %d, got %d", created.Id(), found.Id())
	}
	if found.ItemId() != created.ItemId() {
		t.Fatalf("Expected ItemId %d, got %d", created.ItemId(), found.ItemId())
	}
}

func TestProcessor_GetById_NonExistent(t *testing.T) {
	resetRegistry()
	ctx, _ := createTestContext(t)
	l := createTestLogger()

	p := NewProcessor(l, ctx)

	_, err := p.GetById(999999)
	if err == nil {
		t.Fatal("Expected error when getting non-existent drop")
	}
}

func TestProcessor_GetForMap_ReturnsFilteredDrops(t *testing.T) {
	resetRegistry()
	ctx, ten := createTestContext(t)
	l := createTestLogger()

	p := NewProcessor(l, ctx)
	buf := message.NewBuffer()

	mb1 := NewModelBuilder(ten, 1, 1, 100000000).SetItem(1000001, 10)
	mb2 := NewModelBuilder(ten, 1, 1, 100000000).SetItem(1000002, 20)
	mb3 := NewModelBuilder(ten, 1, 1, 200000000).SetItem(1000003, 30)

	drop1, _ := p.SpawnForCharacter(buf)(mb1)
	drop2, _ := p.SpawnForCharacter(buf)(mb2)
	_, _ = p.SpawnForCharacter(buf)(mb3)

	drops, err := p.GetForMap(1, 1, 100000000)
	if err != nil {
		t.Fatalf("Failed to get drops for map: %v", err)
	}

	if len(drops) != 2 {
		t.Fatalf("Expected 2 drops for map 100000000, got %d", len(drops))
	}

	foundIds := make(map[uint32]bool)
	for _, d := range drops {
		foundIds[d.Id()] = true
	}
	if !foundIds[drop1.Id()] || !foundIds[drop2.Id()] {
		t.Fatal("Expected to find both drops for map 100000000")
	}
}

func TestProcessor_ByIdProvider_WorksWithModelProvider(t *testing.T) {
	resetRegistry()
	ctx, ten := createTestContext(t)
	l := createTestLogger()

	p := NewProcessor(l, ctx)
	buf := message.NewBuffer()

	mb := NewModelBuilder(ten, 1, 1, 100000000).SetItem(1000000, 10)
	created, _ := p.SpawnForCharacter(buf)(mb)

	provider := p.ByIdProvider(created.Id())
	found, err := provider()
	if err != nil {
		t.Fatalf("Provider failed: %v", err)
	}

	if found.Id() != created.Id() {
		t.Fatalf("Expected ID %d, got %d", created.Id(), found.Id())
	}
}

func TestProcessor_ForMapProvider_WorksWithSliceProvider(t *testing.T) {
	resetRegistry()
	ctx, ten := createTestContext(t)
	l := createTestLogger()

	p := NewProcessor(l, ctx)
	buf := message.NewBuffer()

	mb1 := NewModelBuilder(ten, 1, 1, 100000000).SetItem(1000001, 10)
	mb2 := NewModelBuilder(ten, 1, 1, 100000000).SetItem(1000002, 20)

	p.SpawnForCharacter(buf)(mb1)
	p.SpawnForCharacter(buf)(mb2)

	provider := p.ForMapProvider(1, 1, 100000000)
	drops, err := provider()
	if err != nil {
		t.Fatalf("Provider failed: %v", err)
	}

	if len(drops) != 2 {
		t.Fatalf("Expected 2 drops, got %d", len(drops))
	}
}

func TestAllProvider_ReturnsAllDrops(t *testing.T) {
	resetRegistry()
	ctx, ten := createTestContext(t)
	l := createTestLogger()

	p := NewProcessor(l, ctx)
	buf := message.NewBuffer()

	mb1 := NewModelBuilder(ten, 1, 1, 100000000).SetItem(1000001, 10)
	mb2 := NewModelBuilder(ten, 1, 2, 200000000).SetItem(1000002, 20)
	mb3 := NewModelBuilder(ten, 2, 1, 300000000).SetItem(1000003, 30)

	p.SpawnForCharacter(buf)(mb1)
	p.SpawnForCharacter(buf)(mb2)
	p.SpawnForCharacter(buf)(mb3)

	drops, err := AllProvider()
	if err != nil {
		t.Fatalf("AllProvider failed: %v", err)
	}

	if len(drops) != 3 {
		t.Fatalf("Expected 3 drops, got %d", len(drops))
	}
}

func TestProcessor_Reserve_WithPetSlot(t *testing.T) {
	resetRegistry()
	ctx, ten := createTestContext(t)
	l := createTestLogger()

	p := NewProcessor(l, ctx)
	spawnBuf := message.NewBuffer()

	mb := NewModelBuilder(ten, 1, 1, 100000000).SetItem(1000000, 10)
	drop, _ := p.SpawnForCharacter(spawnBuf)(mb)

	reserveBuf := message.NewBuffer()
	txId := uuid.New()
	petSlot := int8(2)

	reserved, err := p.Reserve(reserveBuf)(txId, 1, 1, 100000000, drop.Id(), uint32(12345), petSlot)
	if err != nil {
		t.Fatalf("Failed to reserve drop with pet slot: %v", err)
	}

	if reserved.PetSlot() != petSlot {
		t.Fatalf("Expected petSlot %d, got %d", petSlot, reserved.PetSlot())
	}
}

func TestProcessor_MultipleOperationsSequence(t *testing.T) {
	resetRegistry()
	ctx, ten := createTestContext(t)
	l := createTestLogger()

	p := NewProcessor(l, ctx)

	spawnBuf := message.NewBuffer()
	mb := NewModelBuilder(ten, 1, 1, 100000000).SetItem(1000000, 10)
	drop, _ := p.SpawnForCharacter(spawnBuf)(mb)

	found, err := p.GetById(drop.Id())
	if err != nil {
		t.Fatalf("Step 1 failed - GetById: %v", err)
	}
	if found.Status() != StatusAvailable {
		t.Fatalf("Step 1 - Expected status %s, got %s", StatusAvailable, found.Status())
	}

	reserveBuf := message.NewBuffer()
	txId := uuid.New()
	characterId := uint32(12345)
	_, err = p.Reserve(reserveBuf)(txId, 1, 1, 100000000, drop.Id(), characterId, -1)
	if err != nil {
		t.Fatalf("Step 2 failed - Reserve: %v", err)
	}

	found, _ = p.GetById(drop.Id())
	if found.Status() != StatusReserved {
		t.Fatalf("Step 2 - Expected status %s, got %s", StatusReserved, found.Status())
	}

	cancelBuf := message.NewBuffer()
	err = p.CancelReservation(cancelBuf)(txId, 1, 1, 100000000, drop.Id(), characterId)
	if err != nil {
		t.Fatalf("Step 3 failed - CancelReservation: %v", err)
	}

	found, _ = p.GetById(drop.Id())
	if found.Status() != StatusAvailable {
		t.Fatalf("Step 3 - Expected status %s, got %s", StatusAvailable, found.Status())
	}

	gatherBuf := message.NewBuffer()
	_, err = p.Gather(gatherBuf)(txId, 1, 1, 100000000, drop.Id(), characterId)
	if err != nil {
		t.Fatalf("Step 4 failed - Gather: %v", err)
	}

	_, err = p.GetById(drop.Id())
	if err == nil {
		t.Fatal("Step 4 - Expected drop to be removed")
	}
}

func TestCreatedEventStatusProvider_ReturnsValidMessages(t *testing.T) {
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)

	m, _ := NewModelBuilder(ten, 1, 2, 100000000).
		SetId(12345).
		SetTransactionId(uuid.New()).
		SetItem(1000000, 10).
		SetMeso(0).
		SetType(1).
		SetPosition(100, 200).
		SetOwner(99999, 0).
		SetDropper(88888, 50, 150).
		SetPlayerDrop(false).
		SetStatus(StatusAvailable).
		Build()

	provider := createdEventStatusProvider(m)
	messages, err := provider()
	if err != nil {
		t.Fatalf("createdEventStatusProvider failed: %v", err)
	}
	if len(messages) == 0 {
		t.Fatal("Expected at least one message")
	}
}

func TestExpiredEventStatusProvider_ReturnsValidMessages(t *testing.T) {
	txId := uuid.New()

	provider := expiredEventStatusProvider(txId, 1, 2, 100000000, 12345)
	messages, err := provider()
	if err != nil {
		t.Fatalf("expiredEventStatusProvider failed: %v", err)
	}
	if len(messages) == 0 {
		t.Fatal("Expected at least one message")
	}
}

func TestPickedUpEventStatusProvider_ReturnsValidMessages(t *testing.T) {
	txId := uuid.New()

	provider := pickedUpEventStatusProvider(txId, 1, 2, 100000000, 12345, 99999, 1000000, 0, 10, 0, -1)
	messages, err := provider()
	if err != nil {
		t.Fatalf("pickedUpEventStatusProvider failed: %v", err)
	}
	if len(messages) == 0 {
		t.Fatal("Expected at least one message")
	}
}

func TestReservedEventStatusProvider_ReturnsValidMessages(t *testing.T) {
	txId := uuid.New()

	provider := reservedEventStatusProvider(txId, 1, 2, 100000000, 12345, 99999, 1000000, 0, 10, 0)
	messages, err := provider()
	if err != nil {
		t.Fatalf("reservedEventStatusProvider failed: %v", err)
	}
	if len(messages) == 0 {
		t.Fatal("Expected at least one message")
	}
}

func TestReservationFailureEventStatusProvider_ReturnsValidMessages(t *testing.T) {
	txId := uuid.New()

	provider := reservationFailureEventStatusProvider(txId, 1, 2, 100000000, 12345, 99999)
	messages, err := provider()
	if err != nil {
		t.Fatalf("reservationFailureEventStatusProvider failed: %v", err)
	}
	if len(messages) == 0 {
		t.Fatal("Expected at least one message")
	}
}

func TestProcessor_Gather_NonExistentDrop(t *testing.T) {
	resetRegistry()
	ctx, _ := createTestContext(t)
	l := createTestLogger()

	p := NewProcessor(l, ctx)
	gatherBuf := message.NewBuffer()
	txId := uuid.New()

	gathered, err := p.Gather(gatherBuf)(txId, 1, 1, 100000000, 999999, uint32(12345))
	// RemoveDrop returns empty model for non-existent drop without error
	if gathered.Id() != 0 {
		t.Fatal("Expected zero-value model for non-existent drop")
	}
	// Should not buffer message when drop doesn't exist (d.Id() == 0)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestProcessor_Expire_NonExistentDrop(t *testing.T) {
	resetRegistry()
	ctx, ten := createTestContext(t)
	l := createTestLogger()

	p := NewProcessor(l, ctx)
	expireBuf := message.NewBuffer()

	// Create a model that references a non-existent drop in registry
	m, _ := NewModelBuilder(ten, 1, 1, 100000000).
		SetId(999999).
		SetStatus(StatusAvailable).
		Build()

	err := p.Expire(expireBuf)(m)
	// RemoveDrop returns nil error for non-existent drops (empty model returned)
	// So Expire should succeed but not buffer any message since the drop wasn't in registry
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestProcessor_GetForMap_EmptyMap(t *testing.T) {
	resetRegistry()
	ctx, _ := createTestContext(t)
	l := createTestLogger()

	p := NewProcessor(l, ctx)

	drops, err := p.GetForMap(1, 1, 999999999)
	if err != nil {
		t.Fatalf("Failed to get drops for empty map: %v", err)
	}
	if len(drops) != 0 {
		t.Fatalf("Expected 0 drops for empty map, got %d", len(drops))
	}
}

func TestProcessor_Reserve_NonExistentDrop(t *testing.T) {
	resetRegistry()
	ctx, _ := createTestContext(t)
	l := createTestLogger()

	p := NewProcessor(l, ctx)
	reserveBuf := message.NewBuffer()
	txId := uuid.New()

	_, err := p.Reserve(reserveBuf)(txId, 1, 1, 100000000, 999999, uint32(12345), -1)
	if err == nil {
		t.Fatal("Expected error when reserving non-existent drop")
	}

	// Should still buffer a failure message
	messages := reserveBuf.GetAll()
	if len(messages) == 0 {
		t.Fatal("Expected failure message to be buffered")
	}
}

func TestProcessor_CancelReservation_NonExistentDrop(t *testing.T) {
	resetRegistry()
	ctx, _ := createTestContext(t)
	l := createTestLogger()

	p := NewProcessor(l, ctx)
	cancelBuf := message.NewBuffer()
	txId := uuid.New()

	// This should not error - just silently ignore
	err := p.CancelReservation(cancelBuf)(txId, 1, 1, 100000000, 999999, uint32(12345))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should still buffer a message
	messages := cancelBuf.GetAll()
	if len(messages) == 0 {
		t.Fatal("Expected message to be buffered")
	}
}

func TestProcessor_Gather_WithMesoDrop(t *testing.T) {
	resetRegistry()
	ctx, ten := createTestContext(t)
	l := createTestLogger()

	p := NewProcessor(l, ctx)
	spawnBuf := message.NewBuffer()

	// Create a meso drop
	mb := NewModelBuilder(ten, 1, 1, 100000000).
		SetMeso(1000).
		SetType(3)

	drop, _ := p.SpawnForCharacter(spawnBuf)(mb)

	gatherBuf := message.NewBuffer()
	txId := uuid.New()

	gathered, err := p.Gather(gatherBuf)(txId, 1, 1, 100000000, drop.Id(), uint32(12345))
	if err != nil {
		t.Fatalf("Failed to gather meso drop: %v", err)
	}
	if gathered.Meso() != 1000 {
		t.Fatalf("Expected meso 1000, got %d", gathered.Meso())
	}
}

func TestProcessor_ByIdProvider_NonExistent(t *testing.T) {
	resetRegistry()
	ctx, _ := createTestContext(t)
	l := createTestLogger()

	p := NewProcessor(l, ctx)

	provider := p.ByIdProvider(999999)
	_, err := provider()
	if err == nil {
		t.Fatal("Expected error for non-existent drop")
	}
}

func TestProcessor_ForMapProvider_EmptyMap(t *testing.T) {
	resetRegistry()
	ctx, _ := createTestContext(t)
	l := createTestLogger()

	p := NewProcessor(l, ctx)

	provider := p.ForMapProvider(1, 1, 999999999)
	drops, err := provider()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(drops) != 0 {
		t.Fatalf("Expected 0 drops, got %d", len(drops))
	}
}

func TestAllProvider_EmptyRegistry(t *testing.T) {
	resetRegistry()

	drops, err := AllProvider()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(drops) != 0 {
		t.Fatalf("Expected 0 drops, got %d", len(drops))
	}
}

func TestProcessor_Gather_WithItemDrop(t *testing.T) {
	resetRegistry()
	ctx, ten := createTestContext(t)
	l := createTestLogger()

	p := NewProcessor(l, ctx)
	spawnBuf := message.NewBuffer()

	mb := NewModelBuilder(ten, 1, 1, 100000000).
		SetItem(2000000, 5).
		SetType(1).
		SetOwner(12345, 67890)

	drop, _ := p.SpawnForCharacter(spawnBuf)(mb)

	gatherBuf := message.NewBuffer()
	txId := uuid.New()

	gathered, err := p.Gather(gatherBuf)(txId, 1, 1, 100000000, drop.Id(), uint32(12345))
	if err != nil {
		t.Fatalf("Failed to gather item drop: %v", err)
	}
	if gathered.ItemId() != 2000000 {
		t.Fatalf("Expected itemId 2000000, got %d", gathered.ItemId())
	}
	if gathered.Quantity() != 5 {
		t.Fatalf("Expected quantity 5, got %d", gathered.Quantity())
	}
}
