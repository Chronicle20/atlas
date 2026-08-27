package configuration

import "time"

// RestModel is the JSON:API representation of the imprint configuration
// fetched from atlas-tenants. PendingExpiryHours defaults to the Go zero
// value (0) when atlas-tenants has not provisioned the resource or a stored
// row leaves it unset; Extract folds a zero (or negative) value back to the
// default so a partial/absent config never yields an instant-expiry pending
// change.
type RestModel struct {
	Id                 string `json:"-"`
	PendingExpiryHours int    `json:"pendingExpiryHours"`
}

func (r RestModel) GetName() string {
	return "imprint-configs"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(id string) error {
	r.Id = id
	return nil
}

// Extract converts the fetched RestModel into the immutable domain Model,
// substituting DefaultPendingExpiry for a zero (or negative) PendingExpiryHours.
// A tenant with no imprint-configs resource at all never reaches Extract — the
// fetch fails and the registry substitutes DefaultConfig directly.
func Extract(r RestModel) Model {
	return DefaultConfig().WithPendingExpiry(time.Duration(r.PendingExpiryHours) * time.Hour)
}

// Transform is the inverse of Extract: it converts the domain Model back into
// the RestModel representation, expressing PendingExpiry in whole hours.
func Transform(m Model) RestModel {
	return RestModel{
		PendingExpiryHours: int(m.pendingExpiry.Hours()),
	}
}
