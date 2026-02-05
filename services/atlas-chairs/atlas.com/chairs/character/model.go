package character

import (
	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-tenant"
)

type MapKey struct {
	Tenant tenant.Model
	Field  field.Model
}
