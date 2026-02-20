package monster

import (
	"context"
	"os"
	"sync"
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

func TestMain(m *testing.M) {
	mr, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	defer mr.Close()

	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	InitIdAllocator(rc)
	InitCooldownRegistry(rc)

	os.Exit(m.Run())
}

func testContext(t tenant.Model) context.Context {
	return context.Background()
}

func TestSunnyDay(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := testContext(ten)
	r.Clear(ctx)
	worldId := world.Id(0)
	channelId := channel.Id(0)
	mapId := _map.Id(40000)
	f := field.NewBuilder(worldId, channelId, mapId).Build()
	monsterId := uint32(9300018)
	x := int16(0)
	y := int16(0)
	fh := int16(0)
	stance := byte(0)
	team := int8(0)
	hp := uint32(50)
	mp := uint32(50)

	m := r.CreateMonster(ctx, ten, f, monsterId, x, y, fh, stance, team, hp, mp)
	if !valid(f, monsterId, x, y, fh, stance, team, hp, mp)(m) {
		t.Fatal("Monster created with incorrect properties.")
	}
	if m.ControlCharacterId() != 0 {
		t.Fatal("Unexpected Control CharacterId.")
	}

	controlId := uint32(100)
	var err error
	m, err = r.ControlMonster(ten, m.UniqueId(), controlId)
	if err != nil {
		t.Fatalf("Unable to control monster. err %s", err.Error())
	}
	if m.ControlCharacterId() != controlId {
		t.Fatal("Unexpected Control CharacterId.")
	}

	m, err = r.ClearControl(ten, m.UniqueId())
	if err != nil {
		t.Fatalf("Unable to clear monster control. err %s", err.Error())
	}
	if m.ControlCharacterId() != 0 {
		t.Fatal("Unexpected Control CharacterId.")
	}

	m2 := r.CreateMonster(ctx, ten, f, monsterId, 50, y, fh, stance, team, hp, mp)
	if !valid(f, monsterId, 50, y, fh, stance, team, hp, mp)(m2) {
		t.Fatal("Monster created with incorrect properties.")
	}
	m3 := r.CreateMonster(ctx, ten, f, monsterId, 100, y, fh, stance, team, hp, mp)
	if !valid(f, monsterId, 100, y, fh, stance, team, hp, mp)(m3) {
		t.Fatal("Monster created with incorrect properties.")
	}

	irm, err := r.GetMonster(ten, m.UniqueId())
	if err != nil {
		t.Fatalf("Unable to get monster. err %s", err.Error())
	}
	if !compare(irm)(m) {
		t.Fatal("Monster retrieved with incorrect properties.")
	}

	imms := r.GetMonstersInMap(ten, f)
	if len(imms) != 3 {
		t.Fatal("Monsters in map not correct.")
	}
	for _, imm := range imms {
		if compare(imm)(m) {
			continue
		}
		if compare(imm)(m2) {
			continue
		}
		if compare(imm)(m3) {
			continue
		}
		t.Fatalf("Monster retrieved with incorrect properties.")
	}

	_, err = r.RemoveMonster(ctx, ten, m.UniqueId())
	if err != nil {
		t.Fatalf("Unable to remove monster. err %s", err.Error())
	}
	imms = r.GetMonstersInMap(ten, f)
	if len(imms) != 2 {
		t.Fatal("Monsters in map not correct.")
	}
	for _, imm := range imms {
		if compare(imm)(m2) {
			continue
		}
		if compare(imm)(m3) {
			continue
		}
		t.Fatalf("Monster retrieved with incorrect properties.")
	}
}

func TestIdReuse(t *testing.T) {
	r := GetMonsterRegistry()
	tenant1, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	tenant2, _ := tenant.Create(uuid.New(), "GMS", 87, 1)
	ctx1 := testContext(tenant1)
	ctx2 := testContext(tenant2)
	r.Clear(ctx1)
	worldId := world.Id(0)
	channelId := channel.Id(0)
	mapId := _map.Id(40000)
	f := field.NewBuilder(worldId, channelId, mapId).Build()
	monsterId := uint32(9300018)
	x := int16(0)
	y := int16(0)
	fh := int16(0)
	stance := byte(0)
	team := int8(0)
	hp := uint32(50)
	mp := uint32(50)

	m := r.CreateMonster(ctx1, tenant1, f, monsterId, x, y, fh, stance, team, hp, mp)
	if !valid(f, monsterId, x, y, fh, stance, team, hp, mp)(m) {
		t.Fatal("Monster created with incorrect properties.")
	}

	m2 := r.CreateMonster(ctx2, tenant2, f, monsterId, x, y, fh, stance, team, hp, mp)
	if !valid(f, monsterId, x, y, fh, stance, team, hp, mp)(m2) {
		t.Fatal("Monster created with incorrect properties.")
	}

	m3 := r.CreateMonster(ctx1, tenant1, f, monsterId, x, y, fh, stance, team, hp, mp)
	if !valid(f, monsterId, x, y, fh, stance, team, hp, mp)(m3) {
		t.Fatal("Monster created with incorrect properties.")
	}
	// Verify IDs are unique per tenant (separate Redis counters)
	if m.UniqueId() == m3.UniqueId() {
		t.Fatal("Monster IDs should be unique within tenant.")
	}
}

func valid(f field.Model, monsterId uint32, x int16, y int16, fh int16, stance byte, team int8, hp uint32, mp uint32) func(m Model) bool {
	return func(m Model) bool {
		if m.WorldId() != f.WorldId() {
			return false
		}
		if m.ChannelId() != f.ChannelId() {
			return false
		}
		if m.MapId() != f.MapId() {
			return false
		}
		if m.Instance() != f.Instance() {
			return false
		}
		if m.MonsterId() != monsterId {
			return false
		}
		if m.X() != x {
			return false
		}
		if m.Y() != y {
			return false
		}
		if m.Fh() != fh {
			return false
		}
		if m.Stance() != stance {
			return false
		}
		if m.Team() != team {
			return false
		}
		if m.Hp() != hp {
			return false
		}
		if m.Mp() != mp {
			return false
		}
		return true
	}
}

func compare(m Model) func(o Model) bool {
	return func(o Model) bool {
		if m.UniqueId() != o.UniqueId() {
			return false
		}
		if m.WorldId() != o.WorldId() {
			return false
		}
		if m.ChannelId() != o.ChannelId() {
			return false
		}
		if m.MapId() != o.MapId() {
			return false
		}
		if m.Hp() != o.Hp() {
			return false
		}
		if m.Mp() != o.Mp() {
			return false
		}
		if m.X() != o.X() {
			return false
		}
		if m.Y() != o.Y() {
			return false
		}
		if m.MonsterId() != o.MonsterId() {
			return false
		}
		if m.ControlCharacterId() != o.ControlCharacterId() {
			return false
		}
		return true
	}
}

func TestDestroyAll(t *testing.T) {
	r := GetMonsterRegistry()
	tenant1, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	tenant2, _ := tenant.Create(uuid.New(), "GMS", 87, 1)
	ctx1 := testContext(tenant1)
	ctx2 := testContext(tenant2)
	r.Clear(ctx1)
	worldId := world.Id(0)
	channelId := channel.Id(0)
	mapId := _map.Id(40000)
	f := field.NewBuilder(worldId, channelId, mapId).Build()
	monsterId := uint32(9300018)
	x := int16(0)
	y := int16(0)
	fh := int16(0)
	stance := byte(0)
	team := int8(0)
	hp := uint32(50)
	mp := uint32(50)

	_ = r.CreateMonster(ctx1, tenant1, f, monsterId, x, y, fh, stance, team, hp, mp)
	_ = r.CreateMonster(ctx2, tenant2, f, monsterId, x, y, fh, stance, team, hp, mp)
	_ = r.CreateMonster(ctx1, tenant1, f, monsterId, x, y, fh, stance, team, hp, mp)

	ms := r.GetMonsters()
	count := 0
	for _, v := range ms {
		count += len(v)
	}
	if count != 3 {
		t.Fatal("Expected 3 Monsters, got ", count)
	}
}

func TestIdRecyclingAfterRemoval(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := testContext(ten)
	r.Clear(ctx)
	worldId := world.Id(0)
	channelId := channel.Id(0)
	mapId := _map.Id(40000)
	f := field.NewBuilder(worldId, channelId, mapId).Build()
	monsterId := uint32(9300018)
	x := int16(0)
	y := int16(0)
	fh := int16(0)
	stance := byte(0)
	team := int8(0)
	hp := uint32(50)
	mp := uint32(50)

	// Create first monster
	m1 := r.CreateMonster(ctx, ten, f, monsterId, x, y, fh, stance, team, hp, mp)
	firstId := m1.UniqueId()

	// Create second monster
	m2 := r.CreateMonster(ctx, ten, f, monsterId, x, y, fh, stance, team, hp, mp)
	if m2.UniqueId() == firstId {
		t.Fatalf("Expected second monster to have different ID from first")
	}

	// Remove first monster - ID should be released
	_, err := r.RemoveMonster(ctx, ten, m1.UniqueId())
	if err != nil {
		t.Fatalf("Failed to remove monster: %v", err)
	}

	// Create third monster - should reuse recycled ID
	m3 := r.CreateMonster(ctx, ten, f, monsterId, x, y, fh, stance, team, hp, mp)
	if m3.UniqueId() != firstId {
		t.Fatalf("Expected recycled monster ID to be %d, got %d", firstId, m3.UniqueId())
	}

	// Create fourth monster - should get next sequential
	m4 := r.CreateMonster(ctx, ten, f, monsterId, x, y, fh, stance, team, hp, mp)
	if m4.UniqueId() == firstId || m4.UniqueId() == m2.UniqueId() {
		t.Fatalf("Expected fourth monster to have a new unique ID")
	}
}

func TestIdRecyclingLIFOOrder(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := testContext(ten)
	r.Clear(ctx)
	worldId := world.Id(0)
	channelId := channel.Id(0)
	mapId := _map.Id(40000)
	f := field.NewBuilder(worldId, channelId, mapId).Build()
	monsterId := uint32(9300018)
	x := int16(0)
	y := int16(0)
	fh := int16(0)
	stance := byte(0)
	team := int8(0)
	hp := uint32(50)
	mp := uint32(50)

	// Create 3 monsters
	m1 := r.CreateMonster(ctx, ten, f, monsterId, x, y, fh, stance, team, hp, mp)
	m2 := r.CreateMonster(ctx, ten, f, monsterId, x, y, fh, stance, team, hp, mp)
	m3 := r.CreateMonster(ctx, ten, f, monsterId, x, y, fh, stance, team, hp, mp)

	// Remove in order: m1, m2, m3
	r.RemoveMonster(ctx, ten, m1.UniqueId())
	r.RemoveMonster(ctx, ten, m2.UniqueId())
	r.RemoveMonster(ctx, ten, m3.UniqueId())

	// Create new monsters - should get IDs back in LIFO order: m3, m2, m1
	n1 := r.CreateMonster(ctx, ten, f, monsterId, x, y, fh, stance, team, hp, mp)
	if n1.UniqueId() != m3.UniqueId() {
		t.Fatalf("Expected LIFO recycled ID %d, got %d", m3.UniqueId(), n1.UniqueId())
	}

	n2 := r.CreateMonster(ctx, ten, f, monsterId, x, y, fh, stance, team, hp, mp)
	if n2.UniqueId() != m2.UniqueId() {
		t.Fatalf("Expected LIFO recycled ID %d, got %d", m2.UniqueId(), n2.UniqueId())
	}

	n3 := r.CreateMonster(ctx, ten, f, monsterId, x, y, fh, stance, team, hp, mp)
	if n3.UniqueId() != m1.UniqueId() {
		t.Fatalf("Expected LIFO recycled ID %d, got %d", m1.UniqueId(), n3.UniqueId())
	}
}

func TestConcurrentMonsterCreation(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := testContext(ten)
	r.Clear(ctx)
	worldId := world.Id(0)
	channelId := channel.Id(0)
	mapId := _map.Id(40000)
	f := field.NewBuilder(worldId, channelId, mapId).Build()
	monsterId := uint32(9300018)
	x := int16(0)
	y := int16(0)
	fh := int16(0)
	stance := byte(0)
	team := int8(0)
	hp := uint32(50)
	mp := uint32(50)

	numGoroutines := 50
	monstersPerGoroutine := 20

	var wg sync.WaitGroup
	idChan := make(chan uint32, numGoroutines*monstersPerGoroutine)

	// Spawn goroutines that create monsters concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < monstersPerGoroutine; j++ {
				m := r.CreateMonster(ctx, ten, f, monsterId, x, y, fh, stance, team, hp, mp)
				idChan <- m.UniqueId()
			}
		}()
	}

	wg.Wait()
	close(idChan)

	// Verify no duplicate IDs
	seen := make(map[uint32]bool)
	for id := range idChan {
		if seen[id] {
			t.Fatalf("Duplicate monster ID allocated: %d", id)
		}
		seen[id] = true
	}

	expectedCount := numGoroutines * monstersPerGoroutine
	if len(seen) != expectedCount {
		t.Fatalf("Expected %d unique monster IDs, got %d", expectedCount, len(seen))
	}
}

func TestConcurrentCreateAndRemove(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := testContext(ten)
	r.Clear(ctx)
	worldId := world.Id(0)
	channelId := channel.Id(0)
	mapId := _map.Id(40000)
	f := field.NewBuilder(worldId, channelId, mapId).Build()
	monsterId := uint32(9300018)
	x := int16(0)
	y := int16(0)
	fh := int16(0)
	stance := byte(0)
	team := int8(0)
	hp := uint32(50)
	mp := uint32(50)

	numGoroutines := 20
	iterations := 50

	var wg sync.WaitGroup

	// Spawn goroutines that create and remove monsters
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				m := r.CreateMonster(ctx, ten, f, monsterId, x, y, fh, stance, team, hp, mp)
				r.RemoveMonster(ctx, ten, m.UniqueId())
			}
		}()
	}

	wg.Wait()

	// Verify registry is in consistent state
	monsters := r.GetMonstersInMap(ten, f)
	if len(monsters) != 0 {
		t.Fatalf("Expected 0 monsters after create/remove cycles, got %d", len(monsters))
	}
}
