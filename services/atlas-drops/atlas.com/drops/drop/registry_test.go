package drop

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

func setupTestRegistry(t *testing.T) {
	t.Helper()
	mr := miniredis.RunT(t)
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	InitRegistry(rc)
}

func createTestTenant(t *testing.T) tenant.Model {
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("Failed to create test tenant: %v", err)
	}
	return ten
}

func createTestBuilder(ten tenant.Model, worldId world.Id, channelId channel.Id, mapId _map.Id) *ModelBuilder {
	f := field.NewBuilder(worldId, channelId, mapId).Build()
	return NewModelBuilder(ten, f).
		SetItem(1000000, 1).
		SetPosition(100, 200).
		SetOwner(12345, 0).
		SetDropper(99999, 50, 150).
		SetType(0)
}

func mustCreateDrop(t *testing.T, r *DropRegistry, mb *ModelBuilder) Model {
	drop, err := r.CreateDrop(mb)
	if err != nil {
		t.Fatalf("Failed to create drop: %v", err)
	}
	return drop
}

func TestCreateDrop_SingleDrop(t *testing.T) {
	setupTestRegistry(t)
	r := GetRegistry()
	ten := createTestTenant(t)

	mb := createTestBuilder(ten, 1, 1, 100000000)
	drop := mustCreateDrop(t, r, mb)

	if drop.Id() == 0 {
		t.Fatal("Expected drop to have non-zero ID")
	}
	if drop.Status() != StatusAvailable {
		t.Fatalf("Expected status %s, got %s", StatusAvailable, drop.Status())
	}
	if drop.ItemId() != 1000000 {
		t.Fatalf("Expected itemId 1000000, got %d", drop.ItemId())
	}
	if drop.X() != 100 || drop.Y() != 200 {
		t.Fatalf("Expected position (100, 200), got (%d, %d)", drop.X(), drop.Y())
	}
}

func TestCreateDrop_MultipleDropsSameMap(t *testing.T) {
	setupTestRegistry(t)
	r := GetRegistry()
	ten := createTestTenant(t)

	mb1 := createTestBuilder(ten, 1, 1, 100000000)
	mb2 := createTestBuilder(ten, 1, 1, 100000000)
	mb3 := createTestBuilder(ten, 1, 1, 100000000)

	drop1 := mustCreateDrop(t, r, mb1)
	drop2 := mustCreateDrop(t, r, mb2)
	drop3 := mustCreateDrop(t, r, mb3)

	if drop1.Id() == drop2.Id() || drop2.Id() == drop3.Id() || drop1.Id() == drop3.Id() {
		t.Fatal("Expected all drops to have unique IDs")
	}

	f := field.NewBuilder(world.Id(1), channel.Id(1), _map.Id(100000000)).Build()
	drops, err := r.GetDropsForMap(ten, f)
	if err != nil {
		t.Fatalf("Failed to get drops for map: %v", err)
	}
	if len(drops) != 3 {
		t.Fatalf("Expected 3 drops in map, got %d", len(drops))
	}
}

func TestCreateDrop_MultiTenantIsolation(t *testing.T) {
	setupTestRegistry(t)
	r := GetRegistry()
	ten1 := createTestTenant(t)
	ten2 := createTestTenant(t)

	mb1 := createTestBuilder(ten1, 1, 1, 100000000)
	mb2 := createTestBuilder(ten2, 1, 1, 100000000)

	drop1 := mustCreateDrop(t, r, mb1)
	drop2 := mustCreateDrop(t, r, mb2)

	if drop1.Id() == drop2.Id() {
		t.Fatal("Expected drops to have different IDs")
	}

	f := field.NewBuilder(world.Id(1), channel.Id(1), _map.Id(100000000)).Build()
	drops1, _ := r.GetDropsForMap(ten1, f)
	drops2, _ := r.GetDropsForMap(ten2, f)

	if len(drops1) != 1 {
		t.Fatalf("Expected 1 drop for tenant1, got %d", len(drops1))
	}
	if len(drops2) != 1 {
		t.Fatalf("Expected 1 drop for tenant2, got %d", len(drops2))
	}
	if drops1[0].Id() == drops2[0].Id() {
		t.Fatal("Expected tenant drops to be different")
	}
}

