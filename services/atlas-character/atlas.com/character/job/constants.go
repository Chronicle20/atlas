package job

//goland:noinspection GoUnusedConst
const (
	Beginner uint16 = 0

	Warrior uint16 = 100

	Fighter  uint16 = 110
	Crusader uint16 = 111
	Hero     uint16 = 112

	Page        uint16 = 120
	WhiteKnight uint16 = 121
	Paladin     uint16 = 122

	Spearman     uint16 = 130
	DragonKnight uint16 = 131
	DarkKnight   uint16 = 132

	Magician uint16 = 200

	FirePoisonWizard       uint16 = 210
	FirePoisonMagician     uint16 = 211
	FirePoisonArchMagician uint16 = 212

	IceLighteningWizard       uint16 = 220
	IceLighteningMagician     uint16 = 221
	IceLighteningArchMagician uint16 = 222

	Cleric uint16 = 230
	Priest uint16 = 231
	Bishop uint16 = 232

	Bowman uint16 = 300

	Hunter    uint16 = 310
	Ranger    uint16 = 311
	BowMaster uint16 = 312

	CrossBowman uint16 = 320
	Sniper      uint16 = 321
	Marksman    uint16 = 322

	Thief uint16 = 400

	Assassin  uint16 = 410
	Hermit    uint16 = 411
	NightLord uint16 = 412

	Bandit      uint16 = 420
	ChiefBandit uint16 = 421
	Shadower    uint16 = 422

	Pirate uint16 = 500

	Brawler   uint16 = 510
	Marauder  uint16 = 511
	Buccaneer uint16 = 512

	Gunslinger uint16 = 520
	Outlaw     uint16 = 521
	Corsair    uint16 = 522

	MapleLeafBrigadier uint16 = 800

	GM uint16 = 900

	SUPERGM uint16 = 910

	Noblesse uint16 = 1000

	DawnWarrior1 uint16 = 1100
	DawnWarrior2 uint16 = 1110
	DawnWarrior3 uint16 = 1111
	DawnWarrior4 uint16 = 1112

	BlazeWizard1 uint16 = 1200
	BlazeWizard2 uint16 = 1210
	BlazeWizard3 uint16 = 1211
	BlazeWizard4 uint16 = 1212

	WindArcher1 uint16 = 1300
	WindArcher2 uint16 = 1310
	WindArcher3 uint16 = 1311
	WindArcher4 uint16 = 1312

	NightWalker1 uint16 = 1400
	NightWalker2 uint16 = 1410
	NightWalker3 uint16 = 1411
	NightWalker4 uint16 = 1412

	ThunderBreaker1 uint16 = 1500
	ThunderBreaker2 uint16 = 1510
	ThunderBreaker3 uint16 = 1511
	ThunderBreaker4 uint16 = 1512

	Legend uint16 = 2000

	Aran1 uint16 = 2100
	Aran2 uint16 = 2110
	Aran3 uint16 = 2111
	Aran4 uint16 = 2112

	Evan uint16 = 2001

	Evan1  uint16 = 2200
	Evan2  uint16 = 2210
	Evan3  uint16 = 2211
	Evan4  uint16 = 2212
	Evan5  uint16 = 2213
	Evan6  uint16 = 2214
	Evan7  uint16 = 2215
	Evan8  uint16 = 2216
	Evan9  uint16 = 2217
	Evan10 uint16 = 2218
)

func IsA(characterJobId uint16, referenceJobId ...uint16) bool {
	is := false
	for _, jobId := range referenceJobId {
		if isA(characterJobId, jobId) {
			is = true
		}
	}
	return is
}

func isA(characterJobId uint16, referenceJobId uint16) bool {
	characterBranch := characterJobId / 10
	referenceBranch := referenceJobId / 10
	return characterBranch == referenceBranch && characterJobId >= referenceJobId || referenceBranch%10 == 0 && characterJobId/100 == referenceJobId/100
}

func GetJobStyle(jobId uint16, strength uint16, dexterity uint16) uint16 {
	if strength > dexterity {
		return getJobStyleInternal(jobId, 0x80)
	} else {
		return getJobStyleInternal(jobId, 0x40)
	}
}

func getJobStyleInternal(jobId uint16, opt byte) uint16 {
	if IsA(jobId, Warrior, DawnWarrior1, Aran1) {
		return Warrior
	} else if IsA(jobId, Magician, BlazeWizard1, Evan1) {
		return Magician
	} else if isA(jobId, CrossBowman) {
		return CrossBowman
	} else if IsA(jobId, Bowman, WindArcher1) {
		return Bowman
	} else if IsA(jobId, Thief, NightWalker1) {
		return Thief
	} else if IsA(jobId, Pirate, ThunderBreaker1) {
		if opt == 0x80 {
			return Brawler
		} else {
			return Gunslinger
		}
	}
	return Beginner
}

func GetJobFromIndex(jobIndex uint32) (uint16, bool) {
	if jobIndex == 0 {
		return Noblesse, true
	} else if jobIndex == 1 {
		return Beginner, true
	} else if jobIndex == 2 {
		return Legend, true
	}
	return 0, false
}
