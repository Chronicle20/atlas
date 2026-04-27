package monster

import (
	"context"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
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

	// Remove first monster - counter must advance rather than recycle the oid
	// while it's still in the fresh range, or the client can see a newly
	// spawned monster with the same oid as one it just despawned.
	_, err := r.RemoveMonster(ctx, ten, m1.UniqueId())
	if err != nil {
		t.Fatalf("Failed to remove monster: %v", err)
	}

	// Create third monster - should NOT reuse firstId; counter must advance.
	m3 := r.CreateMonster(ctx, ten, f, monsterId, x, y, fh, stance, team, hp, mp)
	if m3.UniqueId() == firstId || m3.UniqueId() == m2.UniqueId() {
		t.Fatalf("Expected fresh monster ID, got recycled %d (firstId=%d, m2=%d)", m3.UniqueId(), firstId, m2.UniqueId())
	}

	// Create fourth monster - should get another fresh ID.
	m4 := r.CreateMonster(ctx, ten, f, monsterId, x, y, fh, stance, team, hp, mp)
	if m4.UniqueId() == firstId || m4.UniqueId() == m2.UniqueId() || m4.UniqueId() == m3.UniqueId() {
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

	// Remove in order: m1, m2, m3. While the counter is still fresh, released
	// oids must stay out of circulation so the client never sees two objects
	// with the same oid in quick succession.
	r.RemoveMonster(ctx, ten, m1.UniqueId())
	r.RemoveMonster(ctx, ten, m2.UniqueId())
	r.RemoveMonster(ctx, ten, m3.UniqueId())

	removed := map[uint32]struct{}{
		m1.UniqueId(): {},
		m2.UniqueId(): {},
		m3.UniqueId(): {},
	}

	for i := 0; i < 3; i++ {
		n := r.CreateMonster(ctx, ten, f, monsterId, x, y, fh, stance, team, hp, mp)
		if _, reused := removed[n.UniqueId()]; reused {
			t.Fatalf("Expected fresh monster id, got recycled %d", n.UniqueId())
		}
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
	ds, err := r.ApplyDamage(ten, 1, 30, m.UniqueId(), time.Now().UnixMilli())
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
	ds, err = r.ApplyDamage(ten, 2, 200, m.UniqueId(), time.Now().UnixMilli())
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
				r.ApplyDamage(ten, charId, damagePerHit, m.UniqueId(), time.Now().UnixMilli())
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
	if len(got.DamageEntries()) != numAttackers {
		t.Fatalf("Expected %d aggregated damage entries, got %d", numAttackers, len(got.DamageEntries()))
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
	skillId := byte(100)

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
	cr.SetCooldown(ctx, ten, monsterId, byte(100), 10*time.Second)
	cr.SetCooldown(ctx, ten, monsterId, byte(200), 10*time.Second)
	cr.SetCooldown(ctx, ten, monsterId, byte(55), 10*time.Second)

	if !cr.IsOnCooldown(ctx, ten, monsterId, byte(100)) {
		t.Fatal("Expected skill 100 on cooldown")
	}
	if !cr.IsOnCooldown(ctx, ten, monsterId, byte(200)) {
		t.Fatal("Expected skill 200 on cooldown")
	}

	// Clear all cooldowns for this monster
	cr.ClearCooldowns(ctx, ten, monsterId)

	if cr.IsOnCooldown(ctx, ten, monsterId, byte(100)) {
		t.Fatal("Expected skill 100 cooldown cleared")
	}
	if cr.IsOnCooldown(ctx, ten, monsterId, byte(200)) {
		t.Fatal("Expected skill 200 cooldown cleared")
	}
	if cr.IsOnCooldown(ctx, ten, monsterId, byte(55)) {
		t.Fatal("Expected skill 55 cooldown cleared")
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

// TestLoadMonsterWithCjsonEmptyObjectArrays reproduces the production corruption
// where Redis' Lua cjson re-encodes empty arrays as "{}". The loader must tolerate
// that shape for both statusEffects and damageEntries.
func TestLoadMonsterWithCjsonEmptyObjectArrays(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := testContext(ten)
	r.Clear(ctx)
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()

	m := r.CreateMonster(ctx, ten, f, 9300018, 10, 20, 0, 5, 0, 100, 50)

	corrupted := `{"uniqueId":` + strconv.FormatUint(uint64(m.UniqueId()), 10) +
		`,"tenantId":"` + ten.Id().String() + `","tenantRegion":"GMS"` +
		`,"tenantMajorVersion":83,"tenantMinorVersion":1` +
		`,"worldId":0,"channelId":0,"mapId":40000` +
		`,"instance":"00000000-0000-0000-0000-000000000000"` +
		`,"maxHp":100,"hp":100,"maxMp":50,"mp":50` +
		`,"monsterId":9300018,"controlCharacterId":0` +
		`,"x":10,"y":20,"fh":0,"stance":5,"team":0` +
		`,"damageEntries":{},"statusEffects":{}}`
	testMiniRedis.Set(monsterKey(ten, m.UniqueId()), corrupted)

	got, err := r.GetMonster(ten, m.UniqueId())
	if err != nil {
		t.Fatalf("GetMonster failed on cjson-corrupted record: %v", err)
	}
	if got.UniqueId() != m.UniqueId() || got.Hp() != 100 {
		t.Fatalf("Unexpected monster state: id=%d hp=%d", got.UniqueId(), got.Hp())
	}
	if len(got.DamageEntries()) != 0 || len(got.StatusEffects()) != 0 {
		t.Fatalf("Expected empty arrays, got %d damage / %d status",
			len(got.DamageEntries()), len(got.StatusEffects()))
	}

	ds, err := r.ApplyDamage(ten, 1, 30, m.UniqueId(), time.Now().UnixMilli())
	if err != nil {
		t.Fatalf("ApplyDamage failed after recovering from corruption: %v", err)
	}
	if ds.Monster.Hp() != 70 {
		t.Fatalf("Expected HP=70 after 30 damage, got %d", ds.Monster.Hp())
	}
}

// TestControllerHasAggroRoundTrip verifies that storedMonster.ControllerHasAggro
// round-trips through Redis and that legacy blobs missing the field default to
// false.
func TestControllerHasAggroRoundTrip(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := testContext(ten)
	r.Clear(ctx)

	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, ten, f, 9300018, 0, 0, 0, 5, 0, 100, 50)
	if m.ControllerHasAggro() {
		t.Fatal("freshly spawned monster should default ControllerHasAggro=false")
	}

	updated := Clone(m).SetControllerHasAggro(true).Build()
	r.UpdateMonster(ten, m.UniqueId(), updated)
	got, err := r.GetMonster(ten, m.UniqueId())
	if err != nil {
		t.Fatalf("GetMonster failed: %v", err)
	}
	if !got.ControllerHasAggro() {
		t.Fatal("expected ControllerHasAggro=true after persisted update")
	}

	// Legacy blob missing the field
	legacy := `{"uniqueId":` + strconv.FormatUint(uint64(m.UniqueId()), 10) +
		`,"tenantId":"` + ten.Id().String() + `","tenantRegion":"GMS"` +
		`,"tenantMajorVersion":83,"tenantMinorVersion":1` +
		`,"worldId":0,"channelId":0,"mapId":40000` +
		`,"instance":"00000000-0000-0000-0000-000000000000"` +
		`,"maxHp":100,"hp":100,"maxMp":50,"mp":50` +
		`,"monsterId":9300018,"controlCharacterId":0` +
		`,"x":0,"y":0,"fh":0,"stance":5,"team":0` +
		`,"damageEntries":[],"statusEffects":[]}`
	testMiniRedis.Set(monsterKey(ten, m.UniqueId()), legacy)
	got, err = r.GetMonster(ten, m.UniqueId())
	if err != nil {
		t.Fatalf("GetMonster legacy failed: %v", err)
	}
	if got.ControllerHasAggro() {
		t.Fatal("legacy missing field must default to false")
	}
}

// TestApplyDamageWasFirstHit verifies the WasFirstHit flag in DamageSummary.
// First hit on a monster that has a controller flips controllerHasAggro from
// false to true and reports WasFirstHit=true. Subsequent hits report false.
// A monster with no controller never reports WasFirstHit=true.
func TestApplyDamageWasFirstHit(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := testContext(ten)
	r.Clear(ctx)
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()

	// (a) No controller -> WasFirstHit is always false.
	m := r.CreateMonster(ctx, ten, f, 9300018, 0, 0, 0, 5, 0, 1000, 50)
	now := int64(1_000_000)
	s, err := r.ApplyDamage(ten, 1, 10, m.UniqueId(), now)
	if err != nil {
		t.Fatalf("ApplyDamage: %v", err)
	}
	if s.WasFirstHit {
		t.Errorf("WasFirstHit should be false when no controller is set")
	}

	// (b) With controller -> first hit flips aggro and reports true.
	m2 := r.CreateMonster(ctx, ten, f, 9300018, 0, 0, 0, 5, 0, 1000, 50)
	if _, err := r.ControlMonster(ten, m2.UniqueId(), 42); err != nil {
		t.Fatalf("ControlMonster: %v", err)
	}
	s, err = r.ApplyDamage(ten, 7, 10, m2.UniqueId(), now)
	if err != nil {
		t.Fatalf("ApplyDamage: %v", err)
	}
	if !s.WasFirstHit {
		t.Errorf("first hit on controlled monster should report WasFirstHit=true")
	}
	if !s.Monster.ControllerHasAggro() {
		t.Errorf("ControllerHasAggro must flip true after first hit")
	}

	// (c) Second hit on same monster reports false even from a different attacker.
	s, err = r.ApplyDamage(ten, 8, 5, m2.UniqueId(), now+1)
	if err != nil {
		t.Fatalf("ApplyDamage 2: %v", err)
	}
	if s.WasFirstHit {
		t.Errorf("subsequent hits must report WasFirstHit=false")
	}
}

// TestApplyDamageAggregatesByCharacterId verifies that two hits from the same
// character produce a single aggregated entry with summed damage and the most
// recent lastHitMs.
func TestApplyDamageAggregatesByCharacterId(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := testContext(ten)
	r.Clear(ctx)
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, ten, f, 9300018, 0, 0, 0, 5, 0, 1000, 50)

	if _, err := r.ApplyDamage(ten, 1, 10, m.UniqueId(), 100); err != nil {
		t.Fatalf("ApplyDamage 1: %v", err)
	}
	if _, err := r.ApplyDamage(ten, 1, 25, m.UniqueId(), 200); err != nil {
		t.Fatalf("ApplyDamage 2: %v", err)
	}
	got, err := r.GetMonster(ten, m.UniqueId())
	if err != nil {
		t.Fatalf("GetMonster: %v", err)
	}
	entries := got.DamageEntries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 aggregated entry, got %d (%+v)", len(entries), entries)
	}
	if entries[0].CharacterId != 1 || entries[0].Damage != 35 {
		t.Errorf("entry: expected charId=1 damage=35, got %+v", entries[0])
	}
	if entries[0].LastHitMs != 200 {
		t.Errorf("expected LastHitMs=200, got %d", entries[0].LastHitMs)
	}
}

// TestDecayDamageEntriesIdleEntriesDecay verifies that an entry idle past the
// threshold is decayed by AggroDecayMultiplier on each call, and that an entry
// not idle is unchanged.
func TestDecayDamageEntriesIdleEntriesDecay(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := testContext(ten)
	r.Clear(ctx)
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, ten, f, 9300018, 0, 0, 0, 5, 0, 1000, 50)

	// Two entries: one idle (lastHitMs=0), one fresh (lastHitMs=now).
	now := int64(20_000)
	if _, err := r.ApplyDamage(ten, 1, 100, m.UniqueId(), 0); err != nil {
		t.Fatalf("ApplyDamage 1: %v", err)
	}
	if _, err := r.ApplyDamage(ten, 2, 50, m.UniqueId(), now); err != nil {
		t.Fatalf("ApplyDamage 2: %v", err)
	}

	summary, err := r.DecayDamageEntries(ten, m.UniqueId(), now)
	if err != nil {
		t.Fatalf("DecayDamageEntries: %v", err)
	}
	if summary.AggroFlippedOff {
		t.Error("aggroFlippedOff should be false (entries remain)")
	}
	entries := summary.Monster.DamageEntries()
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	for _, e := range entries {
		if e.CharacterId == 1 && e.Damage != 85 {
			t.Errorf("idle entry should decay 100 -> 85, got %d", e.Damage)
		}
		if e.CharacterId == 2 && e.Damage != 50 {
			t.Errorf("fresh entry should remain 50, got %d", e.Damage)
		}
	}
}

// TestDecayDamageEntriesPrunesBelowFloor verifies the deterministic decay
// sequence and that entries below AggroDecayFloor are pruned.
func TestDecayDamageEntriesPrunesBelowFloor(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := testContext(ten)
	r.Clear(ctx)
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, ten, f, 9300018, 0, 0, 0, 5, 0, 1000, 50)

	if _, err := r.ApplyDamage(ten, 1, 100, m.UniqueId(), 0); err != nil {
		t.Fatalf("ApplyDamage: %v", err)
	}

	expected := []uint32{85, 72, 61, 51, 43, 36, 30, 25, 21, 17, 14, 11, 9, 7, 5, 4, 3, 2, 1}
	now := int64(20_000)
	for i, want := range expected {
		summary, err := r.DecayDamageEntries(ten, m.UniqueId(), now)
		if err != nil {
			t.Fatalf("Decay %d: %v", i, err)
		}
		entries := summary.Monster.DamageEntries()
		if len(entries) != 1 {
			t.Fatalf("Decay %d: expected 1 entry, got %d", i, len(entries))
		}
		if entries[0].Damage != want {
			t.Errorf("Decay %d: damage want=%d got=%d", i, want, entries[0].Damage)
		}
	}
	// Next decay drops below floor and prunes.
	summary, err := r.DecayDamageEntries(ten, m.UniqueId(), now)
	if err != nil {
		t.Fatalf("Decay final: %v", err)
	}
	if len(summary.Monster.DamageEntries()) != 0 {
		t.Fatalf("expected pruned (0 entries), got %d", len(summary.Monster.DamageEntries()))
	}
}

