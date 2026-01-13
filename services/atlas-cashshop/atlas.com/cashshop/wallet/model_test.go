package wallet

import (
	"github.com/google/uuid"
	"testing"
)

func TestModelAccessors(t *testing.T) {
	id := uuid.New()
	accountId := uint32(12345)
	credit := uint32(1000)
	points := uint32(500)
	prepaid := uint32(250)

	m := Model{
		id:        id,
		accountId: accountId,
		credit:    credit,
		points:    points,
		prepaid:   prepaid,
	}

	if m.Id() != id {
		t.Errorf("Id mismatch: expected %v, got %v", id, m.Id())
	}
	if m.AccountId() != accountId {
		t.Errorf("AccountId mismatch: expected %d, got %d", accountId, m.AccountId())
	}
	if m.Credit() != credit {
		t.Errorf("Credit mismatch: expected %d, got %d", credit, m.Credit())
	}
	if m.Points() != points {
		t.Errorf("Points mismatch: expected %d, got %d", points, m.Points())
	}
	if m.Prepaid() != prepaid {
		t.Errorf("Prepaid mismatch: expected %d, got %d", prepaid, m.Prepaid())
	}
}

func TestBalanceReturnsCorrectCurrency(t *testing.T) {
	m := Model{
		credit:  1000,
		points:  500,
		prepaid: 250,
	}

	// Currency 1 = credit
	if m.Balance(1) != 1000 {
		t.Errorf("Balance(1) should return credit, got %d", m.Balance(1))
	}

	// Currency 2 = points
	if m.Balance(2) != 500 {
		t.Errorf("Balance(2) should return points, got %d", m.Balance(2))
	}

	// Currency 3 = prepaid (default)
	if m.Balance(3) != 250 {
		t.Errorf("Balance(3) should return prepaid, got %d", m.Balance(3))
	}

	// Unknown currency defaults to prepaid
	if m.Balance(99) != 250 {
		t.Errorf("Balance(99) should return prepaid (default), got %d", m.Balance(99))
	}
}

func TestPurchaseDeductsCredit(t *testing.T) {
	original := Model{
		id:        uuid.New(),
		accountId: 12345,
		credit:    1000,
		points:    500,
		prepaid:   250,
	}

	// Purchase using credit (currency 1)
	result := original.Purchase(1, 100)

	// Original should be unchanged (immutability)
	if original.Credit() != 1000 {
		t.Error("Original model should be unchanged after Purchase")
	}

	// Result should have deducted credit
	if result.Credit() != 900 {
		t.Errorf("Credit should be 900 after purchase, got %d", result.Credit())
	}

	// Other currencies unchanged
	if result.Points() != 500 {
		t.Errorf("Points should be unchanged, got %d", result.Points())
	}
	if result.Prepaid() != 250 {
		t.Errorf("Prepaid should be unchanged, got %d", result.Prepaid())
	}
}

func TestPurchaseDeductsPoints(t *testing.T) {
	original := Model{
		id:        uuid.New(),
		accountId: 12345,
		credit:    1000,
		points:    500,
		prepaid:   250,
	}

	// Purchase using points (currency 2)
	result := original.Purchase(2, 100)

	// Original should be unchanged
	if original.Points() != 500 {
		t.Error("Original model should be unchanged after Purchase")
	}

	// Result should have deducted points
	if result.Points() != 400 {
		t.Errorf("Points should be 400 after purchase, got %d", result.Points())
	}

	// Other currencies unchanged
	if result.Credit() != 1000 {
		t.Errorf("Credit should be unchanged, got %d", result.Credit())
	}
	if result.Prepaid() != 250 {
		t.Errorf("Prepaid should be unchanged, got %d", result.Prepaid())
	}
}

func TestPurchaseDeductsPrepaid(t *testing.T) {
	original := Model{
		id:        uuid.New(),
		accountId: 12345,
		credit:    1000,
		points:    500,
		prepaid:   250,
	}

	// Purchase using prepaid (currency 3)
	result := original.Purchase(3, 50)

	// Original should be unchanged
	if original.Prepaid() != 250 {
		t.Error("Original model should be unchanged after Purchase")
	}

	// Result should have deducted prepaid
	if result.Prepaid() != 200 {
		t.Errorf("Prepaid should be 200 after purchase, got %d", result.Prepaid())
	}

	// Other currencies unchanged
	if result.Credit() != 1000 {
		t.Errorf("Credit should be unchanged, got %d", result.Credit())
	}
	if result.Points() != 500 {
		t.Errorf("Points should be unchanged, got %d", result.Points())
	}
}

func TestPurchasePreservesIdAndAccountId(t *testing.T) {
	id := uuid.New()
	accountId := uint32(12345)
	original := Model{
		id:        id,
		accountId: accountId,
		credit:    1000,
		points:    500,
		prepaid:   250,
	}

	result := original.Purchase(1, 100)

	if result.Id() != id {
		t.Errorf("Id should be preserved after purchase: expected %v, got %v", id, result.Id())
	}
	if result.AccountId() != accountId {
		t.Errorf("AccountId should be preserved after purchase: expected %d, got %d", accountId, result.AccountId())
	}
}

func TestPurchaseUnknownCurrencyDeductsPrepaid(t *testing.T) {
	original := Model{
		credit:  1000,
		points:  500,
		prepaid: 250,
	}

	// Unknown currency (99) should default to prepaid
	result := original.Purchase(99, 50)

	if result.Prepaid() != 200 {
		t.Errorf("Unknown currency should deduct from prepaid: expected 200, got %d", result.Prepaid())
	}
}
