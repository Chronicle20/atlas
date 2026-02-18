package monster

import (
	"atlas-monsters/kafka/producer"
	_map "atlas-monsters/map"
	"atlas-monsters/monster/information"
	"atlas-monsters/monster/mobskill"
	"context"
	"errors"
	"math/rand"
	"time"

	"github.com/Chronicle20/atlas-constants/field"
	map2 "github.com/Chronicle20/atlas-constants/map"
	monster2 "github.com/Chronicle20/atlas-constants/monster"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
)

// Processor defines the interface for monster processing operations
type Processor interface {
	// Providers
	ByIdProvider(monsterId uint32) model.Provider[Model]
	ByFieldProvider(f field.Model) model.Provider[[]Model]
	ControlledInFieldProvider(f field.Model) model.Provider[[]Model]
	NotControlledInFieldProvider(f field.Model) model.Provider[[]Model]
	ControlledByCharacterInFieldProvider(f field.Model, characterId uint32) model.Provider[[]Model]

	// Queries
	GetById(monsterId uint32) (Model, error)
	GetInField(f field.Model) ([]Model, error)

	// Commands
	Create(f field.Model, input RestModel) (Model, error)
	StartControl(uniqueId uint32, controllerId uint32) (Model, error)
	StopControl(m Model) error
	FindNextController(idp model.Provider[[]uint32]) model.Operator[Model]
	Damage(id uint32, characterId uint32, damage uint32, attackType byte)
	DamageFriendly(uniqueId uint32, attackerUniqueId uint32, observerUniqueId uint32)
	Move(id uint32, x int16, y int16, stance byte) error
	Destroy(uniqueId uint32) error
	DestroyInField(f field.Model) error
	UseSkill(uniqueId uint32, characterId uint32, skillId uint16, skillLevel uint16)
	UseSkillGM(uniqueId uint32, skillId uint16, skillLevel uint16)
	ApplyStatusEffect(uniqueId uint32, effect StatusEffect) error
	CancelStatusEffect(uniqueId uint32, statusTypes []string) error
	CancelAllStatusEffects(uniqueId uint32) error
}

// ProcessorImpl implements the Processor interface
type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	t   tenant.Model
}

// NewProcessor creates a new Processor
func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		t:   tenant.MustFromContext(ctx),
	}
}

// ByIdProvider returns a provider for a monster by ID
func (p *ProcessorImpl) ByIdProvider(monsterId uint32) model.Provider[Model] {
	return func() (Model, error) {
		return GetMonsterRegistry().GetMonster(p.t, monsterId)
	}
}

// ByFieldProvider returns a provider for monsters in a field
func (p *ProcessorImpl) ByFieldProvider(f field.Model) model.Provider[[]Model] {
	return func() ([]Model, error) {
		return GetMonsterRegistry().GetMonstersInMap(p.t, f), nil
	}
}

// ControlledInFieldProvider returns a provider for controlled monsters in a field
func (p *ProcessorImpl) ControlledInFieldProvider(f field.Model) model.Provider[[]Model] {
	return model.FilteredProvider(p.ByFieldProvider(f), model.Filters(Controlled))
}

// NotControlledInFieldProvider returns a provider for uncontrolled monsters in a field
func (p *ProcessorImpl) NotControlledInFieldProvider(f field.Model) model.Provider[[]Model] {
	return model.FilteredProvider(p.ByFieldProvider(f), model.Filters(NotControlled))
}

// ControlledByCharacterInFieldProvider returns a provider for monsters controlled by a specific character
func (p *ProcessorImpl) ControlledByCharacterInFieldProvider(f field.Model, characterId uint32) model.Provider[[]Model] {
	return model.FilteredProvider(p.ByFieldProvider(f), model.Filters(IsControlledBy(characterId)))
}

// GetById gets a monster by ID
func (p *ProcessorImpl) GetById(monsterId uint32) (Model, error) {
	return p.ByIdProvider(monsterId)()
}

// GetInField gets all monsters in a field
func (p *ProcessorImpl) GetInField(f field.Model) ([]Model, error) {
	return p.ByFieldProvider(f)()
}

