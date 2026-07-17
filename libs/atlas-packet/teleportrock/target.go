package teleportrock

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

// Target models the CWvsContext::RunMapTransferItem request payload shared by
// USE_TELEPORT_ROCK and the cash-item-use teleport-rock branch (design §1 Q1):
//
//	byte bByName
//	  1 -> string targetName (length-prefixed ASCII)
//	  0 -> int dwTargetField (only encoded when a map was actually selected)
//
// The client omits the payload entirely when the dialog resolves with neither
// a name nor a valid map. A trailing 4-byte updateTime always follows the
// payload in both wrapping ops, so Decode budgets those 4 bytes and marks the
// target invalid (never panics) when the remainder is too short.
type Target struct {
	byName     bool
	targetName string
	targetMap  uint32
	valid      bool
}

func NewTargetByMap(mapId uint32) Target {
	return Target{byName: false, targetMap: mapId, valid: true}
}

func NewTargetByName(name string) Target {
	return Target{byName: true, targetName: name, valid: true}
}

func (t Target) ByName() bool       { return t.byName }
func (t Target) TargetName() string { return t.targetName }
func (t Target) TargetMap() uint32  { return t.targetMap }
func (t Target) Valid() bool        { return t.valid }

func (t Target) String() string {
	if t.byName {
		return fmt.Sprintf("Target{byName=true name=%s valid=%v}", t.targetName, t.valid)
	}
	return fmt.Sprintf("Target{byName=false map=%d valid=%v}", t.targetMap, t.valid)
}

// trailingUpdateTimeBytes is the 4-byte updateTime both wrapping ops append
// after the target payload.
const trailingUpdateTimeBytes = 4

func (t *Target) Decode(_ logrus.FieldLogger) func(r *request.Reader) {
	return func(r *request.Reader) {
		t.valid = false
		if r.Available() <= trailingUpdateTimeBytes {
			return // payload omitted entirely
		}
		t.byName = r.ReadBool()
		if t.byName {
			if r.Available() < 2+trailingUpdateTimeBytes {
				return // not even a string length prefix before the updateTime
			}
			t.targetName = r.ReadAsciiString()
			t.valid = len(t.targetName) > 0
			return
		}
		if r.Available() < 4+trailingUpdateTimeBytes {
			return // byName=0 but no map id was encoded (no selection)
		}
		t.targetMap = r.ReadUint32()
		t.valid = true
	}
}

func (t Target) Encode(w *response.Writer) {
	w.WriteBool(t.byName)
	if t.byName {
		w.WriteAsciiString(t.targetName)
		return
	}
	w.WriteInt(t.targetMap)
}