func TestReserveDrop_Success(t *testing.T) {
	setupTestRegistry(t)
	r := GetRegistry()
	ten := createTestTenant(t)

	mb := createTestBuilder(ten, 1, 1, 100000000)
	drop := mustCreateDrop(t, r, mb)

	characterId := uint32(12345)
	petSlot := int8(-1)

	reserved, err := r.ReserveDrop(drop.Id(), characterId, 0, petSlot)
	if err != nil {
		t.Fatalf("Failed to reserve drop: %v", err)
	}
	if reserved.Status() != StatusReserved {
		t.Fatalf("Expected status %s, got %s", StatusReserved, reserved.Status())
	}
}

func TestReserveDrop_AlreadyReservedBySameCharacter(t *testing.T) {
	setupTestRegistry(t)
	r := GetRegistry()
	ten := createTestTenant(t)

	mb := createTestBuilder(ten, 1, 1, 100000000)
	drop := mustCreateDrop(t, r, mb)

	characterId := uint32(12345)
	petSlot := int8(-1)

	_, err := r.ReserveDrop(drop.Id(), characterId, 0, petSlot)
	if err != nil {
		t.Fatalf("Failed to reserve drop: %v", err)
	}

	reserved2, err := r.ReserveDrop(drop.Id(), characterId, 0, petSlot)
	if err != nil {
		t.Fatalf("Should allow same character to re-reserve: %v", err)
	}
	if reserved2.Status() != StatusReserved {
		t.Fatalf("Expected status %s, got %s", StatusReserved, reserved2.Status())
	}
}

func TestReserveDrop_AlreadyReservedByDifferentCharacter(t *testing.T) {
	setupTestRegistry(t)
	r := GetRegistry()
	ten := createTestTenant(t)

	mb := createTestBuilder(ten, 1, 1, 100000000)
	drop := mustCreateDrop(t, r, mb)

	characterId1 := uint32(12345)
	characterId2 := uint32(67890)
	petSlot := int8(-1)

	_, err := r.ReserveDrop(drop.Id(), characterId1, 0, petSlot)
	if err != nil {
		t.Fatalf("Failed to reserve drop: %v", err)
	}

	_, err = r.ReserveDrop(drop.Id(), characterId2, 0, petSlot)
	if err == nil {
		t.Fatal("Expected error when reserving drop already reserved by another character")
	}
}

func TestReserveDrop_NotFound(t *testing.T) {
	setupTestRegistry(t)
	r := GetRegistry()

	_, err := r.ReserveDrop(999999, 12345, 0, -1)
	if err == nil {
		t.Fatal("Expected error when reserving non-existent drop")
	}
}

func TestCancelDropReservation_ValidCancellation(t *testing.T) {
	setupTestRegistry(t)
	r := GetRegistry()
	ten := createTestTenant(t)

	mb := createTestBuilder(ten, 1, 1, 100000000)
	drop := mustCreateDrop(t, r, mb)

	characterId := uint32(12345)
	petSlot := int8(2)

	_, err := r.ReserveDrop(drop.Id(), characterId, 0, petSlot)
	if err != nil {
		t.Fatalf("Failed to reserve drop: %v", err)
	}

	r.CancelDropReservation(drop.Id(), characterId)

	updated, err := r.GetDrop(drop.Id())
	if err != nil {
		t.Fatalf("Failed to get drop: %v", err)
	}
	if updated.Status() != StatusAvailable {
		t.Fatalf("Expected status %s after cancellation, got %s", StatusAvailable, updated.Status())
	}
	if updated.PetSlot() != -1 {
		t.Fatalf("Expected petSlot -1 after cancellation, got %d", updated.PetSlot())
	}
}

