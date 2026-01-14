package account_test

import (
	"atlas-login/account"
	"testing"
)

func TestBuilder_Build(t *testing.T) {
	m := account.NewBuilder().
		SetId(123).
		SetName("testuser").
		SetPassword("testpass").
		SetPin("1234").
		SetPic("pic123").
		SetLoggedIn(1).
		SetLastLogin(1234567890).
		SetGender(1).
		SetBanned(false).
		SetTos(true).
		SetLanguage("en").
		SetCountry("US").
		SetCharacterSlots(6).
		Build()

	if m.Id() != 123 {
		t.Errorf("Id() = %d, want 123", m.Id())
	}
	if m.Name() != "testuser" {
		t.Errorf("Name() = %s, want 'testuser'", m.Name())
	}
	if m.PIN() != "1234" {
		t.Errorf("PIN() = %s, want '1234'", m.PIN())
	}
	if m.PIC() != "pic123" {
		t.Errorf("PIC() = %s, want 'pic123'", m.PIC())
	}
	if m.LoggedIn() != 1 {
		t.Errorf("LoggedIn() = %d, want 1", m.LoggedIn())
	}
	if m.Gender() != 1 {
		t.Errorf("Gender() = %d, want 1", m.Gender())
	}
	if m.CharacterSlots() != 6 {
		t.Errorf("CharacterSlots() = %d, want 6", m.CharacterSlots())
	}
}

func TestModel_ToBuilder(t *testing.T) {
	original := account.NewBuilder().
		SetId(123).
		SetName("testuser").
		SetPin("1234").
		SetPic("pic123").
		SetGender(1).
		SetCharacterSlots(6).
		Build()

	// Clone and modify pin
	cloned := original.ToBuilder().
		SetPin("5678").
		Build()

	// Original should be unchanged
	if original.PIN() != "1234" {
		t.Errorf("Original PIN() = %s, want '1234'", original.PIN())
	}

	// Cloned should have new pin
	if cloned.PIN() != "5678" {
		t.Errorf("Cloned PIN() = %s, want '5678'", cloned.PIN())
	}

	// Other fields should be preserved
	if cloned.Id() != 123 {
		t.Errorf("Cloned Id() = %d, want 123", cloned.Id())
	}
	if cloned.Name() != "testuser" {
		t.Errorf("Cloned Name() = %s, want 'testuser'", cloned.Name())
	}
	if cloned.PIC() != "pic123" {
		t.Errorf("Cloned PIC() = %s, want 'pic123'", cloned.PIC())
	}
	if cloned.Gender() != 1 {
		t.Errorf("Cloned Gender() = %d, want 1", cloned.Gender())
	}
	if cloned.CharacterSlots() != 6 {
		t.Errorf("Cloned CharacterSlots() = %d, want 6", cloned.CharacterSlots())
	}
}

func TestModel_ToBuilder_ImmutableUpdate(t *testing.T) {
	original := account.NewBuilder().
		SetId(100).
		SetName("user1").
		SetPin("0000").
		SetPic("abc").
		SetGender(0).
		SetTos(false).
		Build()

	// Simulate UpdatePin pattern
	updated := original.ToBuilder().SetPin("9999").Build()

	// Verify original is unchanged
	if original.PIN() != "0000" {
		t.Errorf("Original should remain unchanged, PIN() = %s, want '0000'", original.PIN())
	}

	// Verify updated has new value
	if updated.PIN() != "9999" {
		t.Errorf("Updated PIN() = %s, want '9999'", updated.PIN())
	}

	// Simulate UpdatePic pattern
	updated2 := original.ToBuilder().SetPic("xyz").Build()
	if updated2.PIC() != "xyz" {
		t.Errorf("Updated PIC() = %s, want 'xyz'", updated2.PIC())
	}

	// Simulate UpdateTos pattern
	updated3 := original.ToBuilder().SetTos(true).Build()
	// Note: Tos() getter doesn't exist in original model, this tests the builder works

	// Simulate UpdateGender pattern
	updated4 := original.ToBuilder().SetGender(1).Build()
	if updated4.Gender() != 1 {
		t.Errorf("Updated Gender() = %d, want 1", updated4.Gender())
	}

	// Verify original is still unchanged after all updates
	if original.PIN() != "0000" || original.PIC() != "abc" || original.Gender() != 0 {
		t.Error("Original model was modified by updates")
	}
	_ = updated3 // Use the variable to avoid unused error
}

func TestNewBuilder_DefaultValues(t *testing.T) {
	m := account.NewBuilder().Build()

	if m.Id() != 0 {
		t.Errorf("Default Id() = %d, want 0", m.Id())
	}
	if m.Name() != "" {
		t.Errorf("Default Name() = %s, want ''", m.Name())
	}
	if m.PIN() != "" {
		t.Errorf("Default PIN() = %s, want ''", m.PIN())
	}
	if m.PIC() != "" {
		t.Errorf("Default PIC() = %s, want ''", m.PIC())
	}
	if m.LoggedIn() != 0 {
		t.Errorf("Default LoggedIn() = %d, want 0", m.LoggedIn())
	}
	if m.Gender() != 0 {
		t.Errorf("Default Gender() = %d, want 0", m.Gender())
	}
	if m.CharacterSlots() != 0 {
		t.Errorf("Default CharacterSlots() = %d, want 0", m.CharacterSlots())
	}
}
