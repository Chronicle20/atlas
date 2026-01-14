package quest

import (
	"context"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

// Processor defines the interface for quest data processing
type Processor interface {
	GetQuestState(characterId uint32, questId uint32) model.Provider[State]
	GetQuestProgress(characterId uint32, questId uint32, infoNumber uint32) model.Provider[int]
	GetQuest(characterId uint32, questId uint32) model.Provider[Model]
	GetQuestsByCharacter(characterId uint32) model.Provider[[]Model]
	GetStartedQuestsByCharacter(characterId uint32) model.Provider[[]Model]
	GetCompletedQuestsByCharacter(characterId uint32) model.Provider[[]Model]
}

// ProcessorImpl implements the Processor interface
type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

// NewProcessor creates a new quest processor
func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
}

// GetQuestState returns the state of a quest for a character
func (p *ProcessorImpl) GetQuestState(characterId uint32, questId uint32) model.Provider[State] {
	return func() (State, error) {
		questProvider := requests.Provider[RestModel, Model](p.l, p.ctx)(requestById(characterId, questId), Extract)
		quest, err := questProvider()
		if err != nil {
			p.l.WithError(err).Errorf("Failed to get quest state for character %d, quest %d", characterId, questId)
			return StateNotStarted, err
		}
		return quest.State(), nil
	}
}

// GetQuestProgress returns the progress of a quest for a specific info number
func (p *ProcessorImpl) GetQuestProgress(characterId uint32, questId uint32, infoNumber uint32) model.Provider[int] {
	return func() (int, error) {
		questProvider := requests.Provider[RestModel, Model](p.l, p.ctx)(requestById(characterId, questId), Extract)
		quest, err := questProvider()
		if err != nil {
			p.l.WithError(err).Errorf("Failed to get quest progress for character %d, quest %d, infoNumber %d", characterId, questId, infoNumber)
			return 0, err
		}
		if progress, found := quest.GetProgress(infoNumber); found {
			return progress.ProgressInt(), nil
		}
		return 0, nil
	}
}

// GetQuest returns the complete quest model for a character
func (p *ProcessorImpl) GetQuest(characterId uint32, questId uint32) model.Provider[Model] {
	return func() (Model, error) {
		questProvider := requests.Provider[RestModel, Model](p.l, p.ctx)(requestById(characterId, questId), Extract)
		quest, err := questProvider()
		if err != nil {
			p.l.WithError(err).Errorf("Failed to get quest data for character %d, quest %d", characterId, questId)
			return NewModel(characterId, questId, StateNotStarted), err
		}
		return quest, nil
	}
}

// GetQuestsByCharacter returns all quests for a character
func (p *ProcessorImpl) GetQuestsByCharacter(characterId uint32) model.Provider[[]Model] {
	return func() ([]Model, error) {
		questsProvider := requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestByCharacter(characterId), Extract, model.Filters[Model]())
		quests, err := questsProvider()
		if err != nil {
			p.l.WithError(err).Errorf("Failed to get quests for character %d", characterId)
			return nil, err
		}
		return quests, nil
	}
}

// GetStartedQuestsByCharacter returns all started quests for a character
func (p *ProcessorImpl) GetStartedQuestsByCharacter(characterId uint32) model.Provider[[]Model] {
	return func() ([]Model, error) {
		questsProvider := requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestStartedByCharacter(characterId), Extract, model.Filters[Model]())
		quests, err := questsProvider()
		if err != nil {
			p.l.WithError(err).Errorf("Failed to get started quests for character %d", characterId)
			return nil, err
		}
		return quests, nil
	}
}

// GetCompletedQuestsByCharacter returns all completed quests for a character
func (p *ProcessorImpl) GetCompletedQuestsByCharacter(characterId uint32) model.Provider[[]Model] {
	return func() ([]Model, error) {
		questsProvider := requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestCompletedByCharacter(characterId), Extract, model.Filters[Model]())
		quests, err := questsProvider()
		if err != nil {
			p.l.WithError(err).Errorf("Failed to get completed quests for character %d", characterId)
			return nil, err
		}
		return quests, nil
	}
}