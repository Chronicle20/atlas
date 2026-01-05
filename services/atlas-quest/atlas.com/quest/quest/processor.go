package quest

import (
	"atlas-quest/database"
	dataquest "atlas-quest/data/quest"
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

var (
	ErrQuestAlreadyStarted   = errors.New("quest already started")
	ErrQuestAlreadyCompleted = errors.New("quest already completed")
	ErrQuestNotStarted       = errors.New("quest not started")
	ErrIntervalNotElapsed    = errors.New("interval has not elapsed since last completion")
	ErrQuestExpired          = errors.New("quest has expired")
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
	// StartChained starts a quest as part of a chain (skips interval check)
	StartChained(characterId uint32, questId uint32) (Model, error)
	// Complete completes a quest and returns the next quest ID if this is part of a chain (0 if no chain)
	Complete(characterId uint32, questId uint32) (uint32, error)
	Forfeit(characterId uint32, questId uint32) error
	SetProgress(characterId uint32, questId uint32, infoNumber uint32, progress string) error
	DeleteByCharacterId(characterId uint32) error
	// GetQuestDefinition fetches the quest definition from atlas-data
	GetQuestDefinition(questId uint32) (dataquest.RestModel, error)
	// CheckAutoComplete checks if a quest can be auto-completed and completes it if requirements are met
	// Returns the next quest ID if this is part of a chain (0 if no chain), and whether it was completed
	CheckAutoComplete(characterId uint32, questId uint32) (uint32, bool, error)
	// CheckAutoStart checks for auto-start quests that should start for a character on a given map
	// Returns the list of quest IDs that were auto-started
	CheckAutoStart(characterId uint32, mapId uint32) ([]uint32, error)
}

type ProcessorImpl struct {
	l            logrus.FieldLogger
	ctx          context.Context
	db           *gorm.DB
	t            tenant.Model
	dataProcessor dataquest.Processor
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	return &ProcessorImpl{
		l:            l,
		ctx:          ctx,
		db:           db,
		t:            tenant.MustFromContext(ctx),
		dataProcessor: dataquest.NewProcessor(l, ctx),
	}
}

func (p *ProcessorImpl) WithTransaction(tx *gorm.DB) Processor {
	return &ProcessorImpl{
		l:            p.l,
		ctx:          p.ctx,
		db:           tx,
		t:            p.t,
		dataProcessor: p.dataProcessor,
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
	// Fetch quest definition to check interval and time limit
	questDef, err := p.dataProcessor.GetQuestDefinition(questId)
	if err != nil {
		p.l.WithError(err).Warnf("Unable to fetch quest definition for quest [%d], proceeding without interval/time limit checks.", questId)
		// Continue without quest definition - we can still start the quest
		questDef = dataquest.RestModel{}
	}

	// Check if quest already exists
	existing, err := p.GetByCharacterIdAndQuestId(characterId, questId)
	if err == nil {
		// Quest already exists
		if existing.State() == StateStarted {
			p.l.Debugf("Quest [%d] already started for character [%d].", questId, characterId)
			return existing, ErrQuestAlreadyStarted
		}
		if existing.State() == StateCompleted {
			// Check if this is a repeatable quest (has interval requirement)
			interval := questDef.StartRequirements.Interval
			if interval > 0 {
				// Check if enough time has elapsed since last completion
				elapsed := time.Since(existing.CompletedAt())
				requiredDuration := time.Duration(interval) * time.Minute
				if elapsed < requiredDuration {
					p.l.Debugf("Quest [%d] interval not elapsed for character [%d]. Elapsed: %v, Required: %v", questId, characterId, elapsed, requiredDuration)
					return Model{}, ErrIntervalNotElapsed
				}
				// Interval has elapsed, we can restart this quest
				p.l.Debugf("Quest [%d] interval elapsed for character [%d], restarting.", questId, characterId)
			} else {
				p.l.Debugf("Quest [%d] already completed for character [%d] (not repeatable).", questId, characterId)
				return Model{}, ErrQuestAlreadyCompleted
			}
		}
	}

	// Calculate expiration time for time-limited quests
	var expirationTime time.Time
	timeLimit := questDef.TimeLimit
	if timeLimit == 0 {
		timeLimit = questDef.TimeLimit2
	}
	if timeLimit > 0 {
		// TimeLimit is in seconds
		expirationTime = time.Now().Add(time.Duration(timeLimit) * time.Second)
		p.l.Debugf("Quest [%d] has time limit of %d seconds, expires at %v", questId, timeLimit, expirationTime)
	}

	// Collect mob and map requirements for progress initialization
	var mobIds []uint32
	var mapIds []uint32

	// Get mob requirements from end requirements (completion requirements)
	for _, mob := range questDef.EndRequirements.Mobs {
		mobIds = append(mobIds, mob.Id)
	}

	// Get map requirements from end requirements (medal/fieldEnter quests)
	mapIds = append(mapIds, questDef.EndRequirements.FieldEnter...)

	var m Model
	txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
		var err error
		// If quest was previously completed (repeatable), restart it
		if existing.Id() > 0 && existing.State() == StateCompleted {
			m, err = restart(tx, p.t.Id(), existing.Id(), expirationTime)
		} else {
			m, err = create(tx, p.t, characterId, questId, expirationTime)
		}
		if err != nil {
			p.l.WithError(err).Errorf("Unable to create/restart quest [%d] for character [%d].", questId, characterId)
			return err
		}

		// Initialize progress for mob kills and map visits
		if len(mobIds) > 0 || len(mapIds) > 0 {
			if err = initializeProgress(tx, m.Id(), mobIds, mapIds); err != nil {
				p.l.WithError(err).Errorf("Unable to initialize progress for quest [%d] for character [%d].", questId, characterId)
				return err
			}
		}

		return nil
	})
	if txErr != nil {
		return Model{}, txErr
	}

	p.l.Debugf("Started quest [%d] for character [%d].", questId, characterId)
	return m, nil
}

