package item

import (
	"atlas-character/asset"
	"atlas-character/slottable"
	"context"
	"github.com/Chronicle20/atlas-model/model"
	tenant "github.com/Chronicle20/atlas-tenant"
	"gorm.io/gorm"
)

var entityModelMapper = model.SliceMap[entity, Model](makeModel)

func ByInventoryProvider(db *gorm.DB) func(ctx context.Context) func(inventoryId uint32) model.Provider[[]Model] {
	return func(ctx context.Context) func(inventoryId uint32) model.Provider[[]Model] {
		return func(inventoryId uint32) model.Provider[[]Model] {
			t := tenant.MustFromContext(ctx)
			return entityModelMapper(entityByInventory(t.Id())(inventoryId)(db))(model.ParallelMap())
		}
	}
}

func GetByInventory(db *gorm.DB) func(ctx context.Context) func(inventoryId uint32) ([]Model, error) {
	return func(ctx context.Context) func(inventoryId uint32) ([]Model, error) {
		return func(inventoryId uint32) ([]Model, error) {
			return ByInventoryProvider(db)(ctx)(inventoryId)()
		}
	}
}

var ModelAssetMapper = model.Map(ToAsset)

var ModelSliceAssetMapper = model.SliceMap(ToAsset)

func ExtractFunc[A any, B any, C any](f1 func(B) C) func(f2 func(A) B) func(a A) C {
	return func(f2 func(A) B) func(a A) C {
		return func(a A) C {
			return model.Curry(model.Compose[A, B, C])(f1)(f2)(a)
		}
	}
}

var AssetBySlotProvider = ExtractFunc[*gorm.DB](ExtractFunc[context.Context](ExtractFunc[uint32](ExtractFunc[int16](ModelAssetMapper))))(BySlotProvider)

func BySlotProvider(db *gorm.DB) func(ctx context.Context) func(inventoryId uint32) func(slot int16) model.Provider[Model] {
	return func(ctx context.Context) func(inventoryId uint32) func(slot int16) model.Provider[Model] {
		return func(inventoryId uint32) func(slot int16) model.Provider[Model] {
			return func(slot int16) model.Provider[Model] {
				t := tenant.MustFromContext(ctx)
				return model.Map[entity, Model](makeModel)(getBySlot(t.Id(), inventoryId, slot)(db))
			}
		}
	}
}

func GetBySlot(db *gorm.DB) func(ctx context.Context) func(inventoryId uint32, slot int16) (Model, error) {
	return func(ctx context.Context) func(inventoryId uint32, slot int16) (Model, error) {
		return func(inventoryId uint32, slot int16) (Model, error) {
			return BySlotProvider(db)(ctx)(inventoryId)(slot)()
		}
	}
}

func GetById(db *gorm.DB) func(ctx context.Context) func(id uint32) (Model, error) {
	return func(ctx context.Context) func(id uint32) (Model, error) {
		return func(id uint32) (Model, error) {
			t := tenant.MustFromContext(ctx)
			return model.Map[entity, Model](makeModel)(getById(t.Id(), id)(db))()
		}
	}
}

func AssetByItemIdProvider(db *gorm.DB) func(ctx context.Context) func(inventoryId uint32) func(itemId uint32) model.Provider[[]asset.Asset] {
	return func(ctx context.Context) func(inventoryId uint32) func(itemId uint32) model.Provider[[]asset.Asset] {
		return func(inventoryId uint32) func(itemId uint32) model.Provider[[]asset.Asset] {
			return func(itemId uint32) model.Provider[[]asset.Asset] {
				return ModelSliceAssetMapper(ByItemIdProvider(db)(ctx)(inventoryId)(itemId))(model.ParallelMap())
			}
		}
	}
}

func ByItemIdProvider(db *gorm.DB) func(ctx context.Context) func(inventoryId uint32) func(itemId uint32) model.Provider[[]Model] {
	return func(ctx context.Context) func(inventoryId uint32) func(itemId uint32) model.Provider[[]Model] {
		return func(inventoryId uint32) func(itemId uint32) model.Provider[[]Model] {
			return func(itemId uint32) model.Provider[[]Model] {
				t := tenant.MustFromContext(ctx)
				return entityModelMapper(getForCharacter(t.Id(), inventoryId, itemId)(db))(model.ParallelMap())
			}
		}
	}
}

func GetByItemId(db *gorm.DB) func(ctx context.Context) func(inventoryId uint32, itemId uint32) ([]Model, error) {
	return func(ctx context.Context) func(inventoryId uint32, itemId uint32) ([]Model, error) {
		return func(inventoryId uint32, itemId uint32) ([]Model, error) {
			return ByItemIdProvider(db)(ctx)(inventoryId)(itemId)()
		}
	}
}

func UpdateSlot(db *gorm.DB) func(ctx context.Context) func(id uint32, slot int16) error {
	return func(ctx context.Context) func(id uint32, slot int16) error {
		return func(id uint32, slot int16) error {
			return updateSlot(db, id, slot)
		}
	}
}

func UpdateQuantity(db *gorm.DB) func(ctx context.Context) func(id uint32, quantity uint32) error {
	return func(ctx context.Context) func(id uint32, quantity uint32) error {
		return func(id uint32, quantity uint32) error {
			i, err := GetById(db)(ctx)(id)
			if err != nil {
				return err
			}
			return updateQuantity(db, i.Id(), quantity)
		}
	}
}

func CreateItem(db *gorm.DB) func(ctx context.Context) asset.CharacterAssetCreator {
	return func(ctx context.Context) asset.CharacterAssetCreator {
		return func(characterId uint32) asset.InventoryAssetCreator {
			return func(inventoryId uint32, inventoryType int8) asset.ItemCreator {
				return func(itemId uint32) asset.Creator {
					return func(quantity uint32) model.Provider[asset.Asset] {
						t := tenant.MustFromContext(ctx)
						slot, err := slottable.GetNextFreeSlot(model.SliceMap(ToSlottable)(ByInventoryProvider(db)(ctx)(inventoryId))(model.ParallelMap()))
						if err != nil {
							return model.ErrorProvider[asset.Asset](err)
						}
						i, err := createItem(db, t.Id(), inventoryId, itemId, quantity, slot)
						if err != nil {
							return model.ErrorProvider[asset.Asset](err)
						}
						return model.FixedProvider[asset.Asset](i)
					}
				}
			}
		}
	}
}

func ToAsset(m Model) (asset.Asset, error) {
	return m, nil
}

func ToSlottable(m Model) (asset.Slottable, error) {
	return m, nil
}

func RemoveItem(db *gorm.DB) func(characterId uint32, id uint32) error {
	return func(characterId uint32, id uint32) error {
		return remove(db, characterId, id)
	}
}

func DeleteById(db *gorm.DB) func(ctx context.Context) model.Operator[uint32] {
	return func(ctx context.Context) model.Operator[uint32] {
		return func(id uint32) error {
			t := tenant.MustFromContext(ctx)
			return deleteById(db, t.Id(), id)
		}
	}
}
