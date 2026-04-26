package searchindex

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	MaxQueryLen = 128
	MaxLimit    = 50
)

// MigrationOptions describes the optional raw-SQL statements emitted after
// AutoMigrate creates the table. Each statement is Execed in order; each must
// be idempotent (typical shape: CREATE INDEX IF NOT EXISTS ...).
type MigrationOptions struct {
	IndexStatements []string
}

// Migrate enables pg_trgm (idempotent), AutoMigrates the search-index entity,
// then executes each index statement.
func Migrate(db *gorm.DB, entity interface{}, opts MigrationOptions) error {
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS pg_trgm").Error; err != nil {
		return err
	}
	if err := db.AutoMigrate(entity); err != nil {
		return err
	}
	for _, stmt := range opts.IndexStatements {
		if err := db.Exec(stmt).Error; err != nil {
			return err
		}
	}
	return nil
}

// Upsert upserts a single search-index row inside the caller-supplied
// transaction. pkCols are the composite primary-key columns; updateCols are
// non-PK columns to overwrite on conflict.
func Upsert(tx *gorm.DB, row interface{}, pkCols []string, updateCols []string) error {
	cols := make([]clause.Column, 0, len(pkCols))
	for _, c := range pkCols {
		cols = append(cols, clause.Column{Name: c})
	}
	return tx.Clauses(clause.OnConflict{
		Columns:   cols,
		DoUpdates: clause.AssignmentColumns(updateCols),
	}).Create(row).Error
}

// DeleteAllForTenant deletes every row with the given tenant_id from the
// table backing entity.
func DeleteAllForTenant(tx *gorm.DB, tenantId uuid.UUID, entity interface{}) error {
	return tx.Where("tenant_id = ?", tenantId).Delete(entity).Error
}

// ResolveTenantId picks the tenant_id partition this request should query.
// Returns the active tenant id if the active tenant has any rows in the
// resource's search-index table; otherwise returns uuid.Nil.
//
// The check is a single EXISTS lookup against the existing (tenant_id, ...)
// PK / btree. Callers MUST resolve once per request and pass the result into
// Search / SearchWithFilter / Count / CountWithFilter so all queries see the
// same partition.
func ResolveTenantId[E any](db *gorm.DB, ctx context.Context, _ QuerySpec[E]) (uuid.UUID, error) {
	t := tenant.MustFromContext(ctx)
	bypass := database.WithoutTenantFilter(ctx)

	var entity E
	var dummy int
	err := db.WithContext(bypass).
		Model(&entity).
		Select("1").
		Where("tenant_id = ?", t.Id()).
		Limit(1).
		Scan(&dummy).Error
	if err != nil {
		return uuid.Nil, err
	}
	if dummy == 1 {
		return t.Id(), nil
	}
	return uuid.Nil, nil
}

// QuerySpec configures Search / Count calls. IdOf has been removed; the lib
// no longer dedupes Go-side because all queries hit a single tenant partition.
type QuerySpec[E any] struct {
	EntityIdColumn string
	NameColumns    []string
	ExtraPredicate string
	ExtraArgs      []interface{}
	Order          string
}

