package monster

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	atlasredis "github.com/Chronicle20/atlas/libs/atlas-redis"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// storedMonster is the JSON-serializable representation stored in Redis.
type storedMonster struct {
	UniqueId               uint32           `json:"uniqueId"`
	TenantId               string           `json:"tenantId"`
	TenantRegion           string           `json:"tenantRegion"`
	TenantMajorVersion     uint16           `json:"tenantMajorVersion"`
	TenantMinorVersion     uint16           `json:"tenantMinorVersion"`
	WorldId                byte             `json:"worldId"`
	ChannelId              byte             `json:"channelId"`
	MapId                  uint32           `json:"mapId"`
	Instance               string           `json:"instance"`
	MaxHp                  uint32           `json:"maxHp"`
	Hp                     uint32           `json:"hp"`
	MaxMp                  uint32           `json:"maxMp"`
	Mp                     uint32           `json:"mp"`
	MonsterId              uint32           `json:"monsterId"`
	ControlCharacterId     uint32           `json:"controlCharacterId"`
	ControllerHasAggro     bool             `json:"controllerHasAggro"`
	X                      int16            `json:"x"`
	Y                      int16            `json:"y"`
	Fh                     int16            `json:"fh"`
	Stance                 byte             `json:"stance"`
	Team                   int8             `json:"team"`
	DamageEntries          damageEntryList  `json:"damageEntries"`
	StatusEffects          statusEffectList `json:"statusEffects"`
	NextEligibleRepickAtMs int64            `json:"nextEligibleRepickAtMs,omitempty"`
	LastDamageTakenMs      int64            `json:"lastDamageTakenMs,omitempty"`
}

// damageEntryList and statusEffectList tolerate the empty-object form ("{}")
// produced by Redis' Lua cjson when it re-encodes an empty Lua table. Without
// this, the applyDamageScript corrupts a freshly-spawned monster on its first
// hit: an empty statusEffects array round-trips to "{}", and subsequent Go
// unmarshals fail with "cannot unmarshal object into ... []storedStatusEffect".
type damageEntryList []storedDamageEntry

func (l *damageEntryList) UnmarshalJSON(data []byte) error {
	return unmarshalTolerantArray(data, (*[]storedDamageEntry)(l))
}

type statusEffectList []storedStatusEffect

func (l *statusEffectList) UnmarshalJSON(data []byte) error {
	return unmarshalTolerantArray(data, (*[]storedStatusEffect)(l))
}

func unmarshalTolerantArray[T any](data []byte, out *[]T) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 2 && trimmed[0] == '{' && trimmed[1] == '}' {
		*out = nil
		return nil
	}
	return json.Unmarshal(data, out)
}

type storedDamageEntry struct {
	CharacterId uint32 `json:"characterId"`
	Damage      uint32 `json:"damage"`
	LastHitMs   int64  `json:"lastHitMs"`
}

type storedStatusEffect struct {
	EffectId          string           `json:"effectId"`
	SourceType        string           `json:"sourceType"`
	SourceCharacterId uint32           `json:"sourceCharacterId"`
	SourceSkillId     uint32           `json:"sourceSkillId"`
	SourceSkillLevel  uint32           `json:"sourceSkillLevel"`
	Statuses          map[string]int32 `json:"statuses"`
	DurationMs        int64            `json:"durationMs"`
	TickIntervalMs    int64            `json:"tickIntervalMs"`
	LastTickMs        int64            `json:"lastTickMs"`
	CreatedAtMs       int64            `json:"createdAtMs"`
	ExpiresAtMs       int64            `json:"expiresAtMs"`
	ReflectKind       string           `json:"reflectKind,omitempty"`
	ReflectPercent    int32            `json:"reflectPercent,omitempty"`
	ReflectLtX        int16            `json:"reflectLtX,omitempty"`
	ReflectLtY        int16            `json:"reflectLtY,omitempty"`
	ReflectRbX        int16            `json:"reflectRbX,omitempty"`
	ReflectRbY        int16            `json:"reflectRbY,omitempty"`
	ReflectMaxDamage  int32            `json:"reflectMaxDamage,omitempty"`
}

