package version

import (
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type Region string

const (
	GMS Region = "GMS"
	JMS Region = "JMS"
)

type ClientVariant string

const (
	Modified ClientVariant = "modified"
	Stock    ClientVariant = "stock"
)

func RegionOf(t tenant.Model) Region { return Region(t.Region()) }

func AtLeast(t tenant.Model, n uint16) bool  { return t.MajorVersion() >= n }
func LessThan(t tenant.Model, n uint16) bool { return t.MajorVersion() < n }

func Between(t tenant.Model, lo, hi uint16) bool {
	mv := t.MajorVersion()
	return mv >= lo && mv <= hi
}

// VariantOf reads the tenant clientVariant. Returns Modified when the
// underlying model predates the flag (back-compat).
func VariantOf(t tenant.Model) ClientVariant {
	if cv, ok := variantAccessor(t); ok && cv != "" {
		return ClientVariant(cv)
	}
	return Modified
}

func IsStock(t tenant.Model) bool { return VariantOf(t) == Stock }
