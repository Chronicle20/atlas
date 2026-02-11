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

type ScriptProcessor interface {
	Create(model MapScript) (MapScript, error)
	Update(id uuid.UUID, model MapScript) (MapScript, error)
	Delete(id uuid.UUID) error

	ByIdProvider(id uuid.UUID) model.Provider[MapScript]
	ByScriptNameProvider(scriptName string) model.Provider[[]MapScript]
	ByScriptNameAndTypeProvider(scriptName string, scriptType string) model.Provider[MapScript]
	AllProvider() model.Provider[[]MapScript]

	DeleteAllForTenant() (int64, error)
	Seed() (SeedResult, error)

	Process(f field.Model, characterId uint32, scriptName string, scriptType string) ProcessResult
}

type ProcessorImpl struct {
	l         logrus.FieldLogger
	ctx       context.Context
	t         tenant.Model
	db        *gorm.DB
	evaluator *ConditionEvaluator
	executor  *OperationExecutor
}

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

func (p *ProcessorImpl) ByIdProvider(id uuid.UUID) model.Provider[MapScript] {
	return model.Map[Entity, MapScript](Make)(getByIdProvider(p.t.Id())(id)(p.db))
}

func (p *ProcessorImpl) ByScriptNameProvider(scriptName string) model.Provider[[]MapScript] {
	return model.SliceMap[Entity, MapScript](Make)(getByScriptNameProvider(p.t.Id())(scriptName)(p.db))(model.ParallelMap())
}

func (p *ProcessorImpl) ByScriptNameAndTypeProvider(scriptName string, scriptType string) model.Provider[MapScript] {
	return model.Map[Entity, MapScript](Make)(getByScriptNameAndTypeProvider(p.t.Id())(scriptName)(scriptType)(p.db))
}

func (p *ProcessorImpl) AllProvider() model.Provider[[]MapScript] {
	return model.SliceMap[Entity, MapScript](Make)(getAllProvider(p.t.Id())(p.db))(model.ParallelMap())
}

func (p *ProcessorImpl) Create(m MapScript) (MapScript, error) {
	p.l.Debugf("Creating map script [%s] type [%s].", m.ScriptName(), m.ScriptType())

	result, err := createMapScript(p.db)(p.t.Id())(m)
	if err != nil {
		p.l.WithError(err).Errorf("Failed to create map script [%s].", m.ScriptName())
		return MapScript{}, err
	}
	return result, nil
}

func (p *ProcessorImpl) Update(id uuid.UUID, m MapScript) (MapScript, error) {
	p.l.Debugf("Updating map script [%s].", id)

	result, err := updateMapScript(p.db)(p.t.Id())(id)(m)
	if err != nil {
		p.l.WithError(err).Errorf("Failed to update map script [%s].", id)
		return MapScript{}, err
	}
	return result, nil
}

func (p *ProcessorImpl) Delete(id uuid.UUID) error {
	p.l.Debugf("Deleting map script [%s].", id)

	err := deleteMapScript(p.db)(p.t.Id())(id)
	if err != nil {
		p.l.WithError(err).Errorf("Failed to delete map script [%s].", id)
		return err
	}
	return nil
}

func (p *ProcessorImpl) DeleteAllForTenant() (int64, error) {
	p.l.Debugf("Deleting all map scripts for tenant [%s].", p.t.Id())

	count, err := deleteAllMapScripts(p.db)(p.t.Id())
	if err != nil {
		p.l.WithError(err).Errorf("Failed to delete map scripts for tenant [%s].", p.t.Id())
		return 0, err
	}
	p.l.Debugf("Deleted [%d] map scripts for tenant [%s].", count, p.t.Id())
	return count, nil
}

func (p *ProcessorImpl) Seed() (SeedResult, error) {
	p.l.Infof("Seeding map scripts for tenant [%s].", p.t.Id())

	result := SeedResult{}

	deletedCount, err := p.DeleteAllForTenant()
	if err != nil {
		return result, fmt.Errorf("failed to clear existing map scripts: %w", err)
	}
	result.DeletedCount = int(deletedCount)

	scripts, loadErrors := LoadMapScriptFiles()

	for _, err := range loadErrors {
		result.Errors = append(result.Errors, err.Error())
		result.FailedCount++
	}

	for _, script := range scripts {
		_, err = p.Create(script)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s/%s: failed to create: %v", script.ScriptName(), script.ScriptType(), err))
			result.FailedCount++
			continue
		}
		result.CreatedCount++
	}

	p.l.Infof("Seed complete for tenant [%s]: deleted=%d, created=%d, failed=%d.",
		p.t.Id(), result.DeletedCount, result.CreatedCount, result.FailedCount)

	return result, nil
}

func (p *ProcessorImpl) Process(f field.Model, characterId uint32, scriptName string, scriptType string) ProcessResult {
	p.l.Debugf("Processing map script [%s] type [%s] for character [%d].", scriptName, scriptType, characterId)

	ms, err := p.ByScriptNameAndTypeProvider(scriptName, scriptType)()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			p.l.Debugf("No map script found for [%s] type [%s].", scriptName, scriptType)
			return ProcessResult{
				MatchedRule: "no_script",
			}
		}
		p.l.WithError(err).Warnf("Failed to load map script [%s] type [%s].", scriptName, scriptType)
		return ProcessResult{
			Error: err,
		}
	}

	for _, rule := range ms.Rules() {
		matched, err := p.evaluateRule(f, characterId, rule)
		if err != nil {
			p.l.WithError(err).Errorf("Failed to evaluate rule [%s] for script [%s].", rule.Id(), scriptName)
			return ProcessResult{
				MatchedRule: rule.Id(),
				Error:       fmt.Errorf("rule evaluation failed: %w", err),
			}
		}

		if matched {
			p.l.Debugf("Rule [%s] matched for character [%d] on script [%s].", rule.Id(), characterId, scriptName)

			if len(rule.Operations()) > 0 {
				if err := p.executor.ExecuteOperations(f, characterId, rule.Operations()); err != nil {
					p.l.WithError(err).Errorf("Failed to execute operations for rule [%s].", rule.Id())
					return ProcessResult{
						MatchedRule: rule.Id(),
						Operations:  rule.Operations(),
						Error:       fmt.Errorf("operation execution failed: %w", err),
					}
				}
			}

			return ProcessResult{
				MatchedRule: rule.Id(),
				Operations:  rule.Operations(),
			}
		}
	}

	p.l.Debugf("No rules matched for script [%s] character [%d].", scriptName, characterId)
	return ProcessResult{
		MatchedRule: "no_match",
	}
}

func (p *ProcessorImpl) evaluateRule(f field.Model, characterId uint32, rule Rule) (bool, error) {
	conditions := rule.Conditions()

	if len(conditions) == 0 {
		return true, nil
	}

	for _, cond := range conditions {
		passed, err := p.evaluator.EvaluateCondition(f, characterId, cond)
		if err != nil {
			return false, fmt.Errorf("condition evaluation failed: %w", err)
		}
		if !passed {
			return false, nil
		}
	}

	return true, nil
}