func (p *ProcessorImpl) StartChained(characterId uint32, questId uint32) (Model, error) {
	// Chained quests skip interval checking
	// Fetch quest definition for time limit only
	questDef, err := p.dataProcessor.GetQuestDefinition(questId)
	if err != nil {
		p.l.WithError(err).Warnf("Unable to fetch quest definition for quest [%d], proceeding without time limit.", questId)
		questDef = dataquest.RestModel{}
	}

	// Check if quest already exists
	existing, err := p.GetByCharacterIdAndQuestId(characterId, questId)
	if err == nil {
		if existing.State() == StateStarted {
			return existing, nil
		}
		// For chained quests, we allow restarting even if completed
	}

	// Calculate expiration time for time-limited quests
	var expirationTime time.Time
	timeLimit := questDef.TimeLimit
	if timeLimit == 0 {
		timeLimit = questDef.TimeLimit2
	}
	if timeLimit > 0 {
		expirationTime = time.Now().Add(time.Duration(timeLimit) * time.Second)
	}

	// Collect mob and map requirements for progress initialization
	var mobIds []uint32
	var mapIds []uint32
	for _, mob := range questDef.EndRequirements.Mobs {
		mobIds = append(mobIds, mob.Id)
	}
	mapIds = append(mapIds, questDef.EndRequirements.FieldEnter...)

	var m Model
	txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
		var err error
		if existing.Id() > 0 {
			m, err = restart(tx, p.t.Id(), existing.Id(), expirationTime)
		} else {
			m, err = create(tx, p.t, characterId, questId, expirationTime)
		}
		if err != nil {
			return err
		}

		// Initialize progress for mob kills and map visits
		if len(mobIds) > 0 || len(mapIds) > 0 {
			if err = initializeProgress(tx, m.Id(), mobIds, mapIds); err != nil {
				return err
			}
		}

		return nil
	})
	if txErr != nil {
		return Model{}, txErr
	}

	p.l.Debugf("Started chained quest [%d] for character [%d].", questId, characterId)
	return m, nil
}

func (p *ProcessorImpl) Complete(characterId uint32, questId uint32) (uint32, error) {
	existing, err := p.GetByCharacterIdAndQuestId(characterId, questId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to find quest [%d] for character [%d] to complete.", questId, characterId)
		return 0, err
	}

	if existing.State() == StateCompleted {
		p.l.Debugf("Quest [%d] already completed for character [%d].", questId, characterId)
		return 0, nil
	}

	if existing.State() != StateStarted {
		p.l.Debugf("Quest [%d] not in started state for character [%d].", questId, characterId)
		return 0, ErrQuestNotStarted
	}

	// Check if quest has expired
	if existing.IsExpired() {
		p.l.Debugf("Quest [%d] has expired for character [%d].", questId, characterId)
		return 0, ErrQuestExpired
	}

	txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
		return completeQuest(tx, p.t.Id(), existing.Id())
	})
	if txErr != nil {
		p.l.WithError(txErr).Errorf("Unable to complete quest [%d] for character [%d].", questId, characterId)
		return 0, txErr
	}

	p.l.Debugf("Completed quest [%d] for character [%d].", questId, characterId)

	// Check for quest chain (next quest)
	questDef, err := p.dataProcessor.GetQuestDefinition(questId)
	if err != nil {
		p.l.WithError(err).Warnf("Unable to fetch quest definition for quest [%d] to check for chain.", questId)
		return 0, nil
	}

	nextQuestId := questDef.EndActions.NextQuest
	if nextQuestId > 0 {
		p.l.Debugf("Quest [%d] has next quest [%d] in chain.", questId, nextQuestId)
	}

	return nextQuestId, nil
}