// Create creates a new monster in a field
func (p *ProcessorImpl) Create(f field.Model, input RestModel) (Model, error) {
	p.l.Debugf("Attempting to create monster [%d] in field [%s].", input.MonsterId, f.Id())
	ma, err := information.GetById(p.l)(p.ctx)(input.MonsterId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to retrieve information necessary to create monster [%d].", input.MonsterId)
		return Model{}, err
	}

	m := GetMonsterRegistry().CreateMonster(p.t, f, input.MonsterId, input.X, input.Y, input.Fh, 5, input.Team, ma.Hp(), ma.Mp())

	cid, err := p.getControllerCandidate(f, _map.CharacterIdsInFieldProvider(p.l)(p.ctx)(f))
	if err == nil {
		p.l.Debugf("Created monster [%d] with id [%d] will be controlled by [%d].", m.MonsterId(), m.UniqueId(), cid)
		m, err = p.StartControl(m.UniqueId(), cid)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to start [%d] controlling [%d] in field [%s].", cid, m.UniqueId(), m.Field().Id())
		}
	}

	if ma.Friendly() && ma.DropPeriod() > 0 {
		interval := time.Duration(ma.DropPeriod()/3) * time.Millisecond
		p.l.Debugf("Registering friendly monster [%d] (template [%d]) with drop period [%s].", m.UniqueId(), m.MonsterId(), interval)
		now := time.Now()
		GetDropTimerRegistry().Register(p.t, m.UniqueId(), DropTimerEntry{
			monsterId:    m.MonsterId(),
			field:        f,
			dropPeriod:   interval,
			weaponAttack: ma.WeaponAttack(),
			maxHp:        ma.Hp(),
			lastDropAt:   now,
			lastHitAt:    time.Time{},
		})
	}

	p.l.Debugf("Created monster [%d] in field [%s]. Emitting Monster Status.", input.MonsterId, f.Id())
	_ = producer.ProviderImpl(p.l)(p.ctx)(EnvEventTopicMonsterStatus)(createdStatusEventProvider(m))
	return m, nil
}

// getControllerCandidate finds the best character to control monsters in a field
func (p *ProcessorImpl) getControllerCandidate(f field.Model, idp model.Provider[[]uint32]) (uint32, error) {
	p.l.Debugf("Identifying controller candidate for monsters in field [%s].", f.Id())

	controlCounts, err := model.CollectToMap(idp, characterIdKey, zeroValue)()
	if err != nil {
		p.l.WithError(err).Errorf("Unable to initialize controller candidate map.")
		return 0, err
	}
	err = model.ForEachSlice(p.ControlledInFieldProvider(f), func(m Model) error {
		controlCounts[m.ControlCharacterId()] += 1
		return nil
	})

	var index = uint32(0)
	for key, val := range controlCounts {
		if index == 0 {
			index = key
		} else if val < controlCounts[index] {
			index = key
		}
	}

	if index == 0 {
		return 0, errors.New("should not get here")
	}
	p.l.Debugf("Controller candidate has been determined. Character [%d].", index)
	return index, nil
}

// FindNextController returns an operator that finds and assigns the next controller for a monster
func (p *ProcessorImpl) FindNextController(idp model.Provider[[]uint32]) model.Operator[Model] {
	return func(m Model) error {
		cid, err := p.getControllerCandidate(m.Field(), idp)
		if err != nil {
			return err
		}

		_, err = p.StartControl(m.UniqueId(), cid)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to start [%d] controlling [%d] in field [%s].", cid, m.UniqueId(), m.Field().Id())
		}
		return err
	}
}

// StartControl starts a character controlling a monster
func (p *ProcessorImpl) StartControl(uniqueId uint32, controllerId uint32) (Model, error) {
	m, err := p.GetById(uniqueId)
	if err != nil {
		return Model{}, err
	}

	if m.ControlCharacterId() != 0 {
		err = p.StopControl(m)
		if err != nil {
			return Model{}, err
		}
	}

	m, err = p.GetById(uniqueId)
	if err != nil {
		return Model{}, err
	}

	m, err = GetMonsterRegistry().ControlMonster(p.t, m.UniqueId(), controllerId)
	if err == nil {
		_ = producer.ProviderImpl(p.l)(p.ctx)(EnvEventTopicMonsterStatus)(startControlStatusEventProvider(m))
	}
	return m, err
}

// StopControl stops a character from controlling a monster
func (p *ProcessorImpl) StopControl(m Model) error {
	oldControllerId := m.ControlCharacterId()
	m, err := GetMonsterRegistry().ClearControl(p.t, m.UniqueId())
	if err == nil {
		_ = producer.ProviderImpl(p.l)(p.ctx)(EnvEventTopicMonsterStatus)(stopControlStatusEventProvider(m, oldControllerId))
	}
	return err
}

