package seed

import (
	continentdrop "atlas-drops-information/continent/drop"
	monsterdrop "atlas-drops-information/monster/drop"
	"atlas-drops-information/reactor"
	reactordrop "atlas-drops-information/reactor/drop"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Chronicle20/atlas-tenant"
	"github.com/jtumidanski/api2go/jsonapi"
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

	// Seed reactor drops
	reactorResult, err := p.seedReactorDrops()
	if err != nil {
		return result, fmt.Errorf("failed to seed reactor drops: %w", err)
	}
	result.ReactorDrops = reactorResult

	p.l.Infof("Seed complete for tenant [%s]: monster_deleted=%d, monster_created=%d, continent_deleted=%d, continent_created=%d, reactor_deleted=%d, reactor_created=%d",
		p.t.Id(),
		result.MonsterDrops.DeletedCount, result.MonsterDrops.CreatedCount,
		result.ContinentDrops.DeletedCount, result.ContinentDrops.CreatedCount,
		result.ReactorDrops.DeletedCount, result.ReactorDrops.CreatedCount)

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
		m, err := monsterdrop.NewMonsterDropBuilder(p.t.Id(), 0).
			SetMonsterId(jm.MonsterId).
			SetItemId(jm.ItemId).
			SetMinimumQuantity(jm.MinimumQuantity).
			SetMaximumQuantity(jm.MaximumQuantity).
			SetQuestId(jm.QuestId).
			SetChance(jm.Chance).
			Build()
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("failed to build monster drop model: %v", err))
			result.FailedCount++
			continue
		}
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
		m, err := continentdrop.NewContinentDropBuilder(p.t.Id(), 0).
			SetContinentId(jm.ContinentId).
			SetItemId(jm.ItemId).
			SetMinimumQuantity(jm.MinimumQuantity).
			SetMaximumQuantity(jm.MaximumQuantity).
			SetChance(jm.Chance).
			Build()
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("failed to build continent drop model: %v", err))
			result.FailedCount++
			continue
		}
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

func (p *ProcessorImpl) seedReactorDrops() (SeedResult, error) {
	result := SeedResult{}

	// Delete all existing reactor drops for this tenant
	deletedCount, err := reactordrop.DeleteAllForTenant(p.db, p.t.Id())
	if err != nil {
		return result, fmt.Errorf("failed to clear existing reactor drops: %w", err)
	}
	result.DeletedCount = int(deletedCount)

	// Load reactor drop files from the filesystem using JSON:API format
	dropsPath := reactordrop.GetReactorDropsPath()
	entries, err := os.ReadDir(dropsPath)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("failed to read reactor drops directory: %v", err))
		return result, nil
	}

	var models []reactordrop.Model
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		filePath := filepath.Join(dropsPath, entry.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: failed to read file: %v", entry.Name(), err))
			result.FailedCount++
			continue
		}

		// Use jsonapi.Unmarshal to parse the JSON:API format
		var rm reactor.RestModel
		err = jsonapi.Unmarshal(data, &rm)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: failed to parse JSON:API: %v", entry.Name(), err))
			result.FailedCount++
			continue
		}

		// Extract reactor ID and drops from the RestModel
		reactorId, drops, err := reactor.Extract(rm)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: failed to extract reactor data: %v", entry.Name(), err))
			result.FailedCount++
			continue
		}

		// Convert each drop RestModel to a domain Model
		for _, dr := range drops {
			_, itemId, questId, chance := reactordrop.Extract(dr)
			m, err := reactordrop.NewReactorDropBuilder(p.t.Id(), 0).
				SetReactorId(reactorId).
				SetItemId(itemId).
				SetQuestId(questId).
				SetChance(chance).
				Build()
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("%s: failed to build reactor drop model: %v", entry.Name(), err))
				result.FailedCount++
				continue
			}
			models = append(models, m)
		}
	}

	if len(models) > 0 {
		err = reactordrop.BulkCreateReactorDrop(p.db, models)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("bulk create failed: %v", err))
			result.FailedCount += len(models)
		} else {
			result.CreatedCount = len(models)
		}
	}

	p.l.Debugf("Reactor drops seed: deleted=%d, created=%d, failed=%d", result.DeletedCount, result.CreatedCount, result.FailedCount)

	return result, nil
}