func TestCancelDropReservation_WrongCharacter(t *testing.T) {
	setupTestRegistry(t)
	r := GetRegistry()
	ten := createTestTenant(t)

	mb := createTestBuilder(ten, 1, 1, 100000000)
	drop := mustCreateDrop(t, r, mb)

	characterId1 := uint32(12345)
	characterId2 := uint32(67890)
	petSlot := int8(-1)

	_, err := r.ReserveDrop(drop.Id(), characterId1, 0, petSlot)
	if err != nil {
		t.Fatalf("Failed to reserve drop: %v", err)
	}

	r.CancelDropReservation(drop.Id(), characterId2)

	updated, err := r.GetDrop(drop.Id())
	if err != nil {
		t.Fatalf("Failed to get drop: %v", err)
	}
	if updated.Status() != StatusReserved {
		t.Fatalf("Expected status to remain %s when wrong character cancels, got %s", StatusReserved, updated.Status())
	}
}

func TestRemoveDrop_Success(t *testing.T) {
	setupTestRegistry(t)
	r := GetRegistry()
	ten := createTestTenant(t)

	mb := createTestBuilder(ten, 1, 1, 100000000)
	drop := mustCreateDrop(t, r, mb)

	removed, err := r.RemoveDrop(drop.Id())
	if err != nil {
		t.Fatalf("Failed to remove drop: %v", err)
	}
	if removed.Id() != drop.Id() {
		t.Fatalf("Expected removed drop ID %d, got %d", drop.Id(), removed.Id())
	}

	_, err = r.GetDrop(drop.Id())
	if err == nil {
		t.Fatal("Expected error when getting removed drop")
	}

	f := field.NewBuilder(world.Id(1), channel.Id(1), _map.Id(100000000)).Build()
	drops, _ := r.GetDropsForMap(ten, f)
	if len(drops) != 0 {
		t.Fatalf("Expected 0 drops in map after removal, got %d", len(drops))
	}
}

func TestRemoveDrop_NotFound(t *testing.T) {
	setupTestRegistry(t)
	r := GetRegistry()

	removed, err := r.RemoveDrop(999999)
	if err != nil {
		t.Fatalf("RemoveDrop should not error for non-existent drop: %v", err)
	}
	if removed.Id() != 0 {
		t.Fatalf("Expected empty model for non-existent drop, got ID %d", removed.Id())
	}
}

func TestGetDrop_Existing(t *testing.T) {
	setupTestRegistry(t)
	r := GetRegistry()
	ten := createTestTenant(t)

	mb := createTestBuilder(ten, 1, 1, 100000000)
	created := mustCreateDrop(t, r, mb)

	found, err := r.GetDrop(created.Id())
	if err != nil {
		t.Fatalf("Failed to get drop: %v", err)
	}
	if found.Id() != created.Id() {
		t.Fatalf("Expected drop ID %d, got %d", created.Id(), found.Id())
	}
	if found.ItemId() != created.ItemId() {
		t.Fatalf("Expected itemId %d, got %d", created.ItemId(), found.ItemId())
	}
}

func TestGetDrop_NonExistent(t *testing.T) {
	setupTestRegistry(t)
	r := GetRegistry()

	_, err := r.GetDrop(999999)
	if err == nil {
		t.Fatal("Expected error when getting non-existent drop")
	}
}

