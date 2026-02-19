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
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	return &ProcessorImpl{l: l, ctx: ctx, db: db}
}

func (p *ProcessorImpl) Seed() (CombinedSeedResult, error) {
	t := tenant.MustFromContext(p.ctx)
	p.l.Infof("Seeding gachapons for tenant [%s]", t.Id())

	result := CombinedSeedResult{}

	gachaponResult, err := p.seedGachapons(t)
	if err != nil {
		return result, fmt.Errorf("failed to seed gachapons: %w", err)
	}
	result.Gachapons = gachaponResult

	itemResult, err := p.seedItems(t)
	if err != nil {
		return result, fmt.Errorf("failed to seed items: %w", err)
	}
	result.Items = itemResult

	globalResult, err := p.seedGlobalItems(t)
	if err != nil {
		return result, fmt.Errorf("failed to seed global items: %w", err)
	}
	result.GlobalItems = globalResult

	p.l.Infof("Seed complete for tenant [%s]: gachapons=%d/%d, items=%d/%d, global=%d/%d",
		t.Id(),
		result.Gachapons.CreatedCount, result.Gachapons.DeletedCount,
		result.Items.CreatedCount, result.Items.DeletedCount,
		result.GlobalItems.CreatedCount, result.GlobalItems.DeletedCount)

	return result, nil
}

func (p *ProcessorImpl) seedGachapons(t tenant.Model) (SeedResult, error) {
	result := SeedResult{}

	deletedCount, err := gachapon.DeleteAllForTenant(p.db.WithContext(p.ctx))
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
		m, err := gachapon.NewBuilder(t.Id(), jm.Id).
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
		err = gachapon.BulkCreateGachapon(p.db.WithContext(p.ctx), models)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("bulk create failed: %v", err))
			result.FailedCount += len(models)
		} else {
			result.CreatedCount = len(models)
		}
	}

	return result, nil
}

func (p *ProcessorImpl) seedItems(t tenant.Model) (SeedResult, error) {
	result := SeedResult{}

	deletedCount, err := item.DeleteAllForTenant(p.db.WithContext(p.ctx))
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
		m, err := item.NewBuilder(t.Id(), 0).
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
		err = item.BulkCreateItem(p.db.WithContext(p.ctx), models)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("bulk create failed: %v", err))
			result.FailedCount += len(models)
		} else {
			result.CreatedCount = len(models)
		}
	}

	return result, nil
}

func (p *ProcessorImpl) seedGlobalItems(t tenant.Model) (SeedResult, error) {
	result := SeedResult{}

	deletedCount, err := global.DeleteAllForTenant(p.db.WithContext(p.ctx))
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
		m, err := global.NewBuilder(t.Id(), 0).
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
		err = global.BulkCreateItem(p.db.WithContext(p.ctx), models)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("bulk create failed: %v", err))
			result.FailedCount += len(models)
		} else {
			result.CreatedCount = len(models)
		}
	}

	return result, nil
}
