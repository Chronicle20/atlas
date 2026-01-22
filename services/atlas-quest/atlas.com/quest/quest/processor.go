package quest

import (
	"atlas-quest/database"
	dataquest "atlas-quest/data/quest"
	"atlas-quest/data/validation"
	"atlas-quest/kafka/message/saga"
	sagaproducer "atlas-quest/kafka/producer/saga"
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

var (
	ErrQuestAlreadyStarted      = errors.New("quest already started")
	ErrQuestAlreadyCompleted    = errors.New("quest already completed")
	ErrQuestNotStarted          = errors.New("quest not started")
	ErrIntervalNotElapsed       = errors.New("interval has not elapsed since last completion")
	ErrQuestExpired             = errors.New("quest has expired")
	ErrStartRequirementsNotMet  = errors.New("start requirements not met")
	ErrEndRequirementsNotMet    = errors.New("end requirements not met")
	ErrValidationFailed         = errors.New("validation request failed")
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
	// Start starts a quest with validation and processes start actions
	// Returns the quest model and any failed conditions (empty if validation passed)
	// field is needed for exp/meso start actions
	// If skipValidation is true, start requirements are not checked
	// transactionId is used for saga correlation (use uuid.Nil for non-saga initiated starts)
	Start(transactionId uuid.UUID, characterId uint32, questId uint32, f field.Model, skipValidation bool) (Model, []string, error)
	// StartChained starts a quest as part of a chain (skips interval check but still validates)
	StartChained(transactionId uuid.UUID, characterId uint32, questId uint32, f field.Model) (Model, error)
	// Complete completes a quest with validation and processes rewards via saga
	// Returns the next quest ID if this is part of a chain (0 if no chain)
	// field is needed for meso/exp/fame rewards
	// If skipValidation is true, end requirements are not checked
	// transactionId is used for saga correlation (use uuid.Nil for non-saga initiated completes)
	Complete(transactionId uuid.UUID, characterId uint32, questId uint32, f field.Model, skipValidation bool) (uint32, error)
	Forfeit(transactionId uuid.UUID, characterId uint32, questId uint32) error
	SetProgress(transactionId uuid.UUID, characterId uint32, questId uint32, infoNumber uint32, progress string) error
	DeleteByCharacterId(characterId uint32) error
	// GetQuestDefinition fetches the quest definition from atlas-data
	GetQuestDefinition(questId uint32) (dataquest.RestModel, error)
	// CheckAutoComplete checks if a quest can be auto-completed and completes it if requirements are met
	// Returns the next quest ID if this is part of a chain (0 if no chain), and whether it was completed
	CheckAutoComplete(characterId uint32, questId uint32, f field.Model) (uint32, bool, error)
	// CheckAutoStart checks for auto-start quests that should start for a character on a given map
	// Returns the list of quest IDs that were auto-started
	CheckAutoStart(characterId uint32, f field.Model) ([]uint32, error)
}

type ProcessorImpl struct {
	l                   logrus.FieldLogger
	ctx                 context.Context
	db                  *gorm.DB
	t                   tenant.Model
	dataProcessor       dataquest.Processor
	validationProcessor validation.Processor
	eventEmitter        EventEmitter
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	return &ProcessorImpl{
		l:                   l,
		ctx:                 ctx,
		db:                  db,
		t:                   tenant.MustFromContext(ctx),
		dataProcessor:       dataquest.NewProcessor(l, ctx),
		validationProcessor: validation.NewProcessor(l, ctx),
		eventEmitter:        NewKafkaEventEmitter(l, ctx),
	}
}

// NewProcessorWithDependencies creates a processor with custom dependencies (for testing)
func NewProcessorWithDependencies(l logrus.FieldLogger, ctx context.Context, db *gorm.DB, dataProc dataquest.Processor, validationProc validation.Processor, eventEmitter EventEmitter) Processor {
	return &ProcessorImpl{
		l:                   l,
		ctx:                 ctx,
		db:                  db,
		t:                   tenant.MustFromContext(ctx),
		dataProcessor:       dataProc,
		validationProcessor: validationProc,
		eventEmitter:        eventEmitter,
	}
}

func (p *ProcessorImpl) WithTransaction(tx *gorm.DB) Processor {
	return &ProcessorImpl{
		l:                   p.l,
		ctx:                 p.ctx,
		db:                  tx,
		t:                   p.t,
		dataProcessor:       p.dataProcessor,
		validationProcessor: p.validationProcessor,
		eventEmitter:        p.eventEmitter,
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

func (p *ProcessorImpl) Start(transactionId uuid.UUID, characterId uint32, questId uint32, f field.Model, skipValidation bool) (Model, []string, error) {
	// Fetch quest definition to check interval and time limit
	questDef, err := p.dataProcessor.GetQuestDefinition(questId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to fetch quest definition for quest [%d], cannot start quest.", questId)
		return Model{}, nil, fmt.Errorf("unable to fetch quest definition: %w", err)
	}

	return p.startWithDefinition(transactionId, characterId, questId, questDef, f, skipValidation)
}

// startWithDefinition starts a quest using the provided quest definition
// This is used internally when the quest definition is already available (e.g., from CheckAutoStart)
func (p *ProcessorImpl) startWithDefinition(transactionId uuid.UUID, characterId uint32, questId uint32, questDef dataquest.RestModel, f field.Model, skipValidation bool) (Model, []string, error) {
	// Validate start requirements (unless skipped)
	if !skipValidation {
		passed, failedConditions, err := p.validationProcessor.ValidateStartRequirements(characterId, questDef)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to validate start requirements for quest [%d] character [%d], failing closed.", questId, characterId)
			return Model{}, nil, ErrValidationFailed
		}
		if !passed {
			p.l.Debugf("Start requirements not met for quest [%d] character [%d]. Failed: %v", questId, characterId, failedConditions)
			return Model{}, failedConditions, ErrStartRequirementsNotMet
		}
	}

	// Start the quest (core logic)
	m, err := p.startCore(characterId, questId, questDef)
	if err != nil {
		return Model{}, nil, err
	}

	// Process start actions (items consumed, exp given on start, etc.)
	if err := p.processStartActions(characterId, questId, questDef, f); err != nil {
		p.l.WithError(err).Warnf("Unable to process start actions for quest [%d] character [%d].", questId, characterId)
		// Don't fail the quest start, just log the error
	}

	// Reload the quest to get the full progress after initialization
	updated, err := p.GetByCharacterIdAndQuestId(characterId, questId)
	if err != nil {
		p.l.WithError(err).Warnf("Unable to reload quest [%d] after start for character [%d].", questId, characterId)
		updated = m
	}

	// Emit quest started event
	if err := p.eventEmitter.EmitQuestStarted(transactionId, characterId, byte(f.WorldId()), questId, updated.ProgressString()); err != nil {
		p.l.WithError(err).Warnf("Unable to emit quest started event for quest [%d] character [%d].", questId, characterId)
	}

	return updated, nil, nil
}

// startCore handles the core quest start logic without validation or action processing
func (p *ProcessorImpl) startCore(characterId uint32, questId uint32, questDef dataquest.RestModel) (Model, error) {
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
	// Note: Item tracking is handled client-side, not server-side
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
		// If quest record exists (completed or forfeited), restart it; otherwise create new
		if existing.Id() > 0 {
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
			if err = initializeProgress(tx, p.t.Id(), m.Id(), mobIds, mapIds); err != nil {
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

func (p *ProcessorImpl) StartChained(transactionId uuid.UUID, characterId uint32, questId uint32, f field.Model) (Model, error) {
	// Chained quests skip interval checking but still validate and process start actions
	questDef, err := p.dataProcessor.GetQuestDefinition(questId)
	if err != nil {
		p.l.WithError(err).Warnf("Unable to fetch quest definition for quest [%d], proceeding without validation.", questId)
		questDef = dataquest.RestModel{}
	}

	// Check if quest already exists and is started
	existing, err := p.GetByCharacterIdAndQuestId(characterId, questId)
	if err == nil && existing.State() == StateStarted {
		return existing, nil
	}

	// Validate start requirements (chained quests still need to meet requirements)
	passed, _, err := p.validationProcessor.ValidateStartRequirements(characterId, questDef)
	if err != nil {
		p.l.WithError(err).Warnf("Unable to validate start requirements for chained quest [%d], proceeding anyway.", questId)
	} else if !passed {
		p.l.Warnf("Start requirements not met for chained quest [%d], proceeding anyway.", questId)
		// For chained quests, we proceed even if requirements aren't met
	}

	// Start the quest using startChainedCore (skips interval check)
	m, err := p.startChainedCore(characterId, questId, questDef)
	if err != nil {
		return Model{}, err
	}

	// Process start actions
	if err := p.processStartActions(characterId, questId, questDef, f); err != nil {
		p.l.WithError(err).Warnf("Unable to process start actions for chained quest [%d].", questId)
	}

	// Reload the quest to get the full progress after initialization
	updated, err := p.GetByCharacterIdAndQuestId(characterId, questId)
	if err != nil {
		p.l.WithError(err).Warnf("Unable to reload chained quest [%d] after start for character [%d].", questId, characterId)
		updated = m
	}

	// Emit quest started event
	if err := p.eventEmitter.EmitQuestStarted(transactionId, characterId, byte(f.WorldId()), questId, updated.ProgressString()); err != nil {
		p.l.WithError(err).Warnf("Unable to emit quest started event for chained quest [%d] character [%d].", questId, characterId)
	}

	p.l.Debugf("Started chained quest [%d] for character [%d].", questId, characterId)
	return updated, nil
}

// startChainedCore handles chained quest start (skips interval check)
func (p *ProcessorImpl) startChainedCore(characterId uint32, questId uint32, questDef dataquest.RestModel) (Model, error) {
	existing, _ := p.GetByCharacterIdAndQuestId(characterId, questId)

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
	// Note: Item tracking is handled client-side, not server-side
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
			if err = initializeProgress(tx, p.t.Id(), m.Id(), mobIds, mapIds); err != nil {
				return err
			}
		}

		return nil
	})
	if txErr != nil {
		return Model{}, txErr
	}

	return m, nil
}

func (p *ProcessorImpl) Complete(transactionId uuid.UUID, characterId uint32, questId uint32, f field.Model, skipValidation bool) (uint32, error) {
	// Fetch quest definition for requirements and rewards
	questDef, err := p.dataProcessor.GetQuestDefinition(questId)
	if err != nil {
		p.l.WithError(err).Warnf("Unable to fetch quest definition for quest [%d].", questId)
		questDef = dataquest.RestModel{}
	}

	// Validate end requirements (unless skipped)
	if !skipValidation {
		passed, failedConditions, err := p.validationProcessor.ValidateEndRequirements(characterId, questDef)
		if err != nil {
			p.l.WithError(err).Warnf("Unable to validate end requirements for quest [%d] character [%d], proceeding anyway.", questId, characterId)
		} else if !passed {
			p.l.Debugf("End requirements not met for quest [%d] character [%d]. Failed: %v", questId, characterId, failedConditions)
			return 0, ErrEndRequirementsNotMet
		}
	}

	// Complete the quest (core logic)
	nextQuestId, completedAt, err := p.completeCore(characterId, questId, questDef)
	if err != nil {
		return 0, err
	}

	// Process end actions/rewards
	if err := p.processEndActions(characterId, questId, questDef, f); err != nil {
		p.l.WithError(err).Warnf("Unable to process end actions for quest [%d] character [%d].", questId, characterId)
		// Don't fail the completion, just log the error
	}

	// Emit quest completed event
	if err := p.eventEmitter.EmitQuestCompleted(transactionId, characterId, byte(f.WorldId()), questId, completedAt); err != nil {
		p.l.WithError(err).Warnf("Unable to emit quest completed event for quest [%d] character [%d].", questId, characterId)
	}

	return nextQuestId, nil
}

// completeCore handles the core quest completion logic without validation or reward processing
func (p *ProcessorImpl) completeCore(characterId uint32, questId uint32, questDef dataquest.RestModel) (uint32, time.Time, error) {
	existing, err := p.GetByCharacterIdAndQuestId(characterId, questId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to find quest [%d] for character [%d] to complete.", questId, characterId)
		return 0, time.Time{}, err
	}

	if existing.State() == StateCompleted {
		p.l.Debugf("Quest [%d] already completed for character [%d].", questId, characterId)
		return 0, existing.CompletedAt(), nil
	}

	if existing.State() != StateStarted {
		p.l.Debugf("Quest [%d] not in started state for character [%d].", questId, characterId)
		return 0, time.Time{}, ErrQuestNotStarted
	}

	// Check if quest has expired
	if existing.IsExpired() {
		p.l.Debugf("Quest [%d] has expired for character [%d].", questId, characterId)
		return 0, time.Time{}, ErrQuestExpired
	}

	var completedAt time.Time
	txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
		var err error
		completedAt, err = completeQuest(tx, p.t.Id(), existing.Id())
		return err
	})
	if txErr != nil {
		p.l.WithError(txErr).Errorf("Unable to complete quest [%d] for character [%d].", questId, characterId)
		return 0, time.Time{}, txErr
	}

	p.l.Debugf("Completed quest [%d] for character [%d].", questId, characterId)

	// Return next quest ID if this is part of a chain
	nextQuestId := questDef.EndActions.NextQuest
	if nextQuestId > 0 {
		p.l.Debugf("Quest [%d] has next quest [%d] in chain.", questId, nextQuestId)
	}

	return nextQuestId, completedAt, nil
}

