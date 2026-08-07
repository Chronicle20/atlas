package character

import "github.com/Chronicle20/atlas/libs/atlas-constants/job"

// AP Reset (item 5050000) server policy. Values verbatim from PRD §4.3.
// Fixed values — not tenant-configurable (design §10).
const (
	pointResetPrimaryFloor = uint16(4)     // post-swap floor; source must be >= 5
	pointResetPrimaryCap   = uint16(32767) // AP hard cap
	pointResetPoolCap      = uint16(30000) // HP/MP pool reject bound
)

type pointResetPolicy struct {
	takeHp uint16 // MaxHP loss when resetting OUT of HP
	takeMp uint16 // MaxMP loss when resetting OUT of MP
	gainHp uint16 // MaxHP gain when resetting INTO HP (deterministic AP-reset path)
	gainMp uint16 // MaxMP gain when resetting INTO MP
}

// Branch rows use job.IsAIdentity semantics against branch-root reference
// identities; first match wins, default (Beginner/Noblesse/Legend) last.
// Explorer roots are the canonical branch identities: Warrior, Magician,
// Bowman, Rogue, Pirate.
//
// DIVERGENT (task-187 audit): the Pirate row below (job.Pirate, wire 500 at
// v0.61+) collides with wire 500/510 GM/SuperGM at v0.48 (job 900/910 don't
// exist until v0.61) -- a raw job.Id(500) row, matched via a raw job.Is
// against the character's wire c.JobId(), would misclassify a v0.48 GM
// resetting AP as a Pirate. Every row/ref here is Identity-typed so callers
// resolve c.JobId() once (see TransferAP) and match against the
// version-blind identity space instead.
var pointResetPolicyRows = []struct {
	refs   []job.Identity
	policy pointResetPolicy
}{
	{refs: []job.Identity{job.Warrior, job.DawnWarriorStage1, job.AranStage1}, policy: pointResetPolicy{takeHp: 54, takeMp: 4, gainHp: 20, gainMp: 2}},
	// Magician takeMp (31) is a fallback only: the client scales the magician
	// MP-reset-out loss with effective INT (see pointResetMagicianTakeMp). All
	// other magician values (takeHp, gainHp, gainMp) are fixed and match the client.
	{refs: []job.Identity{job.Magician, job.BlazeWizardStage1}, policy: pointResetPolicy{takeHp: 10, takeMp: 31, gainHp: 6, gainMp: 18}},
	{refs: []job.Identity{job.Bowman, job.WindArcherStage1}, policy: pointResetPolicy{takeHp: 20, takeMp: 12, gainHp: 16, gainMp: 10}},
	{refs: []job.Identity{job.Rogue, job.NightWalkerStage1}, policy: pointResetPolicy{takeHp: 20, takeMp: 12, gainHp: 16, gainMp: 10}},
	{refs: []job.Identity{job.Pirate, job.ThunderBreakerStage1}, policy: pointResetPolicy{takeHp: 42, takeMp: 16, gainHp: 18, gainMp: 14}},
}

var pointResetDefaultPolicy = pointResetPolicy{takeHp: 12, takeMp: 8, gainHp: 8, gainMp: 6}

// pointResetPolicyFor takes a job Identity: callers resolve the character's
// version-specific wire job id through p.set().Job.Resolve first (task-187 --
// see pointResetPolicyRows' DIVERGENT note above). An unresolved id (the
// caller passing the zero Identity on a failed resolve) matches no row here
// and falls through to the default policy, same as any other non-match.
func pointResetPolicyFor(jid job.Identity) pointResetPolicy {
	for _, row := range pointResetPolicyRows {
		if job.IsAIdentity(jid, row.refs...) {
			return row.policy
		}
	}
	return pointResetDefaultPolicy
}

// isPointResetMagician reports whether jid is on the magician branch, using
// the same branch-root refs as the magician pointResetPolicy row
// (job.IsAIdentity on a branch root also matches its sub-lines). Deliberately
// identical to that row so a character receives the INT-scaled MP loss
// (pointResetMagicianTakeMp) IFF it also receives the magician gain/HP/min
// policy — never a hybrid. The client's raw branch classifier (sub_A0EC6B:
// job%1000/100==2) additionally sweeps in Evan (22xx), but Evan is v84+ and
// its reset policy is unverified against the client, so it is left on the
// default policy here consistently rather than given magician MP loss with
// beginner HP/gain values.
func isPointResetMagician(jid job.Identity) bool {
	return job.IsAIdentity(jid, job.Magician, job.BlazeWizardStage1)
}

