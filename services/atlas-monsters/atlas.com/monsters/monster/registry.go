package monster

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

const maxRetries = 10

// storedMonster is the JSON-serializable representation stored in Redis.
type storedMonster struct {
	UniqueId             uint32           `json:"uniqueId"`
	TenantId             string           `json:"tenantId"`
	TenantRegion         string           `json:"tenantRegion"`
	TenantMajorVersion   uint16           `json:"tenantMajorVersion"`
	TenantMinorVersion   uint16           `json:"tenantMinorVersion"`
	WorldId              byte             `json:"worldId"`
	ChannelId            byte             `json:"channelId"`
	MapId                uint32           `json:"mapId"`
	Instance             string           `json:"instance"`
	MaxHp                uint32           `json:"maxHp"`
	Hp                   uint32           `json:"hp"`
	MaxMp                uint32           `json:"maxMp"`
	Mp                   uint32           `json:"mp"`
	MonsterId            uint32           `json:"monsterId"`
	ControlCharacterId   uint32           `json:"controlCharacterId"`
	ControllerHasAggro   bool             `json:"controllerHasAggro"`
	X                    int16            `json:"x"`
	Y                    int16            `json:"y"`
	Fh                   int16            `json:"fh"`
	Stance               byte             `json:"stance"`
	Team                 int8             `json:"team"`
	DamageEntries        damageEntryList  `json:"damageEntries"`
	StatusEffects        statusEffectList `json:"statusEffects"`
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
		})
	}
	return storedMonster{
		UniqueId:           m.uniqueId,
		TenantId:           t.Id().String(),
		TenantRegion:       t.Region(),
		TenantMajorVersion: t.MajorVersion(),
		TenantMinorVersion: t.MinorVersion(),
		WorldId:            byte(m.worldId),
		ChannelId:          byte(m.channelId),
		MapId:              uint32(m.mapId),
		Instance:           m.instance.String(),
		MaxHp:              m.maxHp,
		Hp:                 m.hp,
		MaxMp:              m.maxMp,
		Mp:                 m.mp,
		MonsterId:          m.monsterId,
		ControlCharacterId: m.controlCharacterId,
		ControllerHasAggro: m.controllerHasAggro,
		X:                  m.x,
		Y:                  m.y,
		Fh:                 m.fh,
		Stance:             m.stance,
		Team:               m.team,
		DamageEntries:      des,
		StatusEffects:      ses,
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
	}, nil
}

type Registry struct {
	client *goredis.Client
}

var registry *Registry
var once sync.Once

func InitMonsterRegistry(rc *goredis.Client) {
	once.Do(func() {
		registry = &Registry{client: rc}
	})
}

func GetMonsterRegistry() *Registry {
	return registry
}

func monsterKey(t tenant.Model, uniqueId uint32) string {
	return fmt.Sprintf("atlas:monster:%s:%d", t.Id().String(), uniqueId)
}

func mapIndexKey(t tenant.Model, f field.Model) string {
	return fmt.Sprintf("atlas:monster-map:%s:%d:%d:%d:%s",
		t.Id().String(), f.WorldId(), f.ChannelId(), f.MapId(), f.Instance().String())
}

func mapIndexKeyFromModel(t tenant.Model, m Model) string {
	return fmt.Sprintf("atlas:monster-map:%s:%d:%d:%d:%s",
		t.Id().String(), m.worldId, m.channelId, m.mapId, m.instance.String())
}

func (r *Registry) storeMonster(ctx context.Context, t tenant.Model, m Model) error {
	sm := toStored(t, m)
	data, err := json.Marshal(sm)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, monsterKey(t, m.uniqueId), data, 0).Err()
}

