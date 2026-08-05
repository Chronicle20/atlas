package report

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}

// Entity is the durable report row. Id is a surrogate uuid generated in Go
// at create time (never a business-value PK; works across Postgres and the
// sqlite test driver). ServerTranscript is a marshaled []TranscriptLine
// snapshot taken at creation; nil when atlas-messages was unreachable.
type Entity struct {
	Id               uuid.UUID `gorm:"primaryKey;type:uuid"`
	TenantId         uuid.UUID `gorm:"not null;index:idx_reports_tenant_status,priority:1"`
	Kind             string    `gorm:"not null"`
	ReporterId       uint32    `gorm:"not null"`
	ReporterName     string    `gorm:"not null"`
	AccusedId        uint32    `gorm:"not null"`
	AccusedName      string    `gorm:"not null"`
	ReasonType       byte      `gorm:"not null"`
	Description      string    `gorm:"type:text;not null"`
	ChatLog          *string   `gorm:"type:text"`
	ServerTranscript []byte    `gorm:"type:jsonb"`
	Status           string    `gorm:"not null;default:open;index:idx_reports_tenant_status,priority:2"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (e Entity) TableName() string {
	return "reports"
}
