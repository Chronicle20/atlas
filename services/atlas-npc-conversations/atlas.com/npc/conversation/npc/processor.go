package npc

import (
	"atlas-npc-conversations/conversation"
	"atlas-npc-conversations/conversation/recipe"
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
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

	// Count returns the number of NPC conversations for the current tenant and the max updated_at timestamp.
	// Returns (0, nil, nil) when the tenant has no rows.
	Count() (int64, *time.Time, error)
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
	return model.Map[Entity, Model](Make)(getByIdProvider(id)(p.db.WithContext(p.ctx)))
}

// ByNpcIdProvider returns a provider for retrieving an NPC conversation by NPC ID
func (p *ProcessorImpl) ByNpcIdProvider(npcId uint32) model.Provider[Model] {
	return model.Map[Entity, Model](Make)(getByNpcIdProvider(npcId)(p.db.WithContext(p.ctx)))
}

// AllProvider returns a provider for retrieving all NPC conversations
func (p *ProcessorImpl) AllProvider() model.Provider[[]Model] {
	return model.SliceMap[Entity, Model](Make)(getAllProvider(p.db.WithContext(p.ctx)))(model.ParallelMap())
}

// AllByNpcIdProvider returns a provider for retrieving all NPC conversations for a specific NPC ID
func (p *ProcessorImpl) AllByNpcIdProvider(npcId uint32) model.Provider[[]Model] {
	return model.SliceMap[Entity, Model](Make)(getAllByNpcIdProvider(npcId)(p.db.WithContext(p.ctx)))(model.ParallelMap())
}

// createWithSkipTracking is the seed-time variant of Create: it runs the same
// txn (conversation insert + recipe rebuild) but returns the rebuild's skip
// information so the seed loop can accumulate skips across conversations.
func (p *ProcessorImpl) createWithSkipTracking(m Model, result *SeedResult) (Model, error) {
	var saved Model
	err := p.db.WithContext(p.ctx).Transaction(func(tx *gorm.DB) error {
		created, err := createNpcConversation(tx)(p.t.Id())(m)
		if err != nil {
			return err
		}
		rebuild, err := recipe.NewProcessor(p.l, p.ctx, p.db).RebuildForConversation(tx)(created.NpcId(), created.Id(), created.States())
		if err != nil {
			return err
		}
		result.SkippedRecipes += rebuild.Skipped
		result.SkippedRecipeDetails = append(result.SkippedRecipeDetails, rebuild.SkippedDetails...)
		saved = created
		return nil
	})
	return saved, err
}

// Create creates a new NPC conversation and rebuilds its derived recipe rows
// inside the same transaction.
func (p *ProcessorImpl) Create(m Model) (Model, error) {
	p.l.Debugf("Creating NPC conversation for NPC [%d]", m.NpcId())

	var result Model
	err := p.db.WithContext(p.ctx).Transaction(func(tx *gorm.DB) error {
		created, err := createNpcConversation(tx)(p.t.Id())(m)
		if err != nil {
			return err
		}
		if _, err := recipe.NewProcessor(p.l, p.ctx, p.db).RebuildForConversation(tx)(created.NpcId(), created.Id(), created.States()); err != nil {
			return err
		}
		result = created
		return nil
	})
	if err != nil {
		p.l.WithError(err).Errorf("Failed to create NPC conversation for NPC [%d]", m.NpcId())
		return Model{}, err
	}
	return result, nil
}

// Update updates an existing NPC conversation and rebuilds its derived recipe
// rows inside the same transaction.
func (p *ProcessorImpl) Update(id uuid.UUID, m Model) (Model, error) {
	p.l.Debugf("Updating NPC conversation [%s]", id)

	var result Model
	err := p.db.WithContext(p.ctx).Transaction(func(tx *gorm.DB) error {
		updated, err := updateNpcConversation(tx)(id)(m)
		if err != nil {
			return err
		}
		if _, err := recipe.NewProcessor(p.l, p.ctx, p.db).RebuildForConversation(tx)(updated.NpcId(), updated.Id(), updated.States()); err != nil {
			return err
		}
		result = updated
		return nil
	})
	if err != nil {
		p.l.WithError(err).Errorf("Failed to update NPC conversation [%s]", id)
		return Model{}, err
	}
	return result, nil
}

// Delete deletes an NPC conversation and the recipe rows derived from it,
// inside the same transaction.
func (p *ProcessorImpl) Delete(id uuid.UUID) error {
	p.l.Debugf("Deleting NPC conversation [%s]", id)

	err := p.db.WithContext(p.ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := recipe.NewProcessor(p.l, p.ctx, p.db).RebuildForConversation(tx)(0, id, nil); err != nil {
			return err
		}
		return deleteNpcConversation(tx)(id)
	})
	if err != nil {
		p.l.WithError(err).Errorf("Failed to delete NPC conversation [%s]", id)
		return err
	}
	return nil
}

// DeleteAllForTenant deletes every NPC conversation for the active tenant and
// every derived recipe row, inside the same transaction.
func (p *ProcessorImpl) DeleteAllForTenant() (int64, error) {
	p.l.Debugf("Deleting all NPC conversations for tenant [%s]", p.t.Id())

	var count int64
	err := p.db.WithContext(p.ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := recipe.NewProcessor(p.l, p.ctx, p.db).ClearForTenant(tx); err != nil {
			return err
		}
		c, err := deleteAllNpcConversations(tx)
		if err != nil {
			return err
		}
		count = c
		return nil
	})
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
		m, err := Extract(rm)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("npc_%d: failed to extract model: %v", rm.NpcId, err))
			result.FailedCount++
			continue
		}

		if _, err := p.createWithSkipTracking(m, &result); err != nil {
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

// Count returns the number of NPC conversations for the current tenant and the max updated_at timestamp.
// The tenant filter is applied automatically via the registered tenant callbacks on the GORM context.
func (p *ProcessorImpl) Count() (int64, *time.Time, error) {
	var count int64
	if err := p.db.WithContext(p.ctx).Model(&Entity{}).Count(&count).Error; err != nil {
		return 0, nil, err
	}
	if count == 0 {
		return 0, nil, nil
	}
	row := p.db.WithContext(p.ctx).Model(&Entity{}).Select("MAX(updated_at)").Row()
	var raw sql.NullString
	if err := row.Scan(&raw); err != nil {
		return 0, nil, err
	}
	if !raw.Valid || raw.String == "" {
		return count, nil, nil
	}
	t, err := parseDBTime(raw.String)
	if err != nil || t.IsZero() {
		return count, nil, nil
	}
	return count, &t, nil
}

func parseDBTime(s string) (time.Time, error) {
	formats := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.999999999 -0700 MST",
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, nil
}
