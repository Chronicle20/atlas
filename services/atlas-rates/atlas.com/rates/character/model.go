package character

import (
	"atlas-rates/rate"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-tenant"
)

// Model holds all rate factors for a character
type Model struct {
	tenant      tenant.Model
	worldId     world.Id
	channelId   channel.Id
	characterId uint32
	factors     []rate.Factor
}

func (m Model) Tenant() tenant.Model {
	return m.tenant
}

func (m Model) WorldId() world.Id {
	return m.worldId
}

func (m Model) ChannelId() channel.Id {
	return m.channelId
}

func (m Model) CharacterId() uint32 {
	return m.characterId
}

func (m Model) Factors() []rate.Factor {
	// Return defensive copy
	result := make([]rate.Factor, len(m.factors))
	copy(result, m.factors)
	return result
}

// ComputedRates calculates the final rates from all factors
func (m Model) ComputedRates() rate.Computed {
	return rate.ComputeFromFactors(m.factors)
}

// NewModel creates a new character rate model
func NewModel(t tenant.Model, ch channel.Model, characterId uint32) Model {
	return Model{
		tenant:      t,
		worldId:     ch.WorldId(),
		channelId:   ch.Id(),
		characterId: characterId,
		factors:     make([]rate.Factor, 0),
	}
}

// WithFactor returns a new model with the factor added/updated
func (m Model) WithFactor(f rate.Factor) Model {
	// Remove existing factor with same source and type, then add new one
	newFactors := make([]rate.Factor, 0, len(m.factors)+1)
	for _, existing := range m.factors {
		if existing.Source() != f.Source() || existing.RateType() != f.RateType() {
			newFactors = append(newFactors, existing)
		}
	}
	newFactors = append(newFactors, f)

	return Model{
		tenant:      m.tenant,
		worldId:     m.worldId,
		channelId:   m.channelId,
		characterId: m.characterId,
		factors:     newFactors,
	}
}

// WithoutFactor returns a new model with the factor removed
func (m Model) WithoutFactor(source string, rateType rate.Type) Model {
	newFactors := make([]rate.Factor, 0, len(m.factors))
	for _, existing := range m.factors {
		if existing.Source() != source || existing.RateType() != rateType {
			newFactors = append(newFactors, existing)
		}
	}

	return Model{
		tenant:      m.tenant,
		worldId:     m.worldId,
		channelId:   m.channelId,
		characterId: m.characterId,
		factors:     newFactors,
	}
}

// WithoutFactorsBySource removes all factors from a specific source
func (m Model) WithoutFactorsBySource(source string) Model {
	newFactors := make([]rate.Factor, 0, len(m.factors))
	for _, existing := range m.factors {
		if existing.Source() != source {
			newFactors = append(newFactors, existing)
		}
	}

	return Model{
		tenant:      m.tenant,
		worldId:     m.worldId,
		channelId:   m.channelId,
		characterId: m.characterId,
		factors:     newFactors,
	}
}
