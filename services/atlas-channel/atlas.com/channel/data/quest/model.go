package quest

type Model struct {
	id                uint32
	name              string
	parent            string
	area              uint32
	order             uint32
	autoStart         bool
	autoPreComplete   bool
	autoComplete      bool
	timeLimit         uint32
	timeLimit2        uint32
	selectedMob       bool
	summary           string
	demandSummary     string
	rewardSummary     string
	startRequirements RequirementsModel
	endRequirements   RequirementsModel
	startActions      ActionsModel
	endActions        ActionsModel
}

func (m Model) Id() uint32                          { return m.id }
func (m Model) Name() string                        { return m.name }
func (m Model) Parent() string                      { return m.parent }
func (m Model) Area() uint32                        { return m.area }
func (m Model) Order() uint32                       { return m.order }
func (m Model) AutoStart() bool                     { return m.autoStart }
func (m Model) AutoPreComplete() bool               { return m.autoPreComplete }
func (m Model) AutoComplete() bool                  { return m.autoComplete }
func (m Model) TimeLimit() uint32                   { return m.timeLimit }
func (m Model) TimeLimit2() uint32                  { return m.timeLimit2 }
func (m Model) SelectedMob() bool                   { return m.selectedMob }
func (m Model) Summary() string                     { return m.summary }
func (m Model) DemandSummary() string               { return m.demandSummary }
func (m Model) RewardSummary() string               { return m.rewardSummary }
func (m Model) StartRequirements() RequirementsModel { return m.startRequirements }
func (m Model) EndRequirements() RequirementsModel   { return m.endRequirements }
func (m Model) StartActions() ActionsModel           { return m.startActions }
func (m Model) EndActions() ActionsModel             { return m.endActions }

type RequirementsModel struct {
	npcId           uint32
	levelMin        uint16
	levelMax        uint16
	fameMin         int16
	mesoMin         uint32
	mesoMax         uint32
	jobs            []uint16
	quests          []QuestRequirementModel
	items           []ItemRequirementModel
	mobs            []MobRequirementModel
	fieldEnter      []uint32
	pet             []uint32
	petTamenessMin  int16
	dayOfWeek       []string
	start           string
	end             string
	interval        uint32
	startScript     string
	endScript       string
	infoNumber      uint32
	normalAutoStart bool
	completionCount uint32
}

func (m RequirementsModel) NpcId() uint32                      { return m.npcId }
func (m RequirementsModel) LevelMin() uint16                   { return m.levelMin }
func (m RequirementsModel) LevelMax() uint16                   { return m.levelMax }
func (m RequirementsModel) FameMin() int16                     { return m.fameMin }
func (m RequirementsModel) MesoMin() uint32                    { return m.mesoMin }
func (m RequirementsModel) MesoMax() uint32                    { return m.mesoMax }
func (m RequirementsModel) Jobs() []uint16                     { return m.jobs }
func (m RequirementsModel) Quests() []QuestRequirementModel    { return m.quests }
func (m RequirementsModel) Items() []ItemRequirementModel      { return m.items }
func (m RequirementsModel) Mobs() []MobRequirementModel        { return m.mobs }
func (m RequirementsModel) FieldEnter() []uint32               { return m.fieldEnter }
func (m RequirementsModel) Pet() []uint32                      { return m.pet }
func (m RequirementsModel) PetTamenessMin() int16              { return m.petTamenessMin }
func (m RequirementsModel) DayOfWeek() []string                { return m.dayOfWeek }
func (m RequirementsModel) Start() string                      { return m.start }
func (m RequirementsModel) End() string                        { return m.end }
func (m RequirementsModel) Interval() uint32                   { return m.interval }
func (m RequirementsModel) StartScript() string                { return m.startScript }
func (m RequirementsModel) EndScript() string                  { return m.endScript }
func (m RequirementsModel) InfoNumber() uint32                 { return m.infoNumber }
func (m RequirementsModel) NormalAutoStart() bool              { return m.normalAutoStart }
func (m RequirementsModel) CompletionCount() uint32            { return m.completionCount }

type QuestRequirementModel struct {
	id    uint32
	state uint8
}

func (m QuestRequirementModel) Id() uint32   { return m.id }
func (m QuestRequirementModel) State() uint8 { return m.state }

type ItemRequirementModel struct {
	id    uint32
	count int32
}

func (m ItemRequirementModel) Id() uint32   { return m.id }
func (m ItemRequirementModel) Count() int32 { return m.count }

type MobRequirementModel struct {
	id    uint32
	count uint32
}

func (m MobRequirementModel) Id() uint32    { return m.id }
func (m MobRequirementModel) Count() uint32 { return m.count }

type ActionsModel struct {
	npcId      uint32
	exp        int32
	money      int32
	fame       int16
	items      []ItemRewardModel
	skills     []SkillRewardModel
	nextQuest  uint32
	buffItemId uint32
	interval   uint32
	levelMin   uint16
}

func (m ActionsModel) NpcId() uint32              { return m.npcId }
func (m ActionsModel) Exp() int32                 { return m.exp }
func (m ActionsModel) Money() int32               { return m.money }
func (m ActionsModel) Fame() int16                { return m.fame }
func (m ActionsModel) Items() []ItemRewardModel   { return m.items }
func (m ActionsModel) Skills() []SkillRewardModel { return m.skills }
func (m ActionsModel) NextQuest() uint32          { return m.nextQuest }
func (m ActionsModel) BuffItemId() uint32         { return m.buffItemId }
func (m ActionsModel) Interval() uint32           { return m.interval }
func (m ActionsModel) LevelMin() uint16           { return m.levelMin }

type ItemRewardModel struct {
	id         uint32
	count      int32
	job        int32
	gender     int8
	prop       int32
	period     uint32
	dateExpire string
	variable   uint32
}

func (m ItemRewardModel) Id() uint32         { return m.id }
func (m ItemRewardModel) Count() int32       { return m.count }
func (m ItemRewardModel) Job() int32         { return m.job }
func (m ItemRewardModel) Gender() int8       { return m.gender }
func (m ItemRewardModel) Prop() int32        { return m.prop }
func (m ItemRewardModel) Period() uint32     { return m.period }
func (m ItemRewardModel) DateExpire() string { return m.dateExpire }
func (m ItemRewardModel) Variable() uint32   { return m.variable }

type SkillRewardModel struct {
	id          uint32
	level       int32
	masterLevel uint32
	jobs        []uint16
}

func (m SkillRewardModel) Id() uint32          { return m.id }
func (m SkillRewardModel) Level() int32        { return m.level }
func (m SkillRewardModel) MasterLevel() uint32 { return m.masterLevel }
func (m SkillRewardModel) Jobs() []uint16      { return m.jobs }