// pointResetMagicianTakeMp is the MaxMP lost when a magician resets one point
// OUT of MP. Unlike every other branch (a fixed takeMp), the client scales the
// magician MP loss with EFFECTIVE INT (base + equipment):
//
//	takeMp = 3*effectiveInt/40 + 30   (integer division)
//
// Verified against the GMS v83 client: the reset-dialog MP-loss calc
// (sub_8CE5BD @0x8ce5bd, branch-2 arm) reads the cached effective INT at
// CWvsContext+0x20F8 and the reset-dialog button gate (sub_8CBDDB @0x8cbddb)
// disables MP-as-source using this same value. PRD §4.3's flat 31 is only
// correct at effectiveInt≈14 (3*14/40+30 == 31); a higher-INT mage would drop
// more MaxMP client-side than the server applies, desyncing the pool until
// relog. HP loss and all gain values stay constant (they match the client).
func pointResetMagicianTakeMp(effectiveInt uint16) uint16 {
	return uint16(3*int(effectiveInt)/40 + 30)
}

// Minimum pool after a reset-out: mult*level + off (PRD §4.3 min table).
// Rows are ordered narrowest-first because job.IsAIdentity on a branch root
// also matches its sub-lines. Offsets can be negative; callers compare as
// int.
//
// DIVERGENT (task-187 audit): the "Brawler/Gunslinger lines, TB2+" and
// "Pirate base, TB1" rows below (job.Brawler/job.Gunslinger wire 510/520,
// job.Pirate wire 500) collide with wire 500/510 GM/SuperGM at v0.48 -- see
// pointResetPolicyRows' DIVERGENT note above. Identity-typed for the same
// reason.
type poolMinRow struct {
	refs []job.Identity
	mult int
	off  int
}

var pointResetMinHpRows = []poolMinRow{
	{refs: []job.Identity{job.Fighter, job.DawnWarriorStage2, job.AranStage2}, mult: 24, off: 418},                                                 // Fighter-line, DW2+, Aran2+
	{refs: []job.Identity{job.Warrior, job.DawnWarriorStage1, job.AranStage1}, mult: 24, off: 118},                                                 // rest of the warrior branch (incl. Page/Spearman lines)
	{refs: []job.Identity{job.Magician, job.BlazeWizardStage1}, mult: 10, off: 54},                                                                 // Magician-line, Blaze Wizard
	{refs: []job.Identity{job.Hunter, job.Crossbowman, job.Assassin, job.Bandit, job.WindArcherStage2, job.NightWalkerStage2}, mult: 20, off: 358}, // 2nd-job+ bowman/thief lines
	{refs: []job.Identity{job.Bowman, job.Rogue, job.WindArcherStage1, job.NightWalkerStage1}, mult: 20, off: 58},                                  // bowman/thief base
	{refs: []job.Identity{job.Brawler, job.Gunslinger, job.ThunderBreakerStage2}, mult: 22, off: 338},                                              // Brawler/Gunslinger lines, TB2+
	{refs: []job.Identity{job.Pirate, job.ThunderBreakerStage1}, mult: 22, off: 38},                                                                // Pirate base, TB1
}

var pointResetMinMpRows = []poolMinRow{
	{refs: []job.Identity{job.Page, job.Spearman}, mult: 4, off: 155},                                                                              // Page-/Spearman-line
	{refs: []job.Identity{job.Warrior, job.DawnWarriorStage1, job.AranStage1}, mult: 4, off: 55},                                                   // Warrior, Fighter-line, DW, Aran
	{refs: []job.Identity{job.FirePoisonWizard, job.IceLightningWizard, job.Cleric, job.BlazeWizardStage2}, mult: 22, off: 449},                    // Magician 2nd job+
	{refs: []job.Identity{job.Magician, job.BlazeWizardStage1}, mult: 22, off: -1},                                                                 // Magician base, BW1
	{refs: []job.Identity{job.Hunter, job.Crossbowman, job.Assassin, job.Bandit, job.WindArcherStage2, job.NightWalkerStage2}, mult: 14, off: 135}, // bowman/thief 2nd job+
	{refs: []job.Identity{job.Bowman, job.Rogue, job.WindArcherStage1, job.NightWalkerStage1}, mult: 14, off: -15},                                 // bowman/thief base
	{refs: []job.Identity{job.Brawler, job.Gunslinger, job.ThunderBreakerStage2}, mult: 18, off: 95},                                               // Brawler/Gunslinger lines, TB2+
	{refs: []job.Identity{job.Pirate, job.ThunderBreakerStage1}, mult: 18, off: -55},                                                               // Pirate base, TB1
}

func resolvePoolMin(rows []poolMinRow, defaultMult int, defaultOff int, jid job.Identity, level byte) int {
	for _, row := range rows {
		if job.IsAIdentity(jid, row.refs...) {
			return row.mult*int(level) + row.off
		}
	}
	return defaultMult*int(level) + defaultOff
}

func pointResetMinHp(jid job.Identity, level byte) int {
	return resolvePoolMin(pointResetMinHpRows, 12, 38, jid, level) // default: Beginner/Noblesse
}

func pointResetMinMp(jid job.Identity, level byte) int {
	return resolvePoolMin(pointResetMinMpRows, 10, -5, jid, level) // default: Beginner/Noblesse
}
