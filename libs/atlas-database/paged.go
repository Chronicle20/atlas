// libs/atlas-database/paged.go
package database

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// PagedQuery runs a COUNT plus an OFFSET/LIMIT Find against the same scoped
// *gorm.DB, so the tenant-filter callback and all Where clauses apply
// identically to both. A schema-derived primary-key ordering is appended
// after any caller-supplied ordering so pages form a total order.
// page.Number is 1-based. Lazy: nothing executes until the provider is invoked.
func PagedQuery[E any](db *gorm.DB, page model.Page) model.Provider[model.Paged[E]] {
	return func() (model.Paged[E], error) {
		if page.Number < 1 || page.Size < 1 {
			return model.Paged[E]{}, fmt.Errorf("invalid page number=%d size=%d", page.Number, page.Size)
		}

		var e E
		// Count on a session clone with ORDER BY stripped explicitly —
		// GORM's own order-stripping inside Count is an implementation
		// detail we do not rely on (design §3.2). The clause map is copied
		// explicitly so deleting the ORDER BY entry cannot mutate the
		// original db's Statement.Clauses (they share the same underlying
		// map by default under Session()).
		countDB := db.Session(&gorm.Session{}).Model(&e)
		clauses := make(map[string]clause.Clause, len(countDB.Statement.Clauses))
		for k, v := range countDB.Statement.Clauses {
			clauses[k] = v
		}
		delete(clauses, "ORDER BY")
		countDB.Statement.Clauses = clauses
		var total int64
		if err := countDB.Count(&total).Error; err != nil {
			return model.Paged[E]{}, err
		}

		stmt := &gorm.Statement{DB: db}
		if err := stmt.Parse(&e); err != nil {
			return model.Paged[E]{}, err
		}
		pk := stmt.Schema.PrioritizedPrimaryField
		if pk == nil {
			return model.Paged[E]{}, fmt.Errorf("entity for table %s has no primary key; stable paging requires one", stmt.Schema.Table)
		}

		var results []E
		err := db.Session(&gorm.Session{}).
			Order(clause.OrderByColumn{Column: clause.Column{Name: pk.DBName}}).
			Offset((page.Number - 1) * page.Size).
			Limit(page.Size).
			Find(&results).Error
		if err != nil {
			return model.Paged[E]{}, err
		}
		return model.Paged[E]{Items: results, Total: int(total), Page: page}, nil
	}
}
