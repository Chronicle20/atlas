package character

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type MapKey struct {
	Tenant tenant.Model
	Field  field.Model
}
