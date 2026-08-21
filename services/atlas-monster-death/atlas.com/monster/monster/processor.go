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
	"sync"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
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

// DistributeExperience resolves, plans, and awards experience for a KILLED
// event: resolve the monster and field, resolve parties for in-field
// damagers, plan the split with planDistribution, then award and finally send
// level-gate hints. Hints run last so a publish failure there can never
// affect an EXP award (FR-6.10).
func (p *ProcessorImpl) DistributeExperience(f field.Model, monsterId uint32, damageEntries []DamageEntryModel) error {
	// 1. RESOLVE damage
	damages := aggregateDamageEntries(damageEntries)
	var totalDamage uint32
	for _, d := range damages {
		totalDamage += d.Damage
	}
	if totalDamage == 0 {
		p.l.Warnf("Monster [%d] died with no recorded damage. No experience distributed.", monsterId)
		return nil
	}

	// monster information and the field character list are independent;
	// issue them concurrently.
	var wg sync.WaitGroup
	var mi information.Model
	var miErr error
	var fieldCharacterIds []uint32
	var idsErr error

	wg.Add(2)
	go func() {
		defer wg.Done()
		mi, miErr = p.ip.GetById(monsterId)
	}()
	go func() {
		defer wg.Done()
		fieldCharacterIds, idsErr = p.fp.CharacterIdsInField(f)
	}()
	wg.Wait()

	if miErr != nil {
		p.l.WithError(miErr).Errorf("Unable to locate monster information [%d] for distributing experience from monster death.", monsterId)
		return miErr
	}
	if idsErr != nil {
		p.l.WithError(idsErr).Errorf("Unable to locate field characters for monster [%d] for distributing experience from monster death.", monsterId)
		return idsErr
	}

	inField := make(map[uint32]bool, len(fieldCharacterIds))
	for _, id := range fieldCharacterIds {
		inField[id] = true
	}

	// 2. RESOLVE parties for in-field damagers, memoising by partyId.
	parties := make(map[uint32]party.Model)
	partyOf := make(map[uint32]uint32)
	var solos []SoloInput

	for _, d := range damages {
		characterId := d.CharacterId
		if !inField[characterId] {
			// An out-of-field damager is never party-resolved (D12) -- it
			// closes the tag-and-walk-away leech vector and still counts
			// toward totalDamage and totalEntries via planDistribution.
			continue
		}
		if _, ok := partyOf[characterId]; ok {
			// Already accounted for by a previously fetched party.
			continue
		}

		pt, err := p.pp.GetByMemberId(characterId)
		if err != nil {
			// A party-service outage must degrade to today's solo
			// behaviour, never to zero EXP (FR-2.3).
			p.l.WithError(err).Warnf("Unable to locate party for character [%d]; treating as solo.", characterId)
			solos = append(solos, p.soloInputFor(characterId))
			continue
		}
		if pt.Id() == 0 {
			solos = append(solos, p.soloInputFor(characterId))
			continue
		}

		parties[pt.Id()] = pt
		for _, m := range pt.Members() {
			partyOf[m.Id()] = pt.Id()
		}
	}

	partyInputs := make([]PartyInput, 0, len(parties))
	for _, pt := range parties {
		members := make([]MemberInput, 0, len(pt.Members()))
		for _, m := range pt.Members() {
			if !inField[m.Id()] {
				continue
			}
			members = append(members, MemberInput{CharacterId: m.Id(), Level: m.Level()})
		}
		partyInputs = append(partyInputs, PartyInput{PartyId: pt.Id(), Members: members})
	}

	// 3. PLAN
	plan := planDistribution(ExperienceInput{
		MonsterExperience: mi.Experience(),
		MonsterLevel:      mi.Level(),
		Damages:           damages,
		Solos:             solos,
		Parties:           partyInputs,
	}, p.cfg)

	// 4. AWARD
	for _, r := range plan.Recipients {
		rate := p.rp.GetForCharacter(f.Channel(), r.CharacterId)
		personal, bonus, guarded := computeAward(r, rate.ExpRate(), p.cfg)
		if guarded {
			p.l.Warnf("Computed experience award for monster [%d] character [%d] was not representable and was guarded.", monsterId, r.CharacterId)
		}
		if err := p.cp.AwardExperience(f.Channel(), r.CharacterId, r.White, personal, bonus); err != nil {
			// One recipient's failure must not abort the others (FR-9.2).
			p.l.WithError(err).Warnf("Unable to award experience to character [%d] for monster [%d] death.", r.CharacterId, monsterId)
			continue
		}
	}

	// 5. HINTS, always last so a hint failure cannot affect EXP.
	t := tenant.MustFromContext(p.ctx)
	tenantId := t.Id()
	for _, e := range plan.Exclusions {
		if !p.ht.Allow(tenantId, e.CharacterId) {
			continue
		}
		if err := p.smp.ShowHint(uuid.New(), f.Channel(), e.CharacterId, levelGateHintText(mi.Name(), mi.Level()), 0, 0); err != nil {
			p.l.WithError(err).Warnf("Unable to publish level-gate hint to character [%d] for monster [%d] death.", e.CharacterId, monsterId)
		}
	}

	return nil
}

// soloInputFor fetches a character's level for solo attribution. On error
// this matches today's behaviour: log and skip only that character, never
// aborting the rest of the distribution.
func (p *ProcessorImpl) soloInputFor(characterId uint32) SoloInput {
	c, err := p.cp.GetById(characterId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to locate character [%d] for distributing experience from monster death.", characterId)
		return SoloInput{CharacterId: characterId, Level: 0}
	}
	return SoloInput{CharacterId: characterId, Level: c.Level()}
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
