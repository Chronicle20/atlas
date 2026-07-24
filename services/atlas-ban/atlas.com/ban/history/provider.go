package history

import (
	database "github.com/Chronicle20/atlas/libs/atlas-database"

	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func entitiesByAccountId(accountId uint32, page model.Page) database.EntityProvider[model.Paged[Entity]] {
	return func(db *gorm.DB) model.Provider[model.Paged[Entity]] {
		return database.PagedQuery[Entity](db.Where("account_id = ?", accountId).Order("created_at desc"), page)
	}
}

func entitiesByIP(ip string, page model.Page) database.EntityProvider[model.Paged[Entity]] {
	return func(db *gorm.DB) model.Provider[model.Paged[Entity]] {
		return database.PagedQuery[Entity](db.Where("ip_address = ?", ip).Order("created_at desc"), page)
	}
}

func entitiesByHWID(hwid string, page model.Page) database.EntityProvider[model.Paged[Entity]] {
	return func(db *gorm.DB) model.Provider[model.Paged[Entity]] {
		return database.PagedQuery[Entity](db.Where("hw_id = ?", hwid).Order("created_at desc"), page)
	}
}

func entitiesByTenant(page model.Page) database.EntityProvider[model.Paged[Entity]] {
	return func(db *gorm.DB) model.Provider[model.Paged[Entity]] {
		return database.PagedQuery[Entity](db.Order("created_at desc"), page)
	}
}
