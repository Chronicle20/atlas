package commodities

import (
	"github.com/google/uuid"
)

type Model struct {
	id              uuid.UUID
	npcId           uint32
	templateId      uint32
	mesoPrice       uint32
	discountRate    byte
	tokenTemplateId uint32
	tokenPrice      uint32
	period          uint32
	levelLimit      uint32
	unitPrice       float64
	slotMax         uint32
}

// Id returns the model's id
func (m Model) Id() uuid.UUID {
	return m.id
}

// TemplateId returns the model's templateId
func (m Model) TemplateId() uint32 {
	return m.templateId
}

// MesoPrice returns the model's mesoPrice
func (m Model) MesoPrice() uint32 {
	return m.mesoPrice
}

// DiscountRate returns the model's discountRate
func (m Model) DiscountRate() byte {
	return m.discountRate
}

// TokenTemplateId returns the model's tokenTemplateId
func (m Model) TokenTemplateId() uint32 {
	return m.tokenTemplateId
}

// TokenPrice returns the model's tokenPrice
func (m Model) TokenPrice() uint32 {
	return m.tokenPrice
}

// Period returns the model's period
func (m Model) Period() uint32 {
	return m.period
}

// LevelLimit returns the model's levelLimit
func (m Model) LevelLimit() uint32 {
	return m.levelLimit
}

// NpcId returns the model's npcId
func (m Model) NpcId() uint32 {
	return m.npcId
}

// UnitPrice returns the model's unitPrice
func (m Model) UnitPrice() float64 {
	return m.unitPrice
}

// SlotMax returns the model's slotMax
func (m Model) SlotMax() uint32 {
	return m.slotMax
}
