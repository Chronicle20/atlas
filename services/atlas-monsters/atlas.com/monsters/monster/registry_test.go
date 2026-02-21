package monster

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/field"
	_map "github.com/Chronicle20/atlas-constants/map"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

var testMiniRedis *miniredis.Miniredis

func TestMain(m *testing.M) {
	mr, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	defer mr.Close()
	testMiniRedis = mr

	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	InitIdAllocator(rc)
	InitCooldownRegistry(rc)
	InitMonsterRegistry(rc)
	InitDropTimerRegistry(rc)

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

func TestCreateMoveDamageKill(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := testContext(ten)
	r.Clear(ctx)
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()

	m := r.CreateMonster(ctx, ten, f, 9300018, 0, 0, 0, 5, 0, 100, 50)
	if m.Hp() != 100 || m.Mp() != 50 {
		t.Fatalf("Expected HP=100 MP=50, got HP=%d MP=%d", m.Hp(), m.Mp())
	}

	// Move
	moved := r.MoveMonster(ten, m.UniqueId(), 50, 75, 3)
	if moved.X() != 50 || moved.Y() != 75 || moved.Stance() != 3 {
		t.Fatalf("Move failed: X=%d Y=%d Stance=%d", moved.X(), moved.Y(), moved.Stance())
	}
	// Verify persisted
	got, _ := r.GetMonster(ten, m.UniqueId())
	if got.X() != 50 || got.Y() != 75 {
		t.Fatal("Move not persisted in Redis")
	}

	// Damage (30 HP)
	ds, err := r.ApplyDamage(ten, 1, 30, m.UniqueId())
	if err != nil {
		t.Fatalf("ApplyDamage failed: %v", err)
	}
	if ds.Monster.Hp() != 70 || ds.Killed {
		t.Fatalf("Expected HP=70 alive, got HP=%d killed=%v", ds.Monster.Hp(), ds.Killed)
	}
	if ds.CharacterId != 1 || ds.VisibleDamage != 30 {
		t.Fatalf("DamageSummary fields wrong")
	}

	// Damage to kill (200 > remaining 70)
	ds, err = r.ApplyDamage(ten, 2, 200, m.UniqueId())
	if err != nil {
		t.Fatalf("ApplyDamage failed: %v", err)
	}
	if ds.Monster.Hp() != 0 || !ds.Killed {
		t.Fatalf("Expected HP=0 killed, got HP=%d killed=%v", ds.Monster.Hp(), ds.Killed)
	}

	// Verify damage entries accumulated
	if len(ds.Monster.DamageEntries()) != 2 {
		t.Fatalf("Expected 2 damage entries, got %d", len(ds.Monster.DamageEntries()))
	}
}

func TestConcurrentDamage(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := testContext(ten)
	r.Clear(ctx)
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()

	m := r.CreateMonster(ctx, ten, f, 9300018, 0, 0, 0, 5, 0, 10000, 50)

	numAttackers := 10
	hitsPerAttacker := 50
	damagePerHit := uint32(1)

	var wg sync.WaitGroup
	for i := 0; i < numAttackers; i++ {
		wg.Add(1)
		go func(charId uint32) {
			defer wg.Done()
			for j := 0; j < hitsPerAttacker; j++ {
				r.ApplyDamage(ten, charId, damagePerHit, m.UniqueId())
			}
		}(uint32(i + 1))
	}
	wg.Wait()

	got, err := r.GetMonster(ten, m.UniqueId())
	if err != nil {
		t.Fatalf("GetMonster failed: %v", err)
	}

	totalDamage := uint32(numAttackers * hitsPerAttacker)
	expectedHp := uint32(10000) - totalDamage
	if got.Hp() != expectedHp {
		t.Fatalf("Expected HP=%d after %d damage, got HP=%d", expectedHp, totalDamage, got.Hp())
	}
	if len(got.DamageEntries()) != numAttackers*hitsPerAttacker {
		t.Fatalf("Expected %d damage entries, got %d", numAttackers*hitsPerAttacker, len(got.DamageEntries()))
	}
}

func TestStatusEffectLifecycle(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := testContext(ten)
	r.Clear(ctx)
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()

	m := r.CreateMonster(ctx, ten, f, 9300018, 0, 0, 0, 5, 0, 1000, 50)

	// Apply status effect
	effect := NewStatusEffect(SourceTypePlayerSkill, 100, 2111003, 20,
		map[string]int32{"POISON": 5}, 10*time.Second, 1*time.Second)
	updated, err := r.ApplyStatusEffect(ten, m.UniqueId(), effect)
	if err != nil {
		t.Fatalf("ApplyStatusEffect failed: %v", err)
	}
	if len(updated.StatusEffects()) != 1 {
		t.Fatalf("Expected 1 status effect, got %d", len(updated.StatusEffects()))
	}
	if !updated.HasStatusEffect("POISON") {
		t.Fatal("Expected POISON status effect")
	}

	// Verify persisted
	got, _ := r.GetMonster(ten, m.UniqueId())
	if len(got.StatusEffects()) != 1 {
		t.Fatalf("Status effect not persisted, got %d effects", len(got.StatusEffects()))
	}

	// Cancel by ID
	cancelled, err := r.CancelStatusEffect(ten, m.UniqueId(), effect.EffectId())
	if err != nil {
		t.Fatalf("CancelStatusEffect failed: %v", err)
	}
	if len(cancelled.StatusEffects()) != 0 {
		t.Fatalf("Expected 0 effects after cancel, got %d", len(cancelled.StatusEffects()))
	}
}

func TestStatusEffectVenomStacking(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := testContext(ten)
	r.Clear(ctx)
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()

	m := r.CreateMonster(ctx, ten, f, 9300018, 0, 0, 0, 5, 0, 1000, 50)

	// Apply 4 VENOM effects - should cap at 3 (removing oldest)
	for i := 0; i < 4; i++ {
		effect := NewStatusEffect(SourceTypePlayerSkill, uint32(100+i), 4120005, 30,
			map[string]int32{"VENOM": int32(10 + i)}, 10*time.Second, 1*time.Second)
		_, err := r.ApplyStatusEffect(ten, m.UniqueId(), effect)
		if err != nil {
			t.Fatalf("ApplyStatusEffect %d failed: %v", i, err)
		}
	}

	got, _ := r.GetMonster(ten, m.UniqueId())
	if len(got.StatusEffects()) != 3 {
		t.Fatalf("Expected 3 VENOM stacks, got %d", len(got.StatusEffects()))
	}
}

func TestDeductMp(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := testContext(ten)
	r.Clear(ctx)
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()

	m := r.CreateMonster(ctx, ten, f, 9300018, 0, 0, 0, 5, 0, 100, 50)

	// Deduct 20 MP
	updated, err := r.DeductMp(ten, m.UniqueId(), 20)
	if err != nil {
		t.Fatalf("DeductMp failed: %v", err)
	}
	if updated.Mp() != 30 {
		t.Fatalf("Expected MP=30, got %d", updated.Mp())
	}

	// Deduct more than remaining (should cap at 0)
	updated, err = r.DeductMp(ten, m.UniqueId(), 100)
	if err != nil {
		t.Fatalf("DeductMp failed: %v", err)
	}
	if updated.Mp() != 0 {
		t.Fatalf("Expected MP=0, got %d", updated.Mp())
	}
}

func TestCancelAllStatusEffects(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := testContext(ten)
	r.Clear(ctx)
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()

	m := r.CreateMonster(ctx, ten, f, 9300018, 0, 0, 0, 5, 0, 1000, 50)

	// Apply two different effects
	e1 := NewStatusEffect(SourceTypePlayerSkill, 100, 2111003, 20,
		map[string]int32{"POISON": 5}, 10*time.Second, 1*time.Second)
	r.ApplyStatusEffect(ten, m.UniqueId(), e1)
	e2 := NewStatusEffect(SourceTypeMonsterSkill, 0, 100, 1,
		map[string]int32{"SPEED": -20}, 5*time.Second, 0)
	r.ApplyStatusEffect(ten, m.UniqueId(), e2)

	got, _ := r.GetMonster(ten, m.UniqueId())
	if len(got.StatusEffects()) != 2 {
		t.Fatalf("Expected 2 effects, got %d", len(got.StatusEffects()))
	}

	// Cancel all
	cleared, err := r.CancelAllStatusEffects(ten, m.UniqueId())
	if err != nil {
		t.Fatalf("CancelAllStatusEffects failed: %v", err)
	}
	if len(cleared.StatusEffects()) != 0 {
		t.Fatalf("Expected 0 effects, got %d", len(cleared.StatusEffects()))
	}
}

func TestDropTimerRegistration(t *testing.T) {
	dr := GetDropTimerRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()

	now := time.Now()
	uniqueId := uint32(100)
	entry := DropTimerEntry{
		monsterId:    9300018,
		field:        f,
		dropPeriod:   5 * time.Second,
		weaponAttack: 100,
		maxHp:        5000,
		lastDropAt:   now,
		lastHitAt:    time.Time{},
	}

	dr.Register(ctx, ten, uniqueId, entry)

	all := dr.GetAll(ctx)
	if len(all) == 0 {
		t.Fatal("Expected at least 1 drop timer entry")
	}

	found := false
	for mk, e := range all {
		if mk.MonsterId == uniqueId {
			found = true
			if e.MonsterId() != 9300018 {
				t.Fatalf("Expected monsterId=9300018, got %d", e.MonsterId())
			}
			if e.DropPeriod() != 5*time.Second {
				t.Fatalf("Expected dropPeriod=5s, got %s", e.DropPeriod())
			}
			if e.WeaponAttack() != 100 {
				t.Fatalf("Expected weaponAttack=100, got %d", e.WeaponAttack())
			}
			if !e.LastHitAt().IsZero() {
				t.Fatalf("Expected zero lastHitAt, got %v", e.LastHitAt())
			}
			if !e.Field().Equals(f) {
				t.Fatal("Field mismatch")
			}
		}
	}
	if !found {
		t.Fatal("Did not find registered drop timer entry")
	}

	// Unregister
	dr.Unregister(ctx, ten, uniqueId)
	all = dr.GetAll(ctx)
	for mk := range all {
		if mk.MonsterId == uniqueId {
			t.Fatal("Entry should be unregistered")
		}
	}
}

func TestDropTimerRecordHitAndUpdateDrop(t *testing.T) {
	dr := GetDropTimerRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()

	now := time.Now()
	uniqueId := uint32(200)
	entry := DropTimerEntry{
		monsterId:    9300019,
		field:        f,
		dropPeriod:   3 * time.Second,
		weaponAttack: 50,
		maxHp:        3000,
		lastDropAt:   now,
		lastHitAt:    time.Time{},
	}
	dr.Register(ctx, ten, uniqueId, entry)

	// Record a hit
	hitTime := now.Add(1 * time.Second)
	dr.RecordHit(ctx, ten, uniqueId, hitTime)

	all := dr.GetAll(ctx)
	for mk, e := range all {
		if mk.MonsterId == uniqueId {
			if e.LastHitAt().IsZero() {
				t.Fatal("Expected lastHitAt to be set after RecordHit")
			}
			// Check within 1ms precision (millisecond storage)
			diff := e.LastHitAt().Sub(hitTime)
			if diff < -time.Millisecond || diff > time.Millisecond {
				t.Fatalf("Expected lastHitAt close to %v, got %v (diff=%v)", hitTime, e.LastHitAt(), diff)
			}
		}
	}

	// Update last drop
	dropTime := now.Add(4 * time.Second)
	dr.UpdateLastDrop(ctx, ten, uniqueId, dropTime)

	all = dr.GetAll(ctx)
	for mk, e := range all {
		if mk.MonsterId == uniqueId {
			diff := e.LastDropAt().Sub(dropTime)
			if diff < -time.Millisecond || diff > time.Millisecond {
				t.Fatalf("Expected lastDropAt close to %v, got %v", dropTime, e.LastDropAt())
			}
		}
	}

	// Clean up
	dr.Unregister(ctx, ten, uniqueId)
}

func TestCooldownExpiration(t *testing.T) {
	cr := GetCooldownRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	monsterId := uint32(9300018)
	skillId := uint16(100)

	// Set a cooldown with 2 second TTL
	cr.SetCooldown(ctx, ten, monsterId, skillId, 2*time.Second)

	// Should be on cooldown immediately
	if !cr.IsOnCooldown(ctx, ten, monsterId, skillId) {
		t.Fatal("Expected monster to be on cooldown")
	}

	// Fast forward 1 second - still on cooldown
	testMiniRedis.FastForward(1 * time.Second)
	if !cr.IsOnCooldown(ctx, ten, monsterId, skillId) {
		t.Fatal("Expected monster to still be on cooldown after 1s")
	}

	// Fast forward past expiration
	testMiniRedis.FastForward(2 * time.Second)
	if cr.IsOnCooldown(ctx, ten, monsterId, skillId) {
		t.Fatal("Expected cooldown to have expired after 3s total")
	}
}

func TestCooldownClearAll(t *testing.T) {
	cr := GetCooldownRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	monsterId := uint32(9300019)

	// Set multiple cooldowns
	cr.SetCooldown(ctx, ten, monsterId, 100, 10*time.Second)
	cr.SetCooldown(ctx, ten, monsterId, 200, 10*time.Second)
	cr.SetCooldown(ctx, ten, monsterId, 300, 10*time.Second)

	if !cr.IsOnCooldown(ctx, ten, monsterId, 100) {
		t.Fatal("Expected skill 100 on cooldown")
	}
	if !cr.IsOnCooldown(ctx, ten, monsterId, 200) {
		t.Fatal("Expected skill 200 on cooldown")
	}

	// Clear all cooldowns for this monster
	cr.ClearCooldowns(ctx, ten, monsterId)

	if cr.IsOnCooldown(ctx, ten, monsterId, 100) {
		t.Fatal("Expected skill 100 cooldown cleared")
	}
	if cr.IsOnCooldown(ctx, ten, monsterId, 200) {
		t.Fatal("Expected skill 200 cooldown cleared")
	}
	if cr.IsOnCooldown(ctx, ten, monsterId, 300) {
		t.Fatal("Expected skill 300 cooldown cleared")
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
