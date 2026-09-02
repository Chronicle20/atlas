package job

type Id uint16

// Jobs is the registry of every job id the server knows about. It carries no
// skill data: the job→skill mapping is version-varying and now lives in
// per-tenant JOB documents served by atlas-data (task-185). The key set is the
// "does this job exist" check used by atlas-configurations' preset validators.
var Jobs = map[Id]Job{
	BeginnerId:                 {id: BeginnerId},
	WarriorId:                  {id: WarriorId},
	FighterId:                  {id: FighterId},
	CrusaderId:                 {id: CrusaderId},
	HeroId:                     {id: HeroId, fourthJob: true},
	PageId:                     {id: PageId},
	WhiteKnightId:              {id: WhiteKnightId},
	PaladinId:                  {id: PaladinId, fourthJob: true},
	SpearmanId:                 {id: SpearmanId},
	DragonKnightId:             {id: DragonKnightId},
	DarkKnightId:               {id: DarkKnightId, fourthJob: true},
	MagicianId:                 {id: MagicianId},
	FirePoisonWizardId:         {id: FirePoisonWizardId},
	FirePoisonMagicianId:       {id: FirePoisonMagicianId},
	FirePoisonArchMagicianId:   {id: FirePoisonArchMagicianId, fourthJob: true},
	IceLightningWizardId:       {id: IceLightningWizardId},
	IceLightningMagicianId:     {id: IceLightningMagicianId},
	IceLightningArchMagicianId: {id: IceLightningArchMagicianId, fourthJob: true},
	ClericId:                   {id: ClericId},
	PriestId:                   {id: PriestId},
	BishopId:                   {id: BishopId, fourthJob: true},
	BowmanId:                   {id: BowmanId},
	HunterId:                   {id: HunterId},
	RangerId:                   {id: RangerId},
	BowmasterId:                {id: BowmasterId, fourthJob: true},
	CrossbowmanId:              {id: CrossbowmanId},
	SniperId:                   {id: SniperId},
	MarksmanId:                 {id: MarksmanId, fourthJob: true},
	RogueId:                    {id: RogueId},
	AssassinId:                 {id: AssassinId},
	HermitId:                   {id: HermitId},
	NightLordId:                {id: NightLordId, fourthJob: true},
	BanditId:                   {id: BanditId},
	ChiefBanditId:              {id: ChiefBanditId},
	ShadowerId:                 {id: ShadowerId, fourthJob: true},
	PirateId:                   {id: PirateId},
	BrawlerId:                  {id: BrawlerId},
	MarauderId:                 {id: MarauderId},
	BuccaneerId:                {id: BuccaneerId, fourthJob: true},
	GunslingerId:               {id: GunslingerId},
	OutlawId:                   {id: OutlawId},
	CorsairId:                  {id: CorsairId, fourthJob: true},
	MapleLeafBrigadierId:       {id: MapleLeafBrigadierId},
	GmId:                       {id: GmId},
	SuperGmId:                  {id: SuperGmId},
	NoblesseId:                 {id: NoblesseId},
	DawnWarriorStage1Id:        {id: DawnWarriorStage1Id},
	DawnWarriorStage2Id:        {id: DawnWarriorStage2Id},
	DawnWarriorStage3Id:        {id: DawnWarriorStage3Id},
	DawnWarriorStage4Id:        {id: DawnWarriorStage4Id, fourthJob: true},
	BlazeWizardStage1Id:        {id: BlazeWizardStage1Id},
	BlazeWizardStage2Id:        {id: BlazeWizardStage2Id},
	BlazeWizardStage3Id:        {id: BlazeWizardStage3Id},
	BlazeWizardStage4Id:        {id: BlazeWizardStage4Id, fourthJob: true},
	WindArcherStage1Id:         {id: WindArcherStage1Id},
	WindArcherStage2Id:         {id: WindArcherStage2Id},
	WindArcherStage3Id:         {id: WindArcherStage3Id},
	WindArcherStage4Id:         {id: WindArcherStage4Id, fourthJob: true},
	NightWalkerStage1Id:        {id: NightWalkerStage1Id},
	NightWalkerStage2Id:        {id: NightWalkerStage2Id},
	NightWalkerStage3Id:        {id: NightWalkerStage3Id},
	NightWalkerStage4Id:        {id: NightWalkerStage4Id, fourthJob: true},
	ThunderBreakerStage1Id:     {id: ThunderBreakerStage1Id},
	ThunderBreakerStage2Id:     {id: ThunderBreakerStage2Id},
	ThunderBreakerStage3Id:     {id: ThunderBreakerStage3Id},
	ThunderBreakerStage4Id:     {id: ThunderBreakerStage4Id, fourthJob: true},
	LegendId:                   {id: LegendId},
	AranStage1Id:               {id: AranStage1Id},
	AranStage2Id:               {id: AranStage2Id},
	AranStage3Id:               {id: AranStage3Id},
	AranStage4Id:               {id: AranStage4Id, fourthJob: true},
	EvanId:                     {id: EvanId},
	EvanStage1Id:               {id: EvanStage1Id},
	EvanStage2Id:               {id: EvanStage2Id},
	EvanStage3Id:               {id: EvanStage3Id},
	EvanStage4Id:               {id: EvanStage4Id},
	EvanStage5Id:               {id: EvanStage5Id},
	EvanStage6Id:               {id: EvanStage6Id, fourthJob: true},
	EvanStage7Id:               {id: EvanStage7Id, fourthJob: true},
	EvanStage8Id:               {id: EvanStage8Id, fourthJob: true},
	EvanStage9Id:               {id: EvanStage9Id, fourthJob: true},
	EvanStage10Id:              {id: EvanStage10Id, fourthJob: true},
	CitizenId:                  {id: CitizenId},
}

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
	// CitizenId is the Resistance/Citizen beginner job. It is human-supplied,
	// not read from any client binary: the character-creation packet never
	// carries a job id (findings.md, "New job constants required", task-283
	// execution session, 2026-08-28). See
	// docs/tasks/task-283-race-index-job-mapping/findings.md.
	CitizenId = Id(3000)
)

type Type uint16

const (
	TypeExplorer = Type(0)
	TypeCygnus   = Type(1)
	TypeLegend   = Type(2)
)
