package ban

import (
	"net"
	"time"

	tenant "github.com/Chronicle20/atlas-tenant"
	"gorm.io/gorm"
)

type EntityUpdateFunction func() ([]string, func(e *Entity))

func create(db *gorm.DB) func(tenant tenant.Model, banType BanType, value string, reason string, reasonCode byte, permanent bool, expiresAt time.Time, issuedBy string) (Model, error) {
	return func(tenant tenant.Model, banType BanType, value string, reason string, reasonCode byte, permanent bool, expiresAt time.Time, issuedBy string) (Model, error) {
		a := &Entity{
			TenantId:   tenant.Id(),
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

func deleteById(db *gorm.DB) func(tenant tenant.Model, id uint32) error {
	return func(tenant tenant.Model, id uint32) error {
		return db.Where(&Entity{TenantId: tenant.Id(), ID: id}).Delete(&Entity{}).Error
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
