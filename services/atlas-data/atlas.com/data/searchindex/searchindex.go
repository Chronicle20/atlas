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

// QuerySpec configures a Search call for a specific search-index table.
//
// The entity type E is the GORM-mapped search-index row; IdOf extracts its
// entity-id for dedup across the tenant-scoped and global query phases.
type QuerySpec[E any] struct {
	// EntityIdColumn is the per-resource id column name (e.g. "map_id", "npc_id").
	EntityIdColumn string
	// NameColumns is the ordered list of text columns to match case-insensitively.
	NameColumns []string
	// ExtraPredicate is an optional SQL fragment AND-joined into every query.
	// It is parameterized: use ? placeholders bound to ExtraArgs.
	ExtraPredicate string
	ExtraArgs      []interface{}
	// Order overrides the default `name ASC, <EntityIdColumn> ASC`.
	Order string
	// IdOf extracts the entity-id used to dedup results across tenant phases.
	IdOf func(E) uint64
}

// Search runs the two-phase tenant lookup: active tenant first, then uuid.Nil
// fallback. Numeric queries perform an exact-id lookup before the substring
// match and are unioned ahead of substring results. Results are deduplicated
// by IdOf, with tenant-scoped rows winning over globals.
func Search[E any](db *gorm.DB, ctx context.Context, q string, limit int, spec QuerySpec[E]) ([]E, error) {
	t := tenant.MustFromContext(ctx)
	bypass := database.WithoutTenantFilter(ctx)

	results, seen, err := runTenantQueries(db, bypass, t.Id(), q, limit, spec)
	if err != nil {
		return nil, err
	}
	if len(results) >= limit {
		return results[:limit], nil
	}

	globalResults, _, err := runTenantQueries(db, bypass, uuid.Nil, q, limit, spec)
	if err != nil {
		return nil, err
	}
	for _, gr := range globalResults {
		id := spec.IdOf(gr)
		if _, ok := seen[id]; ok {
			continue
		}
		results = append(results, gr)
		seen[id] = struct{}{}
		if len(results) >= limit {
			break
		}
	}
	return results, nil
}

// SearchWithFilter runs the same two-phase lookup as Search but skips the
// substring match entirely. Useful for filter-only fast paths (e.g. the NPC
// `filter[storebank]=true` request with no search term).
func SearchWithFilter[E any](db *gorm.DB, ctx context.Context, limit int, spec QuerySpec[E]) ([]E, error) {
	t := tenant.MustFromContext(ctx)
	bypass := database.WithoutTenantFilter(ctx)

	results, seen, err := runFilterQuery(db, bypass, t.Id(), limit, spec)
	if err != nil {
		return nil, err
	}
	if len(results) >= limit {
		return results[:limit], nil
	}

	globals, _, err := runFilterQuery(db, bypass, uuid.Nil, limit, spec)
	if err != nil {
		return nil, err
	}
	for _, gr := range globals {
		id := spec.IdOf(gr)
		if _, ok := seen[id]; ok {
			continue
		}
		results = append(results, gr)
		seen[id] = struct{}{}
		if len(results) >= limit {
			break
		}
	}
	return results, nil
}

func runTenantQueries[E any](db *gorm.DB, ctx context.Context, tenantId uuid.UUID, q string, limit int, spec QuerySpec[E]) ([]E, map[uint64]struct{}, error) {
	seen := make(map[uint64]struct{})
	results := make([]E, 0, limit)

	if numeric, err := strconv.Atoi(q); err == nil {
		var row E
		where := []string{"tenant_id = ?", spec.EntityIdColumn + " = ?"}
		args := []interface{}{tenantId, numeric}
		if spec.ExtraPredicate != "" {
			where = append(where, "("+spec.ExtraPredicate+")")
			args = append(args, spec.ExtraArgs...)
		}
		exactErr := db.WithContext(ctx).
			Where(strings.Join(where, " AND "), args...).
			Take(&row).Error
		if exactErr == nil {
			results = append(results, row)
			seen[spec.IdOf(row)] = struct{}{}
		}
	}

	if len(results) >= limit {
		return results, seen, nil
	}

	pattern := "%" + strings.ToLower(q) + "%"
	nameOrs := make([]string, 0, len(spec.NameColumns))
	nameArgs := make([]interface{}, 0, len(spec.NameColumns))
	for _, col := range spec.NameColumns {
		nameOrs = append(nameOrs, "LOWER("+col+") LIKE ?")
		nameArgs = append(nameArgs, pattern)
	}
	where := []string{"tenant_id = ?", "(" + strings.Join(nameOrs, " OR ") + ")"}
	args := []interface{}{tenantId}
	args = append(args, nameArgs...)
	if spec.ExtraPredicate != "" {
		where = append(where, "("+spec.ExtraPredicate+")")
		args = append(args, spec.ExtraArgs...)
	}

	order := spec.Order
	if order == "" {
		order = fmt.Sprintf("name ASC, %s ASC", spec.EntityIdColumn)
	}

	var rows []E
	err := db.WithContext(ctx).
		Where(strings.Join(where, " AND "), args...).
		Order(order).
		Limit(limit).
		Find(&rows).Error
	if err != nil {
		return nil, nil, err
	}
	for _, r := range rows {
		id := spec.IdOf(r)
		if _, ok := seen[id]; ok {
			continue
		}
		results = append(results, r)
		seen[id] = struct{}{}
		if len(results) >= limit {
			break
		}
	}
	return results, seen, nil
}

func runFilterQuery[E any](db *gorm.DB, ctx context.Context, tenantId uuid.UUID, limit int, spec QuerySpec[E]) ([]E, map[uint64]struct{}, error) {
	seen := make(map[uint64]struct{})
	results := make([]E, 0, limit)

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
	err := db.WithContext(ctx).
		Where(strings.Join(where, " AND "), args...).
		Order(order).
		Limit(limit).
		Find(&rows).Error
	if err != nil {
		return nil, nil, err
	}
	for _, r := range rows {
		id := spec.IdOf(r)
		if _, ok := seen[id]; ok {
			continue
		}
		results = append(results, r)
		seen[id] = struct{}{}
		if len(results) >= limit {
			break
		}
	}
	return results, seen, nil
}