// Search runs the substring + exact-id-hoist data query against a single
// tenant partition. The exact-id hoist applies only when offset == 0
// (page 1). On pages >= 2 the hoist is skipped and the substring fetch runs
// with OFFSET/LIMIT against natural ordering.
func Search[E any](
	db *gorm.DB, ctx context.Context,
	tenantId uuid.UUID, q string,
	offset, limit int,
	spec QuerySpec[E],
) ([]E, error) {
	bypass := database.WithoutTenantFilter(ctx)
	results := make([]E, 0, limit)

	pattern := "%" + strings.ToLower(q) + "%"
	nameOrs := make([]string, 0, len(spec.NameColumns))
	nameArgs := make([]interface{}, 0, len(spec.NameColumns))
	for _, col := range spec.NameColumns {
		nameOrs = append(nameOrs, "LOWER("+col+") LIKE ?")
		nameArgs = append(nameArgs, pattern)
	}

	order := spec.Order
	if order == "" {
		order = fmt.Sprintf("name ASC, %s ASC", spec.EntityIdColumn)
	}

	if offset == 0 {
		// Page 1: exact-id hoist if q is numeric.
		hoisted := false
		var hoistedNumeric int
		if numeric, err := strconv.Atoi(q); err == nil {
			where := []string{"tenant_id = ?", spec.EntityIdColumn + " = ?"}
			args := []interface{}{tenantId, numeric}
			if spec.ExtraPredicate != "" {
				where = append(where, "("+spec.ExtraPredicate+")")
				args = append(args, spec.ExtraArgs...)
			}
			var row E
			exactErr := db.WithContext(bypass).
				Where(strings.Join(where, " AND "), args...).
				Take(&row).Error
			if exactErr == nil {
				results = append(results, row)
				hoisted = true
				hoistedNumeric = numeric
				if len(results) >= limit {
					return results, nil
				}
			}
		}

		// Substring fetch. Exclude the hoisted id directly in SQL so we
		// don't need a Go-side dedup (and so the LIMIT is exact).
		where := []string{"tenant_id = ?", "(" + strings.Join(nameOrs, " OR ") + ")"}
		args := []interface{}{tenantId}
		args = append(args, nameArgs...)
		if spec.ExtraPredicate != "" {
			where = append(where, "("+spec.ExtraPredicate+")")
			args = append(args, spec.ExtraArgs...)
		}
		if hoisted {
			where = append(where, spec.EntityIdColumn+" <> ?")
			args = append(args, hoistedNumeric)
		}

		fetch := limit - len(results)
		var rows []E
		if err := db.WithContext(bypass).
			Where(strings.Join(where, " AND "), args...).
			Order(order).
			Limit(fetch).
			Find(&rows).Error; err != nil {
			return nil, err
		}
		results = append(results, rows...)
		return results, nil
	}

	// Page >= 2: natural-ordered substring with OFFSET/LIMIT. No hoist.
	where := []string{"tenant_id = ?", "(" + strings.Join(nameOrs, " OR ") + ")"}
	args := []interface{}{tenantId}
	args = append(args, nameArgs...)
	if spec.ExtraPredicate != "" {
		where = append(where, "("+spec.ExtraPredicate+")")
		args = append(args, spec.ExtraArgs...)
	}

	var rows []E
	if err := db.WithContext(bypass).
		Where(strings.Join(where, " AND "), args...).
		Order(order).
		Offset(offset).
		Limit(limit).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// SearchWithFilter runs the filter-only data query (no substring match)
// against a single tenant partition with OFFSET/LIMIT.
func SearchWithFilter[E any](
	db *gorm.DB, ctx context.Context,
	tenantId uuid.UUID,
	offset, limit int,
	spec QuerySpec[E],
) ([]E, error) {
	bypass := database.WithoutTenantFilter(ctx)

	where := []string{"tenant_id = ?"}
	args := []interface{}{tenantId}
	if spec.ExtraPredicate != "" {
		where = append(where, "("+spec.ExtraPredicate+")")
		args = append(args, spec.ExtraArgs...)
	}
	order := spec.Order
	if order == "" {
		order = fmt.Sprintf("name ASC, %s ASC", spec.EntityIdColumn)
	}

	var rows []E
	if err := db.WithContext(bypass).
		Where(strings.Join(where, " AND "), args...).
		Order(order).
		Offset(offset).
		Limit(limit).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// Count returns the total matching row count for the substring + (optional)
// exact-id predicate against a single tenant partition. The exact-id branch
// is included only when q parses as a non-negative integer.
func Count[E any](db *gorm.DB, ctx context.Context, tenantId uuid.UUID, q string, spec QuerySpec[E]) (int, error) {
	bypass := database.WithoutTenantFilter(ctx)

	pattern := "%" + strings.ToLower(q) + "%"
	nameOrs := make([]string, 0, len(spec.NameColumns))
	nameArgs := make([]interface{}, 0, len(spec.NameColumns))
	for _, col := range spec.NameColumns {
		nameOrs = append(nameOrs, "LOWER("+col+") LIKE ?")
		nameArgs = append(nameArgs, pattern)
	}

	matchClause := "(" + strings.Join(nameOrs, " OR ") + ")"
	args := []interface{}{tenantId}
	args = append(args, nameArgs...)

	if numeric, err := strconv.Atoi(q); err == nil {
		matchClause = "(" + matchClause + " OR " + spec.EntityIdColumn + " = ?)"
		args = append(args, numeric)
	}

	where := []string{"tenant_id = ?", matchClause}
	if spec.ExtraPredicate != "" {
		where = append(where, "("+spec.ExtraPredicate+")")
		args = append(args, spec.ExtraArgs...)
	}

	var entity E
	var n int64
	err := db.WithContext(bypass).
		Model(&entity).
		Where(strings.Join(where, " AND "), args...).
		Count(&n).Error
	if err != nil {
		return 0, err
	}
	return int(n), nil
}

// CountWithFilter returns the total matching row count for the filter-only
// predicate against a single tenant partition.
func CountWithFilter[E any](db *gorm.DB, ctx context.Context, tenantId uuid.UUID, spec QuerySpec[E]) (int, error) {
	bypass := database.WithoutTenantFilter(ctx)

	where := []string{"tenant_id = ?"}
	args := []interface{}{tenantId}
	if spec.ExtraPredicate != "" {
		where = append(where, "("+spec.ExtraPredicate+")")
		args = append(args, spec.ExtraArgs...)
	}

	var entity E
	var n int64
	err := db.WithContext(bypass).
		Model(&entity).
		Where(strings.Join(where, " AND "), args...).
		Count(&n).Error
	if err != nil {
		return 0, err
	}
	return int(n), nil
}
