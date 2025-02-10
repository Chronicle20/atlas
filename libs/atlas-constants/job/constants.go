package job

type Id uint16

const (
	BeginnerId                 = Id(0)
	WarriorId                  = Id(100)
	FighterId                  = Id(110)
	CrusaderId                 = Id(111)
	HeroId                     = Id(112)
	PageId                     = Id(120)
	WhiteKnightId              = Id(121)
	PaladinId                  = Id(122)
	SpearmanId                 = Id(130)
	DragonKnightId             = Id(131)
	DarkKnightId               = Id(132)
	MagicianId                 = Id(200)
	FirePoisonWizardId         = Id(210)
	FirePoisonMagicianId       = Id(211)
	FirePoisonArchMagicianId   = Id(212)
	IceLightningWizardId       = Id(220)
	IceLightningMagicianId     = Id(221)
	IceLightningArchMagicianId = Id(222)
	ClericId                   = Id(230)
	PriestId                   = Id(231)
	BishopId                   = Id(232)
	BowmanId                   = Id(300)
	HunterId                   = Id(310)
	RangerId                   = Id(311)
	BowmasterId                = Id(312)
	CrossbowmanId              = Id(320)
	SniperId                   = Id(321)
	MarksmanId                 = Id(322)
	RogueId                    = Id(400)
	AssassinId                 = Id(410)
	HermitId                   = Id(411)
	NightLordId                = Id(412)
	BanditId                   = Id(420)
	ChiefBanditId              = Id(421)
	ShadowerId                 = Id(422)
	PirateId                   = Id(500)
	BrawlerId                  = Id(510)
	MarauderId                 = Id(511)
	BuccaneerId                = Id(512)
	GunslingerId               = Id(520)
	OutlawId                   = Id(521)
	CorsairId                  = Id(522)
	MapleLeafBrigadierId       = Id(800)
	GmId                       = Id(900)
	SuperGmId                  = Id(910)
	NoblesseId                 = Id(1000)
	DawnWarriorStage1Id        = Id(1100)
	DawnWarriorStage2Id        = Id(1110)
	DawnWarriorStage3Id        = Id(1111)
	DawnWarriorStage4Id        = Id(1112)
	BlazeWizardStage1Id        = Id(1200)
	BlazeWizardStage2Id        = Id(1210)
	BlazeWizardStage3Id        = Id(1211)
	BlazeWizardStage4Id        = Id(1212)
	WindArcherStage1Id         = Id(1300)
	WindArcherStage2Id         = Id(1310)
	WindArcherStage3Id         = Id(1311)
	WindArcherStage4Id         = Id(1312)
	NightWalkerStage1Id        = Id(1400)
	NightWalkerStage2Id        = Id(1410)
	NightWalkerStage3Id        = Id(1411)
	NightWalkerStage4Id        = Id(1412)
	ThunderBreakerStage1Id     = Id(1500)
	ThunderBreakerStage2Id     = Id(1510)
	ThunderBreakerStage3Id     = Id(1511)
	ThunderBreakerStage4Id     = Id(1512)
	LegendId                   = Id(2000)
	AranStage1Id               = Id(2100)
	AranStage2Id               = Id(2110)
	AranStage3Id               = Id(2111)
	AranStage4Id               = Id(2112)
	EvanId                     = Id(2001)
	EvanStage1Id               = Id(2200)
	EvanStage2Id               = Id(2210)
	EvanStage3Id               = Id(2211)
	EvanStage4Id               = Id(2212)
	EvanStage5Id               = Id(2213)
	EvanStage6Id               = Id(2214)
	EvanStage7Id               = Id(2215)
	EvanStage8Id               = Id(2216)
	EvanStage9Id               = Id(2217)
	EvanStage10Id              = Id(2218)
)

func IsA(jobId Id, referenceJobId ...Id) bool {
	is := false
	for _, jobId := range referenceJobId {
		if Is(jobId, jobId) {
			is = true
		}
	}
	return is
}

func Is(jobId Id, referenceJobId Id) bool {
	characterBranch := jobId / 10
	referenceBranch := referenceJobId / 10
	return characterBranch == referenceBranch && jobId >= referenceJobId || referenceBranch%10 == 0 && jobId/100 == referenceJobId/100
}

func IsFourthJob(jobId Id) bool {
	// TODO Cygnus, Aran, or Evan
	return IsA(jobId, HeroId, PaladinId, DarkKnightId, FirePoisonArchMagicianId, IceLightningArchMagicianId, BishopId, BowmasterId, MarksmanId, NightLordId, ChiefBanditId, BuccaneerId, CorsairId)
}