// Damage applies damage to a monster
func (p *ProcessorImpl) Damage(id uint32, characterId uint32, damage uint32, attackType byte) {
	m, err := GetMonsterRegistry().GetMonster(p.t, id)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to get monster [%d].", id)
		return
	}
	if !m.Alive() {
		p.l.Debugf("Character [%d] trying to apply damage to an already dead monster [%d].", characterId, id)
		return
	}

	// Check for damage reflection
	p.checkReflect(m, characterId, attackType)

	// Fetch monster info for boss flag and revives
	var isBoss bool
	var revives []uint32
	ma, infoErr := information.GetById(p.l)(p.ctx)(m.MonsterId())
	if infoErr == nil {
		isBoss = ma.Boss()
		revives = ma.Revives()
	}

	s, err := GetMonsterRegistry().ApplyDamage(p.t, characterId, damage, m.UniqueId())
	if err != nil {
		p.l.WithError(err).Errorf("Error applying damage to monster %d from character %d.", m.UniqueId(), characterId)
		return
	}

	if s.Killed {
		// Clear cooldowns and drop timer on death
		GetCooldownRegistry().ClearCooldowns(id)
		GetDropTimerRegistry().Unregister(p.t, id)

		// Emit cancellation events for any active status effects before death
		for _, se := range s.Monster.StatusEffects() {
			_ = producer.ProviderImpl(p.l)(p.ctx)(EnvEventTopicMonsterStatus)(statusEffectCancelledEventProvider(s.Monster, se))
		}

		err = producer.ProviderImpl(p.l)(p.ctx)(EnvEventTopicMonsterStatus)(killedStatusEventProvider(s.Monster, s.CharacterId, isBoss, s.Monster.DamageSummary()))
		if err != nil {
			p.l.WithError(err).Errorf("Monster [%d] killed, but unable to display that for the characters in the field.", s.Monster.UniqueId())
		}
		_, err = GetMonsterRegistry().RemoveMonster(p.t, s.Monster.UniqueId())
		if err != nil {
			p.l.WithError(err).Errorf("Monster [%d] killed, but not removed from registry.", s.Monster.UniqueId())
		}

		// Boss revive: spawn next phase monsters
		if len(revives) > 0 {
			p.spawnRevives(s.Monster, revives)
		}
		return
	}

	if characterId != s.Monster.ControlCharacterId() {
		dl := s.Monster.DamageLeader() == characterId
		p.l.Debugf("Character [%d] has become damage leader. They should now control the monster.", characterId)
		if dl {
			m, err := p.GetById(s.Monster.UniqueId())
			if err != nil {
				return
			}

			err = p.StopControl(m)
			if err != nil {
				p.l.WithError(err).Errorf("Unable to stop [%d] from controlling monster [%d].", s.Monster.ControlCharacterId(), s.Monster.UniqueId())
			}
			_, err = p.StartControl(m.UniqueId(), characterId)
			if err != nil {
				p.l.WithError(err).Errorf("Unable to start [%d] controlling monster [%d].", characterId, m.UniqueId())
			}
		}
	}

	err = producer.ProviderImpl(p.l)(p.ctx)(EnvEventTopicMonsterStatus)(damagedStatusEventProvider(s.Monster, s.CharacterId, s.CharacterId, isBoss, s.Monster.DamageSummary()))
	if err != nil {
		p.l.WithError(err).Errorf("Monster [%d] damaged, but unable to display that for the characters in the field.", s.Monster.UniqueId())
	}
}

// DamageFriendly applies damage from a hostile monster to a friendly monster and resets the drop timer.
func (p *ProcessorImpl) DamageFriendly(uniqueId uint32, attackerUniqueId uint32, observerUniqueId uint32) {
	m, err := GetMonsterRegistry().GetMonster(p.t, uniqueId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to get friendly monster [%d].", uniqueId)
		return
	}
	if !m.Alive() {
		return
	}

	// Look up attacker info to calculate damage
	attacker, err := GetMonsterRegistry().GetMonster(p.t, attackerUniqueId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to get attacking monster [%d].", attackerUniqueId)
		return
	}

	ma, err := information.GetById(p.l)(p.ctx)(attacker.MonsterId())
	if err != nil {
		p.l.WithError(err).Errorf("Unable to get information for attacking monster [%d].", attacker.MonsterId())
		return
	}

	// Damage formula: rand.Intn(((maxHp/13 + PADamage*10))*2 + 500) / 10
	base := int(ma.Hp()/13+ma.WeaponAttack()*10)*2 + 500
	damage := uint32(rand.Intn(base) / 10)
	if damage == 0 {
		damage = 1
	}

	now := time.Now()
	GetDropTimerRegistry().RecordHit(p.t, uniqueId, now)

	s, err := GetMonsterRegistry().ApplyDamage(p.t, attackerUniqueId, damage, uniqueId)
	if err != nil {
		p.l.WithError(err).Errorf("Error applying friendly damage to monster [%d].", uniqueId)
		return
	}

	if s.Killed {
		GetCooldownRegistry().ClearCooldowns(uniqueId)
		GetDropTimerRegistry().Unregister(p.t, uniqueId)

		for _, se := range s.Monster.StatusEffects() {
			_ = producer.ProviderImpl(p.l)(p.ctx)(EnvEventTopicMonsterStatus)(statusEffectCancelledEventProvider(s.Monster, se))
		}

		err = producer.ProviderImpl(p.l)(p.ctx)(EnvEventTopicMonsterStatus)(killedStatusEventProvider(s.Monster, 0, false, s.Monster.DamageSummary()))
		if err != nil {
			p.l.WithError(err).Errorf("Friendly monster [%d] killed, but unable to emit killed event.", s.Monster.UniqueId())
		}
		_, err = GetMonsterRegistry().RemoveMonster(p.t, s.Monster.UniqueId())
		if err != nil {
			p.l.WithError(err).Errorf("Friendly monster [%d] killed, but not removed from registry.", s.Monster.UniqueId())
		}
		return
	}

	_ = producer.ProviderImpl(p.l)(p.ctx)(EnvEventTopicMonsterStatus)(damagedStatusEventProvider(s.Monster, observerUniqueId, attackerUniqueId, false, s.Monster.DamageSummary()))
}

