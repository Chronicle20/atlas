package quest

import (
	"atlas-npc-conversations/conversation/quest/status"
	"context"
	"errors"
	"fmt"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Processor interface {
	// Create creates a new quest conversation
	Create(model Model) (Model, error)

	// Update updates an existing quest conversation
	Update(id uuid.UUID, model Model) (Model, error)

	// Delete deletes a quest conversation
	Delete(id uuid.UUID) error

	// ByIdProvider returns a provider for retrieving a quest conversation by ID
	ByIdProvider(id uuid.UUID) model.Provider[Model]

	// ByQuestIdProvider returns a provider for retrieving a quest conversation by quest ID
	ByQuestIdProvider(questId uint32) model.Provider[Model]

	// AllProvider returns a provider for retrieving all quest conversations
	AllProvider() model.Provider[[]Model]

	// DeleteAllForTenant deletes all quest conversations for the current tenant
	DeleteAllForTenant() (int64, error)

	// Seed clears existing quest conversations and loads them from the quest-conversations directory
	Seed() (SeedResult, error)

	// GetStateMachineForCharacter returns the appropriate state machine for a character's quest status
	// Returns startStateMachine for NOT_STARTED quests, endStateMachine for STARTED quests
	// Returns error for COMPLETED quests or if quest conversation not found
	GetStateMachineForCharacter(questId uint32, characterId uint32) (StateMachine, error)
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

// ByIdProvider returns a provider for retrieving a quest conversation by ID
func (p *ProcessorImpl) ByIdProvider(id uuid.UUID) model.Provider[Model] {
	return model.Map[Entity, Model](Make)(getByIdProvider(id)(p.db.WithContext(p.ctx)))
}

// ByQuestIdProvider returns a provider for retrieving a quest conversation by quest ID
func (p *ProcessorImpl) ByQuestIdProvider(questId uint32) model.Provider[Model] {
	return model.Map[Entity, Model](Make)(getByQuestIdProvider(questId)(p.db.WithContext(p.ctx)))
}

// AllProvider returns a provider for retrieving all quest conversations
func (p *ProcessorImpl) AllProvider() model.Provider[[]Model] {
	return model.SliceMap[Entity, Model](Make)(getAllProvider(p.db.WithContext(p.ctx)))(model.ParallelMap())
}

// Create creates a new quest conversation
func (p *ProcessorImpl) Create(m Model) (Model, error) {
	p.l.Debugf("Creating quest conversation for quest [%d]", m.QuestId())

	result, err := createQuestConversation(p.db.WithContext(p.ctx))(p.t.Id())(m)
	if err != nil {
		p.l.WithError(err).Errorf("Failed to create quest conversation for quest [%d]", m.QuestId())
		return Model{}, err
	}
	return result, nil
}

// Update updates an existing quest conversation
func (p *ProcessorImpl) Update(id uuid.UUID, m Model) (Model, error) {
	p.l.Debugf("Updating quest conversation [%s]", id)

	result, err := updateQuestConversation(p.db.WithContext(p.ctx))(id)(m)
	if err != nil {
		p.l.WithError(err).Errorf("Failed to update quest conversation [%s]", id)
		return Model{}, err
	}
	return result, nil
}

// Delete deletes a quest conversation
func (p *ProcessorImpl) Delete(id uuid.UUID) error {
	p.l.Debugf("Deleting quest conversation [%s]", id)

	err := deleteQuestConversation(p.db.WithContext(p.ctx))(id)
	if err != nil {
		p.l.WithError(err).Errorf("Failed to delete quest conversation [%s]", id)
		return err
	}
	return nil
}

// DeleteAllForTenant deletes all quest conversations for the current tenant
func (p *ProcessorImpl) DeleteAllForTenant() (int64, error) {
	p.l.Debugf("Deleting all quest conversations for tenant [%s]", p.t.Id())

	count, err := deleteAllQuestConversations(p.db.WithContext(p.ctx))
	if err != nil {
		p.l.WithError(err).Errorf("Failed to delete all quest conversations for tenant [%s]", p.t.Id())
		return 0, err
	}
	return count, nil
}

// Seed clears existing quest conversations and loads them from the quest-conversations directory
func (p *ProcessorImpl) Seed() (SeedResult, error) {
	p.l.Infof("Seeding quest conversations for tenant [%s]", p.t.Id())

	result := SeedResult{}

	// Delete all existing quest conversations for this tenant
	deletedCount, err := p.DeleteAllForTenant()
	if err != nil {
		return result, fmt.Errorf("failed to clear existing quest conversations: %w", err)
	}
	result.DeletedCount = int(deletedCount)

	// Load quest conversation files from the filesystem
	models, loadErrors := LoadQuestConversationFiles()

	// Track load errors
	for _, err := range loadErrors {
		result.Errors = append(result.Errors, err.Error())
		result.FailedCount++
	}

	// Create each quest conversation
	for _, rm := range models {
		// Extract domain model from REST model
		m, err := Extract(rm)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("quest_%d: failed to extract model: %v", rm.QuestId, err))
			result.FailedCount++
			continue
		}

		// Create the quest conversation
		_, err = p.Create(m)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("quest_%d: failed to create: %v", rm.QuestId, err))
			result.FailedCount++
			continue
		}

		result.CreatedCount++
	}

	p.l.Infof("Seed complete for tenant [%s]: deleted=%d, created=%d, failed=%d",
		p.t.Id(), result.DeletedCount, result.CreatedCount, result.FailedCount)

	return result, nil
}

// GetStateMachineForCharacter returns the appropriate state machine for a character's quest status
func (p *ProcessorImpl) GetStateMachineForCharacter(questId uint32, characterId uint32) (StateMachine, error) {
	p.l.Debugf("Getting state machine for quest [%d] character [%d]", questId, characterId)

	// Get the quest conversation
	questConversation, err := p.ByQuestIdProvider(questId)()
	if err != nil {
		p.l.WithError(err).Errorf("Failed to retrieve quest conversation for quest [%d]", questId)
		return StateMachine{}, fmt.Errorf("quest conversation not found: %w", err)
	}

	// Get the character's quest status from atlas-quest service
	questStatus, err := status.RequestByCharacterAndQuest(characterId, questId)(p.l, p.ctx)
	if err != nil {
		// If quest status not found, treat as NOT_STARTED
		p.l.Debugf("Quest status not found for character [%d] quest [%d], treating as NOT_STARTED", characterId, questId)
		return questConversation.StartStateMachine(), nil
	}

	// Route to appropriate state machine based on quest status
	switch {
	case questStatus.IsNotStarted():
		p.l.Debugf("Quest [%d] is NOT_STARTED, using startStateMachine", questId)
		return questConversation.StartStateMachine(), nil

	case questStatus.IsStarted():
		p.l.Debugf("Quest [%d] is STARTED, using endStateMachine", questId)
		if !questConversation.HasEndStateMachine() {
			p.l.Warnf("Quest [%d] is STARTED but has no endStateMachine defined", questId)
			return StateMachine{}, errors.New("quest is in progress but no completion dialogue defined")
		}
		return *questConversation.EndStateMachine(), nil

	case questStatus.IsCompleted():
		p.l.Debugf("Quest [%d] is already COMPLETED", questId)
		return StateMachine{}, errors.New("quest is already completed")

	default:
		p.l.Errorf("Unknown quest status [%d] for quest [%d]", questStatus.State, questId)
		return StateMachine{}, fmt.Errorf("unknown quest status: %d", questStatus.State)
	}
}