func toStored(t tenant.Model, m Model) storedMonster {
	des := make([]storedDamageEntry, 0, len(m.damageEntries))
	for _, e := range m.damageEntries {
		des = append(des, storedDamageEntry{
			CharacterId: e.CharacterId,
			Damage:      e.Damage,
			LastHitMs:   e.LastHitMs,
		})
	}
	ses := make([]storedStatusEffect, 0, len(m.statusEffects))
	for _, se := range m.statusEffects {
		ses = append(ses, storedStatusEffect{
			EffectId:          se.effectId.String(),
			SourceType:        se.sourceType,
			SourceCharacterId: se.sourceCharacterId,
			SourceSkillId:     se.sourceSkillId,
			SourceSkillLevel:  se.sourceSkillLevel,
			Statuses:          se.statuses,
			DurationMs:        se.duration.Milliseconds(),
			TickIntervalMs:    se.tickInterval.Milliseconds(),
			LastTickMs:        se.lastTick.UnixMilli(),
			CreatedAtMs:       se.createdAt.UnixMilli(),
			ExpiresAtMs:       se.expiresAt.UnixMilli(),
			ReflectKind:       se.reflectKind,
			ReflectPercent:    se.reflectPercent,
			ReflectLtX:        se.reflectLtX,
			ReflectLtY:        se.reflectLtY,
			ReflectRbX:        se.reflectRbX,
			ReflectRbY:        se.reflectRbY,
			ReflectMaxDamage:  se.reflectMaxDamage,
		})
	}
	return storedMonster{
		UniqueId:               m.uniqueId,
		TenantId:               t.Id().String(),
		TenantRegion:           t.Region(),
		TenantMajorVersion:     t.MajorVersion(),
		TenantMinorVersion:     t.MinorVersion(),
		WorldId:                byte(m.worldId),
		ChannelId:              byte(m.channelId),
		MapId:                  uint32(m.mapId),
		Instance:               m.instance.String(),
		MaxHp:                  m.maxHp,
		Hp:                     m.hp,
		MaxMp:                  m.maxMp,
		Mp:                     m.mp,
		MonsterId:              m.monsterId,
		ControlCharacterId:     m.controlCharacterId,
		ControllerHasAggro:     m.controllerHasAggro,
		X:                      m.x,
		Y:                      m.y,
		Fh:                     m.fh,
		Stance:                 m.stance,
		Team:                   m.team,
		DamageEntries:          des,
		StatusEffects:          ses,
		NextEligibleRepickAtMs: m.nextSkillDecision.nextEligibleRepickAtMs,
		LastDamageTakenMs:      m.lastDamageTakenMs,
	}
}