// spawnRevives spawns the revive/next-phase monsters when a monster dies
func (p *ProcessorImpl) spawnRevives(m Model, revives []uint32) {
	for _, reviveMonsterId := range revives {
		input := RestModel{
			MonsterId: reviveMonsterId,
			X:         m.X(),
			Y:         m.Y(),
			Fh:        m.Fh(),
			Team:      m.Team(),
		}
		_, err := p.Create(m.Field(), input)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to spawn revive monster [%d] from monster [%d].", reviveMonsterId, m.UniqueId())
		}
	}
}

// Move moves a monster to a new position
func (p *ProcessorImpl) Move(id uint32, x int16, y int16, stance byte) error {
	GetMonsterRegistry().MoveMonster(p.t, id, x, y, stance)
	return nil
}

// UseSkill validates and executes a monster skill
func (p *ProcessorImpl) UseSkill(uniqueId uint32, characterId uint32, skillId uint16, skillLevel uint16) {
	m, err := GetMonsterRegistry().GetMonster(p.t, uniqueId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to get monster [%d] for skill use.", uniqueId)
		return
	}
	if !m.Alive() {
		return
	}

	// Check seal status - sealed monsters cannot use skills
	if m.HasStatusEffect("SEAL") {
		p.l.Debugf("Monster [%d] is sealed and cannot use skill [%d].", uniqueId, skillId)
		return
	}

	// Fetch skill definition from data service
	sd, err := mobskill.GetByIdAndLevel(p.l)(p.ctx)(skillId, skillLevel)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to retrieve mob skill [%d] level [%d].", skillId, skillLevel)
		return
	}

	// Check cooldown
	if GetCooldownRegistry().IsOnCooldown(uniqueId, skillId) {
		p.l.Debugf("Monster [%d] skill [%d] is on cooldown.", uniqueId, skillId)
		return
	}

	// Check HP threshold (skill's hp field = max HP% to use skill, default 100 = always)
	if sd.Hp() > 0 && m.HpPercentage() > sd.Hp() {
		p.l.Debugf("Monster [%d] HP [%d%%] above skill [%d] threshold [%d%%].", uniqueId, m.HpPercentage(), skillId, sd.Hp())
		return
	}

	// Check MP
	if sd.MpCon() > 0 && m.Mp() < sd.MpCon() {
		p.l.Debugf("Monster [%d] insufficient MP [%d] for skill [%d] cost [%d].", uniqueId, m.Mp(), skillId, sd.MpCon())
		return
	}

	// Deduct MP
	if sd.MpCon() > 0 {
		_, err = GetMonsterRegistry().DeductMp(p.t, uniqueId, sd.MpCon())
		if err != nil {
			p.l.WithError(err).Errorf("Unable to deduct MP from monster [%d].", uniqueId)
			return
		}
	}

	// Register cooldown
	if sd.Interval() > 0 {
		GetCooldownRegistry().SetCooldown(uniqueId, skillId, time.Duration(sd.Interval())*time.Second)
	}

	// Probability check
	if sd.Prop() < 100 {
		if rand.Intn(100) >= int(sd.Prop()) {
			p.l.Debugf("Monster [%d] skill [%d] probability check failed [%d%%].", uniqueId, skillId, sd.Prop())
			return
		}
	}

	// Stacking check for reflect/immunity - cannot apply if already active
	category := monster2.SkillCategory(skillId)
	if category == monster2.SkillCategoryImmunity || category == monster2.SkillCategoryReflect {
		statusName := monster2.SkillTypeToStatusName(skillId)
		if statusName != "" && m.HasStatusEffect(string(statusName)) {
			p.l.Debugf("Monster [%d] already has active [%s]. Skill [%d] rejected.", uniqueId, statusName, skillId)
			return
		}
	}

	// Determine animation delay from monster data
	var animDelay time.Duration
	ma, err := information.GetById(p.l)(p.ctx)(m.MonsterId())
	if err == nil {
		if d, ok := ma.AnimationTimes()["skill1"]; ok && d > 0 {
			animDelay = time.Duration(d) * time.Millisecond
		}
	}

	executeEffect := func() {
		switch category {
		case monster2.SkillCategoryStatBuff, monster2.SkillCategoryImmunity, monster2.SkillCategoryReflect:
			p.executeStatBuff(m, sd, skillId, skillLevel)
		case monster2.SkillCategoryHeal:
			p.executeHeal(m, characterId, sd)
		case monster2.SkillCategoryDebuff:
			p.executeDebuff(m, sd, skillId, skillLevel)
		case monster2.SkillCategorySummon:
			p.executeSummon(m, sd)
		default:
			p.l.Warnf("Monster [%d] unknown skill category for skill [%d].", uniqueId, skillId)
		}
	}

	if animDelay > 0 {
		go func() {
			time.Sleep(animDelay)
			executeEffect()
		}()
	} else {
		executeEffect()
	}
}

