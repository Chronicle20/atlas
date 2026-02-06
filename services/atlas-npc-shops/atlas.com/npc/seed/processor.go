package seed

import (
	"atlas-npc/commodities"
	"atlas-npc/shops"
	"context"
	"fmt"

	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Processor interface {
	Seed() (SeedResult, error)
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

func (p *ProcessorImpl) Seed() (SeedResult, error) {
	p.l.Infof("Seeding shops for tenant [%s]", p.t.Id())

	result := SeedResult{}

	// Delete all existing commodities for this tenant first (due to foreign key relationship)
	deletedCommodities, err := commodities.DeleteAllCommoditiesForTenant(p.db, p.t.Id())
	if err != nil {
		return result, fmt.Errorf("failed to clear existing commodities: %w", err)
	}
	result.DeletedCommodities = int(deletedCommodities)

	// Delete all existing shops for this tenant
	deletedShops, err := shops.DeleteAllShopsForTenant(p.db, p.t.Id())
	if err != nil {
		return result, fmt.Errorf("failed to clear existing shops: %w", err)
	}
	result.DeletedShops = int(deletedShops)

	// Load shop files from the filesystem
	jsonModels, loadErrors := shops.LoadShopFiles()

	// Track load errors
	for _, err := range loadErrors {
		result.Errors = append(result.Errors, err.Error())
		result.FailedCount++
	}

	// Convert JSON models to domain models and bulk create
	var shopModels []shops.Model
	var commodityModels []commodities.Model

	for _, jm := range jsonModels {
		// Build shop model
		shopModel, err := shops.NewBuilder(jm.NpcId).
			SetRecharger(jm.Recharger).
			Build()
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("failed to build shop for NPC %d: %v", jm.NpcId, err))
			result.FailedCount++
			continue
		}
		shopModels = append(shopModels, shopModel)

		// Build commodity models for this shop
		for _, cjm := range jm.Commodities {
			commodityModel, err := (&commodities.ModelBuilder{}).
				SetNpcId(jm.NpcId).
				SetTemplateId(cjm.TemplateId).
				SetMesoPrice(cjm.MesoPrice).
				SetDiscountRate(cjm.DiscountRate).
				SetTokenTemplateId(cjm.TokenTemplateId).
				SetTokenPrice(cjm.TokenPrice).
				SetPeriod(cjm.Period).
				SetLevelLimit(cjm.LevelLimit).
				Build()
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("failed to build commodity for NPC %d, template %d: %v", jm.NpcId, cjm.TemplateId, err))
				result.FailedCount++
				continue
			}
			commodityModels = append(commodityModels, commodityModel)
		}
	}

	// Bulk create shops
	if len(shopModels) > 0 {
		err = shops.BulkCreateShops(p.db, p.t.Id(), shopModels)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("bulk create shops failed: %v", err))
			result.FailedCount += len(shopModels)
		} else {
			result.CreatedShops = len(shopModels)
		}
	}

	// Bulk create commodities
	if len(commodityModels) > 0 {
		err = commodities.BulkCreateCommodities(p.db, p.t.Id(), commodityModels)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("bulk create commodities failed: %v", err))
			result.FailedCount += len(commodityModels)
		} else {
			result.CreatedCommodities = len(commodityModels)
		}
	}

	p.l.Infof("Seed complete for tenant [%s]: shops_deleted=%d, shops_created=%d, commodities_deleted=%d, commodities_created=%d, failed=%d",
		p.t.Id(),
		result.DeletedShops, result.CreatedShops,
		result.DeletedCommodities, result.CreatedCommodities,
		result.FailedCount)

	return result, nil
}
