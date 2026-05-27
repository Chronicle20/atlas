package script

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// ScriptProcessor defines the interface for portal script processing
type ScriptProcessor interface {
	// CRUD operations
	Create(model PortalScript) (PortalScript, error)
	Update(id uuid.UUID, model PortalScript) (PortalScript, error)
	Delete(id uuid.UUID) error

	// Query operations
	ByIdProvider(id uuid.UUID) model.Provider[PortalScript]
	ByPortalIdProvider(portalId string) model.Provider[PortalScript]
	AllProvider() model.Provider[[]PortalScript]

	// Count returns the number of portal scripts for the current tenant and the max updated_at timestamp.
	// Returns (0, nil, nil) when the tenant has no rows.
	Count() (int64, *time.Time, error)

	// Execution
	Process(f field.Model, characterId uint32, portalName string, portalId uint32) ProcessResult
}

// ProcessorImpl implements ScriptProcessor using database storage
type ProcessorImpl struct {
	l         logrus.FieldLogger
	ctx       context.Context
	t         tenant.Model
	db        *gorm.DB
	evaluator *ConditionEvaluator
	executor  *OperationExecutor
}

// NewProcessor creates a new script processor with context-driven evaluator/executor
func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) ScriptProcessor {
	t := tenant.MustFromContext(ctx)
	evaluator := NewConditionEvaluator(l, ctx)
	executor := NewOperationExecutor(l, ctx)

	return &ProcessorImpl{
		l:         l,
		ctx:       ctx,
		t:         t,
		db:        db,
		evaluator: evaluator,
		executor:  executor,
	}
}

// ByIdProvider returns a provider for retrieving a portal script by ID
func (p *ProcessorImpl) ByIdProvider(id uuid.UUID) model.Provider[PortalScript] {
	return model.Map[Entity, PortalScript](Make)(getByIdProvider(id)(p.db.WithContext(p.ctx)))
}

// ByPortalIdProvider returns a provider for retrieving a portal script by portal ID
func (p *ProcessorImpl) ByPortalIdProvider(portalId string) model.Provider[PortalScript] {
	return model.Map[Entity, PortalScript](Make)(getByPortalIdProvider(portalId)(p.db.WithContext(p.ctx)))
}

// AllProvider returns a provider for retrieving all portal scripts
func (p *ProcessorImpl) AllProvider() model.Provider[[]PortalScript] {
	return model.SliceMap[Entity, PortalScript](Make)(getAllProvider(p.db.WithContext(p.ctx)))(model.ParallelMap())
}

// Create creates a new portal script
func (p *ProcessorImpl) Create(m PortalScript) (PortalScript, error) {
	p.l.Debugf("Creating portal script [%s]", m.PortalId())

	result, err := createPortalScript(p.db.WithContext(p.ctx))(p.t.Id())(m)
	if err != nil {
		p.l.WithError(err).Errorf("Failed to create portal script [%s]", m.PortalId())
		return PortalScript{}, err
	}
	return result, nil
}

// Update updates an existing portal script
func (p *ProcessorImpl) Update(id uuid.UUID, m PortalScript) (PortalScript, error) {
	p.l.Debugf("Updating portal script [%s]", id)

	result, err := updatePortalScript(p.db.WithContext(p.ctx))(id)(m, p.t.Id())
	if err != nil {
		p.l.WithError(err).Errorf("Failed to update portal script [%s]", id)
		return PortalScript{}, err
	}
	return result, nil
}

// Delete deletes a portal script
func (p *ProcessorImpl) Delete(id uuid.UUID) error {
	p.l.Debugf("Deleting portal script [%s]", id)

	err := deletePortalScript(p.db.WithContext(p.ctx))(id)
	if err != nil {
		p.l.WithError(err).Errorf("Failed to delete portal script [%s]", id)
		return err
	}
	return nil
}

// Process processes a portal entry request
// portalName is the string name used to look up the script, portalId is the numeric ID for operations
func (p *ProcessorImpl) Process(f field.Model, characterId uint32, portalName string, portalId uint32) ProcessResult {
	p.l.Debugf("Processing portal script [%s] (id=%d) for character [%d]", portalName, portalId, characterId)

	// Load the portal script from database
	script, err := p.ByPortalIdProvider(portalName)()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			p.l.Debugf("No portal script found for [%s] - allowing by default", portalName)
			// No script found - allow by default (portal without script)
			return ProcessResult{
				Allow:       true,
				MatchedRule: "no_script",
				Error:       nil,
			}
		}
		p.l.WithError(err).Warnf("Failed to load portal script [%s]", portalName)
		return ProcessResult{
			Allow:       true,
			MatchedRule: "no_script",
			Error:       nil,
		}
	}

	// Evaluate rules in order - first matching rule wins
	for _, rule := range script.Rules() {
		matched, err := p.evaluateRule(characterId, rule)
		if err != nil {
			p.l.WithError(err).Errorf("Failed to evaluate rule [%s] for portal [%s]", rule.Id(), portalName)
			return ProcessResult{
				Allow:       false,
				MatchedRule: rule.Id(),
				Error:       fmt.Errorf("rule evaluation failed: %w", err),
			}
		}

		if matched {
			p.l.Debugf("Rule [%s] matched for character [%d] on portal [%s]", rule.Id(), characterId, portalName)

			// Execute operations (pass portalId for operations like block_portal)
			outcome := rule.OnMatch()
			if len(outcome.Operations()) > 0 {
				if err := p.executor.ExecuteOperations(f, characterId, portalId, outcome.Operations()); err != nil {
					p.l.WithError(err).Errorf("Failed to execute operations for rule [%s]", rule.Id())
					return ProcessResult{
						Allow:       outcome.Allow(),
						MatchedRule: rule.Id(),
						Operations:  outcome.Operations(),
						Error:       fmt.Errorf("operation execution failed: %w", err),
					}
				}
			}

			return ProcessResult{
				Allow:       outcome.Allow(),
				MatchedRule: rule.Id(),
				Operations:  outcome.Operations(),
				Error:       nil,
			}
		}
	}

	// No rules matched - deny by default for safety
	p.l.Warnf("No rules matched for portal [%s], character [%d] - denying by default", portalName, characterId)
	return ProcessResult{
		Allow:       false,
		MatchedRule: "no_match",
		Error:       nil,
	}
}

// Count returns the number of portal scripts for the current tenant and the max updated_at timestamp.
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

// evaluateRule evaluates all conditions for a rule (AND logic)
func (p *ProcessorImpl) evaluateRule(characterId uint32, rule Rule) (bool, error) {
	conditions := rule.Conditions()

	// Empty conditions = always match (default rule)
	if len(conditions) == 0 {
		return true, nil
	}

	// All conditions must pass (AND logic)
	for _, cond := range conditions {
		passed, err := p.evaluator.EvaluateCondition(characterId, cond)
		if err != nil {
			return false, fmt.Errorf("condition evaluation failed: %w", err)
		}
		if !passed {
			return false, nil
		}
	}

	return true, nil
}