func fromStored(sm storedMonster) (tenant.Model, Model, error) {
	tenantId, err := uuid.Parse(sm.TenantId)
	if err != nil {
		return tenant.Model{}, Model{}, err
	}
	t, err := tenant.Create(tenantId, sm.TenantRegion, sm.TenantMajorVersion, sm.TenantMinorVersion)
	if err != nil {
		return tenant.Model{}, Model{}, err
	}
	inst, err := uuid.Parse(sm.Instance)
	if err != nil {
		return tenant.Model{}, Model{}, err
	}

	agg := make(map[uint32]*entry)
	order := make([]uint32, 0, len(sm.DamageEntries))
	for _, de := range sm.DamageEntries {
		if existing, ok := agg[de.CharacterId]; ok {
			existing.Damage += de.Damage
			// Take the latest non-zero lastHitMs; legacy rows have 0.
			if de.LastHitMs > existing.LastHitMs {
				existing.LastHitMs = de.LastHitMs
			}
			continue
		}
		agg[de.CharacterId] = &entry{
			CharacterId: de.CharacterId,
			Damage:      de.Damage,
			LastHitMs:   de.LastHitMs,
		}
		order = append(order, de.CharacterId)
	}
	des := make([]entry, 0, len(order))
	for _, cid := range order {
		des = append(des, *agg[cid])
	}
	ses := make([]StatusEffect, 0, len(sm.StatusEffects))
	for _, sse := range sm.StatusEffects {
		eid, err := uuid.Parse(sse.EffectId)
		if err != nil {
			return tenant.Model{}, Model{}, err
		}
		ses = append(ses, StatusEffect{
			effectId:          eid,
			sourceType:        sse.SourceType,
			sourceCharacterId: sse.SourceCharacterId,
			sourceSkillId:     sse.SourceSkillId,
			sourceSkillLevel:  sse.SourceSkillLevel,
			statuses:          sse.Statuses,
			duration:          time.Duration(sse.DurationMs) * time.Millisecond,
			tickInterval:      time.Duration(sse.TickIntervalMs) * time.Millisecond,
			lastTick:          time.UnixMilli(sse.LastTickMs),
			createdAt:         time.UnixMilli(sse.CreatedAtMs),
			expiresAt:         time.UnixMilli(sse.ExpiresAtMs),
			reflectKind:       sse.ReflectKind,
			reflectPercent:    sse.ReflectPercent,
			reflectLtX:        sse.ReflectLtX,
			reflectLtY:        sse.ReflectLtY,
			reflectRbX:        sse.ReflectRbX,
			reflectRbY:        sse.ReflectRbY,
			reflectMaxDamage:  sse.ReflectMaxDamage,
		})
	}

	return t, Model{
		uniqueId:           sm.UniqueId,
		worldId:            world.Id(sm.WorldId),
		channelId:          channel.Id(sm.ChannelId),
		mapId:              _map.Id(sm.MapId),
		instance:           inst,
		maxHp:              sm.MaxHp,
		hp:                 sm.Hp,
		maxMp:              sm.MaxMp,
		mp:                 sm.Mp,
		monsterId:          sm.MonsterId,
		controlCharacterId: sm.ControlCharacterId,
		controllerHasAggro: sm.ControllerHasAggro,
		x:                  sm.X,
		y:                  sm.Y,
		fh:                 sm.Fh,
		stance:             sm.Stance,
		team:               sm.Team,
		damageEntries:      des,
		statusEffects:      ses,
		nextSkillDecision: nextSkillDecision{
			nextEligibleRepickAtMs: sm.NextEligibleRepickAtMs,
		},
		lastDamageTakenMs: sm.LastDamageTakenMs,
	}, nil
}

// errMonsterNotFound is the package-internal sentinel returned when a monster
// key is absent. Preserves the exact error string the raw-redis implementation
// surfaced to callers (no caller string-matches it; they only check err != nil).
var errMonsterNotFound = errors.New("monster not found")

type Registry struct {
	// reg backs the monster store via the tenant-scoped API (D7): the stored
	// key is atlas:monster:<tenantId>:<region>:<major>.<minor>:<uniqueId>.
	reg *atlasredis.TenantRegistry[uint32, storedMonster]
	// mapIdx backs the per-field membership SET, tenant-scoped: the SET key is
	// atlas:monster-map:<tenantId>:<region>:<major>.<minor>:<world>:<channel>:<map>:<instance>.
	mapIdx *atlasredis.TenantKeyedSet[string]
	// scan and scanMapIdx are bare (non-tenant-scoped) registries pointed at
	// the SAME client/namespace as reg/mapIdx. They observe exactly the keys
	// reg/mapIdx write: tenantEntityKey(ns, t, suffix) joins
	// [prefix, ns, TenantKey(t), suffix] with ":", and a bare Registry/KeyedSet
	// with an identity keyFn fed the pre-joined "TenantKey(t):suffix" string
	// produces the identical final key. They exist ONLY for the two operations
	// the tenant-scoped API has no equivalent for: GetMonsters() (the periodic
	// sweep tasks have no tenant to loop over — they need every tenant with
	// live monster data) and Clear()/ClaimMonster's atomic delete-if-exists
	// (Clear is a test-only full-namespace wipe; TenantRegistry.Clear requires
	// a specific tenant). Do not reach for these where a tenant.Model is
	// already in hand — use reg/mapIdx instead.
	scan       *atlasredis.Registry[string, storedMonster]
	scanMapIdx *atlasredis.KeyedSet[string]
}

var (
	registry *Registry
	once     sync.Once
)

func InitMonsterRegistry(rc *goredis.Client) {
	once.Do(func() {
		registry = &Registry{
			reg:        atlasredis.NewTenantRegistry[uint32, storedMonster](rc, "monster", func(id uint32) string { return strconv.FormatUint(uint64(id), 10) }),
			mapIdx:     atlasredis.NewTenantKeyedSet[string](rc, "monster-map", func(s string) string { return s }),
			scan:       atlasredis.NewRegistry[string, storedMonster](rc, "monster", func(s string) string { return s }),
			scanMapIdx: atlasredis.NewKeyedSet[string](rc, "monster-map", func(s string) string { return s }),
		}
	})
}

