package mock_test

import (
	"atlas-login/account"
	"atlas-login/account/mock"
	"errors"
	"testing"
)

func TestMockProcessor_GetById(t *testing.T) {
	expectedModel := account.NewBuilder().
		SetId(123).
		SetName("testuser").
		Build()

	m := &mock.MockProcessor{
		GetByIdFunc: func(id uint32) (account.Model, error) {
			if id == 123 {
				return expectedModel, nil
			}
			return account.Model{}, errors.New("not found")
		},
	}

	// Test found case
	result, err := m.GetById(123)
	if err != nil {
		t.Errorf("GetById(123) unexpected error: %v", err)
	}
	if result.Id() != 123 {
		t.Errorf("GetById(123) Id = %d, want 123", result.Id())
	}

	// Test not found case
	_, err = m.GetById(999)
	if err == nil {
		t.Error("GetById(999) expected error")
	}
}

func TestMockProcessor_UpdatePin(t *testing.T) {
	var capturedId uint32
	var capturedPin string

	m := &mock.MockProcessor{
		UpdatePinFunc: func(id uint32, pin string) error {
			capturedId = id
			capturedPin = pin
			return nil
		},
	}

	err := m.UpdatePin(123, "5678")
	if err != nil {
		t.Errorf("UpdatePin() unexpected error: %v", err)
	}
	if capturedId != 123 {
		t.Errorf("capturedId = %d, want 123", capturedId)
	}
	if capturedPin != "5678" {
		t.Errorf("capturedPin = %s, want '5678'", capturedPin)
	}
}

func TestMockProcessor_UpdatePic(t *testing.T) {
	var capturedId uint32
	var capturedPic string

	m := &mock.MockProcessor{
		UpdatePicFunc: func(id uint32, pic string) error {
			capturedId = id
			capturedPic = pic
			return nil
		},
	}

	err := m.UpdatePic(123, "newpic")
	if err != nil {
		t.Errorf("UpdatePic() unexpected error: %v", err)
	}
	if capturedId != 123 {
		t.Errorf("capturedId = %d, want 123", capturedId)
	}
	if capturedPic != "newpic" {
		t.Errorf("capturedPic = %s, want 'newpic'", capturedPic)
	}
}

func TestMockProcessor_IsLoggedIn(t *testing.T) {
	loggedInAccounts := map[uint32]bool{
		123: true,
		456: false,
	}

	m := &mock.MockProcessor{
		IsLoggedInFunc: func(id uint32) bool {
			return loggedInAccounts[id]
		},
	}

	if !m.IsLoggedIn(123) {
		t.Error("IsLoggedIn(123) = false, want true")
	}
	if m.IsLoggedIn(456) {
		t.Error("IsLoggedIn(456) = true, want false")
	}
	if m.IsLoggedIn(999) {
		t.Error("IsLoggedIn(999) = true, want false")
	}
}

func TestMockProcessor_DefaultBehavior(t *testing.T) {
	// Mock with no functions set should return defaults
	m := &mock.MockProcessor{}

	result, err := m.GetById(123)
	if err != nil {
		t.Errorf("GetById() unexpected error: %v", err)
	}
	if result.Id() != 0 {
		t.Errorf("Default GetById() Id = %d, want 0", result.Id())
	}

	if m.IsLoggedIn(123) {
		t.Error("Default IsLoggedIn() = true, want false")
	}

	err = m.UpdatePin(123, "1234")
	if err != nil {
		t.Errorf("Default UpdatePin() unexpected error: %v", err)
	}
}
