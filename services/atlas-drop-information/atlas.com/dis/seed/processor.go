package seed

import (
	continentdrop "atlas-drops-information/continent/drop"
	monsterdrop "atlas-drops-information/monster/drop"
	"context"
	"fmt"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Processor interface {
	Seed() (CombinedSeedResult, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	db  *gorm.DB
	t   tenant.Model
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	t := tenant.MustFromContext(ctx)
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		db:  db,
		t:   t,
	}
}

func (p *ProcessorImpl) Seed() (CombinedSeedResult, error) {
	p.l.Infof("Seeding drops for tenant [%s]", p.t.Id())

	result := CombinedSeedResult{}

	// Seed monster drops
	monsterResult, err := p.seedMonsterDrops()
	if err != nil {
		return result, fmt.Errorf("failed to seed monster drops: %w", err)
	}
	result.MonsterDrops = monsterResult

	// Seed continent drops
	continentResult, err := p.seedContinentDrops()
	if err != nil {
		return result, fmt.Errorf("failed to seed continent drops: %w", err)
	}
	result.ContinentDrops = continentResult

	p.l.Infof("Seed complete for tenant [%s]: monster_deleted=%d, monster_created=%d, continent_deleted=%d, continent_created=%d",
		p.t.Id(),
		result.MonsterDrops.DeletedCount, result.MonsterDrops.CreatedCount,
		result.ContinentDrops.DeletedCount, result.ContinentDrops.CreatedCount)

	return result, nil
}

func (p *ProcessorImpl) seedMonsterDrops() (SeedResult, error) {
	result := SeedResult{}

	// Delete all existing monster drops for this tenant
	deletedCount, err := monsterdrop.DeleteAllForTenant(p.db, p.t.Id())
	if err != nil {
		return result, fmt.Errorf("failed to clear existing monster drops: %w", err)
	}
	result.DeletedCount = int(deletedCount)

	// Load monster drop files from the filesystem
	jsonModels, loadErrors := monsterdrop.LoadMonsterDropFiles()

	// Track load errors
	for _, err := range loadErrors {
		result.Errors = append(result.Errors, err.Error())
		result.FailedCount++
	}

	// Convert JSON models to domain models and bulk create
	var models []monsterdrop.Model
	for _, jm := range jsonModels {
		m := monsterdrop.NewMonsterDropBuilder(p.t.Id(), 0).
			SetMonsterId(jm.MonsterId).
			SetItemId(jm.ItemId).
			SetMinimumQuantity(jm.MinimumQuantity).
			SetMaximumQuantity(jm.MaximumQuantity).
			SetQuestId(jm.QuestId).
			SetChance(jm.Chance).
			Build()
		models = append(models, m)
	}

	if len(models) > 0 {
		err = monsterdrop.BulkCreateMonsterDrop(p.db, models)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("bulk create failed: %v", err))
			result.FailedCount += len(models)
		} else {
			result.CreatedCount = len(models)
		}
	}

	p.l.Debugf("Monster drops seed: deleted=%d, created=%d, failed=%d", result.DeletedCount, result.CreatedCount, result.FailedCount)

	return result, nil
}

func (p *ProcessorImpl) seedContinentDrops() (SeedResult, error) {
	result := SeedResult{}

	// Delete all existing continent drops for this tenant
	deletedCount, err := continentdrop.DeleteAllForTenant(p.db, p.t.Id())
	if err != nil {
		return result, fmt.Errorf("failed to clear existing continent drops: %w", err)
	}
	result.DeletedCount = int(deletedCount)

	// Load continent drop files from the filesystem
	jsonModels, loadErrors := continentdrop.LoadContinentDropFiles()

	// Track load errors
	for _, err := range loadErrors {
		result.Errors = append(result.Errors, err.Error())
		result.FailedCount++
	}

	// Convert JSON models to domain models and bulk create
	var models []continentdrop.Model
	for _, jm := range jsonModels {
		m := continentdrop.NewContinentDropBuilder(p.t.Id(), 0).
			SetContinentId(jm.ContinentId).
			SetItemId(jm.ItemId).
			SetMinimumQuantity(jm.MinimumQuantity).
			SetMaximumQuantity(jm.MaximumQuantity).
			SetChance(jm.Chance).
			Build()
		models = append(models, m)
	}

	if len(models) > 0 {
		err = continentdrop.BulkCreateContinentDrop(p.db, models)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("bulk create failed: %v", err))
			result.FailedCount += len(models)
		} else {
			result.CreatedCount = len(models)
		}
	}

	p.l.Debugf("Continent drops seed: deleted=%d, created=%d, failed=%d", result.DeletedCount, result.CreatedCount, result.FailedCount)

	return result, nil
}
