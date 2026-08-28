package backeffect

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type FieldKey struct {
	Tenant tenant.Model
	Field  field.Model
}

type BackEffectEntry struct {
	Effect  byte
	FieldId uint32
	PageId  byte
	// Duration is the client's fade length in milliseconds, as sent in the
	// SET_BACK_EFFECT packet. It is deliberately not an expiry: there is no
	// reaper for back-effect entries, unlike the jukebox registry's
	// ExpiresAt. A CLEAR_BACK_EFFECT command is what removes an entry.
	Duration uint32
}