// UseSkillGM executes a mob skill on a monster without validation checks (GM command).
func (p *ProcessorImpl) UseSkillGM(uniqueId uint32, skillId uint16, skillLevel uint16) {
	m, err := GetMonsterRegistry().GetMonster(p.t, uniqueId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to get monster [%d] for GM skill use.", uniqueId)
		return
	}
	if !m.Alive() {
		return
	}

	sd, err := mobskill.GetByIdAndLevel(p.l)(p.ctx)(skillId, skillLevel)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to retrieve mob skill [%d] level [%d] for GM command.", skillId, skillLevel)
		return
	}

	category := monster2.SkillCategory(skillId)
	switch category {
	case monster2.SkillCategoryStatBuff, monster2.SkillCategoryImmunity, monster2.SkillCategoryReflect:
		p.executeStatBuff(m, sd, skillId, skillLevel)
	case monster2.SkillCategoryHeal:
		p.executeHeal(m, m.UniqueId(), sd)
	case monster2.SkillCategoryDebuff:
		p.executeDebuff(m, sd, skillId, skillLevel)
	case monster2.SkillCategorySummon:
		p.executeSummon(m, sd)
	default:
		p.l.Warnf("Monster [%d] unknown skill category for GM skill [%d].", uniqueId, skillId)
	}
}

// executeStatBuff applies a stat buff/immunity/reflect to the monster (and nearby monsters for AoE)
func (p *ProcessorImpl) executeStatBuff(m Model, sd mobskill.Model, skillId uint16, skillLevel uint16) {
	statusName := monster2.SkillTypeToStatusName(skillId)
	if statusName == "" {
		p.l.Warnf("No status mapping for skill type [%d].", skillId)
		return
	}

	statuses := map[string]int32{string(statusName): sd.X()}
	duration := time.Duration(sd.Duration()) * time.Second

	applyBuff := func(targetId uint32) {
		effect := NewStatusEffect(
			SourceTypeMonsterSkill,
			0,
			uint32(skillId),
			uint32(skillLevel),
			statuses,
			duration,
			0,
		)
		err := p.ApplyStatusEffect(targetId, effect)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to apply stat buff to monster [%d].", targetId)
		}
	}

	applyBuff(m.UniqueId())

	if monster2.IsAoeSkill(skillId) && sd.HasBoundingBox() {
		_ = model.ForEachSlice(p.ByFieldProvider(m.Field()), func(other Model) error {
			if other.UniqueId() == m.UniqueId() {
				return nil
			}
			dx := int32(other.X()) - int32(m.X())
			dy := int32(other.Y()) - int32(m.Y())
			if dx >= sd.LtX() && dx <= sd.RbX() && dy >= sd.LtY() && dy <= sd.RbY() {
				applyBuff(other.UniqueId())
			}
			return nil
		})
	}
}

