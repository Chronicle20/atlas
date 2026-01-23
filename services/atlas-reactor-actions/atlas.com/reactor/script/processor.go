package script

import (
	"context"
	"errors"
	"fmt"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// ScriptProcessor defines the interface for reactor script processing
type ScriptProcessor interface {
	// CRUD operations
	Create(model ReactorScript) (ReactorScript, error)
	Update(id uuid.UUID, model ReactorScript) (ReactorScript, error)
	Delete(id uuid.UUID) error

	// Query operations
	ByIdProvider(id uuid.UUID) model.Provider[ReactorScript]
	ByReactorIdProvider(reactorId string) model.Provider[ReactorScript]
	AllProvider() model.Provider[[]ReactorScript]

	// Seeding
	DeleteAllForTenant() (int64, error)
	Seed() (SeedResult, error)

	// Execution
	ProcessHit(reactorId string, reactorState int8, characterId uint32) ProcessResult
	ProcessTrigger(reactorId string, reactorState int8, characterId uint32) ProcessResult
}

// ProcessorImpl implements ScriptProcessor using database storage
type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	t   tenant.Model
	db  *gorm.DB
}

// NewProcessor creates a new script processor
func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) ScriptProcessor {
	t := tenant.MustFromContext(ctx)

	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		t:   t,
		db:  db,
	}
}

// ByIdProvider returns a provider for retrieving a reactor script by ID
func (p *ProcessorImpl) ByIdProvider(id uuid.UUID) model.Provider[ReactorScript] {
	return model.Map[Entity, ReactorScript](Make)(getByIdProvider(p.t.Id())(id)(p.db))
}

// ByReactorIdProvider returns a provider for retrieving a reactor script by reactor ID
func (p *ProcessorImpl) ByReactorIdProvider(reactorId string) model.Provider[ReactorScript] {
	return model.Map[Entity, ReactorScript](Make)(getByReactorIdProvider(p.t.Id())(reactorId)(p.db))
}

// AllProvider returns a provider for retrieving all reactor scripts
func (p *ProcessorImpl) AllProvider() model.Provider[[]ReactorScript] {
	return model.SliceMap[Entity, ReactorScript](Make)(getAllProvider(p.t.Id())(p.db))(model.ParallelMap())
}

// Create creates a new reactor script
func (p *ProcessorImpl) Create(m ReactorScript) (ReactorScript, error) {
	p.l.Debugf("Creating reactor script [%s]", m.ReactorId())

	result, err := createReactorScript(p.db)(p.t.Id())(m)
	if err != nil {
		p.l.WithError(err).Errorf("Failed to create reactor script [%s]", m.ReactorId())
		return ReactorScript{}, err
	}
	return result, nil
}

// Update updates an existing reactor script
func (p *ProcessorImpl) Update(id uuid.UUID, m ReactorScript) (ReactorScript, error) {
	p.l.Debugf("Updating reactor script [%s]", id)

	result, err := updateReactorScript(p.db)(p.t.Id())(id)(m)
	if err != nil {
		p.l.WithError(err).Errorf("Failed to update reactor script [%s]", id)
		return ReactorScript{}, err
	}
	return result, nil
}

// Delete deletes a reactor script
func (p *ProcessorImpl) Delete(id uuid.UUID) error {
	p.l.Debugf("Deleting reactor script [%s]", id)

	err := deleteReactorScript(p.db)(p.t.Id())(id)
	if err != nil {
		p.l.WithError(err).Errorf("Failed to delete reactor script [%s]", id)
		return err
	}
	return nil
}

// DeleteAllForTenant deletes all reactor scripts for the current tenant
func (p *ProcessorImpl) DeleteAllForTenant() (int64, error) {
	p.l.Debugf("Deleting all reactor scripts for tenant [%s]", p.t.Id())

	count, err := deleteAllReactorScripts(p.db)(p.t.Id())
	if err != nil {
		p.l.WithError(err).Errorf("Failed to delete reactor scripts for tenant [%s]", p.t.Id())
		return 0, err
	}
	p.l.Debugf("Deleted [%d] reactor scripts for tenant [%s]", count, p.t.Id())
	return count, nil
}