func GetMonsterRegistry() *Registry {
	return registry
}

// monsterKey returns the full Redis key for a monster under the tenant-scoped
// layout: atlas:monster:<tenantId>:<region>:<major>.<minor>:<uniqueId>.
// Retained for tests that seed raw blobs directly into Redis.
func monsterKey(t tenant.Model, uniqueId uint32) string {
	return fmt.Sprintf("%s:monster:%s:%d", atlasredis.KeyPrefix(), atlasredis.TenantKey(t), uniqueId)
}

// claimKey reproduces reg's entity key for uniqueId under tenant t, for use
// with scan.RemoveExisting (see the Registry.scan field comment).
func claimKey(t tenant.Model, uniqueId uint32) string {
	return atlasredis.TenantKey(t) + ":" + strconv.FormatUint(uint64(uniqueId), 10)
}

// mapIndexKey is the tenant-scoped map-index SET's entity key: the tenant
// segment is supplied by TenantKeyedSet, so this carries only the field
// coordinates.
func mapIndexKey(worldId byte, channelId byte, mapId uint32, instance string) string {
	return fmt.Sprintf("%d:%d:%d:%s", worldId, channelId, mapId, instance)
}

func mapIndexKeyFromField(f field.Model) string {
	return mapIndexKey(byte(f.WorldId()), byte(f.ChannelId()), uint32(f.MapId()), f.Instance().String())
}

func mapIndexKeyFromModel(m Model) string {
	return mapIndexKey(byte(m.worldId), byte(m.channelId), uint32(m.mapId), m.instance.String())
}

func (r *Registry) storeMonster(ctx context.Context, t tenant.Model, m Model) error {
	return r.reg.Put(ctx, t, m.uniqueId, toStored(t, m))
}

func (r *Registry) loadMonster(ctx context.Context, t tenant.Model, uniqueId uint32) (Model, error) {
	sm, err := r.reg.Get(ctx, t, uniqueId)
	if errors.Is(err, atlasredis.ErrNotFound) {
		return Model{}, errMonsterNotFound
	}
	if err != nil {
		return Model{}, err
	}
	_, m, err := fromStored(sm)
	return m, err
}

// atomicUpdate applies fn to the monster under optimistic lock (Registry.Update
// does Watch+GET+fn+TxPipelined(SET) with retry on contention). fn must be pure
// in its observable effects — it may run multiple times under retry.
func (r *Registry) atomicUpdate(ctx context.Context, t tenant.Model, uniqueId uint32, fn func(m Model) Model) (Model, error) {
	sm, err := r.reg.Update(ctx, t, uniqueId, func(cur storedMonster) storedMonster {
		_, m, derr := fromStored(cur)
		if derr != nil {
			// Cannot decode current state; leave it untouched. The decode error
			// surfaces below via the re-decode of the returned (unchanged) value.
			return cur
		}
		return toStored(t, fn(m))
	})
	if errors.Is(err, atlasredis.ErrNotFound) {
		return Model{}, errMonsterNotFound
	}
	if err != nil {
		return Model{}, err
	}
	_, m, err := fromStored(sm)
	return m, err
}

func (r *Registry) CreateMonster(ctx context.Context, t tenant.Model, f field.Model, monsterId uint32, x int16, y int16, fh int16, stance byte, team int8, hp uint32, mp uint32) Model {
	uniqueId := GetIdAllocator().Allocate(ctx, t)
	m := NewMonster(f, uniqueId, monsterId, x, y, fh, stance, team, hp, mp)

	// The old pipeline issued Set+SAdd ignoring errors; sequential calls match.
	_ = r.reg.Put(ctx, t, uniqueId, toStored(t, m))
	_ = r.mapIdx.Add(ctx, t, mapIndexKeyFromField(f), strconv.FormatUint(uint64(uniqueId), 10))

	return m
}

func (r *Registry) GetMonster(tenant tenant.Model, uniqueId uint32) (Model, error) {
	return r.loadMonster(context.Background(), tenant, uniqueId)
}

