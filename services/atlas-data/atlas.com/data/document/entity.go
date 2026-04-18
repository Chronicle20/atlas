package document

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&Entity{})
}

type Entity struct {
	Id         uuid.UUID       `gorm:"type:uuid;default:uuid_generate_v4()"`
	TenantId   uuid.UUID       `gorm:"not null;uniqueIndex:idx_documents_tenant_type_docid"`
	Type       string          `gorm:"not null;uniqueIndex:idx_documents_tenant_type_docid"`
	DocumentId uint32          `gorm:"not null;uniqueIndex:idx_documents_tenant_type_docid"`
	Content    json.RawMessage `gorm:"type:json;not null"`
	UpdatedAt  time.Time       `gorm:"autoUpdateTime"`
}

func (e Entity) TableName() string {
	return "documents"
}