// TestDecayDamageEntriesFlipsAggroOffKeepingController verifies the post-fix
// behavior: when all entries prune on a monster with active aggro, the script
// flips controllerHasAggro false but keeps controlCharacterId in place. Losing
// aggro is not the same as losing control — the controller continues driving
// the monster's idle/wander AI on the client.
func TestDecayDamageEntriesFlipsAggroOffKeepingController(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := testContext(ten)
	r.Clear(ctx)
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, ten, f, 9300018, 0, 0, 0, 5, 0, 1000, 50)

	if _, err := r.ControlMonster(ten, m.UniqueId(), 42); err != nil {
		t.Fatalf("ControlMonster: %v", err)
	}
	// First-hit flips controllerHasAggro true server-side (controller exists).
	if _, err := r.ApplyDamage(ten, 1, 1, m.UniqueId(), 0); err != nil {
		t.Fatalf("ApplyDamage: %v", err)
	}
	now := int64(20_000)
	summary, err := r.DecayDamageEntries(ten, m.UniqueId(), now)
	if err != nil {
		t.Fatalf("DecayDamageEntries: %v", err)
	}
	if !summary.AggroFlippedOff {
		t.Fatal("expected AggroFlippedOff=true")
	}
	if summary.ControllerCharacterId != 42 {
		t.Errorf("expected ControllerCharacterId=42, got %d", summary.ControllerCharacterId)
	}
	if summary.Monster.ControlCharacterId() != 42 {
		t.Errorf("expected post-state controller=42 (not cleared), got %d", summary.Monster.ControlCharacterId())
	}
	if summary.Monster.ControllerHasAggro() {
		t.Error("expected post-state controllerHasAggro=false")
	}
}