func (r *Registry) GetMonstersInMap(tenant tenant.Model, f field.Model) []Model {
	ctx := context.Background()
	members, err := r.mapIdx.Members(ctx, tenant, mapIndexKeyFromField(f))
	if err != nil || len(members) == 0 {
		return nil
	}

	var result []Model
	for _, idStr := range members {
		uid, perr := strconv.ParseUint(idStr, 10, 32)
		if perr != nil {
			continue
		}
		sm, gerr := r.reg.Get(ctx, tenant, uint32(uid))
		if gerr != nil {
			continue
		}
		_, m, gerr := fromStored(sm)
		if gerr != nil {
			continue
		}
		result = append(result, m)
	}
	return result
}

func (r *Registry) MoveMonster(tenant tenant.Model, uniqueId uint32, endX int16, endY int16, endFh int16, stance byte) Model {
	m, err := r.atomicUpdate(context.Background(), tenant, uniqueId, func(m Model) Model {
		return m.Move(endX, endY, endFh, stance)
	})
	if err != nil {
		return Model{}
	}
	return m
}

func (r *Registry) ControlMonster(tenant tenant.Model, uniqueId uint32, characterId uint32) (Model, error) {
	return r.atomicUpdate(context.Background(), tenant, uniqueId, func(m Model) Model {
		return m.Control(characterId)
	})
}

// ControlMonsterWithAggro assigns the controller and sets the aggro flag in one
// atomic transition. Used by the forced-control path so the resulting
// START_CONTROL event carries controllerHasAggro = true.
func (r *Registry) ControlMonsterWithAggro(tenant tenant.Model, uniqueId uint32, characterId uint32) (Model, error) {
	return r.atomicUpdate(context.Background(), tenant, uniqueId, func(m Model) Model {
		return m.ControlWithAggro(characterId)
	})
}

func (r *Registry) ClearControl(tenant tenant.Model, uniqueId uint32) (Model, error) {
	return r.atomicUpdate(context.Background(), tenant, uniqueId, func(m Model) Model {
		return m.ClearControl()
	})
}

// ApplyDamage atomically applies clamped damage to the monster, aggregates the
// per-character damage entry (summing damage and stamping lastHitMs=nowMs),
// stamps lastDamageTakenMs=nowMs, and flips controllerHasAggro true on the first
// hit of a controlled monster. Ported from the former applyDamageScript Lua via
// Registry.Update; the closure is pure (wasFirstHit derives only from cur), so
// the captured summary reflects the final successful invocation under retry.
func (r *Registry) ApplyDamage(t tenant.Model, characterId uint32, damage uint32, uniqueId uint32, nowMs int64) (DamageSummary, error) {
	ctx := context.Background()

	var wasFirstHit bool
	sm, err := r.reg.Update(ctx, t, uniqueId, func(cur storedMonster) storedMonster {
		hp := cur.Hp
		// actual = hp - max(hp - damage, 0): clamp at 0, never below.
		var actual uint32
		if damage >= hp {
			actual = hp
		} else {
			actual = damage
		}
		cur.Hp = hp - actual

		found := false
		for i := range cur.DamageEntries {
			if cur.DamageEntries[i].CharacterId == characterId {
				cur.DamageEntries[i].Damage += actual
				cur.DamageEntries[i].LastHitMs = nowMs
				found = true
				break
			}
		}
		if !found {
			cur.DamageEntries = append(cur.DamageEntries, storedDamageEntry{
				CharacterId: characterId,
				Damage:      actual,
				LastHitMs:   nowMs,
			})
		}
		cur.LastDamageTakenMs = nowMs

		wasFirstHit = cur.ControlCharacterId != 0 && !cur.ControllerHasAggro
		if wasFirstHit {
			cur.ControllerHasAggro = true
		}
		return cur
	})
	if err != nil {
		// Old Lua collapsed every failure (including absent key) to "monster not found".
		return DamageSummary{}, errMonsterNotFound
	}

	_, m, err := fromStored(sm)
	if err != nil {
		return DamageSummary{}, err
	}

	return DamageSummary{
		CharacterId:   characterId,
		Monster:       m,
		VisibleDamage: damage,
		Killed:        m.Hp() == 0,
		WasFirstHit:   wasFirstHit,
	}, nil
}