func (p *ProcessorImpl) Forfeit(characterId uint32, questId uint32) error {
	existing, err := p.GetByCharacterIdAndQuestId(characterId, questId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to find quest [%d] for character [%d] to forfeit.", questId, characterId)
		return err
	}

	if existing.State() != StateStarted {
		p.l.Debugf("Quest [%d] not in started state for character [%d], cannot forfeit.", questId, characterId)
		return ErrQuestNotStarted
	}

	txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
		return forfeitQuest(tx, p.t.Id(), existing.Id())
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

func (p *ProcessorImpl) GetQuestDefinition(questId uint32) (dataquest.RestModel, error) {
	return p.dataProcessor.GetQuestDefinition(questId)
}

func (p *ProcessorImpl) CheckAutoComplete(characterId uint32, questId uint32) (uint32, bool, error) {
	// Get the quest status
	existing, err := p.GetByCharacterIdAndQuestId(characterId, questId)
	if err != nil {
		return 0, false, err
	}

	if existing.State() != StateStarted {
		return 0, false, nil
	}

	// Fetch quest definition to check if it supports auto-complete
	questDef, err := p.dataProcessor.GetQuestDefinition(questId)
	if err != nil {
		p.l.WithError(err).Warnf("Unable to fetch quest definition for quest [%d] for auto-complete check.", questId)
		return 0, false, nil
	}

	if !questDef.AutoComplete {
		return 0, false, nil
	}

	// Check if all end requirements are met
	if !p.areEndRequirementsMet(existing, questDef) {
		return 0, false, nil
	}

	// All requirements met, complete the quest
	nextQuestId, err := p.Complete(characterId, questId)
	if err != nil {
		return 0, false, err
	}

	p.l.Infof("Auto-completed quest [%d] for character [%d].", questId, characterId)
	return nextQuestId, true, nil
}

func (p *ProcessorImpl) areEndRequirementsMet(q Model, questDef dataquest.RestModel) bool {
	// Check mob kill requirements
	for _, mobReq := range questDef.EndRequirements.Mobs {
		prog, found := q.GetProgress(mobReq.Id)
		if !found {
			return false
		}
		count := parseProgressValue(prog.Progress())
		if count < mobReq.Count {
			return false
		}
	}

	// Check map visit requirements (fieldEnter)
	for _, mapId := range questDef.EndRequirements.FieldEnter {
		prog, found := q.GetProgress(mapId)
		if !found {
			return false
		}
		if prog.Progress() != "1" {
			return false
		}
	}

	// Note: Item requirements are not checked here as they require fetching character inventory
	// which should be done by the calling service. Auto-complete for item-based quests
	// should be triggered externally when items are obtained.

	return true
}

func parseProgressValue(progress string) uint32 {
	if progress == "" {
		return 0
	}
	val, err := strconv.Atoi(progress)
	if err != nil {
		return 0
	}
	return uint32(val)
}

func (p *ProcessorImpl) CheckAutoStart(characterId uint32, mapId uint32) ([]uint32, error) {
	// Fetch all auto-start quests from atlas-data
	autoStartQuests, err := p.dataProcessor.GetAutoStartQuests(mapId)
	if err != nil {
		p.l.WithError(err).Warnf("Unable to fetch auto-start quests for map [%d].", mapId)
		return nil, nil
	}

	var startedQuests []uint32
	for _, questDef := range autoStartQuests {
		// Check if quest is already started or completed
		existing, err := p.GetByCharacterIdAndQuestId(characterId, questDef.Id)
		if err == nil {
			// Quest exists - check if we can restart it
			if existing.State() == StateStarted {
				continue // Already started
			}
			if existing.State() == StateCompleted {
				// Check if it's repeatable
				interval := questDef.StartRequirements.Interval
				if interval == 0 {
					continue // Not repeatable
				}
				elapsed := time.Since(existing.CompletedAt())
				if elapsed < time.Duration(interval)*time.Minute {
					continue // Interval not elapsed
				}
			}
		}

		// Start the quest
		_, err = p.Start(characterId, questDef.Id)
		if err != nil {
			if !errors.Is(err, ErrQuestAlreadyStarted) && !errors.Is(err, ErrQuestAlreadyCompleted) {
				p.l.WithError(err).Warnf("Unable to auto-start quest [%d] for character [%d].", questDef.Id, characterId)
			}
			continue
		}

		p.l.Infof("Auto-started quest [%d] for character [%d] on map [%d].", questDef.Id, characterId, mapId)
		startedQuests = append(startedQuests, questDef.Id)
	}

	return startedQuests, nil
}
