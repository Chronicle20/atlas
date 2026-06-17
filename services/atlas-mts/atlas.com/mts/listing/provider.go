package listing

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"gorm.io/gorm"
)

func getAll() database.EntityProvider[[]entity] {
	return func(db *gorm.DB) model.Provider[[]entity] {
		return database.SliceQuery[entity](db, &entity{})
	}
}

func getById(id string) database.EntityProvider[entity] {
	return func(db *gorm.DB) model.Provider[entity] {
		return database.Query[entity](db, &entity{Id: parseId(id)})
	}
}

// getBrowse returns the listings for a world filtered by state and category.
//
// The filter is built as an explicit name-keyed map rather than a struct
// condition: GORM's struct-condition Where elides zero-valued fields, so a
// struct condition would silently drop the world_id filter for world 0 (a
// valid world.Id, since world.Id is a byte) and return cross-world rows. The
// map form forces every filter column into the WHERE clause.
func getBrowse(worldId world.Id, state State, category string) database.EntityProvider[[]entity] {
	return func(db *gorm.DB) model.Provider[[]entity] {
		var results []entity
		err := db.Where(map[string]interface{}{
			"world_id": byte(worldId),
			"state":    string(state),
			"category": category,
		}).Find(&results).Error
		if err != nil {
			return model.ErrorProvider[[]entity](err)
		}
		return model.FixedProvider(results)
	}
}

func modelFromEntity(e entity) (Model, error) {
	b := NewBuilder(e.TenantId, world.Id(e.WorldId), e.SellerId).
		SetId(e.Id).
		SetSellerName(e.SellerName).
		SetSaleType(SaleType(e.SaleType)).
		SetState(State(e.State)).
		SetTemplateId(e.TemplateId).
		SetQuantity(e.Quantity).
		SetStrength(e.Strength).
		SetDexterity(e.Dexterity).
		SetIntelligence(e.Intelligence).
		SetLuck(e.Luck).
		SetHP(e.HP).
		SetMP(e.MP).
		SetWeaponAttack(e.WeaponAttack).
		SetMagicAttack(e.MagicAttack).
		SetWeaponDefense(e.WeaponDefense).
		SetMagicDefense(e.MagicDefense).
		SetAccuracy(e.Accuracy).
		SetAvoidability(e.Avoidability).
		SetHands(e.Hands).
		SetSpeed(e.Speed).
		SetJump(e.Jump).
		SetSlots(e.Slots).
		SetLevel(e.Level).
		SetItemLevel(e.ItemLevel).
		SetItemExp(e.ItemExp).
		SetRingId(e.RingId).
		SetViciousCount(e.ViciousCount).
		SetFlags(e.Flags).
		SetListValue(e.ListValue).
		SetBuyNowPrice(e.BuyNowPrice).
		SetCommissionRate(e.CommissionRate).
		SetCategory(e.Category).
		SetSubCategory(e.SubCategory).
		SetEndsAt(e.EndsAt).
		SetCurrentBid(e.CurrentBid).
		SetHighBidderId(e.HighBidderId).
		SetMinIncrement(e.MinIncrement).
		SetCreatedAt(e.CreatedAt).
		SetUpdatedAt(e.UpdatedAt)
	return b.Build()
}