// executeHeal heals the monster (and nearby monsters for AoE)
func (p *ProcessorImpl) executeHeal(m Model, observerId uint32, sd mobskill.Model) {
	healAmount := uint32(sd.X())

	healMonster := func(targetId uint32) {
		target, err := GetMonsterRegistry().GetMonster(p.t, targetId)
		if err != nil || !target.Alive() {
			return
		}
		healed := target.Heal(healAmount)
		GetMonsterRegistry().UpdateMonster(p.t, targetId, healed)
		// Emit a damaged event with 0 damage to trigger HP bar update
		_ = producer.ProviderImpl(p.l)(p.ctx)(EnvEventTopicMonsterStatus)(damagedStatusEventProvider(healed, observerId, m.UniqueId(), false, healed.DamageSummary()))
	}

	healMonster(m.UniqueId())

	if sd.HasBoundingBox() {
		_ = model.ForEachSlice(p.ByFieldProvider(m.Field()), func(other Model) error {
			if other.UniqueId() == m.UniqueId() {
				return nil
			}
			dx := int32(other.X()) - int32(m.X())
			dy := int32(other.Y()) - int32(m.Y())
			if dx >= sd.LtX() && dx <= sd.RbX() && dy >= sd.LtY() && dy <= sd.RbY() {
				healMonster(other.UniqueId())
			}
			return nil
		})
	}
}

// executeDebuff applies a disease to target players
func (p *ProcessorImpl) executeDebuff(m Model, sd mobskill.Model, skillId uint16, skillLevel uint16) {
	// Special handling for dispel
	if skillId == monster2.SkillTypeDispel {
		p.executeDispel(m, sd)
		return
	}

	// Special handling for banish
	if skillId == monster2.SkillTypeBanish {
		p.executeBanish(m, sd)
		return
	}

	diseaseName := monster2.SkillTypeToDiseaseName(skillId)
	if diseaseName == "" {
		p.l.Warnf("No disease mapping for skill type [%d].", skillId)
		return
	}

	value := sd.X()
	duration := int32(sd.Duration())
	targets := p.getDiseaseTargets(m, sd)

	for _, characterId := range targets {
		err := producer.ProviderImpl(p.l)(p.ctx)(EnvCommandTopicCharacterBuff)(applyDiseaseCommandProvider(m.Field(), characterId, skillId, skillLevel, diseaseName, value, duration))
		if err != nil {
			p.l.WithError(err).Errorf("Unable to apply disease [%s] to character [%d] from monster [%d].", diseaseName, characterId, m.UniqueId())
		}
	}
}

// executeBanish warps target players to the monster's banish map
func (p *ProcessorImpl) executeBanish(m Model, sd mobskill.Model) {
	ma, err := information.GetById(p.l)(p.ctx)(m.MonsterId())
	if err != nil {
		p.l.WithError(err).Errorf("Unable to get monster info for banish from monster [%d].", m.UniqueId())
		return
	}

	banishMapId := ma.Banish().MapId
	if banishMapId == 0 {
		p.l.Debugf("Monster [%d] has no banish map configured.", m.UniqueId())
		return
	}

	targets := p.getDiseaseTargets(m, sd)
	for _, characterId := range targets {
		err := producer.ProviderImpl(p.l)(p.ctx)(EnvCommandTopicPortal)(warpCommandProvider(m.Field(), characterId, map2.Id(banishMapId)))
		if err != nil {
			p.l.WithError(err).Errorf("Unable to banish character [%d] from monster [%d] to map [%d].", characterId, m.UniqueId(), banishMapId)
		}
	}
}

// executeDispel removes all buffs from target players
func (p *ProcessorImpl) executeDispel(m Model, sd mobskill.Model) {
	targets := p.getDiseaseTargets(m, sd)
	for _, characterId := range targets {
		err := producer.ProviderImpl(p.l)(p.ctx)(EnvCommandTopicCharacterBuff)(cancelAllBuffsCommandProvider(m.Field(), characterId))
		if err != nil {
			p.l.WithError(err).Errorf("Unable to dispel buffs from character [%d] from monster [%d].", characterId, m.UniqueId())
		}
	}
}

// getDiseaseTargets returns the character IDs that should be affected by a debuff skill
func (p *ProcessorImpl) getDiseaseTargets(m Model, sd mobskill.Model) []uint32 {
	// Single-target: use controlling character
	if !sd.HasBoundingBox() && sd.Count() <= 1 {
		if m.ControlCharacterId() == 0 {
			return nil
		}
		return []uint32{m.ControlCharacterId()}
	}

	// AoE: get all characters in the field
	ids, err := _map.CharacterIdsInFieldProvider(p.l)(p.ctx)(m.Field())()
	if err != nil {
		p.l.WithError(err).Errorf("Unable to get characters in field for monster [%d] disease targeting.", m.UniqueId())
		return nil
	}

	// Apply target limit
	if sd.Count() > 0 && uint32(len(ids)) > sd.Count() {
		rand.Shuffle(len(ids), func(i, j int) { ids[i], ids[j] = ids[j], ids[i] })
		ids = ids[:sd.Count()]
	}

	return ids
}

