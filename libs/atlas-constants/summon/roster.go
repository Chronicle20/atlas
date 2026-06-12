package summon

import "github.com/Chronicle20/atlas/libs/atlas-constants/skill"

type Type string

const (
	TypePuppet   Type = "PUPPET"
	TypeAttacker Type = "ATTACKER"
	TypeBuffAura Type = "BUFF_AURA"
)

type Movement byte

const (
	MovementStationary   Movement = 0
	MovementFollow       Movement = 1
	MovementCircleFollow Movement = 3
)

type Entry struct {
	Type     Type
	Movement Movement
	Stun     bool // applies STUN monster status on hit
	Freeze   bool // applies FREEZE monster status on hit
	OneShot  bool // self-cancels after a single attack (Gaviota)
}

// roster: the 21 v83 summon skills. Adding a summon = one row here, no engine change.
// Keys reference the named skill-id constants in libs/atlas-constants/skill (no import
// cycle: skill imports nothing). Type and movement come from Cosmic StatEffect.java /
// Summon.isPuppet()/isStationary().
var roster = map[uint32]Entry{
	uint32(skill.RangerPuppetId):                        {Type: TypePuppet, Movement: MovementStationary},                    // 3111002 Ranger Puppet
	uint32(skill.SniperPuppetId):                        {Type: TypePuppet, Movement: MovementStationary},                    // 3211002 Sniper Puppet
	uint32(skill.WindArcherStage3PuppetId):              {Type: TypePuppet, Movement: MovementStationary},                    // 13111004 Wind Archer Puppet
	uint32(skill.OutlawOctopusId):                       {Type: TypeAttacker, Movement: MovementStationary},                  // 5211001 Octopus
	uint32(skill.CorsairWrathOfTheOctopiId):             {Type: TypeAttacker, Movement: MovementStationary},                  // 5220002 Wrath of the Octopi
	uint32(skill.RangerSilverHawkId):                    {Type: TypeAttacker, Movement: MovementCircleFollow, Stun: true},    // 3111005 Silver Hawk
	uint32(skill.SniperGoldenEagleId):                   {Type: TypeAttacker, Movement: MovementCircleFollow, Stun: true},    // 3211005 Golden Eagle
	uint32(skill.BowmasterPhoenixId):                    {Type: TypeAttacker, Movement: MovementCircleFollow},                // 3121006 Phoenix
	uint32(skill.MarksmanFrostpreyId):                   {Type: TypeAttacker, Movement: MovementCircleFollow, Freeze: true},  // 3221005 Frostprey
	uint32(skill.PriestSummonDragonId):                  {Type: TypeAttacker, Movement: MovementCircleFollow},                // 2311006 Summon Dragon
	uint32(skill.OutlawGaviotaId):                       {Type: TypeAttacker, Movement: MovementCircleFollow, OneShot: true}, // 5211002 Gaviota
	uint32(skill.FirePoisonArchMagicianElquinesId):      {Type: TypeAttacker, Movement: MovementFollow, Freeze: true},        // 2121005 Elquines
	uint32(skill.IceLightningArchMagicianIfritId):       {Type: TypeAttacker, Movement: MovementFollow},                      // 2221005 Ifrit (I/L)
	uint32(skill.BishopBahamutId):                       {Type: TypeAttacker, Movement: MovementFollow},                      // 2321003 Bahamut
	uint32(skill.DawnWarriorStage1SoulId):               {Type: TypeAttacker, Movement: MovementFollow},                      // 11001004 Dawn Warrior Soul
	uint32(skill.BlazeWizardStage1FlameId):              {Type: TypeAttacker, Movement: MovementFollow},                      // 12001004 Blaze Wizard Flame
	uint32(skill.BlazeWizardStage3IfritId):              {Type: TypeAttacker, Movement: MovementFollow},                      // 12111004 Blaze Wizard Ifrit
	uint32(skill.WindArcherStage1StormId):               {Type: TypeAttacker, Movement: MovementFollow},                      // 13001004 Wind Archer Storm
	uint32(skill.NightWalkerStage1DarknessId):           {Type: TypeAttacker, Movement: MovementFollow},                      // 14001005 Night Walker Darkness
	uint32(skill.ThunderBreakerStage1LightningSpriteId): {Type: TypeAttacker, Movement: MovementFollow},                      // 15001004 Thunder Breaker Lightning
	uint32(skill.DarkKnightBeholderId):                  {Type: TypeBuffAura, Movement: MovementFollow},                      // 1321007 Dark Knight Beholder
}

func Lookup(skillId uint32) (Entry, bool) {
	e, ok := roster[skillId]
	return e, ok
}

func IsSummonSkill(skillId uint32) bool {
	_, ok := roster[skillId]
	return ok
}
