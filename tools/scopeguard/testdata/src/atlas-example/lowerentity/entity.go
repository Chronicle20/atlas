package lowerentity

// The lowercase-`entity` convention — mirrors atlas-monster-book/card,
// atlas-keys/key, etc. Has no TenantId: fails.
type entity struct { // want `data-plane entity without TenantId`
	Id   uint32
	Name string
}
