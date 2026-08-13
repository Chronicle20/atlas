package configuration

import "time"

// TierRestModel is one meso-tax band on the wire. The JSON keys must match the
// `taxTiers` element attributes atlas-tenants writes in
// services/atlas-tenants/atlas.com/tenants/configuration/rest.go
// (TradeConfigRestModel).
type TierRestModel struct {
	Threshold uint32  `json:"threshold"`
	Rate      float64 `json:"rate"`
}

// RestModel is the JSON:API representation of the trade configuration fetched
// from atlas-tenants. Fields default to the zero value when atlas-tenants has
// not provisioned the resource; Extract folds any zero knob back to its default
// so a partial config never yields a nonsensical zero.
type RestModel struct {
	Id                        string          `json:"-"`
	TaxEnabled                bool            `json:"taxEnabled"`
	TaxTiers                  []TierRestModel `json:"taxTiers"`
	MaxStagedItems            int             `json:"maxStagedItems"`
	MinTradeLevel             int             `json:"minTradeLevel"`
	AttestationTimeoutSeconds int             `json:"attestationTimeoutSeconds"`
}

func (r RestModel) GetName() string {
	return "trade-configs"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(id string) error {
	r.Id = id
	return nil
}

// Extract converts the fetched RestModel into the immutable domain Model,
// substituting the default for any knob left at its zero value.
//
// TaxEnabled is deliberately NOT zero-folded: atlas-tenants always serialises
// the flag, so a `false` on the wire is an operator disabling the tax (FR-9.1)
// rather than an absent field. A tenant with no resource at all never reaches
// Extract — the fetch fails and the registry substitutes DefaultConfig.
//
// MinTradeLevel is likewise not folded: its default is zero, so "absent" and
// "explicitly unrestricted" are the same configuration.
func Extract(r RestModel) Model {
	d := DefaultConfig()

	tiers := make([]Tier, 0, len(r.TaxTiers))
	for _, t := range r.TaxTiers {
		tiers = append(tiers, Tier(t))
	}

	m := d.
		WithTaxEnabled(r.TaxEnabled).
		WithTaxTiers(tiers).
		WithMinTradeLevel(r.MinTradeLevel)

	if r.MaxStagedItems != 0 {
		m = m.WithMaxStagedItems(r.MaxStagedItems)
	}
	if r.AttestationTimeoutSeconds != 0 {
		m = m.WithAttestationTimeout(time.Duration(r.AttestationTimeoutSeconds) * time.Second)
	}
	return m
}
