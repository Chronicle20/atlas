package wallet

import (
	"math"

	"github.com/google/uuid"
)

type Model struct {
	id        uuid.UUID
	accountId uint32
	credit    uint32
	points    uint32
	prepaid   uint32
}

func (m Model) Id() uuid.UUID {
	return m.id
}

func (m Model) AccountId() uint32 {
	return m.accountId
}

func (m Model) Credit() uint32 {
	return m.credit
}

func (m Model) Points() uint32 {
	return m.points
}

func (m Model) Prepaid() uint32 {
	return m.prepaid
}

func (m Model) Balance(currency uint32) uint32 {
	if currency == 1 {
		return m.credit
	} else if currency == 2 {
		return m.points
	} else {
		return m.prepaid
	}
}

func (m Model) Purchase(currency uint32, amount uint32) Model {
	newCredit := m.credit
	newPoints := m.points
	newPrepaid := m.prepaid
	if currency == 1 {
		newCredit -= amount
	} else if currency == 2 {
		newPoints -= amount
	} else {
		newPrepaid -= amount
	}

	return Model{
		id:        m.id,
		accountId: m.accountId,
		credit:    newCredit,
		points:    newPoints,
		prepaid:   newPrepaid,
	}
}

// Award credits amount to the given currency and returns a new Model. The add
// SATURATES at math.MaxUint32: balances are uint32, and wrapping a large
// reward around would silently destroy the account's existing funds rather
// than merely over-crediting. Currency ids follow the Balance convention:
// 1 = credit (NX), 2 = Maple Points, anything else = prepaid.
func (m Model) Award(currency uint32, amount uint32) Model {
	add := func(v uint32) uint32 {
		if v > math.MaxUint32-amount {
			return math.MaxUint32
		}
		return v + amount
	}

	newCredit := m.credit
	newPoints := m.points
	newPrepaid := m.prepaid
	switch currency {
	case 1:
		newCredit = add(newCredit)
	case 2:
		newPoints = add(newPoints)
	default:
		newPrepaid = add(newPrepaid)
	}

	return Model{
		id:        m.id,
		accountId: m.accountId,
		credit:    newCredit,
		points:    newPoints,
		prepaid:   newPrepaid,
	}
}
