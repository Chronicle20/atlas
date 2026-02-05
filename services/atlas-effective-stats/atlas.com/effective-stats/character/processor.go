package character

import (
	character2 "atlas-effective-stats/kafka/message/character"
	"atlas-effective-stats/kafka/producer"
	"atlas-effective-stats/stat"
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	GetEffectiveStats(ch channel.Model, characterId uint32) (stat.Computed, []stat.Bonus, error)
	AddBonus(ch channel.Model, characterId uint32, source string, statType stat.Type, amount int32) error
	AddMultiplierBonus(ch channel.Model, characterId uint32, source string, statType stat.Type, multiplier float64) error
	RemoveBonus(characterId uint32, source string, statType stat.Type) error
	RemoveBonusesBySource(characterId uint32, source string) error
	SetBaseStats(ch channel.Model, characterId uint32, base stat.Base) error
	// Equipment bonus methods
	AddEquipmentBonuses(ch channel.Model, characterId uint32, equipmentId uint32, bonuses []stat.Bonus) error
	RemoveEquipmentBonuses(characterId uint32, equipmentId uint32) error
	// Buff bonus methods
	AddBuffBonuses(ch channel.Model, characterId uint32, buffSourceId int32, bonuses []stat.Bonus) error
	RemoveBuffBonuses(characterId uint32, buffSourceId int32) error
	// Passive skill bonus methods
	AddPassiveBonuses(ch channel.Model, characterId uint32, skillId uint32, bonuses []stat.Bonus) error
	RemovePassiveBonuses(characterId uint32, skillId uint32) error
	// Cleanup
	RemoveCharacter(characterId uint32)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	t   tenant.Model
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		t:   tenant.MustFromContext(ctx),
	}
}

// GetEffectiveStats retrieves computed effective stats and bonuses for a character
// If the character hasn't been initialized yet, this will lazily initialize them
func (p *ProcessorImpl) GetEffectiveStats(ch channel.Model, characterId uint32) (stat.Computed, []stat.Bonus, error) {
	// Lazy initialization - ensure character's data is set up
	if !GetRegistry().IsInitialized(p.t, characterId) {
		p.l.Debugf("Character [%d] not initialized, performing lazy initialization.", characterId)
		if err := InitializeCharacter(p.l, p.ctx, characterId, ch); err != nil {
			p.l.WithError(err).Warnf("Failed to initialize character [%d], returning current state.", characterId)
		}
	}

	m := GetRegistry().GetOrCreate(p.t, ch, characterId)

	return m.Computed(), m.Bonuses(), nil
}

// AddBonus adds or updates a flat stat bonus for a character
func (p *ProcessorImpl) AddBonus(ch channel.Model, characterId uint32, source string, statType stat.Type, amount int32) error {
	b := stat.NewBonus(source, statType, amount)
	m := GetRegistry().AddBonus(p.t, ch, characterId, b)
	p.l.Debugf("Added bonus [%s] for character [%d]: %s = %d", source, characterId, statType, amount)
	p.logEffectiveStats(characterId, m.Computed())
	return nil
}

// AddMultiplierBonus adds or updates a percentage stat bonus for a character
func (p *ProcessorImpl) AddMultiplierBonus(ch channel.Model, characterId uint32, source string, statType stat.Type, multiplier float64) error {
	b := stat.NewMultiplierBonus(source, statType, multiplier)
	m := GetRegistry().AddBonus(p.t, ch, characterId, b)
	p.l.Debugf("Added multiplier bonus [%s] for character [%d]: %s = %.2f%%", source, characterId, statType, multiplier*100)
	p.logEffectiveStats(characterId, m.Computed())
	return nil
}

