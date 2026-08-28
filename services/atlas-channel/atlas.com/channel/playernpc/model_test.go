package playernpc

import (
	"testing"

	"github.com/google/uuid"
)

// TestAtPositionRelocatesOnlyThePosition pins the contract the REPOSITIONED
// handler relies on: the event's coordinates replace the read-back model's,
// every other attribute (identity, appearance, dir) survives, and the
// receiver is left untouched.
func TestAtPositionRelocatesOnlyThePosition(t *testing.T) {
	original := NewBuilder(uuid.New(), 7).
		SetName("Statue").
		SetScriptId(9901000).
		SetObjectId(101000).
		SetPosition(100, 200, 1, 50, 150, 1).
		MustBuild()

	moved := original.AtPosition(10, 20, 3, 0, 99)

	if moved.X() != 10 || moved.Cy() != 20 || moved.Fh() != 3 || moved.RX0() != 0 || moved.RX1() != 99 {
		t.Fatalf("moved position = (%d,%d,%d,%d,%d), want (10,20,3,0,99)", moved.X(), moved.Cy(), moved.Fh(), moved.RX0(), moved.RX1())
	}
	if moved.Dir() != original.Dir() {
		t.Fatalf("moved dir = %d, want %d (no reposition carries dir)", moved.Dir(), original.Dir())
	}
	if moved.Id() != original.Id() || moved.ObjectId() != original.ObjectId() || moved.ScriptId() != original.ScriptId() || moved.Name() != original.Name() {
		t.Fatal("AtPosition must not alter identity")
	}
	if original.X() != 100 || original.Cy() != 200 || original.Fh() != 1 || original.RX0() != 50 || original.RX1() != 150 {
		t.Fatal("AtPosition must not mutate its receiver")
	}
}
