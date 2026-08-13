package job

// parents is the job ADVANCEMENT relation: which identity a job advances
// from. It is VERSION-BLIND and identity-keyed, and that is a verified
// property, not a convenience: every job row in
// docs/tasks/task-187-version-aware-id-semantics/audit/divergences.csv is a
// WIRE-BINDING divergence (the same Identity bound to a different wire id in
// a different version) and never a structural re-parenting. Gm's parent is
// Beginner in every version we support; only the wire id differs (500 at
// gms 48.1, 900 at gms 61.1+). Set.ParentWire below composes this table with
// the per-version wire binding to reproduce each version's edges exactly, so
// a new version costs zero new edges here.
//
// FR-3.2 / task-182 -- READ BEFORE "CORRECTING" THE Gm AND SuperGm ENTRIES.
// libs/atlas-constants/job/constants.go models Gm (900) and SuperGm (910) as
// independent roots. That is the REGISTRY view and it is correct there. The
// game PRESENTS them as an advancement line beneath Beginner, which is the
// view this table encodes -- the ADVANCEMENT/DISPLAY view. atlas-ui's
// JOB_GRAPH carried this divergence privately (task-182); task-202 moved it
// here so every client gets the same answer. Reverting Gm -> Beginner and
// SuperGm -> Gm to roots silently regresses the v0.48 acceptance criterion
// ("the Special group shows Gm with Super Gm beneath it").
//
// The table is explicit rather than arithmetic. A formula (id/10*10, then
// id/100*100) covers the Explorer branches but not Gm, Evan, Aran, or the
// Cygnus stage lines; a formula with four exceptions is neither auditable
// nor greppable. Roots -- deliberately absent from this map -- are Beginner,
// MapleLeafBrigadier, Noblesse, Legend and Evan.
var parents = map[Identity]Identity{
	// Warrior
	Warrior:      Beginner,
	Fighter:      Warrior,
	Crusader:     Fighter,
	Hero:         Crusader,
	Page:         Warrior,
	WhiteKnight:  Page,
	Paladin:      WhiteKnight,
	Spearman:     Warrior,
	DragonKnight: Spearman,
	DarkKnight:   DragonKnight,

	// Magician
	Magician:                 Beginner,
	FirePoisonWizard:         Magician,
	FirePoisonMagician:       FirePoisonWizard,
	FirePoisonArchMagician:   FirePoisonMagician,
	IceLightningWizard:       Magician,
	IceLightningMagician:     IceLightningWizard,
	IceLightningArchMagician: IceLightningMagician,
	Cleric:                   Magician,
	Priest:                   Cleric,
	Bishop:                   Priest,

	// Bowman
	Bowman:      Beginner,
	Hunter:      Bowman,
	Ranger:      Hunter,
	Bowmaster:   Ranger,
	Crossbowman: Bowman,
	Sniper:      Crossbowman,
	Marksman:    Sniper,

	// Thief
	Rogue:       Beginner,
	Assassin:    Rogue,
	Hermit:      Assassin,
	NightLord:   Hermit,
	Bandit:      Rogue,
	ChiefBandit: Bandit,
	Shadower:    ChiefBandit,

	// Dual Blade (task-204) -- the third Rogue branch, and the only Explorer
	// line with five advancements instead of three. Rooted at Rogue, NOT at
	// Beginner: quest 2351 ("First Mission: Infiltration", the Dual Blade
	// intro chain) has demandSummary "Make a job advancement as a #bRogue#k"
	// and a job gate of [0,400,410,420,430,...], and every Dual Blade job id
	// appears in the Thief-branch job gates (e.g. quest 2140 "Beginner
	// Thief's First Training Session" gates on
	// [400,410,411,412,420,421,422,430,431,432,433,434]).
	//
	// The chain is linear. WZ job gates order the branch by tier --
	// 400,410,420,430,411,421,431,412,422,432,433,434 -- which puts 430
	// alongside Assassin/Bandit, 431 alongside Hermit/ChiefBandit, 432
	// alongside NightLord/Shadower, and 433/434 beyond the Explorer tiers;
	// the level-70+ gate (quest 3121) admits 432/433/434 but not 430/431.
	// All quest evidence read from the provisioned gms 92.1 tenant via
	// atlas-data GET /api/data/quests (2026-08-09).
	BladeRecruit:    Rogue,
	BladeAcolyte:    BladeRecruit,
	BladeSpecialist: BladeAcolyte,
	BladeLord:       BladeSpecialist,
	BladeMaster:     BladeLord,

	// Pirate
	Pirate:     Beginner,
	Brawler:    Pirate,
	Marauder:   Brawler,
	Buccaneer:  Marauder,
	Gunslinger: Pirate,
	Outlaw:     Gunslinger,
	Corsair:    Outlaw,

	// Admin -- the task-182 display convention; see the FR-3.2 note above.
	Gm:      Beginner,
	SuperGm: Gm,

	// Cygnus Knights
	DawnWarriorStage1:    Noblesse,
	DawnWarriorStage2:    DawnWarriorStage1,
	DawnWarriorStage3:    DawnWarriorStage2,
	DawnWarriorStage4:    DawnWarriorStage3,
	BlazeWizardStage1:    Noblesse,
	BlazeWizardStage2:    BlazeWizardStage1,
	BlazeWizardStage3:    BlazeWizardStage2,
	BlazeWizardStage4:    BlazeWizardStage3,
	WindArcherStage1:     Noblesse,
	WindArcherStage2:     WindArcherStage1,
	WindArcherStage3:     WindArcherStage2,
	WindArcherStage4:     WindArcherStage3,
	NightWalkerStage1:    Noblesse,
	NightWalkerStage2:    NightWalkerStage1,
	NightWalkerStage3:    NightWalkerStage2,
	NightWalkerStage4:    NightWalkerStage3,
	ThunderBreakerStage1: Noblesse,
	ThunderBreakerStage2: ThunderBreakerStage1,
	ThunderBreakerStage3: ThunderBreakerStage2,
	ThunderBreakerStage4: ThunderBreakerStage3,

	// Aran
	AranStage1: Legend,
	AranStage2: AranStage1,
	AranStage3: AranStage2,
	AranStage4: AranStage3,

	// Evan
	EvanStage1:  Evan,
	EvanStage2:  EvanStage1,
	EvanStage3:  EvanStage2,
	EvanStage4:  EvanStage3,
	EvanStage5:  EvanStage4,
	EvanStage6:  EvanStage5,
	EvanStage7:  EvanStage6,
	EvanStage8:  EvanStage7,
	EvanStage9:  EvanStage8,
	EvanStage10: EvanStage9,
}