// Seed clears existing reactor scripts and loads them from the scripts directory
func (p *ProcessorImpl) Seed() (SeedResult, error) {
	p.l.Infof("Seeding reactor scripts for tenant [%s]", p.t.Id())

	result := SeedResult{}

	// Delete all existing scripts for this tenant
	deletedCount, err := p.DeleteAllForTenant()
	if err != nil {
		return result, fmt.Errorf("failed to clear existing reactor scripts: %w", err)
	}
	result.DeletedCount = int(deletedCount)

	// Load script files from the filesystem
	scripts, loadErrors := LoadReactorScriptFiles()

	// Track load errors
	for _, err := range loadErrors {
		result.Errors = append(result.Errors, err.Error())
		result.FailedCount++
	}

	// Create each script
	for _, script := range scripts {
		_, err = p.Create(script)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: failed to create: %v", script.ReactorId(), err))
			result.FailedCount++
			continue
		}
		result.CreatedCount++
	}

	p.l.Infof("Seed complete for tenant [%s]: deleted=%d, created=%d, failed=%d",
		p.t.Id(), result.DeletedCount, result.CreatedCount, result.FailedCount)

	return result, nil
}

// ProcessHit processes a reactor hit event
func (p *ProcessorImpl) ProcessHit(reactorId string, reactorState int8, characterId uint32) ProcessResult {
	p.l.Debugf("Processing reactor hit [%s] state [%d] for character [%d]", reactorId, reactorState, characterId)

	// Load the reactor script from database
	script, err := p.ByReactorIdProvider(reactorId)()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			p.l.Debugf("No reactor script found for [%s] - no action", reactorId)
			return ProcessResult{
				MatchedRule: "no_script",
				Error:       nil,
			}
		}
		p.l.WithError(err).Warnf("Failed to load reactor script [%s]", reactorId)
		return ProcessResult{
			MatchedRule: "no_script",
			Error:       nil,
		}
	}

	// Evaluate hit rules in order - first matching rule wins
	return p.evaluateRules(script.HitRules(), reactorId, reactorState, characterId, "hit")
}

// ProcessTrigger processes a reactor trigger event (reached final state)
func (p *ProcessorImpl) ProcessTrigger(reactorId string, reactorState int8, characterId uint32) ProcessResult {
	p.l.Debugf("Processing reactor trigger [%s] state [%d] for character [%d]", reactorId, reactorState, characterId)

	// Load the reactor script from database
	script, err := p.ByReactorIdProvider(reactorId)()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			p.l.Debugf("No reactor script found for [%s] - no action", reactorId)
			return ProcessResult{
				MatchedRule: "no_script",
				Error:       nil,
			}
		}
		p.l.WithError(err).Warnf("Failed to load reactor script [%s]", reactorId)
		return ProcessResult{
			MatchedRule: "no_script",
			Error:       nil,
		}
	}

	// Evaluate act rules in order - first matching rule wins
	return p.evaluateRules(script.ActRules(), reactorId, reactorState, characterId, "trigger")
}

// evaluateRules evaluates a list of rules and returns the result
func (p *ProcessorImpl) evaluateRules(rules []Rule, reactorId string, reactorState int8, characterId uint32, eventType string) ProcessResult {
	evaluator := NewConditionEvaluator(p.l)

	for _, rule := range rules {
		matched, err := evaluator.EvaluateRule(reactorState, rule)
		if err != nil {
			p.l.WithError(err).Errorf("Failed to evaluate rule [%s] for reactor [%s]", rule.Id(), reactorId)
			return ProcessResult{
				MatchedRule: rule.Id(),
				Error:       fmt.Errorf("rule evaluation failed: %w", err),
			}
		}

		if matched {
			p.l.Debugf("Rule [%s] matched for character [%d] on reactor [%s] (%s)", rule.Id(), characterId, reactorId, eventType)

			return ProcessResult{
				MatchedRule: rule.Id(),
				Operations:  rule.Operations(),
				Error:       nil,
			}
		}
	}

	// No rules matched
	p.l.Debugf("No rules matched for reactor [%s], character [%d] (%s)", reactorId, characterId, eventType)
	return ProcessResult{
		MatchedRule: "no_match",
		Error:       nil,
	}
}