// executeSummon spawns monsters defined by the summon skill
func (p *ProcessorImpl) executeSummon(m Model, sd mobskill.Model) {
	summons := sd.Summons()
	if len(summons) == 0 {
		p.l.Debugf("Monster [%d] summon skill has no summon targets.", m.UniqueId())
		return
	}

	// Check summon limit against currently alive monsters in the field
	if sd.Limit() > 0 {
		existing, _ := p.GetInField(m.Field())
		if uint32(len(existing)) >= sd.Limit() {
			p.l.Debugf("Monster [%d] summon limit reached [%d/%d].", m.UniqueId(), len(existing), sd.Limit())
			return
		}
	}

	for _, summonMonsterId := range summons {
		input := RestModel{
			MonsterId: summonMonsterId,
			X:         m.X(),
			Y:         m.Y(),
			Fh:        m.Fh(),
			Team:      m.Team(),
		}
		_, err := p.Create(m.Field(), input)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to summon monster [%d] from monster [%d].", summonMonsterId, m.UniqueId())
		}
	}
}

// Destroy destroys a monster
func (p *ProcessorImpl) Destroy(uniqueId uint32) error {
	GetDropTimerRegistry().Unregister(p.t, uniqueId)
	m, err := GetMonsterRegistry().RemoveMonster(p.t, uniqueId)
	if err != nil {
		return err
	}

	return producer.ProviderImpl(p.l)(p.ctx)(EnvEventTopicMonsterStatus)(destroyedStatusEventProvider(m))
}

// DestroyInField destroys all monsters in a field
func (p *ProcessorImpl) DestroyInField(f field.Model) error {
	return model.ForEachSlice(model.SliceMap[Model, uint32](IdTransformer)(p.ByFieldProvider(f))(model.ParallelMap()), p.Destroy, model.ParallelExecute())
}

// ApplyStatusEffect applies a status effect to a monster after checking immunities
func (p *ProcessorImpl) ApplyStatusEffect(uniqueId uint32, effect StatusEffect) error {
	m, err := GetMonsterRegistry().GetMonster(p.t, uniqueId)
	if err != nil {
		return err
	}

	// Only check immunities for player-sourced effects
	if effect.SourceType() == SourceTypePlayerSkill {
		info, err := information.GetById(p.l)(p.ctx)(m.MonsterId())
		if err == nil {
			// Elemental immunity check
			if blocked, element := isElementallyImmune(info, effect); blocked {
				p.l.Debugf("Monster [%d] is immune to element [%s]. Status rejected.", uniqueId, element)
				return errors.New("elemental immunity")
			}

			// Boss immunity check
			if info.Boss() && !isBossAllowedStatus(effect) {
				p.l.Debugf("Monster [%d] is a boss. Status rejected.", uniqueId)
				return errors.New("boss immunity")
			}
		}
	}

	m, err = GetMonsterRegistry().ApplyStatusEffect(p.t, uniqueId, effect)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to apply status effect to monster [%d].", uniqueId)
		return err
	}

	_ = producer.ProviderImpl(p.l)(p.ctx)(EnvEventTopicMonsterStatus)(statusEffectAppliedEventProvider(m, effect))
	return nil
}

// isElementallyImmune checks if a monster's resistances block the given status effect
func isElementallyImmune(info information.Model, effect StatusEffect) (bool, string) {
	for statusType := range effect.Statuses() {
		switch statusType {
		case "POISON":
			if info.IsImmuneToElement("P") {
				return true, "poison"
			}
		case "FREEZE":
			if info.IsImmuneToElement("I") {
				return true, "ice"
			}
		}
	}
	return false, ""
}

// isBossAllowedStatus returns true if the given status effect can be applied to boss monsters
func isBossAllowedStatus(effect StatusEffect) bool {
	for statusType := range effect.Statuses() {
		switch statusType {
		case "SPEED", "WEAPON_ATTACK", "WEAPON_DEFENSE", "MAGIC_ATTACK", "MAGIC_DEFENSE",
			"POWER_UP", "MAGIC_UP", "POWER_GUARD_UP", "MAGIC_GUARD_UP",
			"SHOWDOWN", "NINJA_AMBUSH", "VENOM":
			// These can affect bosses
			continue
		default:
			// Other statuses (stun, seal, freeze, poison, etc.) cannot affect bosses
			return false
		}
	}
	return true
}

