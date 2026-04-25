package recipe

import (
	"github.com/google/uuid"
)

// recipeNamespace is a stable namespace UUID used as the seed for UUID v5
// derivation of recipe row ids. Changing this value invalidates every existing
// recipe id, so it is hard-coded.
var recipeNamespace = uuid.MustParse("2f8d6a44-3b1c-4f3a-9e9d-7b6a9f2c4a10")

// Material is one entry of a recipe's material list — an item template id
// paired with the quantity required.
type Material struct {
	ItemId   uint32 `json:"itemId"`
	Quantity uint32 `json:"quantity"`
}

// Model is the immutable domain representation of a single recipe row.
type Model struct {
	id                   uuid.UUID
	tenantId             uuid.UUID
	conversationId       uuid.UUID
	npcId                uint32
	stateId              string
	itemId               uint32
	materials            []Material
	mesoCost             uint32
	stimulatorId         uint32
	stimulatorFailChance float64
}

func (m Model) Id() uuid.UUID               { return m.id }
func (m Model) TenantId() uuid.UUID         { return m.tenantId }
func (m Model) ConversationId() uuid.UUID   { return m.conversationId }
func (m Model) NpcId() uint32               { return m.npcId }
func (m Model) StateId() string             { return m.stateId }
func (m Model) ItemId() uint32              { return m.itemId }
func (m Model) Materials() []Material       { return m.materials }
func (m Model) MesoCost() uint32            { return m.mesoCost }
func (m Model) StimulatorId() uint32        { return m.stimulatorId }
func (m Model) StimulatorFailChance() float64 { return m.stimulatorFailChance }

// ComputeRecipeId returns the deterministic UUID v5 used as the recipes.id
// for the given (tenant, conversation, state) triple.
func ComputeRecipeId(tenantId uuid.UUID, conversationId uuid.UUID, stateId string) uuid.UUID {
	seed := tenantId.String() + ":" + conversationId.String() + ":" + stateId
	return uuid.NewSHA1(recipeNamespace, []byte(seed))
}

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
	if b.m.id == uuid.Nil {
		b.m.id = ComputeRecipeId(b.m.tenantId, b.m.conversationId, b.m.stateId)
	}
	return b.m, nil
}
