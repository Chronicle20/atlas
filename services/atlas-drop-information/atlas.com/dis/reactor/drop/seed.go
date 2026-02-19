package drop

import (
	"os"

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

// DeleteAll deletes all reactor drops for the tenant in context
func DeleteAll(db *gorm.DB) (int64, error) {
	result := db.Unscoped().Where("1 = 1").Delete(&entity{})
	return result.RowsAffected, result.Error
}
