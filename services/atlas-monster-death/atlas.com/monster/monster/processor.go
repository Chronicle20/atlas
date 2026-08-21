package monster

import (
	"atlas-monster-death/character"
	_map "atlas-monster-death/map"
	"atlas-monster-death/monster/drop"
	"atlas-monster-death/monster/information"
	"atlas-monster-death/party"
	"atlas-monster-death/quest"
	"atlas-monster-death/rates"
	"atlas-monster-death/system_message"
	"context"
	"math"
	"math/rand"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type Processor interface {
	CreateDrops(f field.Model, id uint32, monsterId uint32, x int16, y int16, killerId uint32) error
	DistributeExperience(f field.Model, monsterId uint32, damageEntries []DamageEntryModel) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	cp  character.Processor
	pp  party.Processor
	rp  rates.Processor
	ip  information.Processor
	fp  _map.Processor
	smp system_message.Processor
	ht  *system_message.Throttle
	cfg ExperienceConfig
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		cp:  character.NewProcessor(l, ctx),
		pp:  party.NewProcessor(l, ctx),
		rp:  rates.NewProcessor(l, ctx),
		ip:  information.NewProcessor(l, ctx),
		fp:  _map.NewProcessor(l, ctx),
		smp: system_message.NewProcessor(l, ctx),
		ht:  system_message.GetHintThrottle(),
		cfg: LoadExperienceConfig(),
	}
}

var _ Processor = (*ProcessorImpl)(nil)

type ProcessorOption func(*ProcessorImpl)

func WithCharacterProcessor(cp character.Processor) ProcessorOption {
	return func(p *ProcessorImpl) {
		p.cp = cp
	}
}

func WithPartyProcessor(pp party.Processor) ProcessorOption {
	return func(p *ProcessorImpl) {
		p.pp = pp
	}
}

func WithRatesProcessor(rp rates.Processor) ProcessorOption {
	return func(p *ProcessorImpl) {
		p.rp = rp
	}
}

func WithInformationProcessor(ip information.Processor) ProcessorOption {
	return func(p *ProcessorImpl) {
		p.ip = ip
	}
}

func WithFieldProcessor(fp _map.Processor) ProcessorOption {
	return func(p *ProcessorImpl) {
		p.fp = fp
	}
}

func WithSystemMessageProcessor(smp system_message.Processor) ProcessorOption {
	return func(p *ProcessorImpl) {
		p.smp = smp
	}
}

func WithHintThrottle(ht *system_message.Throttle) ProcessorOption {
	return func(p *ProcessorImpl) {
		p.ht = ht
	}
}

func WithExperienceConfig(cfg ExperienceConfig) ProcessorOption {
	return func(p *ProcessorImpl) {
		p.cfg = cfg
	}
}

func (p *ProcessorImpl) With(opts ...ProcessorOption) Processor {
	clone := *p
	cp := &clone
	for _, opt := range opts {
		opt(cp)
	}
	return cp
}

func (p *ProcessorImpl) CreateDrops(f field.Model, id uint32, monsterId uint32, x int16, y int16, killerId uint32) error {
	// TODO determine type of drop
	dropType := byte(0)

	ds, err := drop.NewProcessor(p.l, p.ctx).GetByMonsterId(monsterId)
	if err != nil {
		return err
	}
	p.l.Debugf("Monster [%d] has [%d] drops to evaluate.", monsterId, len(ds))

	// Filter quest-specific drops
	ds = p.filterByQuestState(killerId, ds)
	p.l.Debugf("After quest filtering, [%d] drops remain.", len(ds))

	// Get rates for the killer
	r := p.rp.GetForCharacter(f.Channel(), killerId)
	p.l.Debugf("Character [%d] rates: itemDrop=%.2f, meso=%.2f", killerId, r.ItemDropRate(), r.MesoRate())

	ds = getSuccessfulDrops(ds, r.ItemDropRate())

	var ownerPartyId uint32
	pt, perr := p.pp.GetByMemberId(killerId)
	if perr == nil {
		ownerPartyId = pt.Id()
	}

	for i, d := range ds {
		_ = drop.NewProcessor(p.l, p.ctx).Create(f, i+1, id, x, y, killerId, dropType, d, r.MesoRate(), ownerPartyId)
	}
	return nil
}

func getSuccessfulDrops(options []drop.Model, itemDropRate float64) []drop.Model {
	res := make([]drop.Model, 0)
	for _, d := range options {
		if evaluateSuccess(d, itemDropRate) {
			res = append(res, d)
		}
	}
	return res
}

func evaluateSuccess(d drop.Model, itemDropRate float64) bool {
	// Apply item drop rate multiplier to base chance
	adjustedChance := float64(d.Chance()) * itemDropRate
	chance := int32(math.Min(adjustedChance, math.MaxInt32))
	return rand.Int31n(999999) < chance
}