// TestDecayDamageEntriesNoFlipWhenAggroAlreadyOff verifies that decaying a
// monster whose aggro is already off (e.g. no controller, or already passive)
// does NOT report AggroFlippedOff=true even when the entry list empties.
func TestDecayDamageEntriesNoFlipWhenAggroAlreadyOff(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := testContext(ten)
	r.Clear(ctx)
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, ten, f, 9300018, 0, 0, 0, 5, 0, 1000, 50)

	// No controller set -> ApplyDamage cannot flip aggro on.
	if _, err := r.ApplyDamage(ten, 1, 1, m.UniqueId(), 0); err != nil {
		t.Fatalf("ApplyDamage: %v", err)
	}
	now := int64(20_000)
	summary, err := r.DecayDamageEntries(ten, m.UniqueId(), now)
	if err != nil {
		t.Fatalf("DecayDamageEntries: %v", err)
	}
	if summary.AggroFlippedOff {
		t.Error("expected AggroFlippedOff=false when aggro was already off")
	}
}

// TestDecayDamageEntriesLegacyEntryWithoutLastHitMs verifies the Lua script
// tolerates legacy Redis blobs whose damage entries lack the `lastHitMs` field
// entirely — without throwing 'attempt to perform arithmetic on a nil value'.
func TestDecayDamageEntriesLegacyEntryWithoutLastHitMs(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := testContext(ten)
	r.Clear(ctx)
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, ten, f, 9300018, 0, 0, 0, 5, 0, 1000, 50)

	legacy := `{"uniqueId":` + strconv.FormatUint(uint64(m.UniqueId()), 10) +
		`,"tenantId":"` + ten.Id().String() + `","tenantRegion":"GMS"` +
		`,"tenantMajorVersion":83,"tenantMinorVersion":1` +
		`,"worldId":0,"channelId":0,"mapId":40000` +
		`,"instance":"00000000-0000-0000-0000-000000000000"` +
		`,"maxHp":1000,"hp":1000,"maxMp":50,"mp":50` +
		`,"monsterId":9300018,"controlCharacterId":0` +
		`,"x":0,"y":0,"fh":0,"stance":5,"team":0` +
		`,"damageEntries":[{"characterId":7,"damage":100}]` +
		`,"statusEffects":[]}`
	testMiniRedis.Set(monsterKey(ten, m.UniqueId()), legacy)

	now := int64(20_000)
	summary, err := r.DecayDamageEntries(ten, m.UniqueId(), now)
	if err != nil {
		t.Fatalf("DecayDamageEntries on legacy blob: %v", err)
	}
	entries := summary.Monster.DamageEntries()
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry after decay, got %d", len(entries))
	}
	if entries[0].Damage != 85 {
		t.Errorf("legacy entry should decay 100 -> 85 (treated as idle), got %d", entries[0].Damage)
	}
}

