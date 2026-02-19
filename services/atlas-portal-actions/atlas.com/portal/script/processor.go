package script

import (
	"context"
	"errors"
	"fmt"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
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

	// Seeding
	DeleteAllForTenant() (int64, error)
	Seed() (SeedResult, error)

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

// DeleteAllForTenant deletes all portal scripts for the current tenant
func (p *ProcessorImpl) DeleteAllForTenant() (int64, error) {
	p.l.Debugf("Deleting all portal scripts for tenant [%s]", p.t.Id())

	count, err := deleteAllPortalScripts(p.db.WithContext(p.ctx))
	if err != nil {
		p.l.WithError(err).Errorf("Failed to delete portal scripts for tenant [%s]", p.t.Id())
		return 0, err
	}
	p.l.Debugf("Deleted [%d] portal scripts for tenant [%s]", count, p.t.Id())
	return count, nil
}

// Seed clears existing portal scripts and loads them from the scripts directory
func (p *ProcessorImpl) Seed() (SeedResult, error) {
	p.l.Infof("Seeding portal scripts for tenant [%s]", p.t.Id())

	result := SeedResult{}

	// Delete all existing scripts for this tenant
	deletedCount, err := p.DeleteAllForTenant()
	if err != nil {
		return result, fmt.Errorf("failed to clear existing portal scripts: %w", err)
	}
	result.DeletedCount = int(deletedCount)

	// Load script files from the filesystem
	scripts, loadErrors := LoadPortalScriptFiles()

	// Track load errors
	for _, err := range loadErrors {
		result.Errors = append(result.Errors, err.Error())
		result.FailedCount++
	}

	// Create each script
	for _, script := range scripts {
		_, err = p.Create(script)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: failed to create: %v", script.PortalId(), err))
			result.FailedCount++
			continue
		}
		result.CreatedCount++
	}

	p.l.Infof("Seed complete for tenant [%s]: deleted=%d, created=%d, failed=%d",
		p.t.Id(), result.DeletedCount, result.CreatedCount, result.FailedCount)

	return result, nil
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
