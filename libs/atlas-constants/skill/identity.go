package skill

import "sort"

// Identity is a version-blind skill identity: a stable name for "this
// skill concept" independent of the wire id any particular client version
// binds it to. It is distinct from Id, which is a version's raw wire id --
// the same Identity can bind to different Id values across versions (and
// the same Id value can mean different things in different versions; see
// docs/tasks/task-187-version-aware-id-semantics).
//
// The identity constants in identities_gen.go are keyed by their canonical
// (v83-era) wire id token, which is opaque here: it is only a stable,
// collision-free numeric key for the identity, not necessarily this
// version's wire id. Per-version wireId<->Identity binding tables are
// built on top of this namespace by later tasks.
type Identity uint32

// IdentityName returns the checked-in display name for id, or "" if id has
// no known identity.
func IdentityName(id Identity) string {
	return identityNames[id]
}

// Set is one version's immutable wireId<->Identity binding table, built by
// the generator (see version_<r>_<maj>_<min>_gen.go) from
// docs/tasks/task-187-version-aware-id-semantics's per-version semantics +
// availability manifests. Zero value is a valid, empty Set.
type Set struct {
	byWire     map[Id]Identity
	byIdentity map[Identity]Id
	available  map[Identity]struct{} // this version's release-available identities (task-187 Task 5)
	names      map[Identity]string   // this version's identity -> display name (task-187 Task 5)
}

// Resolve returns the Identity this version's wireId is bound to, or
// (0, false) if wireId is not present in this version's semantics.
func (s Set) Resolve(wireId Id) (Identity, bool) {
	id, ok := s.byWire[wireId]
	return id, ok
}

// Wire returns the wireId this version binds id to, or (0, false) if id has
// no binding in this version's semantics.
func (s Set) Wire(id Identity) (Id, bool) {
	w, ok := s.byIdentity[id]
	return w, ok
}

// Available reports whether id was actually released/playable at this
// version -- a SUBSET of presence (Resolve/Wire): an identity can be
// present in the WZ data as an unreleased stub well before its class
// actually shipped. See task-187 Task 5 and
// docs/tasks/task-187-version-aware-id-semantics/audit/availability.csv.
func (s Set) Available(id Identity) bool {
	_, ok := s.available[id]
	return ok
}

// Name returns the version-independent display name for id, or "" if id
// has no binding in this version's semantics.
func (s Set) Name(id Identity) string {
	return s.names[id]
}

// AvailableIdentities returns every Identity available at this version,
// sorted ascending by this version's wire id.
func (s Set) AvailableIdentities() []Identity {
	out := make([]Identity, 0, len(s.available))
	for id := range s.available {
		out = append(out, id)
	}
	sort.Slice(out, func(i, j int) bool { return s.byIdentity[out[i]] < s.byIdentity[out[j]] })
	return out
}

// ---- Identity-keyed semantic predicates (task-187 Task 7) ----
//
// Every predicate below is a version-INDEPENDENT property of an Identity --
// "does this skill concept behave as a keydown attack / shoot skill / mount
// / point-reset-excluded skill", never "what does wire id N mean in version
// V" (that question is Set.Resolve/Set.Wire above). Bodies are ported
// verbatim from their Id-typed originals in model.go/mount.go/
// point_reset.go: only the types change (Id -> Identity, the *Id
// reference constants -> the identities_gen.go Identity constants),
// because Identity tokens are keyed by the exact same canonical (v83-era)
// wire numbering the Id-typed arithmetic already assumes -- see this
// file's package doc and docs/tasks/task-187-version-aware-id-semantics.
//
// Names are suffixed "Identity" (IsKeyDownSkillIdentity, not
// IsKeyDownSkill) wherever the Id-typed original already owns the bare
// name: Go has no function overloading, and the Id-typed predicates in
// model.go/mount.go/point_reset.go stay untouched (additive-only) per the
// task-187 Task 7 brief. IsIdentity is new (Is is variadic already but
// Id-typed; IsIdentity is its Identity-typed counterpart -- no collision).

// IsIdentity is the Identity form of Is: reports whether id equals any of
// refs.
func IsIdentity(id Identity, refs ...Identity) bool {
	for _, r := range refs {
		if id == r {
			return true
		}
	}
	return false
}

// IsShootSkillNotUsingShootingWeaponIdentity is the Identity form of
// IsShootSkillNotUsingShootingWeapon (model.go).
func IsShootSkillNotUsingShootingWeaponIdentity(id Identity) bool {
	switch id {
	case NightLordTaunt, ShadowerTaunt, BuccaneerEnergyOrb, DawnWarriorStage2SoulBlade, ThunderBreakerStage3Spark, ThunderBreakerStage3SharkWave, AranStage2ComboSmash, AranStage3ComboFenrir, AranStage4ComboTempest:
		return true
	default:
		return false
	}
}