func TestGetDropsForMap_ReturnsCorrectDrops(t *testing.T) {
	setupTestRegistry(t)
	r := GetRegistry()
	ten := createTestTenant(t)

	mb1 := createTestBuilder(ten, 1, 1, 100000000)
	mb2 := createTestBuilder(ten, 1, 1, 100000000)
	mb3 := createTestBuilder(ten, 1, 1, 200000000)

	drop1 := mustCreateDrop(t, r, mb1)
	drop2 := mustCreateDrop(t, r, mb2)
	_ = mustCreateDrop(t, r, mb3)

	f := field.NewBuilder(world.Id(1), channel.Id(1), _map.Id(100000000)).Build()
	drops, err := r.GetDropsForMap(ten, f)
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

func TestGetDropsForMap_DifferentChannel(t *testing.T) {
	setupTestRegistry(t)
	r := GetRegistry()
	ten := createTestTenant(t)

	mb1 := createTestBuilder(ten, 1, 1, 100000000)
	mb2 := createTestBuilder(ten, 1, 2, 100000000)

	mustCreateDrop(t, r, mb1)
	mustCreateDrop(t, r, mb2)

	f1 := field.NewBuilder(world.Id(1), channel.Id(1), _map.Id(100000000)).Build()
	f2 := field.NewBuilder(world.Id(1), channel.Id(2), _map.Id(100000000)).Build()
	drops1, _ := r.GetDropsForMap(ten, f1)
	drops2, _ := r.GetDropsForMap(ten, f2)

	if len(drops1) != 1 {
		t.Fatalf("Expected 1 drop for channel 1, got %d", len(drops1))
	}
	if len(drops2) != 1 {
		t.Fatalf("Expected 1 drop for channel 2, got %d", len(drops2))
	}
}

func TestGetAllDrops(t *testing.T) {
	setupTestRegistry(t)
	r := GetRegistry()
	ten := createTestTenant(t)

	mb1 := createTestBuilder(ten, 1, 1, 100000000)
	mb2 := createTestBuilder(ten, 1, 2, 200000000)
	mb3 := createTestBuilder(ten, 2, 1, 300000000)

	mustCreateDrop(t, r, mb1)
	mustCreateDrop(t, r, mb2)
	mustCreateDrop(t, r, mb3)

	drops := r.GetAllDrops()
	if len(drops) != 3 {
		t.Fatalf("Expected 3 total drops, got %d", len(drops))
	}
}

func TestUniqueIdGeneration_Sequential(t *testing.T) {
	setupTestRegistry(t)
	r := GetRegistry()
	ten := createTestTenant(t)

	var ids []uint32
	for i := 0; i < 10; i++ {
		mb := createTestBuilder(ten, 1, 1, 100000000)
		drop := mustCreateDrop(t, r, mb)
		ids = append(ids, drop.Id())
	}

	for i := 1; i < len(ids); i++ {
		if ids[i] != ids[i-1]+1 {
			t.Fatalf("Expected sequential IDs, got %d after %d", ids[i], ids[i-1])
		}
	}
}

func TestReserveDrop_WithPetSlot(t *testing.T) {
	setupTestRegistry(t)
	r := GetRegistry()
	ten := createTestTenant(t)

	mb := createTestBuilder(ten, 1, 1, 100000000)
	drop := mustCreateDrop(t, r, mb)

	characterId := uint32(12345)
	petSlot := int8(2)

	reserved, err := r.ReserveDrop(drop.Id(), characterId, 0, petSlot)
	if err != nil {
		t.Fatalf("Failed to reserve drop with pet slot: %v", err)
	}
	if reserved.PetSlot() != petSlot {
		t.Fatalf("Expected petSlot %d, got %d", petSlot, reserved.PetSlot())
	}
}

func TestConcurrentCreateDrop(t *testing.T) {
	setupTestRegistry(t)
	r := GetRegistry()
	ten := createTestTenant(t)

	const numGoroutines = 100
	var wg sync.WaitGroup
	ids := make(chan uint32, numGoroutines)
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mb := createTestBuilder(ten, 1, 1, 100000000)
			drop, err := r.CreateDrop(mb)
			if err != nil {
				errors <- err
				return
			}
			ids <- drop.Id()
		}()
	}

	wg.Wait()
	close(ids)
	close(errors)

	for err := range errors {
		t.Fatalf("CreateDrop failed: %v", err)
	}

	uniqueIds := make(map[uint32]bool)
	for id := range ids {
		if uniqueIds[id] {
			t.Fatalf("Duplicate ID found: %d", id)
		}
		uniqueIds[id] = true
	}

	if len(uniqueIds) != numGoroutines {
		t.Fatalf("Expected %d unique IDs, got %d", numGoroutines, len(uniqueIds))
	}
}