// RemoveBonus removes a specific stat bonus for a character
func (p *ProcessorImpl) RemoveBonus(characterId uint32, source string, statType stat.Type) error {
	// Get current model to capture old MaxHP/MaxMP before removal
	oldModel, err := GetRegistry().Get(p.t, characterId)
	if err != nil {
		return err
	}
	oldComputed := oldModel.Computed()

	// Remove bonus and recompute
	newModel, err := GetRegistry().RemoveBonus(p.t, characterId, source, statType)
	if err != nil {
		return err
	}
	newComputed := newModel.Computed()

	p.l.Debugf("Removed bonus [%s] type [%s] for character [%d]", source, statType, characterId)
	p.logEffectiveStats(characterId, newComputed)

	// Check for MaxHP/MaxMP decreases and publish clamp commands if needed
	p.checkAndPublishClampCommands(newModel, oldComputed, newComputed)

	return nil
}

// RemoveBonusesBySource removes all bonuses from a specific source for a character
func (p *ProcessorImpl) RemoveBonusesBySource(characterId uint32, source string) error {
	// Get current model to capture old MaxHP/MaxMP before removal
	oldModel, err := GetRegistry().Get(p.t, characterId)
	if err != nil {
		return err
	}
	oldComputed := oldModel.Computed()

	// Remove bonuses and recompute
	newModel, err := GetRegistry().RemoveBonusesBySource(p.t, characterId, source)
	if err != nil {
		return err
	}
	newComputed := newModel.Computed()

	p.l.Debugf("Removed all bonuses from source [%s] for character [%d]", source, characterId)
	p.logEffectiveStats(characterId, newComputed)

	// Check for MaxHP/MaxMP decreases and publish clamp commands if needed
	p.checkAndPublishClampCommands(newModel, oldComputed, newComputed)

	return nil
}

// SetBaseStats sets the base stats for a character
func (p *ProcessorImpl) SetBaseStats(ch channel.Model, characterId uint32, base stat.Base) error {
	m := GetRegistry().SetBaseStats(p.t, ch, characterId, base)
	p.l.Debugf("Set base stats for character [%d]: STR=%d, DEX=%d, INT=%d, LUK=%d, MaxHP=%d, MaxMP=%d",
		characterId, base.Strength(), base.Dexterity(), base.Intelligence(), base.Luck(), base.MaxHP(), base.MaxMP())
	p.logEffectiveStats(characterId, m.Computed())
	return nil
}

// AddEquipmentBonuses adds stat bonuses from equipment
func (p *ProcessorImpl) AddEquipmentBonuses(ch channel.Model, characterId uint32, equipmentId uint32, bonuses []stat.Bonus) error {
	source := fmt.Sprintf("equipment:%d", equipmentId)
	sourcedBonuses := make([]stat.Bonus, 0, len(bonuses))
	for _, b := range bonuses {
		sourcedBonuses = append(sourcedBonuses, stat.NewFullBonus(source, b.StatType(), b.Amount(), b.Multiplier()))
	}
	m := GetRegistry().AddBonuses(p.t, ch, characterId, sourcedBonuses)
	p.l.Debugf("Added equipment [%d] bonuses for character [%d]: %d stats", equipmentId, characterId, len(bonuses))
	p.logEffectiveStats(characterId, m.Computed())
	return nil
}

// RemoveEquipmentBonuses removes all stat bonuses from equipment
func (p *ProcessorImpl) RemoveEquipmentBonuses(characterId uint32, equipmentId uint32) error {
	source := fmt.Sprintf("equipment:%d", equipmentId)
	return p.RemoveBonusesBySource(characterId, source)
}

// AddBuffBonuses adds stat bonuses from a buff
func (p *ProcessorImpl) AddBuffBonuses(ch channel.Model, characterId uint32, buffSourceId int32, bonuses []stat.Bonus) error {
	source := fmt.Sprintf("buff:%d", buffSourceId)
	sourcedBonuses := make([]stat.Bonus, 0, len(bonuses))
	for _, b := range bonuses {
		sourcedBonuses = append(sourcedBonuses, stat.NewFullBonus(source, b.StatType(), b.Amount(), b.Multiplier()))
	}
	m := GetRegistry().AddBonuses(p.t, ch, characterId, sourcedBonuses)
	p.l.Debugf("Added buff [%d] bonuses for character [%d]: %d stats", buffSourceId, characterId, len(bonuses))
	p.logEffectiveStats(characterId, m.Computed())
	return nil
}