func (r *Registry) loadMonster(ctx context.Context, t tenant.Model, uniqueId uint32) (Model, error) {
	data, err := r.client.Get(ctx, monsterKey(t, uniqueId)).Result()
	if err == goredis.Nil {
		return Model{}, errors.New("monster not found")
	}
	if err != nil {
		return Model{}, err
	}
	var sm storedMonster
	if err := json.Unmarshal([]byte(data), &sm); err != nil {
		return Model{}, err
	}
	_, m, err := fromStored(sm)
	return m, err
}

func (r *Registry) atomicUpdate(ctx context.Context, t tenant.Model, uniqueId uint32, fn func(m Model) Model) (Model, error) {
	key := monsterKey(t, uniqueId)
	var result Model

	for i := 0; i < maxRetries; i++ {
		err := r.client.Watch(ctx, func(tx *goredis.Tx) error {
			data, err := tx.Get(ctx, key).Result()
			if err == goredis.Nil {
				return errors.New("monster not found")
			}
			if err != nil {
				return err
			}
			var sm storedMonster
			if err := json.Unmarshal([]byte(data), &sm); err != nil {
				return err
			}
			_, m, err := fromStored(sm)
			if err != nil {
				return err
			}

			result = fn(m)

			updatedSm := toStored(t, result)
			updatedData, err := json.Marshal(updatedSm)
			if err != nil {
				return err
			}
			_, err = tx.TxPipelined(ctx, func(pipe goredis.Pipeliner) error {
				pipe.Set(ctx, key, updatedData, 0)
				return nil
			})
			return err
		}, key)

		if err == nil {
			return result, nil
		}
		if err == goredis.TxFailedErr {
			continue
		}
		return Model{}, err
	}
	return Model{}, errors.New("optimistic lock failed after max retries")
}

func (r *Registry) CreateMonster(ctx context.Context, t tenant.Model, f field.Model, monsterId uint32, x int16, y int16, fh int16, stance byte, team int8, hp uint32, mp uint32) Model {
	uniqueId := GetIdAllocator().Allocate(ctx, t)
	m := NewMonster(f, uniqueId, monsterId, x, y, fh, stance, team, hp, mp)

	sm := toStored(t, m)
	data, _ := json.Marshal(sm)

	pipe := r.client.Pipeline()
	pipe.Set(ctx, monsterKey(t, uniqueId), data, 0)
	pipe.SAdd(ctx, mapIndexKey(t, f), strconv.FormatUint(uint64(uniqueId), 10))
	pipe.Exec(ctx)

	return m
}

func (r *Registry) GetMonster(tenant tenant.Model, uniqueId uint32) (Model, error) {
	return r.loadMonster(context.Background(), tenant, uniqueId)
}

func (r *Registry) GetMonstersInMap(tenant tenant.Model, f field.Model) []Model {
	ctx := context.Background()
	members, err := r.client.SMembers(ctx, mapIndexKey(tenant, f)).Result()
	if err != nil || len(members) == 0 {
		return nil
	}

	pipe := r.client.Pipeline()
	cmds := make([]*goredis.StringCmd, len(members))
	for i, idStr := range members {
		uid, _ := strconv.ParseUint(idStr, 10, 32)
		cmds[i] = pipe.Get(ctx, monsterKey(tenant, uint32(uid)))
	}
	pipe.Exec(ctx)

	var result []Model
	for _, cmd := range cmds {
		data, err := cmd.Result()
		if err != nil {
			continue
		}
		var sm storedMonster
		if err := json.Unmarshal([]byte(data), &sm); err != nil {
			continue
		}
		_, m, err := fromStored(sm)
		if err != nil {
			continue
		}
		result = append(result, m)
	}
	return result
}

