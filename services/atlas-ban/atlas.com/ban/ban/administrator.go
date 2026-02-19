package ban

import (
	"net"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type EntityUpdateFunction func() ([]string, func(e *Entity))

func create(db *gorm.DB) func(tenantId uuid.UUID, banType BanType, value string, reason string, reasonCode byte, permanent bool, expiresAt time.Time, issuedBy string) (Model, error) {
	return func(tenantId uuid.UUID, banType BanType, value string, reason string, reasonCode byte, permanent bool, expiresAt time.Time, issuedBy string) (Model, error) {
		a := &Entity{
			TenantId:   tenantId,
			BanType:    byte(banType),
			Value:      value,
			Reason:     reason,
			ReasonCode: reasonCode,
			Permanent:  permanent,
			ExpiresAt:  expiresAt,
			IssuedBy:   issuedBy,
		}

		err := db.Create(a).Error
		if err != nil {
			return Model{}, err
		}

		return Make(*a)
	}
}

func deleteById(db *gorm.DB) func(id uint32) error {
	return func(id uint32) error {
		return db.Where("id = ?", id).Delete(&Entity{}).Error
	}
}

func ipMatchesCIDR(ip string, cidr string) bool {
	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return false
	}
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}
	return network.Contains(parsedIP)
}

func isCIDR(value string) bool {
	_, _, err := net.ParseCIDR(value)
	return err == nil
}

func updateExpiresAt(db *gorm.DB) func(id uint32, expiresAt time.Time) error {
	return func(id uint32, expiresAt time.Time) error {
		return db.Model(&Entity{}).
			Where("id = ?", id).
			Update("expires_at", expiresAt).Error
	}
}

func Make(e Entity) (Model, error) {
	return NewBuilder(e.TenantId, BanType(e.BanType), e.Value).
		SetId(e.ID).
		SetReason(e.Reason).
		SetReasonCode(e.ReasonCode).
		SetPermanent(e.Permanent).
		SetExpiresAt(e.ExpiresAt).
		SetIssuedBy(e.IssuedBy).
		SetCreatedAt(e.CreatedAt).
		SetUpdatedAt(e.UpdatedAt).
		Build()
}
