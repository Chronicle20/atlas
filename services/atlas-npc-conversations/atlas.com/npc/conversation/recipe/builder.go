package recipe

import (
	"github.com/google/uuid"
)

// Builder mutates a draft Model and returns immutable copies.
type Builder struct {
	m Model
}

func NewBuilder() *Builder { return &Builder{} }

func (b *Builder) SetId(id uuid.UUID) *Builder                { b.m.id = id; return b }
func (b *Builder) SetTenantId(id uuid.UUID) *Builder          { b.m.tenantId = id; return b }
func (b *Builder) SetConversationId(id uuid.UUID) *Builder    { b.m.conversationId = id; return b }
func (b *Builder) SetNpcId(id uint32) *Builder                { b.m.npcId = id; return b }
func (b *Builder) SetStateId(id string) *Builder              { b.m.stateId = id; return b }
func (b *Builder) SetItemId(id uint32) *Builder               { b.m.itemId = id; return b }
func (b *Builder) SetMaterials(materials []Material) *Builder { b.m.materials = materials; return b }
func (b *Builder) SetMesoCost(cost uint32) *Builder           { b.m.mesoCost = cost; return b }
func (b *Builder) SetStimulatorId(id uint32) *Builder         { b.m.stimulatorId = id; return b }
func (b *Builder) SetStimulatorFailChance(c float64) *Builder { b.m.stimulatorFailChance = c; return b }

// Build returns a copy of the assembled Model. If id is unset, it is computed
// deterministically from (tenantId, conversationId, stateId).
func (b *Builder) Build() (Model, error) {
	m := b.m
	if m.id == uuid.Nil {
		m.id = ComputeRecipeId(m.tenantId, m.conversationId, m.stateId)
	}
	return m, nil
}
