package npc

import (
	"atlas-npc-conversations/conversation"
	"context"
	"fmt"

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
	return model.Map[Entity, Model](Make)(getByIdProvider(p.t.Id())(id)(p.db))
}

// ByNpcIdProvider returns a provider for retrieving an NPC conversation by NPC ID
func (p *ProcessorImpl) ByNpcIdProvider(npcId uint32) model.Provider[Model] {
	return model.Map[Entity, Model](Make)(getByNpcIdProvider(p.t.Id())(npcId)(p.db))
}

// AllProvider returns a provider for retrieving all NPC conversations
func (p *ProcessorImpl) AllProvider() model.Provider[[]Model] {
	return model.SliceMap[Entity, Model](Make)(getAllProvider(p.t.Id())(p.db))(model.ParallelMap())
}

// AllByNpcIdProvider returns a provider for retrieving all NPC conversations for a specific NPC ID
func (p *ProcessorImpl) AllByNpcIdProvider(npcId uint32) model.Provider[[]Model] {
	return model.SliceMap[Entity, Model](Make)(getAllByNpcIdProvider(p.t.Id())(npcId)(p.db))(model.ParallelMap())
}

// Create creates a new NPC conversation
func (p *ProcessorImpl) Create(m Model) (Model, error) {
	p.l.Debugf("Creating NPC conversation for NPC [%d]", m.NpcId())

	result, err := createNpcConversation(p.db)(p.t.Id())(m)
	if err != nil {
		p.l.WithError(err).Errorf("Failed to create NPC conversation for NPC [%d]", m.NpcId())
		return Model{}, err
	}
	return result, nil
}

// Update updates an existing NPC conversation
func (p *ProcessorImpl) Update(id uuid.UUID, m Model) (Model, error) {
	p.l.Debugf("Updating NPC conversation [%s]", id)

	result, err := updateNpcConversation(p.db)(p.t.Id())(id)(m)
	if err != nil {
		p.l.WithError(err).Errorf("Failed to update NPC conversation [%s]", id)
		return Model{}, err
	}
	return result, nil
}

// Delete deletes an NPC conversation
func (p *ProcessorImpl) Delete(id uuid.UUID) error {
	p.l.Debugf("Deleting NPC conversation [%s]", id)

	err := deleteNpcConversation(p.db)(p.t.Id())(id)
	if err != nil {
		p.l.WithError(err).Errorf("Failed to delete NPC conversation [%s]", id)
		return err
	}
	return nil
}

// DeleteAllForTenant deletes all NPC conversations for the current tenant using hard delete
func (p *ProcessorImpl) DeleteAllForTenant() (int64, error) {
	p.l.Debugf("Deleting all NPC conversations for tenant [%s]", p.t.Id())

	count, err := deleteAllNpcConversations(p.db)(p.t.Id())
	if err != nil {
		p.l.WithError(err).Errorf("Failed to delete NPC conversations for tenant [%s]", p.t.Id())
		return 0, err
	}
	p.l.Debugf("Deleted [%d] NPC conversations for tenant [%s]", count, p.t.Id())
	return count, nil
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