// CancelStatusEffect cancels status effects by type from a monster
func (p *ProcessorImpl) CancelStatusEffect(uniqueId uint32, statusTypes []string) error {
	m, err := p.GetById(uniqueId)
	if err != nil {
		return err
	}

	for _, st := range statusTypes {
		for _, se := range m.StatusEffects() {
			if se.HasStatus(st) {
				m, err = GetMonsterRegistry().CancelStatusEffect(p.t, uniqueId, se.EffectId())
				if err != nil {
					p.l.WithError(err).Errorf("Unable to cancel status effect [%s] from monster [%d].", se.EffectId(), uniqueId)
					continue
				}
				_ = producer.ProviderImpl(p.l)(p.ctx)(EnvEventTopicMonsterStatus)(statusEffectCancelledEventProvider(m, se))
			}
		}
	}
	return nil
}

// CancelAllStatusEffects cancels all status effects from a monster
func (p *ProcessorImpl) CancelAllStatusEffects(uniqueId uint32) error {
	m, err := p.GetById(uniqueId)
	if err != nil {
		return err
	}

	effects := m.StatusEffects()
	m, err = GetMonsterRegistry().CancelAllStatusEffects(p.t, uniqueId)
	if err != nil {
		return err
	}

	for _, se := range effects {
		_ = producer.ProviderImpl(p.l)(p.ctx)(EnvEventTopicMonsterStatus)(statusEffectCancelledEventProvider(m, se))
	}
	return nil
}

// checkReflect checks if the monster has an active reflect status that matches the attack type
// and emits a reflect event if so. Attack types: 0=melee, 1=ranged, 2=magic, 3=energy.
func (p *ProcessorImpl) checkReflect(m Model, characterId uint32, attackType byte) {
	const attackTypeMagic = byte(2)

	for _, se := range m.StatusEffects() {
		if attackType == attackTypeMagic {
			if val, ok := se.Statuses()[string(monster2.TemporaryStatTypeMagicCounter)]; ok && val > 0 {
				_ = producer.ProviderImpl(p.l)(p.ctx)(EnvEventTopicMonsterStatus)(damageReflectedEventProvider(m, characterId, uint32(val), string(monster2.TemporaryStatTypeMagicCounter)))
				return
			}
		} else {
			if val, ok := se.Statuses()[string(monster2.TemporaryStatTypeWeaponCounter)]; ok && val > 0 {
				_ = producer.ProviderImpl(p.l)(p.ctx)(EnvEventTopicMonsterStatus)(damageReflectedEventProvider(m, characterId, uint32(val), string(monster2.TemporaryStatTypeWeaponCounter)))
				return
			}
		}
	}
}

// Helper functions

func zeroValue(_ uint32) int {
	return 0
}

func characterIdKey(id uint32) uint32 {
	return id
}

func IdTransformer(m Model) (uint32, error) {
	return m.UniqueId(), nil
}

// Filter functions

func Controlled(m Model) bool {
	return m.ControlCharacterId() != 0
}

func NotControlled(m Model) bool {
	return m.ControlCharacterId() == 0
}

func IsControlledBy(id uint32) model.Filter[Model] {
	return func(m Model) bool {
		return m.ControlCharacterId() == id
	}
}

// Lifecycle functions for service shutdown

func allByTenantProvider() model.Provider[map[tenant.Model][]Model] {
	return func() (map[tenant.Model][]Model, error) {
		return GetMonsterRegistry().GetMonsters(), nil
	}
}

func destroyInTenant(l logrus.FieldLogger) func(ctx context.Context) func(t tenant.Model) model.Operator[[]Model] {
	return func(ctx context.Context) func(t tenant.Model) model.Operator[[]Model] {
		return func(t tenant.Model) model.Operator[[]Model] {
			return func(models []Model) error {
				tctx := tenant.WithContext(ctx, t)
				p := NewProcessor(l, tctx)
				idp := model.SliceMap(IdTransformer)(model.FixedProvider(models))(model.ParallelMap())
				return model.ForEachSlice(idp, p.Destroy, model.ParallelExecute())
			}
		}
	}
}

func DestroyAll(l logrus.FieldLogger, ctx context.Context) error {
	return model.ForEachMap(allByTenantProvider(), destroyInTenant(l)(ctx), model.ParallelExecute())
}

func Teardown(l logrus.FieldLogger) func() {
	return func() {
		ctx, span := otel.GetTracerProvider().Tracer("atlas-monsters").Start(context.Background(), "teardown")
		defer span.End()

		err := DestroyAll(l, ctx)
		if err != nil {
			l.WithError(err).Errorf("Error destroying all monsters on teardown.")
		}
	}
}