// IsShootSkillNotConsumingBulletIdentity is the Identity form of
// IsShootSkillNotConsumingBullet (model.go).
func IsShootSkillNotConsumingBulletIdentity(id Identity) bool {
	if IsShootSkillNotUsingShootingWeaponIdentity(id) {
		return true
	}
	switch id {
	case HunterPowerKnockBack, CrossbowmanPowerKnockBack, HermitShadowMeso, WindArcherStage2StormBreak, NightWalkerStage2Vampire:
		return true
	default:
		return false
	}
}

// IsKeyDownSkillIdentity is the Identity form of IsKeyDownSkill (model.go).
func IsKeyDownSkillIdentity(id Identity) bool {
	return IsIdentity(id,
		FirePoisonArchMagicianBigBang,
		IceLightningArchMagicianBigBang,
		BishopBigBang,
		HeroMonsterMagnet,
		PaladinMonsterMagnet,
		DarkKnightMonsterMagnet,
		BowmasterHurricane,
		MarksmanPiercingArrow,
		CorsairRapidFire,
		NightWalkerStage3PoisonBomb,
		WindArcherStage3Hurricane,
		ThunderBreakerStage2CorkscrewBlow,
		EvanStage4IceBreath,
		EvanStage7FireBreath,
		BrawlerCorkscrewBlow, // 5101004 -- IDA-verified keydown v61/v72/v79/v83/v87/v95/jms185 (task-161)
		GunslingerGrenade)    // 5201002 -- IDA-verified keydown v61/v72/v79/v83/v87/v95/jms185 (task-161)
}

// IsTamedMountSkillIdentity is the Identity form of IsTamedMountSkill
// (mount.go): reports whether id is a tamed-monster MonsterRider skill
// (Beginner/Noblesse/Legend/Evan band). Tamed mounts read the equipped
// taming-mob item id as the vehicle; skill-only mounts do not.
func IsTamedMountSkillIdentity(id Identity) bool {
	return IsIdentity(id, BeginnerMonsterRiding, NoblesseMonsterRiding, LegendMonsterRiding, EvanMonsterRiding)
}

// SkillOnlyMountVehicleIdentity is the Identity form of
// SkillOnlyMountVehicleId (mount.go): maps a skill-only mount identity (any
// band) to its fixed vehicle item id. SpaceShip is per-level
// (1932000+level). Returns false for identities that are not skill-only
// mounts. The return value is still a raw item wire id (int32), not an
// Identity -- mount vehicle items are not modeled in this identity space.
func SkillOnlyMountVehicleIdentity(id Identity, level int) (int32, bool) {
	switch id {
	case BeginnerSpaceShip, NoblesseSpaceShip:
		return int32(1932000 + level), true
	case BeginnerYetiMount1, NoblesseYetiMount1, LegendYetiMount1:
		return 1932003, true
	case BeginnerYetiMount2, NoblesseYetiMount2, LegendYetiMount2:
		return 1932004, true
	case BeginnerBroomstick, NoblesseBroomstick, LegendBroomstick:
		return 1932005, true
	case BeginnerBalrogMount, NoblesseBalrogMount, LegendBalrogMount:
		return 1932010, true
	default:
		return 0, false
	}
}

// IsPointResetExcludedIdentity is the Identity form of
// IsPointResetExcluded (point_reset.go): reports whether id may not
// participate in an SP Reset transfer (as source or target): Aran hidden
// combo skills, GM skills, and PQ-granted skills, whose points are not
// pool-backed (see docs/tasks/task-126-ap-sp-reset-items/design.md §4.1).
func IsPointResetExcludedIdentity(id Identity) bool {
	switch id {
	case Identity(21110007), Identity(21110008), Identity(21120009), Identity(21120010): // Aran hidden combo
		return true
	case Identity(10000013), Identity(20001013): // PQ skills (fixed ids)
		return true
	}
	if id >= Identity(9001000) && id <= Identity(9101008) { // GM skills
		return true
	}
	if id >= Identity(8001000) && id <= Identity(8001001) { // GM skills
		return true
	}
	if id >= Identity(20000014) && id <= Identity(20000018) { // PQ skills
		return true
	}
	rem := uint32(id) % 10000000
	if rem >= 1009 && rem <= 1011 { // PQ skills (per-class beginner band)
		return true
	}
	if rem == 1020 { // PQ skill
		return true
	}
	return false
}
