package definition

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Processor interface {
	Create(model Model) (Model, error)
	Update(id uuid.UUID, model Model) (Model, error)
	Delete(id uuid.UUID) error
	ByIdProvider(id uuid.UUID) model.Provider[Model]
	ByQuestIdProvider(questId string) model.Provider[Model]
	AllProvider() model.Provider[[]Model]
	DeleteAllForTenant() (int64, error)
	Seed() (SeedResult, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	t   tenant.Model
	db  *gorm.DB
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	t := tenant.MustFromContext(ctx)

	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		t:   t,
		db:  db,
	}
}

func (p *ProcessorImpl) ByIdProvider(id uuid.UUID) model.Provider[Model] {
	return model.Map[Entity, Model](Make)(getByIdProvider(p.t.Id())(id)(p.db))
}

func (p *ProcessorImpl) ByQuestIdProvider(questId string) model.Provider[Model] {
	return model.Map[Entity, Model](Make)(getByQuestIdProvider(p.t.Id())(questId)(p.db))
}

func (p *ProcessorImpl) AllProvider() model.Provider[[]Model] {
	return model.SliceMap[Entity, Model](Make)(getAllProvider(p.t.Id())(p.db))(model.ParallelMap())
}

func (p *ProcessorImpl) Create(m Model) (Model, error) {
	p.l.Debugf("Creating PQ definition [%s]", m.QuestId())

	result, err := createDefinition(p.db)(p.t.Id())(m)
	if err != nil {
		p.l.WithError(err).Errorf("Failed to create PQ definition [%s]", m.QuestId())
		return Model{}, err
	}
	return result, nil
}

func (p *ProcessorImpl) Update(id uuid.UUID, m Model) (Model, error) {
	p.l.Debugf("Updating PQ definition [%s]", id)

	result, err := updateDefinition(p.db)(p.t.Id())(id)(m)
	if err != nil {
		p.l.WithError(err).Errorf("Failed to update PQ definition [%s]", id)
		return Model{}, err
	}
	return result, nil
}

func (p *ProcessorImpl) Delete(id uuid.UUID) error {
	p.l.Debugf("Deleting PQ definition [%s]", id)

	err := deleteDefinition(p.db)(p.t.Id())(id)
	if err != nil {
		p.l.WithError(err).Errorf("Failed to delete PQ definition [%s]", id)
		return err
	}
	return nil
}

func (p *ProcessorImpl) DeleteAllForTenant() (int64, error) {
	p.l.Debugf("Deleting all PQ definitions for tenant [%s]", p.t.Id())

	count, err := deleteAllDefinitions(p.db)(p.t.Id())
	if err != nil {
		p.l.WithError(err).Errorf("Failed to delete PQ definitions for tenant [%s]", p.t.Id())
		return 0, err
	}
	p.l.Debugf("Deleted [%d] PQ definitions for tenant [%s]", count, p.t.Id())
	return count, nil
}

func (p *ProcessorImpl) Seed() (SeedResult, error) {
	p.l.Infof("Seeding PQ definitions for tenant [%s]", p.t.Id())

	result := SeedResult{}

	deletedCount, err := p.DeleteAllForTenant()
	if err != nil {
		return result, fmt.Errorf("failed to clear existing PQ definitions: %w", err)
	}
	result.DeletedCount = int(deletedCount)

	models, loadErrors := LoadDefinitionFiles()

	for _, err := range loadErrors {
		result.Errors = append(result.Errors, err.Error())
		result.FailedCount++
	}

	for _, rm := range models {
		m, err := Extract(rm)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: failed to extract model: %v", rm.QuestId, err))
			result.FailedCount++
			continue
		}

		_, err = p.Create(m)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: failed to create: %v", rm.QuestId, err))
			result.FailedCount++
			continue
		}

		result.CreatedCount++
	}

	p.l.Infof("Seed complete for tenant [%s]: deleted=%d, created=%d, failed=%d",
		p.t.Id(), result.DeletedCount, result.CreatedCount, result.FailedCount)

	return result, nil
}
