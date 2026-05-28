package script

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
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

	// Count returns the number of reactor scripts for the current tenant and the max updated_at timestamp.
	// Returns (0, nil, nil) when the tenant has no rows.
	Count() (int64, *time.Time, error)

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
	return model.Map[Entity, ReactorScript](Make)(getByIdProvider(id)(p.db.WithContext(p.ctx)))
}

// ByReactorIdProvider returns a provider for retrieving a reactor script by reactor ID
func (p *ProcessorImpl) ByReactorIdProvider(reactorId string) model.Provider[ReactorScript] {
	return model.Map[Entity, ReactorScript](Make)(getByReactorIdProvider(reactorId)(p.db.WithContext(p.ctx)))
}

// AllProvider returns a provider for retrieving all reactor scripts
func (p *ProcessorImpl) AllProvider() model.Provider[[]ReactorScript] {
	return model.SliceMap[Entity, ReactorScript](Make)(getAllProvider(p.db.WithContext(p.ctx)))(model.ParallelMap())
}

// Create creates a new reactor script
func (p *ProcessorImpl) Create(m ReactorScript) (ReactorScript, error) {
	p.l.Debugf("Creating reactor script [%s]", m.ReactorId())

	result, err := createReactorScript(p.db.WithContext(p.ctx))(p.t.Id())(m)
	if err != nil {
		p.l.WithError(err).Errorf("Failed to create reactor script [%s]", m.ReactorId())
		return ReactorScript{}, err
	}
	return result, nil
}

// Update updates an existing reactor script
func (p *ProcessorImpl) Update(id uuid.UUID, m ReactorScript) (ReactorScript, error) {
	p.l.Debugf("Updating reactor script [%s]", id)

	result, err := updateReactorScript(p.db.WithContext(p.ctx))(id)(m, p.t.Id())
	if err != nil {
		p.l.WithError(err).Errorf("Failed to update reactor script [%s]", id)
		return ReactorScript{}, err
	}
	return result, nil
}

// Delete deletes a reactor script
func (p *ProcessorImpl) Delete(id uuid.UUID) error {
	p.l.Debugf("Deleting reactor script [%s]", id)

	err := deleteReactorScript(p.db.WithContext(p.ctx))(id)
	if err != nil {
		p.l.WithError(err).Errorf("Failed to delete reactor script [%s]", id)
		return err
	}
	return nil
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
	evaluator := NewConditionEvaluator(p.l, p.ctx)

	for _, rule := range rules {
		matched, err := evaluator.EvaluateRule(reactorState, characterId, rule)
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

// Count returns the number of reactor scripts for the current tenant and the max updated_at timestamp.
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