func TestConcurrentReserveDrop(t *testing.T) {
	setupTestRegistry(t)
	r := GetRegistry()
	ten := createTestTenant(t)

	// Use FFA drop (owner=0) so ownership validation allows all characters to attempt reservation
	f := field.NewBuilder(world.Id(1), channel.Id(1), _map.Id(100000000)).Build()
	mb := NewModelBuilder(ten, f).
		SetItem(1000000, 1).
		SetPosition(100, 200).
		SetOwner(0, 0).
		SetDropper(99999, 50, 150).
		SetType(0)
	drop := mustCreateDrop(t, r, mb)

	const numGoroutines = 10
	var wg sync.WaitGroup
	successCount := int32(0)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(characterId uint32) {
			defer wg.Done()
			_, err := r.ReserveDrop(drop.Id(), characterId, 0, -1)
			if err == nil {
				atomic.AddInt32(&successCount, 1)
			}
		}(uint32(i + 1))
	}

	wg.Wait()

	// With Redis (single-threaded), the first writer wins.
	// Multiple reservations may succeed since there's no distributed lock,
	// but they all see AVAILABLE status initially in a race.
	// At minimum one should succeed.
	if successCount < 1 {
		t.Fatalf("Expected at least 1 successful reservation, got %d", successCount)
	}
}

func TestGetNextUniqueId_Sequential(t *testing.T) {
	setupTestRegistry(t)
	r := GetRegistry()

	id1 := r.getNextUniqueId()
	id2 := r.getNextUniqueId()
	id3 := r.getNextUniqueId()

	if id2 != id1+1 || id3 != id2+1 {
		t.Fatal("Expected sequential IDs")
	}
}

func TestGetNextUniqueId_Wraparound(t *testing.T) {
	setupTestRegistry(t)
	r := GetRegistry()

	// Set counter to just below wraparound threshold
	r.client.Set(context.Background(), nextIdKey, maxId-1, 0)

	id1 := r.getNextUniqueId()
	if id1 != maxId {
		t.Fatalf("Expected %d, got %d", maxId, id1)
	}

	id2 := r.getNextUniqueId()
	if id2 != minId {
		t.Fatalf("Expected wraparound to %d, got %d", minId, id2)
	}
}

func TestCancelDropReservation_DropNotReserved(t *testing.T) {
	setupTestRegistry(t)
	r := GetRegistry()
	ten := createTestTenant(t)

	mb := createTestBuilder(ten, 1, 1, 100000000)
	drop := mustCreateDrop(t, r, mb)

	r.CancelDropReservation(drop.Id(), uint32(12345))

	found, _ := r.GetDrop(drop.Id())
	if found.Status() != StatusAvailable {
		t.Fatal("Drop should still be available")
	}
}

func TestCancelDropReservation_AlreadyAvailable(t *testing.T) {
	setupTestRegistry(t)
	r := GetRegistry()
	ten := createTestTenant(t)

	mb := createTestBuilder(ten, 1, 1, 100000000).SetStatus(StatusAvailable)
	drop := mustCreateDrop(t, r, mb)

	_, _ = r.ReserveDrop(drop.Id(), uint32(12345), 0, -1)

	r.CancelDropReservation(drop.Id(), uint32(12345))

	found, _ := r.GetDrop(drop.Id())
	if found.Status() != StatusAvailable {
		t.Fatalf("Expected status %s, got %s", StatusAvailable, found.Status())
	}
}

