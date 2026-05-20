package tenantpurge

import (
	"context"
	"errors"
	"fmt"

	"atlas-data/canonical"
	minio "atlas-data/storage/minio"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// ErrCanonicalRefused is returned when the request targets the canonical tenant.
var ErrCanonicalRefused = errors.New("canonical tenant cannot be purged")

// PurgeTables is the per-tenant table set wiped in Postgres.
var PurgeTables = []string{
	"documents",
	"monster_search_index",
	"npc_search_index",
	"reactor_search_index",
	"map_search_index",
	"item_string_search_index",
	"tenant_baselines",
}

// Purge deletes all Postgres rows and best-effort MinIO objects for tenantID.
func Purge(ctx context.Context, l logrus.FieldLogger, db *gorm.DB, mc *minio.Client, tenantID uuid.UUID) error {
	if tenantID.String() == canonical.TenantUUID {
		return ErrCanonicalRefused
	}
	if err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, table := range PurgeTables {
			if err := tx.Exec("DELETE FROM "+table+" WHERE tenant_id = ?", tenantID.String()).Error; err != nil {
				return fmt.Errorf("purge %s: %w", table, err)
			}
		}
		return nil
	}); err != nil {
		return err
	}
	if mc != nil {
		prefix := fmt.Sprintf("tenants/%s/", tenantID.String())
		cfg := mc.Cfg()
		for _, b := range []string{cfg.BucketWZ, cfg.BucketAssets, cfg.BucketRenders} {
			if err := mc.RemovePrefix(ctx, b, prefix); err != nil {
				l.WithError(err).Warnf("partial purge: %s/%s", b, prefix)
			}
		}
	}
	return nil
}
