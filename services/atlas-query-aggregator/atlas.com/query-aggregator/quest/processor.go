package quest

import (
	"context"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

// Processor defines the interface for quest data processing
type Processor interface {
	GetQuestStatus(characterId uint32, questId uint32) model.Provider[QuestStatus]
	GetQuestProgress(characterId uint32, questId uint32, step string) model.Provider[int]
	GetQuest(characterId uint32, questId uint32) model.Provider[Model]
}

// processor implements the Processor interface
type processor struct {
	l   logrus.FieldLogger
	ctx context.Context
}

// NewProcessor creates a new quest processor
func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &processor{
		l:   l,
		ctx: ctx,
	}
}

// GetQuestStatus returns the status of a quest for a character
func (p *processor) GetQuestStatus(characterId uint32, questId uint32) model.Provider[QuestStatus] {
	return func() (QuestStatus, error) {
		questProvider := requests.Provider[RestModel, Model](p.l, p.ctx)(requestById(characterId, questId), Extract)
		quest, err := questProvider()
		if err != nil {
			p.l.WithError(err).Errorf("Failed to get quest status for character %d, quest %d", characterId, questId)
			return UNDEFINED, err
		}
		return quest.Status(), nil
	}
}

// GetQuestProgress returns the progress of a quest step for a character
func (p *processor) GetQuestProgress(characterId uint32, questId uint32, step string) model.Provider[int] {
	return func() (int, error) {
		questProvider := requests.Provider[RestModel, Model](p.l, p.ctx)(requestById(characterId, questId), Extract)
		quest, err := questProvider()
		if err != nil {
			p.l.WithError(err).Errorf("Failed to get quest progress for character %d, quest %d, step %s", characterId, questId, step)
			return 0, err
		}
		return quest.Progress(step), nil
	}
}

// GetQuest returns the complete quest model for a character
func (p *processor) GetQuest(characterId uint32, questId uint32) model.Provider[Model] {
	return func() (Model, error) {
		questProvider := requests.Provider[RestModel, Model](p.l, p.ctx)(requestById(characterId, questId), Extract)
		quest, err := questProvider()
		if err != nil {
			p.l.WithError(err).Errorf("Failed to get quest data for character %d, quest %d", characterId, questId)
			return NewModel(questId, UNDEFINED), err
		}
		return quest, nil
	}
}