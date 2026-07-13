package character

import "github.com/Chronicle20/atlas/libs/atlas-constants/job"

// AP Reset (item 5050000) server policy. Values verbatim from PRD §4.3
// (Cosmic AssignAPProcessor under default config). Fixed reference-config
// parity — not tenant-configurable (design §10).
const (
	pointResetPrimaryFloor = uint16(4)     // post-swap floor; source must be >= 5
	pointResetPrimaryCap   = uint16(32767) // Cosmic MAX_AP
	pointResetPoolCap      = uint16(30000) // Cosmic assignHP/assignMP reject bound
)

type pointResetPolicy struct {
	takeHp uint16 // MaxHP loss when resetting OUT of HP
	takeMp uint16 // MaxMP loss when resetting OUT of MP
	gainHp uint16 // MaxHP gain when resetting INTO HP (deterministic AP-reset path)
	gainMp uint16 // MaxMP gain when resetting INTO MP
}

// Branch rows use job.Is semantics against branch-root reference ids; first
// match wins, default (Beginner/Noblesse/Legend) last. Explorer roots are the
// raw branch ids: 100 warrior, 200 magician, 300 bowman, 400 thief, 500 pirate.
var pointResetPolicyRows = []struct {
	refs   []job.Id
	policy pointResetPolicy
}{
	{refs: []job.Id{job.Id(100), job.DawnWarriorStage1Id, job.AranStage1Id}, policy: pointResetPolicy{takeHp: 54, takeMp: 4, gainHp: 20, gainMp: 2}},
	// Magician takeMp (31) is a fallback only: the client scales the magician
	// MP-reset-out loss with effective INT (see pointResetMagicianTakeMp). All
	// other magician values (takeHp, gainHp, gainMp) are fixed and match the client.
	{refs: []job.Id{job.Id(200), job.BlazeWizardStage1Id}, policy: pointResetPolicy{takeHp: 10, takeMp: 31, gainHp: 6, gainMp: 18}},
	{refs: []job.Id{job.Id(300), job.WindArcherStage1Id}, policy: pointResetPolicy{takeHp: 20, takeMp: 12, gainHp: 16, gainMp: 10}},
	{refs: []job.Id{job.Id(400), job.NightWalkerStage1Id}, policy: pointResetPolicy{takeHp: 20, takeMp: 12, gainHp: 16, gainMp: 10}},
	{refs: []job.Id{job.Id(500), job.ThunderBreakerStage1Id}, policy: pointResetPolicy{takeHp: 42, takeMp: 16, gainHp: 18, gainMp: 14}},
}

var pointResetDefaultPolicy = pointResetPolicy{takeHp: 12, takeMp: 8, gainHp: 8, gainMp: 6}

func pointResetPolicyFor(jobId job.Id) pointResetPolicy {
	for _, row := range pointResetPolicyRows {
		for _, ref := range row.refs {
			if job.Is(jobId, ref) {
				return row.policy
			}
		}
	}
	return pointResetDefaultPolicy
}

// isPointResetMagician reports whether jobId is on the magician branch, using
// the same branch-root refs as the magician pointResetPolicy row (job.Is on a
// branch root also matches its sub-lines). Deliberately identical to that row
// so a character receives the INT-scaled MP loss (pointResetMagicianTakeMp)
// IFF it also receives the magician gain/HP/min policy — never a hybrid. The
// client's raw branch classifier (sub_A0EC6B: job%1000/100==2) additionally
// sweeps in Evan (22xx), but Evan is v84+ and its reset policy is unverified
// against the client, so it is left on the default policy here consistently
// rather than given magician MP loss with beginner HP/gain values.
func isPointResetMagician(jobId job.Id) bool {
	return job.Is(jobId, job.Id(200)) || job.Is(jobId, job.BlazeWizardStage1Id)
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
// Rows are ordered narrowest-first because job.Is on a branch root also
// matches its sub-lines. Offsets can be negative; callers compare as int.
type poolMinRow struct {
	refs []job.Id
	mult int
	off  int
}

var pointResetMinHpRows = []poolMinRow{
	{refs: []job.Id{job.Id(110), job.DawnWarriorStage2Id, job.AranStage2Id}, mult: 24, off: 418},                                              // Fighter-line, DW2+, Aran2+
	{refs: []job.Id{job.Id(100), job.DawnWarriorStage1Id, job.AranStage1Id}, mult: 24, off: 118},                                              // rest of the warrior branch (incl. Page/Spearman lines)
	{refs: []job.Id{job.Id(200), job.BlazeWizardStage1Id}, mult: 10, off: 54},                                                                 // Magician-line, Blaze Wizard
	{refs: []job.Id{job.Id(310), job.Id(320), job.Id(410), job.Id(420), job.WindArcherStage2Id, job.NightWalkerStage2Id}, mult: 20, off: 358}, // 2nd-job+ bowman/thief lines
	{refs: []job.Id{job.Id(300), job.Id(400), job.WindArcherStage1Id, job.NightWalkerStage1Id}, mult: 20, off: 58},                            // bowman/thief base
	{refs: []job.Id{job.Id(510), job.Id(520), job.ThunderBreakerStage2Id}, mult: 22, off: 338},                                                // Brawler/Gunslinger lines, TB2+
	{refs: []job.Id{job.Id(500), job.ThunderBreakerStage1Id}, mult: 22, off: 38},                                                              // Pirate base, TB1
}

var pointResetMinMpRows = []poolMinRow{
	{refs: []job.Id{job.Id(120), job.Id(130)}, mult: 4, off: 155},                                                                             // Page-/Spearman-line
	{refs: []job.Id{job.Id(100), job.DawnWarriorStage1Id, job.AranStage1Id}, mult: 4, off: 55},                                                // Warrior, Fighter-line, DW, Aran
	{refs: []job.Id{job.Id(210), job.Id(220), job.Id(230), job.BlazeWizardStage2Id}, mult: 22, off: 449},                                      // Magician 2nd job+
	{refs: []job.Id{job.Id(200), job.BlazeWizardStage1Id}, mult: 22, off: -1},                                                                 // Magician base, BW1
	{refs: []job.Id{job.Id(310), job.Id(320), job.Id(410), job.Id(420), job.WindArcherStage2Id, job.NightWalkerStage2Id}, mult: 14, off: 135}, // bowman/thief 2nd job+
	{refs: []job.Id{job.Id(300), job.Id(400), job.WindArcherStage1Id, job.NightWalkerStage1Id}, mult: 14, off: -15},                           // bowman/thief base
	{refs: []job.Id{job.Id(510), job.Id(520), job.ThunderBreakerStage2Id}, mult: 18, off: 95},                                                 // Brawler/Gunslinger lines, TB2+
	{refs: []job.Id{job.Id(500), job.ThunderBreakerStage1Id}, mult: 18, off: -55},                                                             // Pirate base, TB1
}

func resolvePoolMin(rows []poolMinRow, defaultMult int, defaultOff int, jobId job.Id, level byte) int {
	for _, row := range rows {
		for _, ref := range row.refs {
			if job.Is(jobId, ref) {
				return row.mult*int(level) + row.off
			}
		}
	}
	return defaultMult*int(level) + defaultOff
}

func pointResetMinHp(jobId job.Id, level byte) int {
	return resolvePoolMin(pointResetMinHpRows, 12, 38, jobId, level) // default: Beginner/Noblesse
}

func pointResetMinMp(jobId job.Id, level byte) int {
	return resolvePoolMin(pointResetMinMpRows, 10, -5, jobId, level) // default: Beginner/Noblesse
}
