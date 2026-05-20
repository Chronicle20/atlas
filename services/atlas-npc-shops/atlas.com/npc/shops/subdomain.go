package shops

import (
	"atlas-npc/commodities"
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	seeder "github.com/Chronicle20/atlas/libs/atlas-seeder"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// compile-time assertion
var _ seeder.Subdomain[JSONModel, ShopRecord] = ShopSubdomain{}

// ShopRecord bundles a shop and its commodities for a single seed file.
type ShopRecord struct {
	Shop        Model
	Commodities []commodities.Model
}

// ShopSubdomain implements seeder.Subdomain for npc shop seed data.
type ShopSubdomain struct{}

func (ShopSubdomain) Name() string { return "npc-shops" }
func (ShopSubdomain) Path() string { return "npc-shops/shops" }
func (ShopSubdomain) Type() string { return "npc-shop" }
func (ShopSubdomain) EntityIDPattern() *regexp.Regexp {
	return regexp.MustCompile(`^shop-(\d+)\.json$`)
}

func (ShopSubdomain) DeleteAllForTenant(db *gorm.DB) (int64, error) {
	tenantId := extractShopTenantId(db)

	// Delete commodities first (foreign-key relationship)
	commoditiesResult := db.Unscoped().Where("tenant_id = ?", tenantId).Delete(&commodities.Entity{})
	if commoditiesResult.Error != nil {
		return 0, commoditiesResult.Error
	}

	shopsResult := db.Unscoped().Where("tenant_id = ?", tenantId).Delete(&Entity{})
	if shopsResult.Error != nil {
		return 0, shopsResult.Error
	}

	return shopsResult.RowsAffected, nil
}

func (ShopSubdomain) Decode(payload []byte) (JSONModel, error) {
	var jm JSONModel
	if err := json.Unmarshal(payload, &jm); err != nil {
		return JSONModel{}, fmt.Errorf("npc-shops: decode: %w", err)
	}
	return jm, nil
}

func (ShopSubdomain) Build(t tenant.Model, _ string, jm JSONModel) ([]ShopRecord, error) {
	_ = t // tenant tracked via GORM context

	shop, err := NewBuilder(jm.NpcId).
		SetRecharger(jm.Recharger).
		Build()
	if err != nil {
		return nil, fmt.Errorf("npc-shops: build shop for NPC %d: %w", jm.NpcId, err)
	}

	var commodityModels []commodities.Model
	for _, cjm := range jm.Commodities {
		commodity, err := commodities.NewBuilder().
			SetId(uuid.New()).
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
			return nil, fmt.Errorf("npc-shops: build commodity for NPC %d template %d: %w", jm.NpcId, cjm.TemplateId, err)
		}
		commodityModels = append(commodityModels, commodity)
	}

	return []ShopRecord{{Shop: shop, Commodities: commodityModels}}, nil
}

func (ShopSubdomain) BulkCreate(db *gorm.DB, records []ShopRecord) error {
	if len(records) == 0 {
		return nil
	}

	tenantId := extractShopTenantId(db)

	var shopModels []Model
	var commodityModels []commodities.Model
	for _, r := range records {
		shopModels = append(shopModels, r.Shop)
		commodityModels = append(commodityModels, r.Commodities...)
	}

	if err := BulkCreateShops(db, tenantId, shopModels); err != nil {
		return fmt.Errorf("npc-shops: bulk create shops: %w", err)
	}
	if len(commodityModels) > 0 {
		if err := commodities.BulkCreateCommodities(db, tenantId, commodityModels); err != nil {
			return fmt.Errorf("npc-shops: bulk create commodities: %w", err)
		}
	}
	return nil
}

func (ShopSubdomain) Count(db *gorm.DB) (int64, *time.Time, error) {
	var count int64
	if err := db.Model(&Entity{}).Count(&count).Error; err != nil {
		return 0, nil, err
	}
	return count, nil, nil
}

// extractShopTenantId retrieves the tenant ID embedded in the GORM context.
func extractShopTenantId(db *gorm.DB) uuid.UUID {
	if db.Statement != nil && db.Statement.Context != nil {
		t := tenant.MustFromContext(db.Statement.Context)
		return t.Id()
	}
	return uuid.Nil
}
