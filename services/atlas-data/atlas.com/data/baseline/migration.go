package baseline

import "gorm.io/gorm"

type tenantBaseline struct {
	TenantID       string `gorm:"primaryKey;type:uuid;column:tenant_id"`
	Region         string `gorm:"not null;column:region"`
	MajorVersion   int    `gorm:"not null;column:major_version"`
	MinorVersion   int    `gorm:"not null;column:minor_version"`
	BaselineSha256 string `gorm:"not null;column:baseline_sha256"`
	RestoredAt     string `gorm:"not null;column:restored_at;default:now()"`
}

func (tenantBaseline) TableName() string { return "tenant_baselines" }

// Migration auto-creates the tenant_baselines table.
func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&tenantBaseline{})
}
