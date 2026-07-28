package battleship

import (
	"atlas-channel/character"
	"atlas-channel/character/buff"
	"atlas-channel/data/skill/effect"
	"context"
	"strconv"
	"time"

	charskill "atlas-channel/character/skill"
	dataskill "atlas-channel/data/skill"

	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	redis "github.com/Chronicle20/atlas/libs/atlas-redis"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const registryNamespace = "battleship-hp"

// fallbackStateTTL bounds orphaned ship-HP entries when the effect-derived
// duration is unavailable (FR-5.2 safety net). Matches the 5221006 WZ buff
// duration (time=2100000 ms = 35 min).
const fallbackStateTTL = 35 * time.Minute

// counterStore is the Redis seam; production is redis.TenantCounter
// (namespace battleship-hp), wired by InitRegistry. Tests inject a fake.
type counterStore interface {
	Set(ctx context.Context, t tenant.Model, key string, value int64, ttl time.Duration) error
	DecrByIfExists(ctx context.Context, t tenant.Model, key string, delta int64, ttl time.Duration) (int64, bool, error)
	InitIfMissingAndDecrBy(ctx context.Context, t tenant.Model, key string, initial int64, delta int64, ttl time.Duration) (int64, error)
	Remove(ctx context.Context, t tenant.Model, key string) error
}

var store counterStore

// InitRegistry wires the production Redis-backed ship-HP store. Call once
// from main() after redis.Connect.
func InitRegistry(client *goredis.Client) {
	store = redis.NewTenantCounter(client, registryNamespace)
}

// ShipHP is the ship's full HP pool. It is VERSION-GATED (R-4): the client
// renders the gauge as remaining ÷ its own max, computed by
// get_max_durability_of_vehicle, and that function changed at v87. Using one
// formula everywhere would desync the bar on gms_87/92/95 and jms_185.
//
//	major <  87 (gms_61…gms_84):  200 × (charLevel + 2×SLV − 120)
//	major >= 87 (gms_87…jms_185): 300 × charLevel + 500 × (SLV − 72)
//
// The pre-87 arm is expressed as 400×SLV + max(charLevel−120,0)×200, which is
// algebraically identical to the client's form for charLevel >= 120 — the only
// reachable range, since Battleship is a 4th-job skill — and clamps instead of
// going negative below it. The v87+ arm is floored at 0 for the same reason.
func ShipHP(t tenant.Model, skillLevel byte, charLevel byte) int32 {
	if isPostBigBangDurability(t) {
		hp := 300*int32(charLevel) + 500*(int32(skillLevel)-72)
		if hp < 0 {
			return 0
		}
		return hp
	}
	hp := 400 * int32(skillLevel)
	if charLevel > 120 {
		hp += (int32(charLevel) - 120) * 200
	}
	return hp
}

// isPostBigBangDurability reports whether the tenant's client uses the newer
// get_max_durability_of_vehicle formula. Follows the established version-gate
// idiom (libs/atlas-packet/field/clientbound/set_field.go:48).
func isPostBigBangDurability(t tenant.Model) bool {
	return (t.IsRegion("GMS") && t.MajorAtLeast(87)) || t.IsRegion("JMS")
}

// Collaborator seams (function vars per the skill/handler/common.go
// precedent) so Drain's break flow is unit-testable offline.
var cancelBuffFunc = func(l logrus.FieldLogger, ctx context.Context, f field.Model, characterId uint32) error {
	return buff.NewProcessor(l, ctx).Cancel(f, characterId, int32(skill2.CorsairBattleshipId))
}

var applyCooldownFunc = func(l logrus.FieldLogger, ctx context.Context, f field.Model, cooldown uint32, characterId uint32) error {
	return charskill.NewProcessor(l, ctx).ApplyCooldown(f, skill2.CorsairBattleshipId, cooldown)(characterId)
}

var effectFunc = func(l logrus.FieldLogger, ctx context.Context, level byte) (effect.Model, error) {
	return dataskill.NewProcessor(l, ctx).GetEffect(uint32(skill2.CorsairBattleshipId), level)
}

var characterLevelFunc = func(l logrus.FieldLogger, ctx context.Context, characterId uint32) (byte, error) {
	c, err := character.NewProcessor(l, ctx).GetById()(characterId)
	if err != nil {
		return 0, err
	}
	return c.Level(), nil
}

type DrainStatus int

const (
	// DrainNotRiding: character has no active battleship ride on this pod.
	DrainNotRiding DrainStatus = iota
	// DrainSkipped: zero/negative damage, Redis unavailable (degraded mode),
	// or a concurrent drain that arrived after another caller's break.
	DrainSkipped
	// DrainDrained: HP decremented, ship intact; RemainingHP is valid.
	DrainDrained
	// DrainBroke: this drain crossed zero; dismount + cooldown were emitted.
	DrainBroke
)

type DrainResult struct {
	Status      DrainStatus
	RemainingHP int32
}

// Processor owns all battleship ride state (mirror + Redis pool). Handlers
// never touch either directly, so a future move to a service-owned store
// only changes this package (PRD NFR "state home caveat").
type Processor interface {
	// InitShipHP seeds a fresh full pool (always full — never carried over).
	InitShipHP(characterId uint32, skillLevel byte, charLevel byte, ttl time.Duration) error
	// StartRide begins a ride: records the mirror entry that IsRiding and the
	// attack/damage hot paths read (mirror write, no I/O). Called by the buff
	// consumer's APPLIED hook when a MONSTER_RIDING change is observed.
	StartRide(characterId uint32, s RideState)
	// IsRiding reports the active ride and its skill level (mirror read, no I/O).
	IsRiding(characterId uint32) (byte, bool)
	// Drain applies damage to the ship pool (FR-3/FR-4). Exactly one caller
	// per depletion observes DrainBroke.
	Drain(f field.Model, characterId uint32, damage int32) DrainResult
	// Clear removes mirror + Redis state; idempotent.
	Clear(characterId uint32)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	t   tenant.Model
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx, t: tenant.MustFromContext(ctx)}
}