// ApplyRecovery atomically applies HP/MP recovery to the monster. Returns the
// updated Model along with flags indicating whether HP and MP were actually
// changed. HP recovery is gated by the idle window: applies only when
// nowMs - lastDamageTakenMs > AggroIdleThresholdMs. MP recovery is unconditional
// (independent of SEAL and other cast-blocking statuses, per design D5).
// A dead mob (hp == 0) is skipped — healing the dead is forbidden.
//
// Ported from the former applyRecoveryScript Lua via Registry.Update. The Lua
// skipped the SET when nothing applied; Update always writes, but in that case
// it writes back identical state, so the observable result is the same. The
// hpApplied/mpApplied flags derive purely from cur, so the captured values
// reflect the final successful invocation under optimistic-lock retry.
func (r *Registry) ApplyRecovery(t tenant.Model, uniqueId uint32, hpRecovery, mpRecovery uint32, nowMs int64) (Model, bool, bool, error) {
	ctx := context.Background()

	var hpApplied, mpApplied bool
	sm, err := r.reg.Update(ctx, t, uniqueId, func(cur storedMonster) storedMonster {
		hpApplied = false
		mpApplied = false

		if cur.Hp == 0 {
			return cur
		}

		if hpRecovery > 0 && cur.Hp < cur.MaxHp {
			if (nowMs - cur.LastDamageTakenMs) > AggroIdleThresholdMs {
				newHp := cur.Hp + hpRecovery
				if newHp > cur.MaxHp {
					newHp = cur.MaxHp
				}
				cur.Hp = newHp
				hpApplied = true
			}
		}

		if mpRecovery > 0 && cur.Mp < cur.MaxMp {
			newMp := cur.Mp + mpRecovery
			if newMp > cur.MaxMp {
				newMp = cur.MaxMp
			}
			cur.Mp = newMp
			mpApplied = true
		}

		return cur
	})
	if errors.Is(err, atlasredis.ErrNotFound) {
		return Model{}, false, false, errMonsterNotFound
	}
	if err != nil {
		return Model{}, false, false, err
	}
	_, m, err := fromStored(sm)
	if err != nil {
		return Model{}, false, false, err
	}
	return m, hpApplied, mpApplied, nil
}

func (r *Registry) RemoveMonster(ctx context.Context, t tenant.Model, uniqueId uint32) (Model, error) {
	sm, err := r.reg.Get(ctx, t, uniqueId)
	if errors.Is(err, atlasredis.ErrNotFound) {
		return Model{}, errMonsterNotFound
	}
	if err != nil {
		return Model{}, err
	}
	_, m, err := fromStored(sm)
	if err != nil {
		return Model{}, err
	}

	// Old pipeline issued Del+SRem ignoring errors; sequential calls match.
	_ = r.reg.Remove(ctx, t, uniqueId)
	_ = r.mapIdx.Remove(ctx, t, mapIndexKeyFromModel(m), strconv.FormatUint(uint64(uniqueId), 10))

	GetIdAllocator().Release(ctx, t, uniqueId)
	return m, nil
}

// ClaimMonster removes a monster and reports whether THIS caller was the one
// that removed it. Unlike RemoveMonster (Get-then-Del with the Del reply
// discarded), the delete is the claim: exactly one of N concurrent callers can
// observe ok=true, so exactly one catch attempt on a given monster can succeed.
// A missing monster and a lost race are both (Model{}, false, nil) — neither is
// an error, and neither should emit anything.
//
// The map-index removal and id release run ONLY for the winner, so a loser
// cannot recycle an id the winner still owns.
func (r *Registry) ClaimMonster(ctx context.Context, t tenant.Model, uniqueId uint32) (Model, bool, error) {
	sm, err := r.reg.Get(ctx, t, uniqueId)
	if errors.Is(err, atlasredis.ErrNotFound) {
		return Model{}, false, nil
	}
	if err != nil {
		return Model{}, false, err
	}
	_, m, err := fromStored(sm)
	if err != nil {
		return Model{}, false, err
	}

	claimed, err := r.scan.RemoveExisting(ctx, claimKey(t, uniqueId))
	if err != nil {
		return Model{}, false, err
	}
	if !claimed {
		return Model{}, false, nil
	}

	_ = r.mapIdx.Remove(ctx, t, mapIndexKeyFromModel(m), strconv.FormatUint(uint64(uniqueId), 10))
	GetIdAllocator().Release(ctx, t, uniqueId)
	return m, true, nil
}

