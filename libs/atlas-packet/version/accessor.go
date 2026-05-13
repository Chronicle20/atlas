package version

import tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"

// variantAccessor is the seam between version.VariantOf and tenant.Model.
// Task 14 will replace this with a direct method call once tenant.Model
// exposes ClientVariant(). For now, use a structural type assertion so this
// task can land before Task 14.
//
// NOTE: tenant.Model methods are all pointer-receiver, so we must pass &t
// for the interface assertion to succeed once ClientVariant() is added.
func variantAccessor(t tenant.Model) (string, bool) {
	type variantAware interface{ ClientVariant() string }
	if va, ok := any(&t).(variantAware); ok {
		return va.ClientVariant(), true
	}
	return "", false
}
