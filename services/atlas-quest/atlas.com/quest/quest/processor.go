package quest

import (
	"atlas-quest/database"
	"context"
	"errors"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Processor interface {
	WithTransaction(*gorm.DB) Processor
	ByIdProvider(id uint32) model.Provider[Model]
	ByCharacterIdProvider(characterId uint32) model.Provider[[]Model]
	ByCharacterIdAndQuestIdProvider(characterId uint32, questId uint32) model.Provider[Model]
	ByCharacterIdAndStateProvider(characterId uint32, state State) model.Provider[[]Model]
	GetById(id uint32) (Model, error)
	GetByCharacterId(characterId uint32) ([]Model, error)
	GetByCharacterIdAndQuestId(characterId uint32, questId uint32) (Model, error)
	GetByCharacterIdAndState(characterId uint32, state State) ([]Model, error)
	Start(characterId uint32, questId uint32) (Model, error)
	Complete(characterId uint32, questId uint32) error
	Forfeit(characterId uint32, questId uint32) error
	SetProgress(characterId uint32, questId uint32, infoNumber uint32, progress string) error
	DeleteByCharacterId(characterId uint32) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	db  *gorm.DB
	t   tenant.Model
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		db:  db,
		t:   tenant.MustFromContext(ctx),
	}
}

func (p *ProcessorImpl) WithTransaction(tx *gorm.DB) Processor {
	return &ProcessorImpl{
		l:   p.l,
		ctx: p.ctx,
		db:  tx,
		t:   p.t,
	}
}

func (p *ProcessorImpl) ByIdProvider(id uint32) model.Provider[Model] {
	return model.Map(Make)(byIdEntityProvider(p.t.Id(), id)(p.db))
}

func (p *ProcessorImpl) ByCharacterIdProvider(characterId uint32) model.Provider[[]Model] {
	return model.SliceMap(Make)(byCharacterIdEntityProvider(p.t.Id(), characterId)(p.db))()
}

func (p *ProcessorImpl) ByCharacterIdAndQuestIdProvider(characterId uint32, questId uint32) model.Provider[Model] {
	return model.Map(Make)(byCharacterIdAndQuestIdEntityProvider(p.t.Id(), characterId, questId)(p.db))
}

func (p *ProcessorImpl) ByCharacterIdAndStateProvider(characterId uint32, state State) model.Provider[[]Model] {
	return model.SliceMap(Make)(byCharacterIdAndStateEntityProvider(p.t.Id(), characterId, state)(p.db))()
}

func (p *ProcessorImpl) GetById(id uint32) (Model, error) {
	return p.ByIdProvider(id)()
}

func (p *ProcessorImpl) GetByCharacterId(characterId uint32) ([]Model, error) {
	return p.ByCharacterIdProvider(characterId)()
}

func (p *ProcessorImpl) GetByCharacterIdAndQuestId(characterId uint32, questId uint32) (Model, error) {
	return p.ByCharacterIdAndQuestIdProvider(characterId, questId)()
}

func (p *ProcessorImpl) GetByCharacterIdAndState(characterId uint32, state State) ([]Model, error) {
	return p.ByCharacterIdAndStateProvider(characterId, state)()
}

func (p *ProcessorImpl) Start(characterId uint32, questId uint32) (Model, error) {
	// Check if quest already exists
	existing, err := p.GetByCharacterIdAndQuestId(characterId, questId)
	if err == nil {
		// Quest already exists
		if existing.State() == StateStarted {
			p.l.Debugf("Quest [%d] already started for character [%d].", questId, characterId)
			return existing, nil
		}
		if existing.State() == StateCompleted {
			p.l.Debugf("Quest [%d] already completed for character [%d].", questId, characterId)
			return Model{}, errors.New("quest already completed")
		}
	}

	var m Model
	txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
		var err error
		m, err = create(tx, p.t, characterId, questId)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to create quest [%d] for character [%d].", questId, characterId)
			return err
		}
		return nil
	})
	if txErr != nil {
		return Model{}, txErr
	}

	p.l.Debugf("Started quest [%d] for character [%d].", questId, characterId)
	return m, nil
}

func (p *ProcessorImpl) Complete(characterId uint32, questId uint32) error {
	existing, err := p.GetByCharacterIdAndQuestId(characterId, questId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to find quest [%d] for character [%d] to complete.", questId, characterId)
		return err
	}

	if existing.State() == StateCompleted {
		p.l.Debugf("Quest [%d] already completed for character [%d].", questId, characterId)
		return nil
	}

	if existing.State() != StateStarted {
		p.l.Debugf("Quest [%d] not in started state for character [%d].", questId, characterId)
		return errors.New("quest not started")
	}

	txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
		return updateState(tx, p.t.Id(), existing.Id(), StateCompleted)
	})
	if txErr != nil {
		p.l.WithError(txErr).Errorf("Unable to complete quest [%d] for character [%d].", questId, characterId)
		return txErr
	}

	p.l.Debugf("Completed quest [%d] for character [%d].", questId, characterId)
	return nil
}

func (p *ProcessorImpl) Forfeit(characterId uint32, questId uint32) error {
	existing, err := p.GetByCharacterIdAndQuestId(characterId, questId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to find quest [%d] for character [%d] to forfeit.", questId, characterId)
		return err
	}

	if existing.State() != StateStarted {
		p.l.Debugf("Quest [%d] not in started state for character [%d], cannot forfeit.", questId, characterId)
		return errors.New("quest not started")
	}

	txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
		return deleteWithProgress(tx, p.t.Id(), existing.Id())
	})
	if txErr != nil {
		p.l.WithError(txErr).Errorf("Unable to forfeit quest [%d] for character [%d].", questId, characterId)
		return txErr
	}

	p.l.Debugf("Forfeited quest [%d] for character [%d].", questId, characterId)
	return nil
}

func (p *ProcessorImpl) SetProgress(characterId uint32, questId uint32, infoNumber uint32, progressValue string) error {
	existing, err := p.GetByCharacterIdAndQuestId(characterId, questId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to find quest [%d] for character [%d] to set progress.", questId, characterId)
		return err
	}

	if existing.State() != StateStarted {
		p.l.Debugf("Quest [%d] not in started state for character [%d], cannot set progress.", questId, characterId)
		return errors.New("quest not started")
	}

	txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
		return setProgress(tx, p.t.Id(), existing.Id(), infoNumber, progressValue)
	})
	if txErr != nil {
		p.l.WithError(txErr).Errorf("Unable to set progress for quest [%d] for character [%d].", questId, characterId)
		return txErr
	}

	p.l.Debugf("Set progress for quest [%d], infoNumber [%d] for character [%d].", questId, infoNumber, characterId)
	return nil
}

func (p *ProcessorImpl) DeleteByCharacterId(characterId uint32) error {
	txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
		return deleteByCharacterIdWithProgress(tx, p.t.Id(), characterId)
	})
	if txErr != nil {
		p.l.WithError(txErr).Errorf("Unable to delete quests for character [%d].", characterId)
		return txErr
	}

	p.l.Debugf("Deleted all quests for character [%d].", characterId)
	return nil
}