func (p *ProcessorImpl) filterByQuestState(characterId uint32, drops []drop.Model) []drop.Model {
	// Check if any drops require quest filtering
	hasQuestDrops := false
	for _, d := range drops {
		if d.QuestId() != 0 {
			hasQuestDrops = true
			break
		}
	}

	// Skip quest lookup if no quest-specific drops
	if !hasQuestDrops {
		return drops
	}

	// Fetch started quest IDs for character
	startedQuests, err := quest.GetStartedQuestIds(p.l)(p.ctx)(characterId)
	if err != nil {
		p.l.WithError(err).Warnf("Unable to fetch started quests for character [%d], excluding all quest drops.", characterId)
		// On error, exclude all quest-specific drops for safety
		startedQuests = make(map[uint32]bool)
	}

	result := make([]drop.Model, 0, len(drops))
	for _, d := range drops {
		if d.QuestId() == 0 {
			// Non-quest item, always include
			result = append(result, d)
		} else if startedQuests[d.QuestId()] {
			// Quest item with started quest
			result = append(result, d)
		}
		// Quest item without started quest is excluded
	}
	return result
}

func (p *ProcessorImpl) DistributeExperience(f field.Model, monsterId uint32, damageEntries []DamageEntryModel) error {
	d, _ := p.produceDistribution(f, monsterId, damageEntries)()
	for k, v := range d.Solo() {
		exp := float64(v) * d.ExperiencePerDamage()
		c, err := p.cp.GetById(k)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to locate character %d whose for distributing experience from monster death.", k)
		} else {
			// Get rates for the character and apply exp rate
			r := p.rp.GetForCharacter(f.Channel(), c.Id())
			exp = exp * r.ExpRate()
			p.l.Debugf("Character [%d] exp rate: %.2f, adjusted exp: %.2f", c.Id(), r.ExpRate(), exp)

			whiteExperienceGain := isWhiteExperienceGain(c.Id(), d.PersonalRatio(), d.StandardDeviationRatio())
			p.distributeCharacterExperience(f, c.Id(), c.Level(), exp, 0.0, c.Level(), true, whiteExperienceGain, false)
		}
	}
	return nil
}

func (p *ProcessorImpl) produceDistribution(f field.Model, monsterId uint32, damageEntries []DamageEntryModel) model.Provider[DamageDistributionModel] {
	mi, err := p.ip.GetById(monsterId)
	if err != nil {
		return model.ErrorProvider[DamageDistributionModel](err)
	}

	fieldCharacterIds, err := p.fp.CharacterIdsInField(f)
	if err != nil {
		return model.ErrorProvider[DamageDistributionModel](err)
	}
	cim := make(map[uint32]bool)
	for _, m := range fieldCharacterIds {
		cim[m] = true
	}

	totalEntries := 0
	// TODO parties
	partyDistribution := make(map[uint32]map[uint32]uint32)
	soloDistribution := make(map[uint32]uint32)

	for _, de := range damageEntries {
		if _, ok := cim[de.characterId]; ok {
			soloDistribution[de.characterId] = de.damage
		}
		totalEntries += 1
	}

	// TODO account for healing
	totalDamage := mi.Hp()
	epd := float64(mi.Experience()) / float64(totalDamage)

	personalRatio := make(map[uint32]float64)
	entryExperienceRatio := make([]float64, 0)

	for k, v := range soloDistribution {
		ratio := float64(v) / float64(totalDamage)
		personalRatio[k] = ratio
		entryExperienceRatio = append(entryExperienceRatio, ratio)
	}

	for _, party := range partyDistribution {
		ratio := 0.0
		for k, v := range party {
			cr := float64(v) / float64(totalDamage)
			personalRatio[k] = cr
			ratio += cr
		}
		entryExperienceRatio = append(entryExperienceRatio, ratio)
	}

	stdr := calculateExperienceStandardDeviationThreshold(entryExperienceRatio, totalEntries)
	m := DamageDistributionModel{
		solo:                   soloDistribution,
		party:                  partyDistribution,
		personalRatio:          personalRatio,
		experiencePerDamage:    epd,
		standardDeviationRatio: stdr,
	}
	return model.FixedProvider(m)
}

func calculateExperienceStandardDeviationThreshold(entryExperienceRatio []float64, totalEntries int) float64 {
	averageExperienceReward := 0.0
	for _, v := range entryExperienceRatio {
		averageExperienceReward += v
	}
	averageExperienceReward /= float64(totalEntries)

	varExperienceReward := 0.0
	for _, v := range entryExperienceRatio {
		varExperienceReward += math.Pow(v-averageExperienceReward, 2)
	}
	varExperienceReward /= float64(len(entryExperienceRatio))

	return averageExperienceReward + math.Sqrt(varExperienceReward)
}

func isWhiteExperienceGain(characterId uint32, personalRatio map[uint32]float64, standardDeviationRatio float64) bool {
	if val, ok := personalRatio[characterId]; ok {
		return val >= standardDeviationRatio
	} else {
		return false
	}
}

func (p *ProcessorImpl) distributeCharacterExperience(f field.Model, characterId uint32, level byte, experience float64, partyBonusMod float64, totalPartyLevel byte, hightestPartyDamage bool, whiteExperienceGain bool, hasPartySharers bool) {
	expSplitCommonMod := 0.8
	characterExperience := (expSplitCommonMod * float64(level)) / float64(totalPartyLevel)
	if hightestPartyDamage {
		characterExperience += 0.2
	}
	characterExperience *= experience
	bonusExperience := partyBonusMod * characterExperience

	_ = p.cp.AwardExperience(f.Channel(), characterId, whiteExperienceGain, uint32(characterExperience), uint32(bonusExperience))
}