func (r *Registry) ApplyStatusEffect(t tenant.Model, uniqueId uint32, effect StatusEffect) (Model, error) {
	return r.atomicUpdate(context.Background(), t, uniqueId, func(m Model) Model {
		return m.ApplyStatus(effect)
	})
}

func (r *Registry) CancelStatusEffect(t tenant.Model, uniqueId uint32, effectId uuid.UUID) (Model, error) {
	return r.atomicUpdate(context.Background(), t, uniqueId, func(m Model) Model {
		return m.CancelStatus(effectId)
	})
}

func (r *Registry) CancelStatusEffectByType(t tenant.Model, uniqueId uint32, statusType string) (Model, error) {
	return r.atomicUpdate(context.Background(), t, uniqueId, func(m Model) Model {
		return m.CancelStatusByType(statusType)
	})
}

func (r *Registry) CancelAllStatusEffects(t tenant.Model, uniqueId uint32) (Model, error) {
	return r.atomicUpdate(context.Background(), t, uniqueId, func(m Model) Model {
		return m.CancelAllStatuses()
	})
}

func (r *Registry) UpdateStatusEffectLastTick(t tenant.Model, uniqueId uint32, effectId uuid.UUID, tickTime time.Time) (Model, error) {
	return r.atomicUpdate(context.Background(), t, uniqueId, func(m Model) Model {
		updated := make([]StatusEffect, 0, len(m.statusEffects))
		for _, se := range m.statusEffects {
			if se.EffectId() == effectId {
				se = se.WithLastTick(tickTime)
			}
			updated = append(updated, se)
		}
		result := Clone(m).Build()
		result.statusEffects = updated
		return result
	})
}

func (r *Registry) DeductMp(t tenant.Model, uniqueId uint32, amount uint32) (Model, error) {
	return r.atomicUpdate(context.Background(), t, uniqueId, func(m Model) Model {
		return m.DeductMp(amount)
	})
}

// SetNextSkillDecision atomically replaces the monster's picker decision.
// The skill choice (skillId, skillLevel, decidedAtMs) is in-memory only and
// dropped on Redis round-trip — atlas-channel's nextSkillInbox is the source
// of truth for what the monster will cast next. Only nextEligibleRepickAtMs
// survives a round-trip, so the sweep task can identify monsters whose
// cooldown-driven next-repick window has elapsed across in-memory rebuilds.
// On rehydration of the skill choice fields, the picker re-runs and emits a
// fresh decision via NEXT_SKILL_DECIDED.
func (r *Registry) SetNextSkillDecision(t tenant.Model, uniqueId uint32, d nextSkillDecision) (Model, error) {
	return r.atomicUpdate(context.Background(), t, uniqueId, func(m Model) Model {
		return Clone(m).SetNextSkillDecision(d).Build()
	})
}

func (r *Registry) UpdateMonster(t tenant.Model, uniqueId uint32, m Model) {
	r.storeMonster(context.Background(), t, m)
}

func (r *Registry) GetMonsters() map[tenant.Model][]Model {
	ctx := context.Background()
	result := make(map[tenant.Model][]Model)

	all, err := r.scan.GetAll(ctx)
	if err != nil {
		return result
	}
	for _, sm := range all {
		t, m, derr := fromStored(sm)
		if derr != nil {
			continue
		}
		result[t] = append(result[t], m)
	}
	return result
}

func (r *Registry) Clear(ctx context.Context) {
	_, _ = r.scan.Clear(ctx)
	_, _ = r.scanMapIdx.ClearAll(ctx)
}

// DecaySummary is returned by DecayDamageEntries. AggroFlippedOff is true when
// the entry list became empty and the monster's controller was switched from
// active to passive (controllerHasAggro flipped true→false). The controller is
// NOT cleared during decay — losing aggro is not the same as losing control;
// the existing controller continues to drive the monster's idle/wander AI on
// the client.
type DecaySummary struct {
	Monster               Model
	ControllerCharacterId uint32
	AggroFlippedOff       bool
}

