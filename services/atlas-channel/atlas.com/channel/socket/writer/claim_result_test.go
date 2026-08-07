package writer

import (
	"bytes"
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func reportTestContext(t *testing.T) context.Context {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	return tenant.WithContext(context.Background(), tm)
}

var reportTestOptions = map[string]interface{}{
	"operations": map[string]interface{}{
		"SUCCESS":          "0x02",
		"TRY_AGAIN":        "0x41",
		"RECHECK_NAME":     "0x42",
		"UNABLE_TO_LOCATE": "0x01",
	},
}

func TestClaimResultSuccessBodyResolvesMode(t *testing.T) {
	l, _ := test.NewNullLogger()
	actual := ClaimResultSuccessBody(true, 100)(l, reportTestContext(t))(reportTestOptions)
	expected := []byte{0x02, 0x01, 0x64, 0x00, 0x00, 0x00}
	if !bytes.Equal(actual, expected) {
		t.Errorf("got %v want %v", actual, expected)
	}
}

func TestClaimResultNoticeBodyResolvesMode(t *testing.T) {
	l, _ := test.NewNullLogger()
	actual := ClaimResultNoticeBody(ClaimResultRecheckName)(l, reportTestContext(t))(reportTestOptions)
	expected := []byte{0x42}
	if !bytes.Equal(actual, expected) {
		t.Errorf("got %v want %v", actual, expected)
	}
}

func TestSueCharacterResultBodyResolvesCode(t *testing.T) {
	l, _ := test.NewNullLogger()
	sueOptions := map[string]interface{}{
		"operations": map[string]interface{}{"UNABLE_TO_LOCATE": "0x01"},
	}
	actual := SueCharacterResultBody(SueResultUnableToLocate)(l, reportTestContext(t))(sueOptions)
	expected := []byte{0x01}
	if !bytes.Equal(actual, expected) {
		t.Errorf("got %v want %v", actual, expected)
	}
}

func TestClaimEnableBodies(t *testing.T) {
	l, _ := test.NewNullLogger()
	ctx := reportTestContext(t)
	if got := ClaimAvailableTimeBody(0, 0)(l, ctx)(nil); !bytes.Equal(got, []byte{0x00, 0x00}) {
		t.Errorf("available time: %v", got)
	}
	if got := ClaimSvrStatusChangedBody(true)(l, ctx)(nil); !bytes.Equal(got, []byte{0x01}) {
		t.Errorf("status changed: %v", got)
	}
}