// TestDecayDamageEntriesNoOpWhenAllFresh verifies a no-op when nothing is idle.
func TestDecayDamageEntriesNoOpWhenAllFresh(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := testContext(ten)
	r.Clear(ctx)
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, ten, f, 9300018, 0, 0, 0, 5, 0, 1000, 50)

	now := int64(20_000)
	if _, err := r.ApplyDamage(ten, 1, 100, m.UniqueId(), now); err != nil {
		t.Fatalf("ApplyDamage: %v", err)
	}
	summary, err := r.DecayDamageEntries(ten, m.UniqueId(), now)
	if err != nil {
		t.Fatalf("DecayDamageEntries: %v", err)
	}
	entries := summary.Monster.DamageEntries()
	if len(entries) != 1 || entries[0].Damage != 100 {
		t.Errorf("fresh entry should be untouched: %+v", entries)
	}
}

// TestApplyDamageWritesLastDamageTakenMs verifies that ApplyDamage stamps the
// monster's lastDamageTakenMs with the passed nowMs (drives the recovery
// task's HP-regen idle gate).
func TestApplyDamageWritesLastDamageTakenMs(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := testContext(ten)
	r.Clear(ctx)
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, ten, f, 9300018, 0, 0, 0, 5, 0, 1000, 50)

	now := int64(1_700_000_000_000)
	if _, err := r.ApplyDamage(ten, 1, 10, m.UniqueId(), now); err != nil {
		t.Fatalf("ApplyDamage: %v", err)
	}

	got, err := r.GetMonster(ten, m.UniqueId())
	if err != nil {
		t.Fatalf("GetMonster: %v", err)
	}
	if got.LastDamageTakenMs() != now {
		t.Errorf("expected lastDamageTakenMs=%d after damage; got %d", now, got.LastDamageTakenMs())
	}
}