func shipKey(characterId uint32) string {
	return strconv.FormatUint(uint64(characterId), 10)
}

func (p *ProcessorImpl) InitShipHP(characterId uint32, skillLevel byte, charLevel byte, ttl time.Duration) error {
	if store == nil {
		p.l.Warnf("Battleship store not initialized; ship HP for character [%d] will lazily re-initialize.", characterId)
		return nil
	}
	if ttl <= 0 {
		ttl = fallbackStateTTL
	}
	return store.Set(p.ctx, p.t, shipKey(characterId), int64(ShipHP(p.t, skillLevel, charLevel)), ttl)
}

func (p *ProcessorImpl) StartRide(characterId uint32, s RideState) {
	GetRideMirror().Put(p.t, characterId, s)
}

func (p *ProcessorImpl) IsRiding(characterId uint32) (byte, bool) {
	rs, ok := GetRideMirror().Get(p.t, characterId)
	return rs.SkillLevel, ok
}

func (p *ProcessorImpl) Drain(f field.Model, characterId uint32, damage int32) DrainResult {
	if damage <= 0 {
		return DrainResult{Status: DrainSkipped}
	}
	rs, riding := GetRideMirror().Get(p.t, characterId)
	if !riding {
		return DrainResult{Status: DrainNotRiding}
	}
	if store == nil {
		p.l.Warnf("Battleship store not initialized; drain skipped for character [%d].", characterId)
		return DrainResult{Status: DrainSkipped}
	}
	ttl := rs.StateTTL
	if ttl <= 0 {
		ttl = fallbackStateTTL
	}

	newHp, existed, err := store.DecrByIfExists(p.ctx, p.t, shipKey(characterId), int64(damage), ttl)
	if err != nil {
		// Degraded mode: never fail damage processing over Redis (NFR).
		p.l.WithError(err).Warnf("Battleship drain skipped for character [%d]; Redis unavailable.", characterId)
		return DrainResult{Status: DrainSkipped}
	}
	if !existed {
		// FR-3.3 lazy re-init: state lost (Redis restart / TTL expiry) is
		// never an error and never a stuck ship — re-derive full HP. The
		// character-level REST fetch happens ONLY on this miss path, never
		// on the steady-state DecrByIfExists hot path above.
		//
		// The seed-and-decrement must be a single atomic Redis operation
		// (InitIfMissingAndDecrBy), not a local "compute full-damage, then
		// Set" pair: two concurrent misses would otherwise each compute
		// against the same full baseline and race a plain Set, either
		// losing one caller's decrement (last write wins) or letting both
		// independently observe the same zero crossing (double break).
		// InitIfMissingAndDecrBy seeds the counter at most once — whichever
		// concurrent caller's script runs first — and every other racing
		// caller decrements the value that caller already seeded.
		charLevel, lerr := characterLevelFunc(p.l, p.ctx, characterId)
		if lerr != nil {
			p.l.WithError(lerr).Warnf("Battleship lazy re-init failed for character [%d]; drain skipped.", characterId)
			return DrainResult{Status: DrainSkipped}
		}
		full := int64(ShipHP(p.t, rs.SkillLevel, charLevel))
		v, serr := store.InitIfMissingAndDecrBy(p.ctx, p.t, shipKey(characterId), full, int64(damage), ttl)
		if serr != nil {
			p.l.WithError(serr).Warnf("Battleship lazy re-init failed for character [%d]; drain skipped.", characterId)
			return DrainResult{Status: DrainSkipped}
		}
		newHp = v
		p.l.Debugf("Battleship ship HP lazily re-initialized for character [%d]: full [%d], damage [%d].", characterId, full, damage)
	}

	if newHp > 0 {
		p.l.Debugf("Battleship drained for character [%d]: damage [%d], remaining [%d].", characterId, damage, newHp)
		return DrainResult{Status: DrainDrained, RemainingHP: int32(newHp)}
	}
	if newHp+int64(damage) > 0 {
		// Exactly-once crossing: only the decrement that moved the value
		// from positive to <= 0 satisfies this predicate (FR-4 / NFR).
		p.breakShip(f, characterId, rs.SkillLevel)
		return DrainResult{Status: DrainBroke}
	}
	// Concurrent drain that landed after another caller's crossing.
	return DrainResult{Status: DrainSkipped}
}

