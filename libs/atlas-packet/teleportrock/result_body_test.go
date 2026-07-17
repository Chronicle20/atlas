package teleportrock

import (
	"bytes"
	"context"
	"testing"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	testlog "github.com/sirupsen/logrus/hooks/test"
)

func testOperations() map[string]interface{} {
	return map[string]interface{}{
		"operations": map[string]interface{}{
			MapTransferModeDeleteList:        "0x02",
			MapTransferModeRegisterList:      "0x03",
			MapTransferModeCannotGo:          "0x05",
			MapTransferModeUnableToLocate:    "0x06",
			MapTransferModeUnableToLocate2:   "0x07",
			MapTransferModeCannotGoContinent: "0x08",
			MapTransferModeCurrentMap:        "0x09",
			MapTransferModeMapNotAvailable:   "0x0A",
			MapTransferModeMapleIslandLevel7: "0x0B",
		},
	}
}

// The mode byte is config-resolved via WithResolvedCode("operations", key) —
// never hard-coded (known crash class when the table is missing).
func TestMapTransferResultListBodyResolvesMode(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	got := MapTransferResultListBody(MapTransferModeRegisterList, false, []_map.Id{100000000})(l, context.Background())(testOperations())
	if got[0] != 0x03 || got[1] != 0x00 {
		t.Fatalf("header: % x", got[:2])
	}
	if len(got) != 2+5*4 {
		t.Fatalf("regular list body must be 22 bytes, got %d", len(got))
	}
}

func TestMapTransferResultErrorBodyResolvesMode(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	got := MapTransferResultErrorBody(MapTransferModeCannotGoContinent, true)(l, context.Background())(testOperations())
	want := []byte{0x08, 0x01}
	if !bytes.Equal(got, want) {
		t.Fatalf("got % x want % x", got, want)
	}
}