// ParentIdentity returns the identity id advances from, or (0, false) if id
// is a branch root (Beginner, MapleLeafBrigadier, Noblesse, Legend, Evan) or
// is not a known identity. Version-blind -- see the parents table doc.
func ParentIdentity(id Identity) (Identity, bool) {
	p, ok := parents[id]
	return p, ok
}

// ParentWire returns THIS version's advancement edge for id, in wire ids.
// It reports (0, false) when id is a root, when id's parent is not Available
// at this version, or when the parent has no wire binding here.
//
// FR-3.4 / design D7: an unavailable parent makes the entry a ROOT. It
// deliberately does NOT walk up to the nearest available ancestor --
// reparenting would invent an edge the game never had, and a synthesised
// grandparent edge is a lie that renders convincingly. If a version ever
// ships a job whose parent it did not ship, "root" is an honest rendering of
// a genuinely odd situation. TestParentWire_D7PolicyGuard fails the day this
// becomes observable, so the choice gets re-made deliberately.
//
// Callers must treat (0, false) as "no parent", never as "parent is wire id
// 0" -- Beginner IS wire id 0 and is a legitimate parent value.
func (s Set) ParentWire(id Identity) (Id, bool) {
	p, ok := parents[id]
	if !ok {
		return 0, false
	}
	if !s.Available(p) {
		return 0, false
	}
	w, ok := s.byIdentity[p]
	if !ok {
		return 0, false
	}
	return w, true
}
