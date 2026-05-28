package script

import (
	"fmt"
	"regexp"
	"time"

	seeder "github.com/Chronicle20/atlas/libs/atlas-seeder"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// compile-time assertion
var _ seeder.Subdomain[jsonReactorScript, ReactorScript] = ReactorSubdomain{}

// ReactorSubdomain implements seeder.Subdomain for reactor action scripts.
type ReactorSubdomain struct{}

func (ReactorSubdomain) Name() string { return "reactor-actions" }
func (ReactorSubdomain) Path() string { return "reactor-actions/reactors" }
func (ReactorSubdomain) Type() string { return "reactor-action" }
func (ReactorSubdomain) EntityIDPattern() *regexp.Regexp {
	return regexp.MustCompile(`^reactor-(.+)\.json$`)
}

func (ReactorSubdomain) DeleteAllForTenant(db *gorm.DB) (int64, error) {
	result := db.Unscoped().Where("1 = 1").Delete(&Entity{})
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

func (ReactorSubdomain) Decode(payload []byte) (jsonReactorScript, error) {
	var attrs jsonReactorScript
	if err := seeder.DecodeAttributes(payload, &attrs); err != nil {
		return jsonReactorScript{}, fmt.Errorf("reactor-actions: decode attributes: %w", err)
	}
	return attrs, nil
}

func (ReactorSubdomain) Build(t tenant.Model, entityID string, attrs jsonReactorScript) ([]ReactorScript, error) {
	_ = t // tenant tracked via GORM context; not embedded in the domain model

	// If the JSON carries a ReactorId use it; otherwise fall back to the entity ID from the filename
	reactorId := attrs.ReactorId
	if reactorId == "" {
		reactorId = entityID
	}

	builder := NewReactorScriptBuilder().
		SetReactorId(reactorId).
		SetDescription(attrs.Description)

	for _, jr := range attrs.HitRules {
		rule, err := convertJsonRule(jr)
		if err != nil {
			return nil, fmt.Errorf("reactor-actions: convert hit rule %q: %w", jr.Id, err)
		}
		builder.AddHitRule(rule)
	}

	for _, jr := range attrs.ActRules {
		rule, err := convertJsonRule(jr)
		if err != nil {
			return nil, fmt.Errorf("reactor-actions: convert act rule %q: %w", jr.Id, err)
		}
		builder.AddActRule(rule)
	}

	return []ReactorScript{builder.Build()}, nil
}

func (ReactorSubdomain) BulkCreate(db *gorm.DB, models []ReactorScript) error {
	if len(models) == 0 {
		return nil
	}

	tenantId := extractReactorTenantId(db)
	entities := make([]Entity, 0, len(models))
	for _, m := range models {
		e, err := ToEntity(m, tenantId)
		if err != nil {
			return err
		}
		e.ID = uuid.New()
		entities = append(entities, e)
	}
	return db.Create(&entities).Error
}

func (ReactorSubdomain) Count(db *gorm.DB) (int64, *time.Time, error) {
	var count int64
	if err := db.Model(&Entity{}).Count(&count).Error; err != nil {
		return 0, nil, err
	}
	return count, nil, nil
}

// extractReactorTenantId retrieves the tenant ID embedded in the GORM context.
func extractReactorTenantId(db *gorm.DB) uuid.UUID {
	if db.Statement != nil && db.Statement.Context != nil {
		t := tenant.MustFromContext(db.Statement.Context)
		return t.Id()
	}
	return uuid.Nil
}