// TestFromStoredCollapsesLegacyDamageEntries verifies that a Redis blob with
// the old multi-row-per-character shape and no lastHitMs round-trips into a
// single aggregated entry per character with LastHitMs == 0.
func TestFromStoredCollapsesLegacyDamageEntries(t *testing.T) {
	r := GetMonsterRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := testContext(ten)
	r.Clear(ctx)

	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, ten, f, 9300018, 0, 0, 0, 5, 0, 100, 50)

	legacy := `{"uniqueId":` + strconv.FormatUint(uint64(m.UniqueId()), 10) +
		`,"tenantId":"` + ten.Id().String() + `","tenantRegion":"GMS"` +
		`,"tenantMajorVersion":83,"tenantMinorVersion":1` +
		`,"worldId":0,"channelId":0,"mapId":40000` +
		`,"instance":"00000000-0000-0000-0000-000000000000"` +
		`,"maxHp":100,"hp":100,"maxMp":50,"mp":50` +
		`,"monsterId":9300018,"controlCharacterId":0` +
		`,"x":0,"y":0,"fh":0,"stance":5,"team":0` +
		`,"damageEntries":[` +
		`{"characterId":7,"damage":10},` +
		`{"characterId":7,"damage":15},` +
		`{"characterId":9,"damage":5}` +
		`],"statusEffects":[]}`
	testMiniRedis.Set(monsterKey(ten, m.UniqueId()), legacy)

	got, err := r.GetMonster(ten, m.UniqueId())
	if err != nil {
		t.Fatalf("GetMonster failed: %v", err)
	}
	entries := got.DamageEntries()
	if len(entries) != 2 {
		t.Fatalf("expected 2 aggregated entries, got %d (%+v)", len(entries), entries)
	}
	for _, e := range entries {
		if e.CharacterId == 7 && e.Damage != 25 {
			t.Errorf("character 7 damage: expected 25, got %d", e.Damage)
		}
		if e.CharacterId == 9 && e.Damage != 5 {
			t.Errorf("character 9 damage: expected 5, got %d", e.Damage)
		}
		if e.LastHitMs != 0 {
			t.Errorf("legacy entry should default LastHitMs=0, got %d", e.LastHitMs)
		}
	}
}
