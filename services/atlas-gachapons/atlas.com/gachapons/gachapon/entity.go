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
	return db.AutoMigrate(&entity{})
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
	TenantId       uuid.UUID   `gorm:"not null"`
	ID             string      `gorm:"primaryKey;not null"`
	Name           string      `gorm:"not null"`
	NpcIds         int64Array  `gorm:"type:integer[];not null"`
	CommonWeight   uint32      `gorm:"not null"`
	UncommonWeight uint32      `gorm:"not null"`
	RareWeight     uint32      `gorm:"not null"`
}

func (e entity) TableName() string {
	return "gachapons"
}