// breakShip performs the FR-4 break: clear state, dismount (foreign cancel
// broadcast comes from the existing atlas-buffs → buff consumer path), and
// apply the 5221006 cooldown with the effect's cooltime via the existing
// atlas-skills SET_COOLDOWN command. cancelBuffFunc and applyCooldownFunc
// each emit unconditionally — a second call is NOT a no-op, it is a second
// dismount broadcast and a second SET_COOLDOWN command. breakShip is
// therefore safe to call only because Drain's atomic Redis crossing
// predicate (both on the steady-state DecrByIfExists path and the
// InitIfMissingAndDecrBy lazy re-init path) guarantees at most one caller
// per depletion ever reaches this function.
func (p *ProcessorImpl) breakShip(f field.Model, characterId uint32, skillLevel byte) {
	p.Clear(characterId)
	if err := cancelBuffFunc(p.l, p.ctx, f, characterId); err != nil {
		p.l.WithError(err).Errorf("Battleship break: unable to cancel mount buff for character [%d].", characterId)
	}
	var cooldown uint32
	if e, err := effectFunc(p.l, p.ctx, skillLevel); err == nil {
		cooldown = e.Cooldown()
	} else {
		p.l.WithError(err).Errorf("Battleship break: unable to load effect level [%d] for character [%d]; cooldown not applied.", skillLevel, characterId)
	}
	if cooldown > 0 {
		if err := applyCooldownFunc(p.l, p.ctx, f, cooldown, characterId); err != nil {
			p.l.WithError(err).Errorf("Battleship break: unable to apply cooldown for character [%d].", characterId)
		}
	}
	p.l.Debugf("Battleship broke for character [%d]: dismounted, cooldown [%d]s.", characterId, cooldown)
}

func (p *ProcessorImpl) Clear(characterId uint32) {
	GetRideMirror().Remove(p.t, characterId)
	if store == nil {
		return
	}
	if err := store.Remove(p.ctx, p.t, shipKey(characterId)); err != nil {
		p.l.WithError(err).Warnf("Battleship state remove failed for character [%d]; TTL will expire it.", characterId)
	}
}