func (r *Registry) MoveMonster(tenant tenant.Model, uniqueId uint32, endX int16, endY int16, stance byte) Model {
	m, err := r.atomicUpdate(context.Background(), tenant, uniqueId, func(m Model) Model {
		return m.Move(endX, endY, stance)
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

func (r *Registry) ClearControl(tenant tenant.Model, uniqueId uint32) (Model, error) {
	return r.atomicUpdate(context.Background(), tenant, uniqueId, func(m Model) Model {
		return m.ClearControl()
	})
}

var applyDamageScript = goredis.NewScript(`
local key = KEYS[1]
local charId = tonumber(ARGV[1])
local damage = tonumber(ARGV[2])
local nowMs = tonumber(ARGV[3])
local j = redis.call('GET', key)
if not j then
    return redis.error_reply("monster not found")
end
local m = cjson.decode(j)
local hp = m.hp
local actual = hp - math.max(hp - damage, 0)
m.hp = hp - actual

local entries = m.damageEntries
if type(entries) ~= 'table' then
    entries = {}
end

local found = false
for _, e in ipairs(entries) do
    if e.characterId == charId then
        e.damage = e.damage + actual
        e.lastHitMs = nowMs
        found = true
        break
    end
end
if not found then
    table.insert(entries, {
        characterId = charId,
        damage = actual,
        lastHitMs = nowMs
    })
end
m.damageEntries = entries

local hadAggro = m.controllerHasAggro
local wasFirstHit = false
if m.controlCharacterId ~= 0 and not hadAggro then
    m.controllerHasAggro = true
    wasFirstHit = true
end

redis.call('SET', key, cjson.encode(m))
return cjson.encode({wasFirstHit = wasFirstHit, monster = m})
`)

func (r *Registry) ApplyDamage(t tenant.Model, characterId uint32, damage uint32, uniqueId uint32, nowMs int64) (DamageSummary, error) {
	ctx := context.Background()
	key := monsterKey(t, uniqueId)

	result, err := applyDamageScript.Run(ctx, r.client, []string{key},
		strconv.FormatUint(uint64(characterId), 10),
		strconv.FormatUint(uint64(damage), 10),
		strconv.FormatInt(nowMs, 10),
	).Result()
	if err != nil {
		return DamageSummary{}, errors.New("monster not found")
	}

	resultStr, ok := result.(string)
	if !ok {
		return DamageSummary{}, errors.New("unexpected response type")
	}

	var env struct {
		WasFirstHit bool          `json:"wasFirstHit"`
		Monster     storedMonster `json:"monster"`
	}
	if err := json.Unmarshal([]byte(resultStr), &env); err != nil {
		return DamageSummary{}, err
	}
	_, m, err := fromStored(env.Monster)
	if err != nil {
		return DamageSummary{}, err
	}

	return DamageSummary{
		CharacterId:   characterId,
		Monster:       m,
		VisibleDamage: damage,
		Killed:        m.Hp() == 0,
		WasFirstHit:   env.WasFirstHit,
	}, nil
}

func (r *Registry) RemoveMonster(ctx context.Context, t tenant.Model, uniqueId uint32) (Model, error) {
	key := monsterKey(t, uniqueId)
	data, err := r.client.Get(ctx, key).Result()
	if err == goredis.Nil {
		return Model{}, errors.New("monster not found")
	}
	if err != nil {
		return Model{}, err
	}

	var sm storedMonster
	if err := json.Unmarshal([]byte(data), &sm); err != nil {
		return Model{}, err
	}
	_, m, err := fromStored(sm)
	if err != nil {
		return Model{}, err
	}

	idxKey := mapIndexKeyFromModel(t, m)
	pipe := r.client.Pipeline()
	pipe.Del(ctx, key)
	pipe.SRem(ctx, idxKey, strconv.FormatUint(uint64(uniqueId), 10))
	pipe.Exec(ctx)

	GetIdAllocator().Release(ctx, t, uniqueId)
	return m, nil
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

// SetNextSkillDecision atomically replaces the monster's in-memory picker
// decision. The decision is dropped on Redis round-trip (storedMonster does
// not carry it); on rehydration the picker re-runs and emits a fresh
// decision.
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

	var cursor uint64
	for {
		keys, nextCursor, err := r.client.Scan(ctx, cursor, "atlas:monster:*", 100).Result()
		if err != nil {
			break
		}
		if len(keys) > 0 {
			pipe := r.client.Pipeline()
			cmds := make([]*goredis.StringCmd, len(keys))
			for i, key := range keys {
				cmds[i] = pipe.Get(ctx, key)
			}
			pipe.Exec(ctx)

			for _, cmd := range cmds {
				data, err := cmd.Result()
				if err != nil {
					continue
				}
				var sm storedMonster
				if err := json.Unmarshal([]byte(data), &sm); err != nil {
					continue
				}
				t, m, err := fromStored(sm)
				if err != nil {
					continue
				}
				result[t] = append(result[t], m)
			}
		}
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	return result
}

func (r *Registry) Clear(ctx context.Context) {
	r.scanAndDelete(ctx, "atlas:monster:*")
	r.scanAndDelete(ctx, "atlas:monster-map:*")
}

func (r *Registry) scanAndDelete(ctx context.Context, pattern string) {
	var cursor uint64
	for {
		keys, nextCursor, err := r.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return
		}
		if len(keys) > 0 {
			r.client.Del(ctx, keys...)
		}
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
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

var decayDamageEntriesScript = goredis.NewScript(`
local key = KEYS[1]
local now = tonumber(ARGV[1])
local idleMs = tonumber(ARGV[2])
local mult = tonumber(ARGV[3])
local floorVal = tonumber(ARGV[4])
local j = redis.call('GET', key)
if not j then
    return redis.error_reply("monster not found")
end
local m = cjson.decode(j)

local entries = m.damageEntries
if type(entries) ~= 'table' then
    entries = {}
end

local kept = {}
for _, e in ipairs(entries) do
    -- Legacy entries written before this branch shipped have no lastHitMs.
    -- Treat them as having a lastHit of 0 (effectively always idle).
    local lastHit = e.lastHitMs or 0
    if (now - lastHit) > idleMs then
        e.damage = math.floor(e.damage * mult)
        e.lastHitMs = lastHit
    end
    if e.damage >= floorVal then
        table.insert(kept, e)
    end
end
m.damageEntries = kept

local hadAggro = m.controllerHasAggro
local aggroFlippedOff = false
if #kept == 0 and hadAggro then
    m.controllerHasAggro = false
    aggroFlippedOff = true
end

redis.call('SET', key, cjson.encode(m))
return cjson.encode({
    aggroFlippedOff = aggroFlippedOff,
    controllerCharacterId = m.controlCharacterId,
    monster = m,
})
`)

func (r *Registry) DecayDamageEntries(t tenant.Model, uniqueId uint32, nowMs int64) (DecaySummary, error) {
	ctx := context.Background()
	key := monsterKey(t, uniqueId)

	result, err := decayDamageEntriesScript.Run(ctx, r.client, []string{key},
		strconv.FormatInt(nowMs, 10),
		strconv.FormatInt(AggroIdleThresholdMs, 10),
		strconv.FormatFloat(AggroDecayMultiplier, 'f', -1, 64),
		strconv.FormatUint(uint64(AggroDecayFloor), 10),
	).Result()
	if err != nil {
		return DecaySummary{}, err
	}
	resultStr, ok := result.(string)
	if !ok {
		return DecaySummary{}, errors.New("unexpected response type")
	}

	var env struct {
		AggroFlippedOff       bool          `json:"aggroFlippedOff"`
		ControllerCharacterId uint32        `json:"controllerCharacterId"`
		Monster               storedMonster `json:"monster"`
	}
	if err := json.Unmarshal([]byte(resultStr), &env); err != nil {
		return DecaySummary{}, err
	}
	_, m, err := fromStored(env.Monster)
	if err != nil {
		return DecaySummary{}, err
	}
	return DecaySummary{
		Monster:               m,
		ControllerCharacterId: env.ControllerCharacterId,
		AggroFlippedOff:       env.AggroFlippedOff,
	}, nil
}

