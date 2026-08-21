package writer

import (
	"bytes"
	"testing"

	"github.com/sirupsen/logrus/hooks/test"
)

// contiMoveTestOptions mirrors the shape RegisterTenantWriterOptions/
// TenantWriterOptions produce from a socket template's ContiMove writer
// entry (DOM-25) -- the byte values are the client wire bytes, resolved per
// tenant rather than carried as free-form seed config.
var contiMoveTestOptions = map[string]interface{}{
	"operations": map[string]interface{}{
		"SHOW_STATE":     float64(10),
		"SHOW_SUB_STATE": float64(4),
		"HIDE_STATE":     float64(10),
		"HIDE_SUB_STATE": float64(5),
	},
}

// TestContiMoveBodyResolvesShowFromTenantOptions pins that ContiMoveBody
// resolves the SHOW state/subState pair from the tenant's writer options
// table rather than taking raw bytes as a parameter.
func TestContiMoveBodyResolvesShowFromTenantOptions(t *testing.T) {
	l, _ := test.NewNullLogger()
	actual := ContiMoveBody(ContiMoveShow)(l, reportTestContext(t))(contiMoveTestOptions)
	expected := []byte{0x0A, 0x04}
	if !bytes.Equal(actual, expected) {
		t.Errorf("got %v want %v", actual, expected)
	}
}

// TestContiMoveBodyResolvesHideFromTenantOptions pins the HIDE pair.
func TestContiMoveBodyResolvesHideFromTenantOptions(t *testing.T) {
	l, _ := test.NewNullLogger()
	actual := ContiMoveBody(ContiMoveHide)(l, reportTestContext(t))(contiMoveTestOptions)
	expected := []byte{0x0A, 0x05}
	if !bytes.Equal(actual, expected) {
		t.Errorf("got %v want %v", actual, expected)
	}
}

// TestContiMoveBodyMissingOptionsFallsBackToLoudSentinel asserts a resolve
// miss falls back to ResolveCode's 99 sentinel (logged, will likely crash
// the client) instead of silently sending a guessed value -- the same
// failure mode every other options-resolved writer in this package uses.
// State 99 isn't one of the three arms that reads a subState byte
// (contiMoveHasSubState in libs/atlas-packet/field/clientbound/conti_move.go),
// so only the one sentinel byte is written.
func TestContiMoveBodyMissingOptionsFallsBackToLoudSentinel(t *testing.T) {
	l, _ := test.NewNullLogger()
	actual := ContiMoveBody(ContiMoveShow)(l, reportTestContext(t))(nil)
	expected := []byte{0x63} // 99
	if !bytes.Equal(actual, expected) {
		t.Errorf("got %v want %v", actual, expected)
	}
}