// RemoveBuffBonuses removes all stat bonuses from a buff
func (p *ProcessorImpl) RemoveBuffBonuses(characterId uint32, buffSourceId int32) error {
	source := fmt.Sprintf("buff:%d", buffSourceId)
	return p.RemoveBonusesBySource(characterId, source)
}

// AddPassiveBonuses adds stat bonuses from a passive skill
func (p *ProcessorImpl) AddPassiveBonuses(ch channel.Model, characterId uint32, skillId uint32, bonuses []stat.Bonus) error {
	source := fmt.Sprintf("passive:%d", skillId)
	sourcedBonuses := make([]stat.Bonus, 0, len(bonuses))
	for _, b := range bonuses {
		sourcedBonuses = append(sourcedBonuses, stat.NewFullBonus(source, b.StatType(), b.Amount(), b.Multiplier()))
	}
	m := GetRegistry().AddBonuses(p.t, ch, characterId, sourcedBonuses)
	p.l.Debugf("Added passive skill [%d] bonuses for character [%d]: %d stats", skillId, characterId, len(bonuses))
	p.logEffectiveStats(characterId, m.Computed())
	return nil
}

// RemovePassiveBonuses removes all stat bonuses from a passive skill
func (p *ProcessorImpl) RemovePassiveBonuses(characterId uint32, skillId uint32) error {
	source := fmt.Sprintf("passive:%d", skillId)
	return p.RemoveBonusesBySource(characterId, source)
}

// RemoveCharacter removes a character from the registry
func (p *ProcessorImpl) RemoveCharacter(characterId uint32) {
	GetRegistry().Delete(p.t, characterId)
	p.l.Debugf("Removed character [%d] from effective stats registry.", characterId)
}

// logEffectiveStats logs the current effective stats for debugging
func (p *ProcessorImpl) logEffectiveStats(characterId uint32, c stat.Computed) {
	p.l.Debugf("Character [%d] effective stats: STR=%d, DEX=%d, INT=%d, LUK=%d, MaxHP=%d, MaxMP=%d, WATK=%d, MATK=%d",
		characterId, c.Strength(), c.Dexterity(), c.Intelligence(), c.Luck(),
		c.MaxHP(), c.MaxMP(), c.WeaponAttack(), c.MagicAttack())
}

// checkAndPublishClampCommands checks if MaxHP or MaxMP decreased and publishes clamp commands if needed
func (p *ProcessorImpl) checkAndPublishClampCommands(m Model, oldComputed, newComputed stat.Computed) {
	transactionId := uuid.New()

	// Check for MaxHP decrease
	if newComputed.MaxHP() < oldComputed.MaxHP() {
		p.l.Debugf("MaxHP decreased for character [%d]: %d -> %d, publishing clamp command", m.CharacterId(), oldComputed.MaxHP(), newComputed.MaxHP())
		_ = producer.ProviderImpl(p.l)(p.ctx)(character2.EnvCommandTopic)(clampHPCommandProvider(transactionId, m.Channel(), m.CharacterId(), uint16(newComputed.MaxHP())))
	}

	// Check for MaxMP decrease
	if newComputed.MaxMP() < oldComputed.MaxMP() {
		p.l.Debugf("MaxMP decreased for character [%d]: %d -> %d, publishing clamp command", m.CharacterId(), oldComputed.MaxMP(), newComputed.MaxMP())
		_ = producer.ProviderImpl(p.l)(p.ctx)(character2.EnvCommandTopic)(clampMPCommandProvider(transactionId, m.Channel(), m.CharacterId(), uint16(newComputed.MaxMP())))
	}
}