// DecayDamageEntries atomically decays idle damage entries, prunes any that
// fall below AggroDecayFloor, and flips controllerHasAggro false when the entry
// list empties on a monster that had aggro. Ported from the former
// decayDamageEntriesScript Lua via Registry.Update. aggroFlippedOff and
// controllerCharacterId derive purely from cur, so the captured values reflect
// the final successful invocation under optimistic-lock retry.
func (r *Registry) DecayDamageEntries(t tenant.Model, uniqueId uint32, nowMs int64) (DecaySummary, error) {
	ctx := context.Background()

	var aggroFlippedOff bool
	var controllerCharacterId uint32
	sm, err := r.reg.Update(ctx, t, uniqueId, func(cur storedMonster) storedMonster {
		aggroFlippedOff = false

		kept := make([]storedDamageEntry, 0, len(cur.DamageEntries))
		for _, e := range cur.DamageEntries {
			// Legacy entries written before lastHitMs shipped default to 0
			// (treated as idle).
			if (nowMs - e.LastHitMs) > AggroIdleThresholdMs {
				// Mirror the Lua math.floor(e.damage * mult).
				e.Damage = uint32(math.Floor(float64(e.Damage) * AggroDecayMultiplier))
			}
			if e.Damage >= AggroDecayFloor {
				kept = append(kept, e)
			}
		}
		cur.DamageEntries = kept

		if len(kept) == 0 && cur.ControllerHasAggro {
			cur.ControllerHasAggro = false
			aggroFlippedOff = true
		}

		controllerCharacterId = cur.ControlCharacterId
		return cur
	})
	if errors.Is(err, atlasredis.ErrNotFound) {
		return DecaySummary{}, errMonsterNotFound
	}
	if err != nil {
		return DecaySummary{}, err
	}
	_, m, err := fromStored(sm)
	if err != nil {
		return DecaySummary{}, err
	}
	return DecaySummary{
		Monster:               m,
		ControllerCharacterId: controllerCharacterId,
		AggroFlippedOff:       aggroFlippedOff,
	}, nil
}

// ClearSummary is returned by ClearDamageEntries. It mirrors DecaySummary
// field-for-field so callers of both converge on the same emit decision.
type ClearSummary struct {
	Monster               Model
	ControllerCharacterId uint32
	AggroFlippedOff       bool
}

// ClearDamageEntries atomically wipes EVERY damage entry on the monster and
// flips controllerHasAggro false when the monster had aggro. This is a full
// wipe, not a decay toward AggroDecayFloor (FR-4.2).
//
// It deliberately converges on the same state DecayDamageEntries reaches when
// its entry list empties, which is what makes the interaction with the decay
// sweep and the controller picker correct (FR-4.4): the sweep's next tick sees
// an empty list and does nothing, and the picker's ControllerHasAggro gate
// behaves identically to a naturally-decayed monster. The controller itself is
// NOT cleared — losing aggro is not losing control.
//
// Sets cur.DamageEntries to a non-nil empty slice (matching the `kept` slice
// DecayDamageEntries produces via make([]storedDamageEntry, 0, n) when it
// prunes everything) rather than nil, so the two operations serialize to the
// identical stored representation.
//
// Written against storedMonster via reg.Update rather than atomicUpdate for the
// same reason DecayDamageEntries is: Model exposes no builder path that clears
// the damage-entry slice. aggroFlippedOff and controllerCharacterId derive
// purely from cur, so the captured values reflect the final successful
// invocation under optimistic-lock retry.
func (r *Registry) ClearDamageEntries(t tenant.Model, uniqueId uint32) (ClearSummary, error) {
	ctx := context.Background()

	var aggroFlippedOff bool
	var controllerCharacterId uint32
	sm, err := r.reg.Update(ctx, t, uniqueId, func(cur storedMonster) storedMonster {
		aggroFlippedOff = false

		cur.DamageEntries = make([]storedDamageEntry, 0, len(cur.DamageEntries))
		if cur.ControllerHasAggro {
			cur.ControllerHasAggro = false
			aggroFlippedOff = true
		}

		controllerCharacterId = cur.ControlCharacterId
		return cur
	})
	if errors.Is(err, atlasredis.ErrNotFound) {
		return ClearSummary{}, errMonsterNotFound
	}
	if err != nil {
		return ClearSummary{}, err
	}
	_, m, err := fromStored(sm)
	if err != nil {
		return ClearSummary{}, err
	}
	return ClearSummary{
		Monster:               m,
		ControllerCharacterId: controllerCharacterId,
		AggroFlippedOff:       aggroFlippedOff,
	}, nil
}
