package handler

import (
	"atlas-channel/battleship"
	"atlas-channel/character"
	"atlas-channel/character/buff"
	"atlas-channel/character/skill"
	"atlas-channel/character/snapshot"
	"atlas-channel/data/skill/effect"
	"atlas-channel/data/skill/effect/statup"
	"atlas-channel/drop"
	"atlas-channel/effective_stats"
	"atlas-channel/monster"
	"atlas-channel/session"
	"atlas-channel/skill/handler"
	"atlas-channel/socket/writer"
	"context"
	"errors"
	"math"
	"math/rand"

	skill2 "atlas-channel/data/skill"

	_map "atlas-channel/map"

	"github.com/sirupsen/logrus"

	charconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/constants"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	monster2 "github.com/Chronicle20/atlas/libs/atlas-constants/monster"
	"github.com/Chronicle20/atlas/libs/atlas-constants/point"
	skill3 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	charpkt "github.com/Chronicle20/atlas/libs/atlas-packet/character/clientbound"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// computeReflect computes the damage that should be reflected back to the
// attacker for one attack damage entry. dx/dy bounds-check (attacker minus
// monster) is inclusive on every edge so a hit on the LtX/LtY/RbX/RbY edge
// still triggers reflect — matching classic v83 behaviour. The reflected
// total is the sum of all damage lines multiplied by Percent and clamped
// to MaxDamage. Kind matching is the caller's responsibility (see
// monster.StatusMirror.GetReflect).
func computeReflect(damages []int32, info monster.ReflectInfo, attackerX, attackerY, monsterX, monsterY int16) (reflected int32, withinRange bool) {
	dx := attackerX - monsterX
	dy := attackerY - monsterY
	if dx < info.LtX || dx > info.RbX || dy < info.LtY || dy > info.RbY {
		return 0, false
	}
	total := int32(0)
	for _, d := range damages {
		total += d
	}
	r := total * info.Percent / 100
	if r > info.MaxDamage {
		r = info.MaxDamage
	}
	return r, true
}

// snapshotVenomDamagePerTick computes the per-tick damage applied by a
// VENOM stack at apply time. Classic formula: round(coef * Luck *
// MagicAttack), where coef is drawn from [0.1, 0.2). The math is pulled
// out of the handler so it can be pinned by unit tests; the production
// site picks the coef via rand.Float64() and feeds the result here.
func snapshotVenomDamagePerTick(luck, magicAttack int, coef float64) int32 {
	return int32(math.Round(coef * float64(luck) * float64(magicAttack)))
}

// attackKindFromAttackType maps a packet AttackType to the reflect kind the
// monster's reflect would have to match for the attack to be reflected.
// Returns the empty string for attack types that cannot be reflected
// (e.g. ENERGY).
func attackKindFromAttackType(at packetmodel.AttackType) string {
	switch at {
	case packetmodel.AttackTypeMelee, packetmodel.AttackTypeRanged:
		return monster2.ReflectKindPhysical
	case packetmodel.AttackTypeMagic:
		return monster2.ReflectKindMagical
	}
	return ""
}

// isDrainSkill reports whether id is one of the four attack-side
// drain-family skills that heal the attacker from damage dealt
// (Assassin Drain, Marauder/Thunder Breaker Energy Drain, Night
// Walker Vampire). Aran Combo Drain is buff-driven and excluded.
func isDrainSkill(id skill3.Id) bool {
	switch id {
	case skill3.AssassinDrainId,
		skill3.MarauderEnergyDrainId,
		skill3.ThunderBreakerStage3EnergyDrainId,
		skill3.NightWalkerStage2VampireId:
		return true
	}
	return false
}

// attackMonsterByIdFn is the REST-fallback seam for the per-swing monster
// resolvers below (precedent: monsterByIdFn in movement).
var attackMonsterByIdFn = func(l logrus.FieldLogger, ctx context.Context, uniqueId uint32) (monster.Model, error) {
	return monster.NewProcessor(l, ctx).GetById(uniqueId)
}

// buildMonsterResolver returns a per-swing memoized monster resolve backed
// by the live mirror with REST fallback + backfill (FR-4.2): one resolve
// per damaged monster serves both the reflect check and MP Eater. Not
// goroutine-safe — processAttack runs single-goroutine per packet.
func buildMonsterResolver(l logrus.FieldLogger, ctx context.Context, t tenant.Model) func(monsterId uint32) (monster.LiveEntry, error) {
	resolved := make(map[uint32]monster.LiveEntry)
	return func(monsterId uint32) (monster.LiveEntry, error) {
		if e, ok := resolved[monsterId]; ok {
			return e, nil
		}
		e, ok := monster.GetLiveMirror().Lookup(t, monsterId)
		if !ok {
			l.Debugf("Live mirror miss for monster [%d] on attack path; falling back to REST.", monsterId)
			mo, err := attackMonsterByIdFn(l, ctx, monsterId)
			if err != nil {
				monster.RecordMirrorFallback(t, false)
				return monster.LiveEntry{}, err
			}
			monster.RecordMirrorFallback(t, true)
			e = monster.LiveEntryFromModel(mo)
			monster.GetLiveMirror().Put(t, monsterId, e)
		}
		resolved[monsterId] = e
		return e, nil
	}
}

// buildMonsterModelResolver returns a per-swing memoized full-monster.Model
// resolver for the three passives that need Hp/MaxHp (drainTryHeal,
// pickPocketTryProc, mortalBlowDeps) — fields the live mirror does not
// carry (R7: mirroring monster HP is out of scope, it mutates on every
// hit and is a far larger change than this plan sized). Every resolve is
// therefore a REST call, but memoization still collapses the up-to-three
// passive reads for one damaged monster into at most one REST call per
// swing, counted via RecordMirrorFallback like the mirror-backed resolver
// above.
func buildMonsterModelResolver(l logrus.FieldLogger, ctx context.Context, t tenant.Model) func(monsterId uint32) (monster.Model, error) {
	resolved := make(map[uint32]monster.Model)
	return func(monsterId uint32) (monster.Model, error) {
		if mo, ok := resolved[monsterId]; ok {
			return mo, nil
		}
		mo, err := attackMonsterByIdFn(l, ctx, monsterId)
		if err != nil {
			monster.RecordMirrorFallback(t, false)
			return monster.Model{}, err
		}
		monster.RecordMirrorFallback(t, true)
		resolved[monsterId] = mo
		return mo, nil
	}
}

// damageInfoEntryDeps groups the per-attack closures and lookups that
// processDamageInfoEntry needs. Wrapping them keeps the helper signature
// readable and lets tests construct fakes with a single struct.
type damageInfoEntryDeps struct {
	getReflect        func(t tenant.Model, monsterId uint32, kind string) (monster.ReflectInfo, bool)
	getMonster        func(monsterId uint32) (monster.LiveEntry, error)
	applyDamage       func(f field.Model, monsterId, characterId uint32, damages []uint32, attackType byte) error
	emitReflectDamage func(f field.Model, uniqueId, templateId, characterId uint32, reflectDamage uint32, reflectType string) error
	applyStatus       func(f field.Model, monsterId, characterId, skillId, skillLevel uint32, statuses map[string]int32, duration uint32) error
	// loadEffectiveStats lazily fetches the caster's buff-inclusive
	// effective stats, at most once per attack. Consumed by the venom
	// DPT snapshot and by the drain-family heal cap.
	loadEffectiveStats func() effective_stats.RestModel
	// onDamageApplied is invoked once per non-reflected DamageInfo after
	// damage and status apply, with the entry's summed damage (clamped to
	// MaxUint32). Optional; nil-safe. Used by passives that fire per
	// damaged monster (MP Eater, drain-family heals, Pick Pocket).
	onDamageApplied func(di packetmodel.DamageInfo, totalDamage uint32)
}

// processDamageInfoEntry handles one DamageInfo from a magic/melee/ranged
// attack packet: damage application or reflect emission, then optional
// monster status apply. All side-effecting calls go through deps so tests
// can drive each branch without constructing a real monster.Processor or
// session.
func processDamageInfoEntry(
	l logrus.FieldLogger,
	di packetmodel.DamageInfo,
	ai packetmodel.AttackInfo,
	se effect.Model,
	skillLevel uint32,
	casterId uint32,
	casterX, casterY int16,
	f field.Model,
	t tenant.Model,
	attackKind string,
	deps damageInfoEntryDeps,
) {
	damages := di.Damages()

	if len(damages) == 0 {
		if len(se.MonsterStatus()) == 0 {
			return
		}
		ms := make(map[string]int32)
		for k, v := range se.MonsterStatus() {
			ms[k] = int32(v)
		}
		if _, isVenom := ms["VENOM"]; isVenom {
			stats := deps.loadEffectiveStats()
			coef := 0.1 + rand.Float64()*0.1
			ms["VENOM"] = snapshotVenomDamagePerTick(int(stats.Luck), int(stats.MagicAttack), coef)
		}
		_ = deps.applyStatus(f, di.MonsterId(), casterId, uint32(ai.SkillId()), skillLevel, ms, uint32(se.Duration()))
		return
	}

	reflected := false
	if attackKind != "" {
		if info, ok := deps.getReflect(t, di.MonsterId(), attackKind); ok {
			mon, mErr := deps.getMonster(di.MonsterId())
			if mErr == nil {
				entry := make([]int32, 0, len(damages))
				for _, d := range damages {
					entry = append(entry, int32(d))
				}
				r, within := computeReflect(entry, info, casterX, casterY, mon.X, mon.Y)
				if within {
					l.Debugf("reflect: char [%d] hit monster [%d] for %d reflected damage.", casterId, di.MonsterId(), r)
					if eErr := deps.emitReflectDamage(f, di.MonsterId(), mon.MonsterId, casterId, uint32(r), info.Kind); eErr != nil {
						l.WithError(eErr).Errorf("Unable to emit DAMAGE_REFLECTED for monster [%d] / character [%d].", di.MonsterId(), casterId)
					}
					reflected = true
				}
			}
		}
	}

	if reflected {
		// On reflect: monster takes no damage AND no monster status is applied
		// for this entry (FREEZE/STUN/etc. would let the player slip through
		// the reflect's intent).
		return
	}

	if err := deps.applyDamage(f, di.MonsterId(), casterId, damages, byte(ai.AttackType())); err != nil {
		l.WithError(err).Errorf("Unable to apply damage to monster [%d] from character [%d].", di.MonsterId(), casterId)
	}

	// Apply monster status effects from skill (e.g., freeze, poison, stun).
	if len(se.MonsterStatus()) > 0 {
		ms := make(map[string]int32)
		for k, v := range se.MonsterStatus() {
			ms[k] = int32(v)
		}
		if _, isVenom := ms["VENOM"]; isVenom {
			stats := deps.loadEffectiveStats()
			coef := 0.1 + rand.Float64()*0.1
			ms["VENOM"] = snapshotVenomDamagePerTick(int(stats.Luck), int(stats.MagicAttack), coef)
		}
		_ = deps.applyStatus(f, di.MonsterId(), casterId, uint32(ai.SkillId()), skillLevel, ms, uint32(se.Duration()))
	}

	if deps.onDamageApplied != nil {
		var total uint64
		for _, d := range damages {
			total += uint64(d)
		}
		if total > math.MaxUint32 {
			total = math.MaxUint32
		}
		deps.onDamageApplied(di, uint32(total))
	}
}

// shouldProc returns true when a prop-gated passive (MP Eater, Pick
// Pocket) should fire given the effect's prop and a single uniform roll
// in [0,1). Mirrors Cosmic's `prop == 1.0 || rand() < prop`. Defensive
// against negative props.
func shouldProc(prop float64, roll float64) bool {
	if prop <= 0 {
		return false
	}
	return prop >= 1.0 || roll < prop
}

// pickPocketWhitelist is the fixed set of skills that can proc Pick
// Pocket (Cosmic AbstractDealDamageHandler parity). Basic attack
// (skillId == 0) is handled in pickPocketWhitelisted.
var pickPocketWhitelist = map[uint32]struct{}{
	uint32(skill3.RogueDoubleStabId):          {},
	uint32(skill3.BanditSavageBlowId):         {},
	uint32(skill3.ChiefBanditAssaulterId):     {},
	uint32(skill3.ChiefBanditBandOfThievesId): {},
	uint32(skill3.ShadowerAssassinateId):      {},
	uint32(skill3.ShadowerTauntId):            {},
	uint32(skill3.ShadowerBoomerangStepId):    {},
}

// pickPocketWhitelisted reports whether skillId can proc Pick Pocket.
func pickPocketWhitelisted(skillId uint32) bool {
	if skillId == 0 {
		return true
	}
	_, ok := pickPocketWhitelist[skillId]
	return ok
}

// pickPocketMesoAmount computes the meso payout for one damage line:
// min(max(damage/20000 * maxmeso, 1), maxmeso), float math then
// truncation, matching Cosmic. Returns 0 when maxmeso <= 0. A 0-damage
// line still yields 1 on a successful roll.
func pickPocketMesoAmount(damage uint32, maxmeso int32) uint32 {
	if maxmeso <= 0 {
		return 0
	}
	v := math.Max(float64(damage)/20000.0*float64(maxmeso), 1)
	if v > float64(maxmeso) {
		return uint32(maxmeso)
	}
	return uint32(v)
}

// pickPocketState is the per-attack Pick Pocket context, resolved once
// before the DamageInfo loop (design §3.3): whitelist gate first (pure,
// no I/O), then at most one buff REST call and one effect lookup.
type pickPocketState struct {
	enabled bool
	maxmeso int32   // PICK_POCKET stat Amount() captured at buff time
	prop    float64 // effect prop at the buff's captured Level()
}

// pickPocketResolveState gates and resolves Pick Pocket for one attack.
// Any failure (buff REST error, effect lookup error) or non-positive
// maxmeso/prop yields a disabled state; errors are logged and swallowed,
// never propagated into the attack pipeline.
func pickPocketResolveState(
	l logrus.FieldLogger,
	getBuffs func(characterId uint32) ([]buff.Model, error),
	getEffect func(uniqueId uint32, level byte) (effect.Model, error),
	skillId uint32,
	characterId uint32,
) pickPocketState {
	if !pickPocketWhitelisted(skillId) {
		return pickPocketState{}
	}

	buffs, err := getBuffs(characterId)
	if err != nil {
		l.WithError(err).Errorf("Pick Pocket: buff lookup failed for character [%d].", characterId)
		return pickPocketState{}
	}

	var maxmeso int32
	var level byte
	found := false
	for _, b := range buffs {
		if b.Expired() {
			continue
		}
		for _, ch := range b.Changes() {
			if ch.Type() == string(charconst.TemporaryStatTypePickPocket) {
				maxmeso = ch.Amount()
				level = b.Level()
				found = true
				break
			}
		}
		if found {
			break
		}
	}
	if !found {
		return pickPocketState{}
	}
	if maxmeso <= 0 {
		l.Debugf("Pick Pocket: non-positive maxmeso [%d] for character [%d]; proc disabled.", maxmeso, characterId)
		return pickPocketState{}
	}

	se, err := getEffect(uint32(skill3.ChiefBanditPickpocketId), level)
	if err != nil {
		l.WithError(err).Errorf("Pick Pocket: effect lookup failed at level [%d] for character [%d].", level, characterId)
		return pickPocketState{}
	}
	if se.Prop() <= 0 {
		l.Debugf("Pick Pocket: non-positive prop at level [%d] for character [%d]; proc disabled.", level, characterId)
		return pickPocketState{}
	}

	return pickPocketState{enabled: true, maxmeso: maxmeso, prop: se.Prop()}
}

// pickPocketTryProc rolls each damage line of one non-reflected
// DamageInfo and emits one meso SPAWN per success. Monster snapshot
// fetch failure skips this monster's procs (Debugf); emit failures are
// logged (Errorf) and swallowed, continuing with the remaining lines.
func pickPocketTryProc(
	l logrus.FieldLogger,
	getMonster func(monsterId uint32) (monster.Model, error),
	spawnMeso func(f field.Model, mesos uint32, x int16, y int16, ownerId uint32, dropperId uint32, dropperX int16, dropperY int16) error,
	state pickPocketState,
	di packetmodel.DamageInfo,
	f field.Model,
	characterId uint32,
) {
	if !state.enabled {
		return
	}

	mon, err := getMonster(di.MonsterId())
	if err != nil {
		l.WithError(err).Debugf("Pick Pocket: monster [%d] snapshot fetch failed; skipping its procs.", di.MonsterId())
		return
	}

	for _, d := range di.Damages() {
		if !shouldProc(state.prop, rand.Float64()) {
			continue
		}
		mesos := pickPocketMesoAmount(d, state.maxmeso)
		l.Debugf("Pick Pocket proc: character=[%d] monster=[%d] mesos=[%d].", characterId, di.MonsterId(), mesos)
		x := mon.X() + int16(rand.Intn(100)-50)
		if sErr := spawnMeso(f, mesos, x, mon.Y(), characterId, di.MonsterId(), mon.X(), mon.Y()); sErr != nil {
			l.WithError(sErr).Errorf("Pick Pocket: SPAWN emit failed for monster [%d] character [%d].", di.MonsterId(), characterId)
		}
	}
}

// mpEaterAbsorbAmount computes the requested drain from monster MaxMp
// and the skill's X (absorb percent). Returns 0 when MaxMp is 0 or X is
// non-positive. atlas-monsters re-clamps to the monster's current MP.
func mpEaterAbsorbAmount(maxMp uint32, x int16) uint32 {
	if maxMp == 0 || x <= 0 {
		return 0
	}
	return uint32(uint64(maxMp) * uint64(x) / 100)
}

// mortalBlowEligible reports whether a monster's (pre-attack snapshot) HP
// is at or below the Mortal Blow threshold: hp ≤ maxHp × x / 100, with
// integer truncating division (Cosmic parity). Widens through uint64 so
// maxHp near MaxUint32 cannot overflow. Defensive: false when x ≤ 0 or
// maxHp == 0 (malformed/absent tenant data means the passive is inert).
func mortalBlowEligible(hp uint32, maxHp uint32, x int16) bool {
	if x <= 0 || maxHp == 0 {
		return false
	}
	return uint64(hp) <= uint64(maxHp)*uint64(x)/100
}

// mortalBlowKillRoll reports whether the instant kill procs for a uniform
// roll in [1,100]: roll ≤ y. Defensive: false when y ≤ 0.
func mortalBlowKillRoll(roll int, y int16) bool {
	if y <= 0 {
		return false
	}
	return roll <= int(y)
}

// isMortalBlowAttack reports whether an attack is a client-side Mortal Blow
// proc: a ranged attack tagged with the Ranger (3110001) or Sniper
// (3210001) passive's skill id. The v83 client only tags an attack with
// these ids on a successful point-blank normal-attack conversion, and the
// upstream ownership guard in processAttack destroys the session for
// unowned skill ids, so this gate is sufficient (no job-range check —
// PRD FR-1).
func isMortalBlowAttack(at packetmodel.AttackType, skillId uint32) bool {
	return at == packetmodel.AttackTypeRanged &&
		(skill3.Id(skillId) == skill3.RangerMortalBlowId || skill3.Id(skillId) == skill3.SniperMortalBlowId)
}

// mortalBlowDeps groups the seams mortalBlowTryProc needs so tests can
// drive every branch (snapshot miss, threshold, roll, emit failure)
// without a real monster.Processor or Kafka — same pattern as
// damageInfoEntryDeps. Production wiring: mp.GetById, mp.Kill, and
// rand.Intn(100)+1.
type mortalBlowDeps struct {
	getMonster func(monsterId uint32) (monster.Model, error)
	emitKill   func(f field.Model, monsterId uint32, characterId uint32) error
	// roll returns a uniform integer in [1,100].
	roll func() int
}

// mortalBlowTryProc evaluates and (on success) emits the Mortal Blow
// instant kill for one damaged monster. Called once per damaged monster
// after damage and status apply, only for ranged attacks tagged with the
// Mortal Blow skill ids (the attack's skill IS the passive, so se is
// already resolved at the character's owned level — no extra effect
// lookup). The threshold reads the channel's monster snapshot, which
// reflects pre-attack HP (damage propagates to atlas-monsters
// asynchronously); that is the specified Cosmic-parity timing (FR-2).
// Boss exclusion is enforced authoritatively by atlas-monsters — the
// snapshot carries no boss flag. Errors are logged at Debugf/Errorf and
// swallowed — never abort the surrounding attack pipeline (FR-5).
//
// Mortal Blow is version-agnostic: it applies to any tenant whose client
// sends skill id 3110001/3210001, including the pre-BB legacy versions
// v48/v61/v72/v79; it is inert on v95/JMS (post-BB redesign) and on any
// version whose client never sends the id. No per-version code exists or
// is needed.
func mortalBlowTryProc(
	l logrus.FieldLogger,
	deps mortalBlowDeps,
	se effect.Model,
	monsterId uint32,
	f field.Model,
	characterId uint32,
	skillId uint32,
) {
	x, y := se.X(), se.Y()
	if x <= 0 || y <= 0 {
		return
	}

	mon, err := deps.getMonster(monsterId)
	if err != nil {
		l.WithError(err).Debugf("Mortal Blow: monster [%d] snapshot fetch failed.", monsterId)
		return
	}

	if !mortalBlowEligible(mon.Hp(), mon.MaxHp(), x) {
		return
	}

	roll := deps.roll()
	l.Debugf("Mortal Blow threshold pass: caster=[%d] skill=[%d] monster=[%d] (hp=%d maxHp=%d x=%d) roll=[%d] y=[%d].",
		characterId, skillId, monsterId, mon.Hp(), mon.MaxHp(), x, roll, y)
	if !mortalBlowKillRoll(roll, y) {
		return
	}

	l.Debugf("Mortal Blow proc: caster=[%d] skill=[%d] monster=[%d] roll=[%d].", characterId, skillId, monsterId, roll)
	if err := deps.emitKill(f, monsterId, characterId); err != nil {
		l.WithError(err).Errorf("Mortal Blow: KILL emit failed for monster [%d] caster [%d].", monsterId, characterId)
	}
}

// sacrificeHpCost computes the self-HP cost of Dragon Knight Sacrifice:
// firstLine × x / 100 (truncating integer division, Cosmic parity),
// clamped so the caster is left with at least 1 HP. Returns 0 when the
// first line is 0 (miss), x is non-positive, or currentHp <= 1. The
// MaxInt16 cap is a defensive narrowing guard: on supported versions max
// HP <= 30000 so the survival clamp already bounds the result, but Hp()
// is uint16 and the call site negates into int16 — the cap makes that
// narrowing safe by construction instead of by data assumption.
func sacrificeHpCost(firstLine uint32, x int16, currentHp uint16) uint16 {
	if firstLine == 0 || x <= 0 || currentHp <= 1 {
		return 0
	}
	cost := uint64(firstLine) * uint64(x) / 100
	if cost >= uint64(currentHp) {
		cost = uint64(currentHp) - 1
	}
	if cost > math.MaxInt16 {
		cost = math.MaxInt16
	}
	return uint16(cost)
}

// sacrificeFirstDamageLine returns the first damage line of the first
// damage entry, or 0 when the attack has no entries or the first entry
// has no lines. Sacrifice's self-HP cost basis is only ever this line —
// additional lines and targets are deliberately ignored (Cosmic
// damageLines().getFirst() parity; PRD FR-2).
func sacrificeFirstDamageLine(ai packetmodel.AttackInfo) uint32 {
	di := ai.DamageInfo()
	if len(di) == 0 || len(di[0].Damages()) == 0 {
		return 0
	}
	return di[0].Damages()[0]
}

// drainHealAmount computes the drain-family HP gain for one damaged
// monster: floor(totalDamage * x / 100), capped by the monster's max HP
// and by half the attacker's effective (buff-inclusive) max HP, then
// defensively clamped to int16 range for ChangeHP. Returns 0 for
// non-positive x, zero damage, or zero effectiveMaxHp (fail-safe when
// the effective-stats fetch failed).
func drainHealAmount(totalDamage uint32, x int16, monsterMaxHp uint32, effectiveMaxHp uint32) int16 {
	if totalDamage == 0 || x <= 0 || effectiveMaxHp == 0 {
		return 0
	}
	heal := uint64(totalDamage) * uint64(x) / 100
	if m := uint64(monsterMaxHp); heal > m {
		heal = m
	}
	if h := uint64(effectiveMaxHp) / 2; heal > h {
		heal = h
	}
	if heal > math.MaxInt16 {
		return math.MaxInt16
	}
	return int16(heal)
}

// mpEaterTryProc evaluates and (on success) emits MP Eater for one
// damaged monster. Called once per damaged monster after status apply.
// Errors are logged at Debugf/Errorf and swallowed — never abort the
// surrounding attack pipeline.
func mpEaterTryProc(
	l logrus.FieldLogger,
	ctx context.Context,
	mp monster.Processor,
	getMonster func(uint32) (monster.LiveEntry, error),
	c character.Model,
	monsterId uint32,
	f field.Model,
	characterId uint32,
) {
	eaterId, ok := job.MpEaterSkillId(c.JobId())
	if !ok {
		return
	}

	var ownedLevel byte
	for _, owned := range c.Skills() {
		if owned.Id() == eaterId {
			ownedLevel = owned.Level()
			break
		}
	}
	if ownedLevel == 0 {
		return
	}

	eaterEffect, err := skill2.NewProcessor(l, ctx).GetEffect(uint32(eaterId), ownedLevel)
	if err != nil {
		l.WithError(err).Errorf("MP Eater: skill effect lookup failed for skill [%d] level [%d].", eaterId, ownedLevel)
		return
	}
	if eaterEffect.Prop() <= 0 {
		return
	}

	mon, err := getMonster(monsterId)
	if err != nil {
		l.WithError(err).Debugf("MP Eater: monster [%d] snapshot fetch failed.", monsterId)
		return
	}
	if mon.MaxMp == 0 || mon.Mp == 0 {
		return
	}

	if !shouldProc(eaterEffect.Prop(), rand.Float64()) {
		return
	}

	amount := mpEaterAbsorbAmount(mon.MaxMp, eaterEffect.X())
	if amount == 0 {
		return
	}

	l.Debugf("MP Eater proc: caster=[%d] skill=[%d] monster=[%d] amount=[%d] (monster MaxMp=%d Mp=%d).",
		characterId, eaterId, monsterId, amount, mon.MaxMp, mon.Mp)

	if err := mp.DrainMp(f, monsterId, characterId, uint32(eaterId), amount); err != nil {
		l.WithError(err).Errorf("MP Eater: DRAIN_MP emit failed for monster [%d] caster [%d].", monsterId, characterId)
	}
}

// drainTryHeal computes and emits the drain-family heal for one damaged
// monster: floor(totalDamage * x / 100), capped by the monster's max HP
// and half the caster's effective max HP. Called once per damaged
// monster after damage apply. All collaborators are injected so flow
// tests can drive every branch; production passes mp.GetById and
// cp.ChangeHP. Errors are logged and swallowed — never abort the
// surrounding attack pipeline.
func drainTryHeal(
	l logrus.FieldLogger,
	getMonster func(monsterId uint32) (monster.Model, error),
	changeHP func(f field.Model, characterId uint32, amount int16) error,
	loadEffectiveStats func() effective_stats.RestModel,
	x int16,
	skillId uint32,
	monsterId uint32,
	totalDamage uint32,
	f field.Model,
	characterId uint32,
) {
	mon, err := getMonster(monsterId)
	if err != nil {
		l.WithError(err).Debugf("Drain heal: monster [%d] snapshot fetch failed; skipping heal for caster [%d].", monsterId, characterId)
		return
	}

	stats := loadEffectiveStats()
	heal := drainHealAmount(totalDamage, x, mon.MaxHp(), stats.MaxHp)
	if heal <= 0 {
		return
	}

	l.Debugf("Drain heal: caster=[%d] skill=[%d] monster=[%d] damage=[%d] x=[%d] heal=[%d].",
		characterId, skillId, monsterId, totalDamage, x, heal)

	if err := changeHP(f, characterId, heal); err != nil {
		l.WithError(err).Errorf("Drain heal: CHANGE_HP emit failed for character [%d] (skill [%d], monster [%d]).", characterId, skillId, monsterId)
	}
}

// beaconApplyDeps groups the emit closures beaconTryApply needs so tests can
// record ordering without Kafka. Mirrors the damageInfoEntryDeps pattern.
type beaconApplyDeps struct {
	monsterExists func(monsterId uint32) bool
	cancelByTypes func(types []string) error
	applyNoExpiry func(sourceId int32, level byte, mobId int32) error
}

// beaconTargetMonsterId picks the beacon lock target: the LAST damage entry
// whose monster id is nonzero and still exists in the field registry. WZ has
// no mobCount for 5211006/5220011 (single-target), so multiple entries only
// occur on malformed packets; last-valid-wins matches Cosmic's per-monster
// loop order (design.md §5.2).
func beaconTargetMonsterId(monsterIds []uint32, exists func(uint32) bool) (uint32, bool) {
	var target uint32
	found := false
	for _, id := range monsterIds {
		if id == 0 || !exists(id) {
			continue
		}
		target = id
		found = true
	}
	return target, found
}

// beaconTryApply handles Homing Beacon (5211006) / Bullseye (5220011): on a
// valid strike it emits CANCEL_BY_TYPES(HOMING_BEACON) then a no-expiry APPLY
// whose statup amount is the struck monster's object id. Both commands share
// the character-keyed buff command topic, so ordering is guaranteed and the
// old lock (either skill id) is always cleared first (FR-1.4). A whiff emits
// nothing (FR-1.5). Failures are logged and swallowed — the attack pipeline
// (damage, projectile, broadcast) is already complete and must not be
// affected (FR-1.6). skill.OutlawHomingBeaconId (5211006) and
// skill.CorsairBullseyeId (5220011) are NOT in the task-187
// divergences.csv (checked: no job/skill row for wireId 5211006 or
// 5220011), so a raw comparison here is not banned by
// tools/skill-job-id-guard.sh.
func beaconTryApply(l logrus.FieldLogger, ai packetmodel.AttackInfo, skillLevel byte, f field.Model, characterId uint32, deps beaconApplyDeps) {
	sid := skill3.Id(ai.SkillId())
	if sid != skill3.OutlawHomingBeaconId && sid != skill3.CorsairBullseyeId {
		return
	}

	ids := make([]uint32, 0, len(ai.DamageInfo()))
	for _, di := range ai.DamageInfo() {
		ids = append(ids, di.MonsterId())
	}
	mobId, ok := beaconTargetMonsterId(ids, deps.monsterExists)
	if !ok {
		return
	}

	if err := deps.cancelByTypes([]string{string(charconst.TemporaryStatTypeHomingBeacon)}); err != nil {
		l.WithError(err).Errorf("Beacon: unable to cancel prior HOMING_BEACON for character [%d]; skipping apply.", characterId)
		return
	}
	if err := deps.applyNoExpiry(int32(ai.SkillId()), skillLevel, int32(mobId)); err != nil {
		l.WithError(err).Errorf("Beacon: unable to apply HOMING_BEACON (mob [%d]) for character [%d].", mobId, characterId)
	}
}

// attackCastTryApply runs the per-skill attack-cast handler registered for
// castId, if any. It is the ATTACK-packet twin of the UseSkill dispatcher in
// skill/handler/common.go, for skills the client delivers on a
// melee/ranged/magic attack packet rather than USE_SKILL.
//
// Poison Mist (2111003) is the motivating case and the reason this exists:
// it carries `damage`/`attackCount`/`mobCount` in Skill.wz, so the v83 client
// sends it on opcode 0x2E (CharacterMagicAttackHandle) and never on USE_SKILL.
// Registered on the use-skill registry it silently never fired -- the mist was
// never created, so neither SPAWN_MIST nor the poison DoT ever happened.
//
// castId is the resolved version-blind Identity (the registry key); wireSkillId
// is the raw per-version id the packet carried, which handlers put back on the
// wire for the client to match against its own WZ.
//
// Failures are logged and swallowed, matching beaconTryApply and the projectile
// / meso-explosion emits: by the time this runs the damage is applied and the
// attack is broadcast, so nothing here may abort the pipeline.
func attackCastTryApply(
	l logrus.FieldLogger,
	ctx context.Context,
	wp writer.Producer,
	f field.Model,
	characterId uint32,
	castId skill3.Identity,
	wireSkillId skill3.Id,
	skillLevel byte,
	e effect.Model,
	castOrigin *point.Model,
) {
	h, ok := handler.LookupAttackCast(castId)
	if !ok {
		return
	}
	if err := h(l)(ctx)(wp, f, characterId, wireSkillId, skillLevel, e, castOrigin); err != nil {
		l.WithError(err).Errorf("Attack-cast handler for skill [%d] failed for character [%d].", wireSkillId, characterId)
	}
}

// attackCastOrigin is the world point an ATTACK-delivered cast should anchor at.
//
// An attack packet always carries a better answer than the server can look up,
// and the two cases differ:
//
//   - A thrown-grenade skill carries where the BOMB landed, a function of how
//     long the attack key was held. The client has already drawn the explosion
//     there, so anchoring anywhere else visibly disagrees with it (task-218
//     field report #4: Poison Bomb's mist sat at the caster's feet no matter how
//     far the bomb flew).
//   - Every other attack carries the caster's own position at the moment of the
//     attack. That beats reading it back from atlas-character, which is updated
//     asynchronously over Kafka (movement.ProcessorImpl.ForCharacter emits
//     COMMAND_CHARACTER_MOVEMENT from a detached goroutine), so a REST read
//     taken while handling this attack can observe a PRE-MOVE position and
//     anchor the effect where the caster used to be (field report #2).
//
// Never nil for an attack-delivered cast, so the caster REST lookup is reserved
// for the USE_SKILL-delivered mists, which have no packet-supplied position.
//
// The grenade case is gated on the skill rather than on a non-zero value:
// AttackInfo.Decode only fills those fields for the grenade arm, and (0,0) is a
// legal coordinate, so a value-based test would both miss a legitimate origin
// and invent one for every other attack.
func attackCastOrigin(ai packetmodel.AttackInfo) *point.Model {
	x, y := int16(ai.CharacterX()), int16(ai.CharacterY())
	if skill3.IsGrenadeSkill(skill3.Id(ai.SkillId())) {
		x, y = int16(ai.GrenadeX()), int16(ai.GrenadeY())
	}
	p := point.NewModel(point.X(x), point.Y(y))
	return &p
}

// resolveAttackSkill finds the owned skill backing an attack's wire skill id,
// reporting false when the character may not attack with it at all.
//
// The direct case is ownership. The indirect case is Aran's hidden combo
// variants (Full Swing / Over Swing at two and three swings): the client sends
// the variant's id once the combo count escalates the swing, but the variant is
// never in the skill book -- no SP is spent on it and it is excluded from SP
// reset. Its level lives on the parent, so the parent's Model is what backs the
// attack, and an unowned parent is still a rejection: the client can only
// produce the variant by escalating a swing it already has.
//
// The variant's own id stays on the wire for every downstream lookup that needs
// it (the effect fetch keys on ai.SkillId(), and WZ carries a per-level effect
// table for the variant at the same maxLevel as its parent).
func resolveAttackSkill(skills []skill.Model, wireId skill3.Id) (skill.Model, bool) {
	find := func(id skill3.Id) (skill.Model, bool) {
		for _, sk := range skills {
			if sk.Id() == id {
				return sk, true
			}
		}
		return skill.Model{}, false
	}

	if sk, ok := find(wireId); ok {
		return sk, true
	}
	if parentId, ok := skill3.AranHiddenComboParent(wireId); ok {
		return find(parentId)
	}
	return skill.Model{}, false
}

func processAttack(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(ai packetmodel.AttackInfo) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(ai packetmodel.AttackInfo) model.Operator[session.Model] {
		return func(wp writer.Producer) func(ai packetmodel.AttackInfo) model.Operator[session.Model] {
			return func(ai packetmodel.AttackInfo) model.Operator[session.Model] {
				return func(s session.Model) error {
					sp := snapshot.NewProcessor(l, ctx)
					c, err := sp.Get(s.CharacterId())
					if err != nil {
						return err
					}

					// Resolved once and reused for every version-sensitive
					// wire-id comparison below (task-187): the registered-check
					// gate that skips the generic HP/MP cost block MUST key on
					// the caster's version-blind skill Identity, not the raw
					// wire id -- a raw compare cannot distinguish a v0.48
					// SuperGM Hide cast (wire 5101004) from a v0.62+ Brawler
					// Corkscrew Blow cast (same wire).
					t := tenant.MustFromContext(ctx)
					set := constants.For(t.Region(), t.MajorVersion(), t.MinorVersion())
					attackId, attackIdOk := set.Skill.Resolve(skill3.Id(ai.SkillId()))

					var sk skill.Model
					var se effect.Model
					var explodedMesoDropIds []uint32

					// cp is only used for ChangeHP/ChangeMP command emission
					// below (cost gate, Sacrifice, Combo Drain) — every
					// character READ on this path now goes through sp (the
					// snapshot).
					cp := character.NewProcessor(l, ctx)

					if ai.SkillId() > 0 {
						// Process skill
						var owned bool
						sk, owned = resolveAttackSkill(c.Skills(), skill3.Id(ai.SkillId()))
						if !owned {
							l.Errorf("Character [%d] attempting to attack with skill [%d] which they do not own.", s.CharacterId(), ai.SkillId())
							return session.NewProcessor(l, ctx).Destroy(s)
						}

						// Battleship Cannon/Torpedo are usable only while
						// riding the battleship (FR-6.1). Soft rejection (no
						// costs, no damage, no broadcast): a briefly desynced
						// legitimate client — e.g. the cast→BUFF_APPLIED
						// mirror window — must not be disconnected. Routed
						// through battleship.Processor.IsRiding, which is
						// itself a pure mirror read: zero I/O in the attack
						// hot path (FR-6.2).
						if !battleshipAttackPermitted(l, ctx, s.CharacterId(), skill3.Id(ai.SkillId())) {
							l.WithFields(logrus.Fields{
								"character_id": s.CharacterId(),
								"skill_id":     ai.SkillId(),
							}).Debug("battleship_attack_rejected_not_riding")
							return nil
						}

						// Energy Blast requires a full energy bar (task-216
						// FR-6). Same soft-rejection posture as the battleship
						// gate — before any cost, damage, or broadcast, and
						// returning nil rather than destroying the session.
						// The bar is NOT consumed by a successful cast; only
						// the charged window's own timer resets it.
						if permitted, bar := energyBlastPermitted(t, s.CharacterId(), attackId, attackIdOk); !permitted {
							l.WithFields(logrus.Fields{
								"character_id": s.CharacterId(),
								"skill_id":     ai.SkillId(),
								"energy_bar":   bar,
							}).Debug("energy_blast_rejected_not_charged")
							energyReannounceAuthoritative(l, ctx, wp, s)
							return nil
						}

						se, err = skill2.NewProcessor(l, ctx).GetEffect(ai.SkillId(), sk.Level())
						if err != nil {
							return err
						}

						// Meso Explosion (task-150): validate the exploded-drop list
						// against the field's drops BEFORE any side effect (FR-6 —
						// rejection must skip cost, damage, broadcast, and destruction).
						// One field-scoped fetch; the map keys structurally enforce the
						// same-field/instance check. Routed through the resolved
						// Identity (task-187) rather than a raw wire compare.
						if attackIdOk && skill3.IsIdentity(attackId, skill3.ChiefBanditMesoExplosion) {
							ds, dErr := drop.NewProcessor(l, ctx).InMapModelProvider(s.Field())()
							if dErr != nil {
								return dErr
							}
							fieldDrops := make(map[uint32]drop.Model, len(ds))
							for _, d := range ds {
								fieldDrops[d.Id()] = d
							}
							if badId, ok := validateMesoExplosion(ai.ExplodedMesoDrops(), fieldDrops, se.AttackCount()); !ok {
								l.Warnf("Character [%d] meso-explosion attack with skill [%d] rejected: drop [%d] failed validation.", s.CharacterId(), ai.SkillId(), badId)
								return nil
							}
							explodedMesoDropIds = ai.ExplodedMesoDrops()
						}

						// Skip the generic cost block when a per-skill
						// dispatcher entry exists — that handler owns
						// HP/MP cost (and any cooldown) on the buff-side
						// CharacterUseSkill packet. Without this gate,
						// dual-packet skills like Heal would
						// double-deduct MP.
						registered := false
						if attackIdOk {
							_, registered = handler.Lookup(attackId)
						}
						if !registered {
							if se.HPConsume() > 0 {
								_ = cp.ChangeHP(s.Field(), s.CharacterId(), -int16(se.HPConsume()))
							}
							if se.MPConsume() > 0 {
								_ = cp.ChangeMP(s.Field(), s.CharacterId(), -int16(se.MPConsume()))
							}
						}
					}

					// One buff snapshot per attack, shared by the projectile
					// consumption gate, Pick Pocket, and post-damage
					// buff-driven effects (Combo Drain). Fetched at most once
					// and only when a consumer actually needs it; a lookup
					// failure is cached as "no buffs active" for every
					// consumer and never aborts the attack. Mirrors
					// loadEffectiveStats below.
					loadBuffs := newAttackBuffLoader(l, sp.GetBuffs)

					// Compute projectile consumption plan before broadcasting so planner
					// errors surface before visible side effects. Emission happens post-broadcast.
					pp := NewProjectileProcessor(l, ctx)
					projectilePlan, hasProjectilePlan := pp.Plan(c, ai, se, loadBuffs)

					mp := monster.NewProcessor(l, ctx)
					mirror := monster.GetStatusMirror()
					attackKind := attackKindFromAttackType(ai.AttackType())

					// Per-swing memoized monster resolvers (FR-4.2/R7): the
					// live-mirror-backed resolver serves the reflect check and
					// MP Eater (Mp/MaxMp/X/Y only); the full-model resolver
					// serves the three passives that need Hp/MaxHp, which the
					// mirror deliberately does not carry. Both collapse N
					// per-swing reads into at most one per damaged monster.
					getMonster := buildMonsterResolver(l, ctx, t)
					getMonsterModel := buildMonsterModelResolver(l, ctx, t)

					// Lazy effective-stats fetch: needed when a damage entry
					// produces a VENOM apply and by drain-family heals.
					// Cached for the duration of one attack.
					var effectiveStats effective_stats.RestModel
					effectiveStatsLoaded := false
					loadEffectiveStats := func() effective_stats.RestModel {
						if effectiveStatsLoaded {
							return effectiveStats
						}
						effectiveStatsLoaded = true
						stats, sErr := effective_stats.NewProcessor(l, ctx).GetByCharacterId(s.WorldId(), s.ChannelId(), s.CharacterId())
						if sErr != nil {
							l.WithError(sErr).Errorf("Unable to fetch effective stats for character [%d]; venom DPT and drain heal will fall back to zero.", s.CharacterId())
							return effective_stats.RestModel{}
						}
						effectiveStats = stats
						return effectiveStats
					}

					// Pick Pocket per-attack state: whitelist gate first
					// (pure, no I/O), then at most one buff REST call and
					// one effect lookup. Failures disable the proc and
					// never abort the attack.
					dp := drop.NewProcessor(l, ctx)
					ppState := pickPocketResolveState(
						l,
						loadBuffs,
						skill2.NewProcessor(l, ctx).GetEffect,
						ai.SkillId(),
						s.CharacterId(),
					)

					deps := damageInfoEntryDeps{
						getReflect:         mirror.GetReflect,
						getMonster:         getMonster,
						applyDamage:        mp.Damage,
						emitReflectDamage:  mp.EmitDamageReflected,
						applyStatus:        mp.ApplyStatus,
						loadEffectiveStats: loadEffectiveStats,
						// Per-monster passives, fired once per non-reflected
						// entry after damage and status apply. Failures are
						// swallowed so the rest of the attack pipeline is
						// unaffected.
						onDamageApplied: func(di packetmodel.DamageInfo, totalDamage uint32) {
							if ai.AttackType() == packetmodel.AttackTypeMagic && ai.SkillId() > 0 {
								mpEaterTryProc(l, ctx, mp, getMonster, c, di.MonsterId(), s.Field(), s.CharacterId())
							}
							if ai.SkillId() > 0 && isDrainSkill(skill3.Id(ai.SkillId())) {
								drainTryHeal(l, getMonsterModel, cp.ChangeHP, loadEffectiveStats, se.X(), ai.SkillId(), di.MonsterId(), totalDamage, s.Field(), s.CharacterId())
							}
							if ppState.enabled {
								pickPocketTryProc(l, getMonsterModel, dp.SpawnMeso, ppState, di, s.Field(), s.CharacterId())
							}
							// Mortal Blow proc: per-monster, ranged attacks tagged with the
							// Ranger/Sniper Mortal Blow skill id only. Ownership was enforced
							// upstream (unowned skill ids destroy the session). Failures swallowed (FR-5).
							if isMortalBlowAttack(ai.AttackType(), ai.SkillId()) {
								mortalBlowTryProc(l, mortalBlowDeps{
									getMonster: getMonsterModel,
									emitKill:   mp.Kill,
									roll:       func() int { return rand.Intn(100) + 1 },
								}, se, di.MonsterId(), s.Field(), s.CharacterId(), ai.SkillId())
							}
						},
					}
					for _, di := range ai.DamageInfo() {
						processDamageInfoEntry(
							l, di, ai, se, uint32(sk.Level()),
							s.CharacterId(), c.X(), c.Y(),
							s.Field(), t, attackKind,
							deps,
						)
					}

					_ = _map.NewProcessor(l, ctx).ForOtherSessionsInMap(s.Field(), s.CharacterId(), func(os session.Model) error {
						var writerName string
						var bodyProducer packet.Encode
						if ai.AttackType() == packetmodel.AttackTypeMelee {
							writerName = charpkt.CharacterAttackMeleeWriter
							bodyProducer = writer.CharacterAttackMeleeBody(c, ai)
						} else if ai.AttackType() == packetmodel.AttackTypeRanged {
							writerName = charpkt.CharacterAttackRangedWriter
							bodyProducer = writer.CharacterAttackRangedBody(c, ai)
						} else if ai.AttackType() == packetmodel.AttackTypeMagic {
							writerName = charpkt.CharacterAttackMagicWriter
							bodyProducer = writer.CharacterAttackMagicBody(c, ai)
						} else if ai.AttackType() == packetmodel.AttackTypeEnergy {
							writerName = charpkt.CharacterAttackEnergyWriter
							bodyProducer = writer.CharacterAttackEnergyBody(c, ai)
						} else {
							return errors.New("unhandled attack type")
						}

						err = session.Announce(l)(ctx)(wp)(writerName)(bodyProducer)(os)
						if err != nil {
							l.WithError(err).Errorf("Unable to announce character [%d] is attacking to character [%d].", s.CharacterId(), os.CharacterId())
							return err
						}
						return nil
					})

					// Projectile reservation + consume emits run fire-and-forget after the
					// broadcast. Classic semantics: the projectile is expended the moment the
					// server accepts the attack, regardless of broadcast success.
					if hasProjectilePlan {
						if perr := pp.Emit(s.CharacterId(), projectilePlan); perr != nil {
							l.WithError(perr).Errorf("Failed to emit projectile consumption for character [%d].", s.CharacterId())
						}
					}

					// Destroy the validated exploded meso drops (Chief Bandit Meso
					// Explosion, task-150). CONSUME removes each drop in atlas-drops
					// without crediting anyone; the drop consumer then announces
					// DropDestroy with the explode animation to the whole field.
					// Same at-least-once posture as the projectile emission above:
					// damage is already applied, so failures log and continue.
					if len(explodedMesoDropIds) > 0 {
						if cErr := drop.NewProcessor(l, ctx).ConsumeAll(s.Field(), explodedMesoDropIds); cErr != nil {
							l.WithError(cErr).Errorf("Unable to emit CONSUME for [%d] exploded meso drops for character [%d].", len(explodedMesoDropIds), s.CharacterId())
						} else {
							l.Debugf("Destroyed [%d] exploded meso drops for character [%d].", len(explodedMesoDropIds), s.CharacterId())
						}
					}

					// TODO apply cooldown
					// TODO cancel dark sight / wind walk
					// Combo orb gain/consume: melee only (close-range attacks,
					// Cosmic CloseRangeDamageHandler parity). Fire-and-forget
					// beside the projectile emit — failures never abort the
					// attack. The character was fetched with SkillModelDecorator,
					// so combo skill levels are already in hand.
					if ai.AttackType() == packetmodel.AttackTypeMelee {
						comboOrbTryUpdate(l, c, ai, comboOrbProductionDeps(l, ctx, s.Field(), s.CharacterId()))
						// Aran combo eligibility rides the same fetch: the
						// client sends ARAN_COMBO_COUNTER from CMob::OnHit at
						// melee-hit frequency, and this keeps that path free
						// of REST (task-217 design.md §3.5).
						aranComboRefreshEligibility(l, ctx, s.Field(), c, skill2.NewProcessor(l, ctx).GetEffect)
					}

					// Energy Charge bar gain (task-216). Wider gate than Combo:
					// every close-range attack, the Energy Charge aura's own
					// touch damage (AttackTypeEnergy), and — on the ranged path
					// only — Thunder Breaker Shark Wave. Fire-and-forget beside
					// the combo emit: at most one Kafka message, zero REST, and
					// no branch can fail the attack (NFR-1 / NFR-2). The
					// character was fetched with SkillModelDecorator, so the
					// energy skill level is already in hand.
					//
					// Note there is deliberately NO "don't refresh Energy Charge
					// on its own touch damage" guard here (Cosmic's
					// AbstractDealDamageHandler.java:183-184). Atlas's attack
					// path applies no skill statups at all, so the aura cannot
					// refresh itself — see TestEnergyChargeIsNotAnAttackCastHandler.
					if energyChargeQualifies(ai.AttackType(), attackId, attackIdOk) {
						energyChargeTryUpdate(l, set.Skill, c, ai, energyChargeProductionDeps(l, ctx, s.Field(), s.CharacterId()))
					}

					// Dragon Knight Sacrifice trades the caster's HP for the hit:
					// firstDamageLine × X / 100, clamped to leave at least 1 HP
					// (Cosmic parity — Sacrifice can never kill the caster). This
					// damage-proportional cost is separate from the generic
					// HPConsume/MPConsume cast cost above, which continues to apply.
					// Fire-and-forget like the projectile emit: failures are
					// logged and never abort the attack pipeline.
					if skill3.Id(ai.SkillId()) == skill3.DragonKnightSacrificeId {
						firstLine := sacrificeFirstDamageLine(ai)
						cost := sacrificeHpCost(firstLine, se.X(), c.Hp())
						if cost > 0 {
							l.Debugf("Sacrifice self-HP cost: caster=[%d] skill=[%d] firstLine=[%d] x=[%d] cost=[%d].",
								s.CharacterId(), ai.SkillId(), firstLine, se.X(), cost)
							if herr := cp.ChangeHP(s.Field(), s.CharacterId(), -int16(cost)); herr != nil {
								l.WithError(herr).Errorf("Sacrifice: CHANGE_HP emit failed for caster [%d] skill [%d].", s.CharacterId(), ai.SkillId())
							}
						}
					}

					// Per-skill attack-cast dispatcher (Poison Mist, ...). This is
					// the ATTACK-packet twin of the UseSkill dispatcher at
					// skill/handler/common.go, for skills the client delivers on
					// a melee/ranged/magic attack packet instead of USE_SKILL.
					// It is a SEPARATE registry from handler.Lookup on purpose --
					// see handler.AttackCastHandler's doc for why folding the two
					// would double-fire dual-packet skills like Heal and zero out
					// the attack-only skills' MP cost.
					//
					// Placed here, after damage/broadcast/projectile, with the
					// same fire-and-forget posture as the emits above: the attack
					// pipeline is already complete and a handler failure must
					// never abort it. The WIRE skill id is passed (not the
					// resolved Identity) because handlers put it on the wire for
					// the client to match against its own WZ.
					if attackIdOk && ai.SkillId() > 0 {
						attackCastTryApply(l, ctx, wp, s.Field(), s.CharacterId(), attackId, skill3.Id(ai.SkillId()), sk.Level(), se, attackCastOrigin(ai))
					}

					// TODO apply attack effect (heal, mp consumption, dispel, cure all, combo reset, etc)
					// TODO apply Bandit Steal
					// TODO Fire Demon ice weaken
					// TODO Ice Demon fire weaken
					if ai.AttackType() == packetmodel.AttackTypeRanged && ai.SkillId() > 0 {
						bp := buff.NewProcessor(l, ctx)
						beaconTryApply(l, ai, sk.Level(), s.Field(), s.CharacterId(), beaconApplyDeps{
							monsterExists: func(monsterId uint32) bool {
								_, gErr := mp.GetById(monsterId)
								return gErr == nil
							},
							cancelByTypes: func(types []string) error {
								return bp.CancelByTypes(s.Field(), s.CharacterId(), types)
							},
							applyNoExpiry: func(sourceId int32, level byte, mobId int32) error {
								return bp.ApplyNoExpiry(s.Field(), s.CharacterId(), sourceId, level,
									[]statup.Model{statup.NewModel(string(charconst.TemporaryStatTypeHomingBeacon), mobId)})(s.CharacterId())
							},
						})
					}
					// TODO Flame Thrower
					// TODO Snow Charge
					// TODO Hamstring
					// TODO Slow
					// TODO Blind
					// TODO Paladin / White Knight charges
					comboDrainTryProc(l, loadBuffs, cp.ChangeHP, s.Field(), s.CharacterId(), ai)
					// TODO Three Snails consumption
					// TODO Heavens Hammer
					// TODO ComboTempest
					// TODO BodyPressure

					return nil
				}
			}
		}
	}
}

// battleshipAttackPermitted gates the battleship-dependent attack skills
// (Cannon 5221007, Torpedo 5221008) on an active battleship ride. Every
// attack entry point (melee/ranged/magic/energy/touch) funnels through
// processAttack, so this single gate covers them all. Skills outside the
// pair always pass. Goes through battleship.Processor.IsRiding (a mirror
// read, no I/O) rather than the mirror directly — battleship.NewProcessor
// is a trivial struct init, so constructing one per attack costs nothing
// beyond the read itself. The rejection stays soft: it returns false (never
// destroys the session), matching the caller's nil-return handling.
func battleshipAttackPermitted(l logrus.FieldLogger, ctx context.Context, characterId uint32, skillId skill3.Id) bool {
	if !skill3.Is(skillId, skill3.CorsairBattleshipCannonId, skill3.CorsairBattleshipTorpedoId) {
		return true
	}
	_, riding := battleship.NewProcessor(l, ctx).IsRiding(characterId)
	return riding
}
