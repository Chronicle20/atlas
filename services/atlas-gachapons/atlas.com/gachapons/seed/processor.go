package seed

import (
	"atlas-gachapons/gachapon"
	"atlas-gachapons/global"
	"atlas-gachapons/item"
	"context"
	"fmt"

	tenant "github.com/Chronicle20/atlas-tenant"
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
	return &ProcessorImpl{l: l, ctx: ctx, db: db, t: t}
}

func (p *ProcessorImpl) Seed() (CombinedSeedResult, error) {
	p.l.Infof("Seeding gachapons for tenant [%s]", p.t.Id())

	result := CombinedSeedResult{}

	gachaponResult, err := p.seedGachapons()
	if err != nil {
		return result, fmt.Errorf("failed to seed gachapons: %w", err)
	}
	result.Gachapons = gachaponResult

	itemResult, err := p.seedItems()
	if err != nil {
		return result, fmt.Errorf("failed to seed items: %w", err)
	}
	result.Items = itemResult

	globalResult, err := p.seedGlobalItems()
	if err != nil {
		return result, fmt.Errorf("failed to seed global items: %w", err)
	}
	result.GlobalItems = globalResult

	p.l.Infof("Seed complete for tenant [%s]: gachapons=%d/%d, items=%d/%d, global=%d/%d",
		p.t.Id(),
		result.Gachapons.CreatedCount, result.Gachapons.DeletedCount,
		result.Items.CreatedCount, result.Items.DeletedCount,
		result.GlobalItems.CreatedCount, result.GlobalItems.DeletedCount)

	return result, nil
}

func (p *ProcessorImpl) seedGachapons() (SeedResult, error) {
	result := SeedResult{}

	deletedCount, err := gachapon.DeleteAllForTenant(p.db, p.t.Id())
	if err != nil {
		return result, fmt.Errorf("failed to clear existing gachapons: %w", err)
	}
	result.DeletedCount = int(deletedCount)

	jsonModels, err := LoadGachapons()
	if err != nil {
		result.Errors = append(result.Errors, err.Error())
		result.FailedCount++
		return result, nil
	}

	var models []gachapon.Model
	for _, jm := range jsonModels {
		m, err := gachapon.NewBuilder(p.t.Id(), jm.Id).
			SetName(jm.Name).
			SetNpcIds(jm.NpcIds).
			SetCommonWeight(jm.CommonWeight).
			SetUncommonWeight(jm.UncommonWeight).
			SetRareWeight(jm.RareWeight).
			Build()
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("failed to build gachapon model: %v", err))
			result.FailedCount++
			continue
		}
		models = append(models, m)
	}

	if len(models) > 0 {
		err = gachapon.BulkCreateGachapon(p.db, models)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("bulk create failed: %v", err))
			result.FailedCount += len(models)
		} else {
			result.CreatedCount = len(models)
		}
	}

	return result, nil
}

func (p *ProcessorImpl) seedItems() (SeedResult, error) {
	result := SeedResult{}

	deletedCount, err := item.DeleteAllForTenant(p.db, p.t.Id())
	if err != nil {
		return result, fmt.Errorf("failed to clear existing items: %w", err)
	}
	result.DeletedCount = int(deletedCount)

	jsonModels, err := LoadItems()
	if err != nil {
		result.Errors = append(result.Errors, err.Error())
		result.FailedCount++
		return result, nil
	}

	var models []item.Model
	for _, jm := range jsonModels {
		m, err := item.NewBuilder(p.t.Id(), 0).
			SetGachaponId(jm.GachaponId).
			SetItemId(jm.ItemId).
			SetQuantity(jm.Quantity).
			SetTier(jm.Tier).
			Build()
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("failed to build item model: %v", err))
			result.FailedCount++
			continue
		}
		models = append(models, m)
	}

	if len(models) > 0 {
		err = item.BulkCreateItem(p.db, models)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("bulk create failed: %v", err))
			result.FailedCount += len(models)
		} else {
			result.CreatedCount = len(models)
		}
	}

	return result, nil
}

func (p *ProcessorImpl) seedGlobalItems() (SeedResult, error) {
	result := SeedResult{}

	deletedCount, err := global.DeleteAllForTenant(p.db, p.t.Id())
	if err != nil {
		return result, fmt.Errorf("failed to clear existing global items: %w", err)
	}
	result.DeletedCount = int(deletedCount)

	jsonModels, err := LoadGlobalItems()
	if err != nil {
		result.Errors = append(result.Errors, err.Error())
		result.FailedCount++
		return result, nil
	}

	var models []global.Model
	for _, jm := range jsonModels {
		m, err := global.NewBuilder(p.t.Id(), 0).
			SetItemId(jm.ItemId).
			SetQuantity(jm.Quantity).
			SetTier(jm.Tier).
			Build()
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("failed to build global item model: %v", err))
			result.FailedCount++
			continue
		}
		models = append(models, m)
	}

	if len(models) > 0 {
		err = global.BulkCreateItem(p.db, models)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("bulk create failed: %v", err))
			result.FailedCount += len(models)
		} else {
			result.CreatedCount = len(models)
		}
	}

	return result, nil
}
