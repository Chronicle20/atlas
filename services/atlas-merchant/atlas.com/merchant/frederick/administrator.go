package frederick

import (
	"time"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func storeItems(tenantId uuid.UUID, characterId uint32, items []StoredItem) database.EntityProvider[bool] {
	return func(db *gorm.DB) model.Provider[bool] {
		err := database.ExecuteTransaction(db, func(tx *gorm.DB) error {
			now := time.Now()
			for _, item := range items {
				entity := &ItemEntity{
					Id:           uuid.New(),
					TenantId:     tenantId,
					CharacterId:  characterId,
					ItemId:       item.ItemId,
					ItemType:     item.ItemType,
					Quantity:     item.Quantity,
					ItemSnapshot: item.ItemSnapshot,
					StoredAt:     now,
				}
				if err := tx.Create(entity).Error; err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return model.ErrorProvider[bool](err)
		}
		return model.FixedProvider(true)
	}
}

func storeMesos(tenantId uuid.UUID, characterId uint32, amount uint32) database.EntityProvider[bool] {
	return func(db *gorm.DB) model.Provider[bool] {
		if amount == 0 {
			return model.FixedProvider(true)
		}
		entity := &MesoEntity{
			Id:          uuid.New(),
			TenantId:    tenantId,
			CharacterId: characterId,
			Amount:      amount,
			StoredAt:    time.Now(),
		}
		err := db.Create(entity).Error
		if err != nil {
			return model.ErrorProvider[bool](err)
		}
		return model.FixedProvider(true)
	}
}

func clearItems(characterId uint32) database.EntityProvider[bool] {
	return func(db *gorm.DB) model.Provider[bool] {
		err := db.Where("character_id = ?", characterId).Delete(&ItemEntity{}).Error
		if err != nil {
			return model.ErrorProvider[bool](err)
		}
		return model.FixedProvider(true)
	}
}

func clearMesos(characterId uint32) database.EntityProvider[bool] {
	return func(db *gorm.DB) model.Provider[bool] {
		err := db.Where("character_id = ?", characterId).Delete(&MesoEntity{}).Error
		if err != nil {
			return model.ErrorProvider[bool](err)
		}
		return model.FixedProvider(true)
	}
}

func createNotification(t tenant.Model, characterId uint32) database.EntityProvider[NotificationEntity] {
	return func(db *gorm.DB) model.Provider[NotificationEntity] {
		entity := &NotificationEntity{
			Id:           uuid.New(),
			TenantId:     t.Id(),
			TenantRegion: t.Region(),
			TenantMajor:  t.MajorVersion(),
			TenantMinor:  t.MinorVersion(),
			CharacterId:  characterId,
			StoredAt:     time.Now(),
			NextDay:      2,
		}
		err := db.Create(entity).Error
		if err != nil {
			return model.ErrorProvider[NotificationEntity](err)
		}
		return model.FixedProvider(*entity)
	}
}

func clearNotifications(characterId uint32) database.EntityProvider[bool] {
	return func(db *gorm.DB) model.Provider[bool] {
		err := db.Where("character_id = ?", characterId).Delete(&NotificationEntity{}).Error
		if err != nil {
			return model.ErrorProvider[bool](err)
		}
		return model.FixedProvider(true)
	}
}

func cleanupExpiredItems(cutoff time.Time) database.EntityProvider[int64] {
	return func(db *gorm.DB) model.Provider[int64] {
		result := db.Where("stored_at < ?", cutoff).Delete(&ItemEntity{})
		if result.Error != nil {
			return model.ErrorProvider[int64](result.Error)
		}
		return model.FixedProvider(result.RowsAffected)
	}
}

func advanceNotification(id uuid.UUID, nextDay uint16) database.EntityProvider[bool] {
	return func(db *gorm.DB) model.Provider[bool] {
		err := db.Model(&NotificationEntity{}).Where("id = ?", id).Update("next_day", nextDay).Error
		if err != nil {
			return model.ErrorProvider[bool](err)
		}
		return model.FixedProvider(true)
	}
}

func deleteNotification(id uuid.UUID) database.EntityProvider[bool] {
	return func(db *gorm.DB) model.Provider[bool] {
		err := db.Where("id = ?", id).Delete(&NotificationEntity{}).Error
		if err != nil {
			return model.ErrorProvider[bool](err)
		}
		return model.FixedProvider(true)
	}
}

func cleanupExpiredMesos(cutoff time.Time) database.EntityProvider[int64] {
	return func(db *gorm.DB) model.Provider[int64] {
		result := db.Where("stored_at < ?", cutoff).Delete(&MesoEntity{})
		if result.Error != nil {
			return model.ErrorProvider[int64](result.Error)
		}
		return model.FixedProvider(result.RowsAffected)
	}
}