func (p *ProcessorImpl) Forfeit(transactionId uuid.UUID, characterId uint32, questId uint32) error {
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

	// Emit quest forfeited event (worldId is 0 as it's not available in forfeit context)
	if err := p.eventEmitter.EmitQuestForfeited(transactionId, characterId, 0, questId); err != nil {
		p.l.WithError(err).Warnf("Unable to emit quest forfeited event for quest [%d] character [%d].", questId, characterId)
	}

	p.l.Debugf("Forfeited quest [%d] for character [%d].", questId, characterId)
	return nil
}

func (p *ProcessorImpl) SetProgress(transactionId uuid.UUID, characterId uint32, questId uint32, infoNumber uint32, progressValue string) error {
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

	// Reload the quest to get the updated progress for emission
	updated, err := p.GetByCharacterIdAndQuestId(characterId, questId)
	if err != nil {
		p.l.WithError(err).Warnf("Unable to reload quest [%d] for character [%d] after progress update.", questId, characterId)
		// Still emit with just the single value as fallback
		if err := p.eventEmitter.EmitProgressUpdated(transactionId, characterId, 0, questId, infoNumber, progressValue); err != nil {
			p.l.WithError(err).Warnf("Unable to emit quest progress updated event for quest [%d] character [%d].", questId, characterId)
		}
	} else {
		// Emit quest progress updated event with the full progress string
		fullProgress := updated.ProgressString()
		if err := p.eventEmitter.EmitProgressUpdated(transactionId, characterId, 0, questId, infoNumber, fullProgress); err != nil {
			p.l.WithError(err).Warnf("Unable to emit quest progress updated event for quest [%d] character [%d].", questId, characterId)
		}
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

func (p *ProcessorImpl) CheckAutoComplete(characterId uint32, questId uint32, f field.Model) (uint32, bool, error) {
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

	// Check if all end requirements are met (internal check for mob kills/map visits)
	if !p.areEndRequirementsMet(existing, questDef) {
		return 0, false, nil
	}

	// All requirements met, complete the quest (skip external validation as we've already checked locally)
	// Use uuid.Nil since auto-complete is not initiated by a saga
	nextQuestId, err := p.Complete(uuid.Nil, characterId, questId, f, true)
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

// arePrerequisiteQuestsMet checks if all prerequisite quests are in the required state
func (p *ProcessorImpl) arePrerequisiteQuestsMet(characterId uint32, questDef dataquest.RestModel) bool {
	for _, prereq := range questDef.StartRequirements.Quests {
		existingQuest, err := p.GetByCharacterIdAndQuestId(characterId, prereq.Id)
		if err != nil {
			// Quest not found - only valid if required state is NotStarted (0)
			if prereq.State != uint8(StateNotStarted) {
				p.l.Debugf("Prerequisite quest [%d] not found for character [%d], required state [%d].", prereq.Id, characterId, prereq.State)
				return false
			}
			continue
		}

		// Check if quest is in the required state
		if uint8(existingQuest.State()) != prereq.State {
			p.l.Debugf("Prerequisite quest [%d] is in state [%d], required state [%d] for character [%d].", prereq.Id, existingQuest.State(), prereq.State, characterId)
			return false
		}
	}
	return true
}

func (p *ProcessorImpl) CheckAutoStart(characterId uint32, f field.Model) ([]uint32, error) {
	// Fetch all auto-start quests from atlas-data
	autoStartQuests, err := p.dataProcessor.GetAutoStartQuests(uint32(f.MapId()))
	if err != nil {
		p.l.WithError(err).Warnf("Unable to fetch auto-start quests for map [%d].", f.MapId())
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

		// Check prerequisite quests before attempting to start
		if !p.arePrerequisiteQuestsMet(characterId, questDef) {
			continue // Prerequisites not met
		}

		// Start the quest (auto-start quests still validate requirements)
		// Use startWithDefinition directly since we already have the quest definition
		// Use uuid.Nil since auto-start is not initiated by a saga
		_, _, err = p.startWithDefinition(uuid.Nil, characterId, questDef.Id, questDef, f, false)
		if err != nil {
			if !errors.Is(err, ErrQuestAlreadyStarted) && !errors.Is(err, ErrQuestAlreadyCompleted) && !errors.Is(err, ErrStartRequirementsNotMet) && !errors.Is(err, ErrValidationFailed) {
				p.l.WithError(err).Warnf("Unable to auto-start quest [%d] for character [%d].", questDef.Id, characterId)
			}
			continue
		}

		p.l.Infof("Auto-started quest [%d] for character [%d] on map [%d].", questDef.Id, characterId, f.MapId())
		startedQuests = append(startedQuests, questDef.Id)
	}

	return startedQuests, nil
}

func (p *ProcessorImpl) processStartActions(characterId uint32, questId uint32, questDef dataquest.RestModel, f field.Model) error {
	actions := questDef.StartActions

	// Build saga for start actions
	builder := sagaproducer.NewBuilder(saga.QuestStart, fmt.Sprintf("quest_%d", questId))

	// Consume required items (negative count items in requirements)
	for _, item := range questDef.StartRequirements.Items {
		if item.Count < 0 {
			builder.AddConsumeItem(characterId, item.Id, uint32(-item.Count))
		}
	}

	// Process item rewards - separate into random selection pool and unconditional items
	var randomPool []dataquest.ItemReward
	var unconditionalItems []dataquest.ItemReward

	for _, item := range actions.Items {
		if item.Prop >= 0 && item.Count > 0 {
			// Items with prop >= 0 and positive count are candidates for random selection
			randomPool = append(randomPool, item)
		} else {
			// Items with prop == -1 or negative count are unconditional
			unconditionalItems = append(unconditionalItems, item)
		}
	}

	// If there's a random pool, select one item based on weighted probability
	if len(randomPool) > 0 {
		selected := selectRandomItem(randomPool)
		if selected != nil {
			builder.AddAwardItem(characterId, selected.Id, uint32(selected.Count))
		}
	}

	// Process unconditional items
	for _, item := range unconditionalItems {
		if item.Count > 0 {
			builder.AddAwardItem(characterId, item.Id, uint32(item.Count))
		} else if item.Count < 0 {
			builder.AddConsumeItem(characterId, item.Id, uint32(-item.Count))
		}
	}

	// Award exp on start
	if actions.Exp > 0 {
		builder.AddAwardExperience(characterId, byte(f.WorldId()), byte(f.ChannelId()), actions.Exp)
	}

	// Award meso on start
	if actions.Money != 0 {
		builder.AddAwardMesos(characterId, byte(f.WorldId()), byte(f.ChannelId()), actions.Money, questId)
	}

	// Emit saga if there are steps
	if builder.HasSteps() {
		s := builder.Build()
		return p.eventEmitter.EmitSaga(s)
	}

	return nil
}

func (p *ProcessorImpl) processEndActions(characterId uint32, questId uint32, questDef dataquest.RestModel, f field.Model) error {
	actions := questDef.EndActions

	// Build saga for end actions
	builder := sagaproducer.NewBuilder(saga.QuestComplete, fmt.Sprintf("quest_%d", questId))

	// Process item rewards - separate into random selection pool and unconditional items
	var randomPool []dataquest.ItemReward
	var unconditionalItems []dataquest.ItemReward

	for _, item := range actions.Items {
		if item.Prop >= 0 && item.Count > 0 {
			// Items with prop >= 0 and positive count are candidates for random selection
			randomPool = append(randomPool, item)
		} else {
			// Items with prop == -1 or negative count are unconditional
			unconditionalItems = append(unconditionalItems, item)
		}
	}

	// If there's a random pool, select one item based on weighted probability
	if len(randomPool) > 0 {
		selected := selectRandomItem(randomPool)
		if selected != nil {
			builder.AddAwardItem(characterId, selected.Id, uint32(selected.Count))
		}
	}

	// Process unconditional items
	for _, item := range unconditionalItems {
		if item.Count > 0 {
			builder.AddAwardItem(characterId, item.Id, uint32(item.Count))
		} else if item.Count < 0 {
			builder.AddConsumeItem(characterId, item.Id, uint32(-item.Count))
		}
	}

	// Award experience
	if actions.Exp > 0 {
		builder.AddAwardExperience(characterId, byte(f.WorldId()), byte(f.ChannelId()), actions.Exp)
	}

	// Award meso
	if actions.Money != 0 {
		builder.AddAwardMesos(characterId, byte(f.WorldId()), byte(f.ChannelId()), actions.Money, questId)
	}

	// Award fame
	if actions.Fame != 0 {
		builder.AddAwardFame(characterId, byte(f.WorldId()), byte(f.ChannelId()), actions.Fame, questId)
	}

	// Award skills
	for _, skill := range actions.Skills {
		builder.AddCreateSkill(characterId, skill.Id, byte(skill.Level), byte(skill.MasterLevel))
	}

	// Emit saga if there are steps
	if builder.HasSteps() {
		s := builder.Build()
		return p.eventEmitter.EmitSaga(s)
	}

	return nil
}

// selectRandomItem selects one item from the pool based on weighted probability (Prop values)
func selectRandomItem(pool []dataquest.ItemReward) *dataquest.ItemReward {
	if len(pool) == 0 {
		return nil
	}
	if len(pool) == 1 {
		return &pool[0]
	}

	// Calculate total weight
	var totalWeight int32
	for _, item := range pool {
		weight := item.Prop
		if weight <= 0 {
			weight = 1 // Treat 0 as 1 for equal probability
		}
		totalWeight += weight
	}

	if totalWeight <= 0 {
		// Fallback to first item if weights are invalid
		return &pool[0]
	}

	// Generate random number and select item
	roll := rand.Int31n(totalWeight)
	var cumulative int32
	for i := range pool {
		weight := pool[i].Prop
		if weight <= 0 {
			weight = 1
		}
		cumulative += weight
		if roll < cumulative {
			return &pool[i]
		}
	}

	// Fallback (shouldn't reach here)
	return &pool[len(pool)-1]
}
