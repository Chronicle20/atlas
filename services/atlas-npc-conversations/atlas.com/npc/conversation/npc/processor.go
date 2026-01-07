package npc

import (
	"atlas-npc-conversations/conversation"
	"context"
	"fmt"
	"time"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func init() {
	// Register the NPC conversation provider factory to break the import cycle
	conversation.SetNpcConversationProviderFactory(func(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) conversation.NpcConversationProvider {
		return &NpcConversationProviderAdapter{processor: NewProcessor(l, ctx, db)}
	})
}

// NpcConversationProviderAdapter adapts npc.Processor to conversation.NpcConversationProvider
type NpcConversationProviderAdapter struct {
	processor Processor
}

// ByNpcIdProvider implements conversation.NpcConversationProvider
func (a *NpcConversationProviderAdapter) ByNpcIdProvider(npcId uint32) func() (conversation.NpcConversation, error) {
	provider := a.processor.ByNpcIdProvider(npcId)
	return func() (conversation.NpcConversation, error) {
		m, err := provider()
		if err != nil {
			return nil, err
		}
		return &m, nil
	}
}

type Processor interface {
	// Create creates a new NPC conversation
	Create(model Model) (Model, error)

	// Update updates an existing NPC conversation
	Update(id uuid.UUID, model Model) (Model, error)

	// Delete deletes an NPC conversation
	Delete(id uuid.UUID) error

	// ByIdProvider returns a provider for retrieving an NPC conversation by ID
	ByIdProvider(id uuid.UUID) model.Provider[Model]

	// ByNpcIdProvider returns a provider for retrieving an NPC conversation by NPC ID
	ByNpcIdProvider(npcId uint32) model.Provider[Model]

	// AllByNpcIdProvider returns a provider for retrieving all NPC conversations for a specific NPC ID
	AllByNpcIdProvider(npcId uint32) model.Provider[[]Model]

	// AllProvider returns a provider for retrieving all NPC conversations
	AllProvider() model.Provider[[]Model]

	// DeleteAllForTenant deletes all NPC conversations for the current tenant
	DeleteAllForTenant() (int64, error)

	// Seed clears existing NPC conversations and loads them from the conversations directory
	Seed() (SeedResult, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	t   tenant.Model
	db  *gorm.DB
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	t := tenant.MustFromContext(ctx)

	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		t:   t,
		db:  db,
	}
}

// ByIdProvider returns a provider for retrieving an NPC conversation by ID
func (p *ProcessorImpl) ByIdProvider(id uuid.UUID) model.Provider[Model] {
	return model.Map[Entity, Model](Make)(GetByIdProvider(p.t.Id())(id)(p.db))
}

// ByNpcIdProvider returns a provider for retrieving an NPC conversation by NPC ID
func (p *ProcessorImpl) ByNpcIdProvider(npcId uint32) model.Provider[Model] {
	return model.Map[Entity, Model](Make)(GetByNpcIdProvider(p.t.Id())(npcId)(p.db))
}

// AllProvider returns a provider for retrieving all NPC conversations
func (p *ProcessorImpl) AllProvider() model.Provider[[]Model] {
	return model.SliceMap[Entity, Model](Make)(GetAllProvider(p.t.Id())(p.db))(model.ParallelMap())
}

// AllByNpcIdProvider returns a provider for retrieving all NPC conversations for a specific NPC ID
func (p *ProcessorImpl) AllByNpcIdProvider(npcId uint32) model.Provider[[]Model] {
	return model.SliceMap[Entity, Model](Make)(GetAllByNpcIdProvider(p.t.Id())(npcId)(p.db))(model.ParallelMap())
}

// Create creates a new NPC conversation
func (p *ProcessorImpl) Create(m Model) (Model, error) {
	p.l.Debugf("Creating NPC conversation for NPC [%d]", m.NpcId())

	// Convert model to entity
	entity, err := ToEntity(m, p.t.Id())
	if err != nil {
		p.l.WithError(err).Errorf("Failed to convert model to entity")
		return Model{}, err
	}

	entity.ID = uuid.New()

	// Save to database
	result := p.db.Create(&entity)
	if result.Error != nil {
		p.l.WithError(result.Error).Errorf("Failed to create NPC conversation")
		return Model{}, result.Error
	}

	// Convert back to model
	return Make(entity)
}

// Update updates an existing NPC conversation
func (p *ProcessorImpl) Update(id uuid.UUID, m Model) (Model, error) {
	p.l.Debugf("Updating NPC conversation [%s]", id)

	// Check if conversation exists
	var existingEntity Entity
	result := p.db.Where("tenant_id = ? AND id = ?", p.t.Id(), id).First(&existingEntity)
	if result.Error != nil {
		p.l.WithError(result.Error).Errorf("Failed to find NPC conversation [%s]", id)
		return Model{}, result.Error
	}

	// Convert model to entity
	entity, err := ToEntity(m, p.t.Id())
	if err != nil {
		p.l.WithError(err).Errorf("Failed to convert model to entity")
		return Model{}, err
	}

	// Ensure ID is preserved
	entity.ID = id

	// Update in database
	result = p.db.Model(&Entity{}).Where("tenant_id = ? AND id = ?", p.t.Id(), id).Updates(map[string]interface{}{
		"npc_id":     entity.NpcID,
		"data":       entity.Data,
		"updated_at": time.Now(),
	})
	if result.Error != nil {
		p.l.WithError(result.Error).Errorf("Failed to update NPC conversation [%s]", id)
		return Model{}, result.Error
	}

	// Retrieve updated entity
	result = p.db.Where("tenant_id = ? AND id = ?", p.t.Id(), id).First(&entity)
	if result.Error != nil {
		p.l.WithError(result.Error).Errorf("Failed to retrieve updated NPC conversation [%s]", id)
		return Model{}, result.Error
	}

	// Convert back to model
	return Make(entity)
}

// Delete deletes an NPC conversation
func (p *ProcessorImpl) Delete(id uuid.UUID) error {
	p.l.Debugf("Deleting NPC conversation [%s]", id)

	// Delete from database
	result := p.db.Where("tenant_id = ? AND id = ?", p.t.Id(), id).Delete(&Entity{})
	if result.Error != nil {
		p.l.WithError(result.Error).Errorf("Failed to delete NPC conversation [%s]", id)
		return result.Error
	}

	return nil
}

// DeleteAllForTenant deletes all NPC conversations for the current tenant using hard delete
func (p *ProcessorImpl) DeleteAllForTenant() (int64, error) {
	p.l.Debugf("Deleting all NPC conversations for tenant [%s]", p.t.Id())
	result := p.db.Unscoped().Where("tenant_id = ?", p.t.Id()).Delete(&Entity{})
	if result.Error != nil {
		p.l.WithError(result.Error).Errorf("Failed to delete NPC conversations for tenant [%s]", p.t.Id())
		return 0, result.Error
	}
	p.l.Debugf("Deleted [%d] NPC conversations for tenant [%s]", result.RowsAffected, p.t.Id())
	return result.RowsAffected, nil
}

// Seed clears existing NPC conversations and loads them from the conversations directory
func (p *ProcessorImpl) Seed() (SeedResult, error) {
	p.l.Infof("Seeding NPC conversations for tenant [%s]", p.t.Id())

	result := SeedResult{}

	// Delete all existing conversations for this tenant
	deletedCount, err := p.DeleteAllForTenant()
	if err != nil {
		return result, fmt.Errorf("failed to clear existing NPC conversations: %w", err)
	}
	result.DeletedCount = int(deletedCount)

	// Load conversation files from the filesystem
	models, loadErrors := LoadConversationFiles()

	// Track load errors
	for _, err := range loadErrors {
		result.Errors = append(result.Errors, err.Error())
		result.FailedCount++
	}

	// Create each conversation
	for _, rm := range models {
		// Extract domain model from REST model
		m, err := Extract(rm)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("npc_%d: failed to extract model: %v", rm.NpcId, err))
			result.FailedCount++
			continue
		}

		// Create the conversation
		_, err = p.Create(m)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("npc_%d: failed to create: %v", rm.NpcId, err))
			result.FailedCount++
			continue
		}

		result.CreatedCount++
	}

	p.l.Infof("Seed complete for tenant [%s]: deleted=%d, created=%d, failed=%d",
		p.t.Id(), result.DeletedCount, result.CreatedCount, result.FailedCount)

	return result, nil
}
