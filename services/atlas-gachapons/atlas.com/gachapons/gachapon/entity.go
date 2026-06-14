package gachapon

import (
	"database/sql/driver"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Migration(db *gorm.DB) error {
	if err := migrateToSurrogatePK(db); err != nil {
		return err
	}
	return db.AutoMigrate(&entity{})
}

// migrateToSurrogatePK rewrites the legacy gachapons primary key — which was the
// slug column "id" alone — into a surrogate uuid primary key ("uid") guarded by a
// (tenant_id, id) unique index. The slug-only key was not tenant-scoped, and
// gachapon slugs ("henesys", "ellinia", ...) are identical across tenants, so the
// second tenant to seed a slug collided on the primary key and got zero rows. The
// surrogate key lets every tenant own its own copy of a slug.
//
// AutoMigrate cannot perform this transformation (it never drops or recreates a
// primary key), so the structural change is done here with explicit DDL. The
// function is idempotent: it no-ops on a fresh database (table absent — AutoMigrate
// then creates the surrogate-key shape directly) and on an already-migrated
// database (the "uid" column is present). The DDL is Postgres-only and runs solely
// against the live schema; unit tests use a fresh sqlite database and take the
// no-op path.
func migrateToSurrogatePK(db *gorm.DB) error {
	m := db.Migrator()
	if !m.HasTable("gachapons") {
		return nil // fresh database: AutoMigrate builds the surrogate-key shape
	}
	if m.HasColumn(&entity{}, "uid") {
		// The "uid" column is added in the same transaction that repoints the
		// primary key and builds the unique index, so its presence means the
		// whole migration committed. (A manually half-applied state is out of
		// scope; the transaction below is all-or-nothing.)
		return nil
	}
	return db.Transaction(func(tx *gorm.DB) error {
		// uid is derived deterministically from (tenant_id, id), which is unique
		// in the legacy table, so the backfill needs no extension (gen_random_uuid
		// is core only on PG13+) and cannot collide.
		stmts := []string{
			`ALTER TABLE gachapons ADD COLUMN uid uuid`,
			`UPDATE gachapons SET uid = md5(tenant_id::text || id)::uuid WHERE uid IS NULL`,
			`ALTER TABLE gachapons ALTER COLUMN uid SET NOT NULL`,
			// Drop the existing primary key by its real name rather than
			// assuming the GORM/Postgres default "gachapons_pkey"; a legacy
			// deployment with a differently-named PK constraint would otherwise
			// abort the migration (and service startup).
			`DO $$
			DECLARE pk_name text;
			BEGIN
				SELECT conname INTO pk_name
				FROM pg_constraint
				WHERE conrelid = 'gachapons'::regclass AND contype = 'p';
				IF pk_name IS NOT NULL THEN
					EXECUTE format('ALTER TABLE gachapons DROP CONSTRAINT %I', pk_name);
				END IF;
			END $$`,
			`ALTER TABLE gachapons ADD PRIMARY KEY (uid)`,
			`CREATE UNIQUE INDEX IF NOT EXISTS idx_gachapons_tenant_slug ON gachapons (tenant_id, id)`,
		}
		for _, s := range stmts {
			if err := tx.Exec(s).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// int64Array implements database/sql Scanner and driver.Valuer for PostgreSQL integer arrays.
type int64Array []int64

func (a int64Array) Value() (driver.Value, error) {
	if a == nil {
		return nil, nil
	}
	parts := make([]string, len(a))
	for i, v := range a {
		parts[i] = strconv.FormatInt(v, 10)
	}
	return "{" + strings.Join(parts, ",") + "}", nil
}

func (a *int64Array) Scan(src interface{}) error {
	if src == nil {
		*a = nil
		return nil
	}
	var s string
	switch v := src.(type) {
	case string:
		s = v
	case []byte:
		s = string(v)
	default:
		return fmt.Errorf("int64Array.Scan: unsupported type %T", src)
	}
	s = strings.Trim(s, "{}")
	if s == "" {
		*a = int64Array{}
		return nil
	}
	parts := strings.Split(s, ",")
	result := make(int64Array, len(parts))
	for i, p := range parts {
		v, err := strconv.ParseInt(strings.TrimSpace(p), 10, 64)
		if err != nil {
			return fmt.Errorf("int64Array.Scan: parsing %q: %w", p, err)
		}
		result[i] = v
	}
	*a = result
	return nil
}

type entity struct {
	// Uid is a surrogate primary key. The business identity of a gachapon is its
	// slug (ID), but slugs are shared across tenants, so the key that enforces
	// row uniqueness must be tenant-independent. See migrateToSurrogatePK.
	Uid            uuid.UUID  `gorm:"column:uid;type:uuid;primaryKey"`
	TenantId       uuid.UUID  `gorm:"not null;uniqueIndex:idx_gachapons_tenant_slug,priority:1"`
	ID             string     `gorm:"not null;uniqueIndex:idx_gachapons_tenant_slug,priority:2"`
	Name           string     `gorm:"not null"`
	NpcIds         int64Array `gorm:"type:integer[];not null"`
	CommonWeight   uint32     `gorm:"not null"`
	UncommonWeight uint32     `gorm:"not null"`
	RareWeight     uint32     `gorm:"not null"`
}

func (e entity) TableName() string {
	return "gachapons"
}
