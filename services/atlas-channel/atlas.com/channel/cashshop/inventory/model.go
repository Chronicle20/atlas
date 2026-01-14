package inventory

import (
	"atlas-channel/cashshop/inventory/compartment"
)

// Model represents a cash shop inventory with multiple compartments
type Model struct {
	accountId    uint32
	compartments map[compartment.CompartmentType]compartment.Model
}

// AccountId returns the account ID associated with this inventory
func (m Model) AccountId() uint32 {
	return m.accountId
}

// Compartments returns all compartments in this inventory
func (m Model) Compartments() []compartment.Model {
	res := make([]compartment.Model, 0)
	for _, v := range m.compartments {
		res = append(res, v)
	}
	return res
}

// CompartmentByType returns a specific compartment by its type
func (m Model) CompartmentByType(ct compartment.CompartmentType) compartment.Model {
	return m.compartments[ct]
}

// Explorer returns the Explorer compartment
func (m Model) Explorer() compartment.Model {
	return m.compartments[compartment.TypeExplorer]
}

// Cygnus returns the Cygnus compartment
func (m Model) Cygnus() compartment.Model {
	return m.compartments[compartment.TypeCygnus]
}

// Legend returns the Legend compartment
func (m Model) Legend() compartment.Model {
	return m.compartments[compartment.TypeLegend]
}