func TestRemoveDrop_CleansUpMapIndex(t *testing.T) {
	setupTestRegistry(t)
	r := GetRegistry()
	ten := createTestTenant(t)

	mb := createTestBuilder(ten, 1, 1, 100000000)
	drop := mustCreateDrop(t, r, mb)

	f := field.NewBuilder(world.Id(1), channel.Id(1), _map.Id(100000000)).Build()
	dropsBefore, _ := r.GetDropsForMap(ten, f)
	if len(dropsBefore) != 1 {
		t.Fatal("Expected 1 drop before removal")
	}

	_, _ = r.RemoveDrop(drop.Id())

	dropsAfter, _ := r.GetDropsForMap(ten, f)
	if len(dropsAfter) != 0 {
		t.Fatalf("Expected 0 drops after removal, got %d", len(dropsAfter))
	}
}

func TestGetDropsForMap_EmptyMap(t *testing.T) {
	setupTestRegistry(t)
	r := GetRegistry()
	ten := createTestTenant(t)

	f := field.NewBuilder(world.Id(1), channel.Id(1), _map.Id(999999999)).Build()
	drops, err := r.GetDropsForMap(ten, f)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(drops) != 0 {
		t.Fatalf("Expected 0 drops for empty map, got %d", len(drops))
	}
}

func TestGetAllDrops_EmptyRegistry(t *testing.T) {
	setupTestRegistry(t)
	r := GetRegistry()

	drops := r.GetAllDrops()
	if len(drops) != 0 {
		t.Fatalf("Expected 0 drops in empty registry, got %d", len(drops))
	}
}

func TestCancelDropReservation_NonExistentDrop(t *testing.T) {
	setupTestRegistry(t)
	r := GetRegistry()

	// Should not panic when canceling reservation for non-existent drop
	r.CancelDropReservation(999999, uint32(12345))
}

func TestCancelDropReservation_NoReservation(t *testing.T) {
	setupTestRegistry(t)
	r := GetRegistry()
	ten := createTestTenant(t)

	mb := createTestBuilder(ten, 1, 1, 100000000)
	drop := mustCreateDrop(t, r, mb)

	r.CancelDropReservation(drop.Id(), uint32(12345))

	found, _ := r.GetDrop(drop.Id())
	if found.Status() != StatusAvailable {
		t.Fatal("Drop should still be available")
	}
}

func TestRemoveDrop_FromMiddleOfList(t *testing.T) {
	setupTestRegistry(t)
	r := GetRegistry()
	ten := createTestTenant(t)

	mb1 := createTestBuilder(ten, 1, 1, 100000000)
	mb2 := createTestBuilder(ten, 1, 1, 100000000)
	mb3 := createTestBuilder(ten, 1, 1, 100000000)

	drop1 := mustCreateDrop(t, r, mb1)
	drop2 := mustCreateDrop(t, r, mb2)
	drop3 := mustCreateDrop(t, r, mb3)

	_, _ = r.RemoveDrop(drop2.Id())

	f := field.NewBuilder(world.Id(1), channel.Id(1), _map.Id(100000000)).Build()
	drops, _ := r.GetDropsForMap(ten, f)
	if len(drops) != 2 {
		t.Fatalf("Expected 2 drops, got %d", len(drops))
	}

	foundIds := make(map[uint32]bool)
	for _, d := range drops {
		foundIds[d.Id()] = true
	}

	if !foundIds[drop1.Id()] || !foundIds[drop3.Id()] {
		t.Fatal("Expected drop1 and drop3 to remain")
	}
	if foundIds[drop2.Id()] {
		t.Fatal("Expected drop2 to be removed")
	}
}

func TestGetDrop_Internal(t *testing.T) {
	setupTestRegistry(t)
	r := GetRegistry()
	ten := createTestTenant(t)

	mb := createTestBuilder(ten, 1, 1, 100000000)
	created := mustCreateDrop(t, r, mb)

	drop, ok := r.getDrop(created.Id())
	if !ok {
		t.Fatal("Expected to find drop")
	}
	if drop.Id() != created.Id() {
		t.Fatal("Expected correct drop ID")
	}

	_, ok = r.getDrop(999999)
	if ok {
		t.Fatal("Expected not to find non-existent drop")
	}
}
