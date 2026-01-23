package drop

import (
	"os"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const defaultReactorDropsPath = "/drops/reactors"

// GetReactorDropsPath returns the path to the reactor drops directory,
// using the REACTOR_DROPS_PATH environment variable if set, otherwise
// falling back to the default path.
func GetReactorDropsPath() string {
	if path := os.Getenv("REACTOR_DROPS_PATH"); path != "" {
		return path
	}
	return defaultReactorDropsPath
}

// DeleteAllForTenant deletes all reactor drops for a specific tenant
func DeleteAllForTenant(db *gorm.DB, tenantId uuid.UUID) (int64, error) {
	result := db.Unscoped().Where("tenant_id = ?", tenantId).Delete(&entity{})
	return result.RowsAffected, result.Error
}
