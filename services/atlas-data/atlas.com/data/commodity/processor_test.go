package commodity

import (
	"os"
	"strings"
	"testing"
)

// TestRegisterCommodityDoesNotWrapInOuterTx asserts processor.go no longer
// wraps the Etc.wz Register call in a single database.ExecuteTransaction.
// Pre-fix that outer transaction made a single-row failure (or a conn
// blip across a multi-thousand-row register) roll back every committed
// row. F2 (task-076) replaces it with per-row commits provided by the
// document Storage.
func TestRegisterCommodityDoesNotWrapInOuterTx(t *testing.T) {
	src, err := os.ReadFile("processor.go")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(src), "database.ExecuteTransaction") {
		t.Fatal("processor.go still wraps Register in database.ExecuteTransaction; F2 requires chunked per-row commits")
	}
}
