package teleportrock

import (
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

// Target models the CWvsContext::RunMapTransferItem request payload shared by
// USE_TELEPORT_ROCK and the cash-item-use teleport-rock branch (design §1 Q1):
//
//	byte bByName
//	  1 -> string targetName (length-prefixed ASCII)
//	  0 -> int dwTargetField (only encoded when a map was actually selected)
//
// The client omits the payload entirely when the dialog resolves with neither
// a name nor a valid map.
//
// Whether a trailing 4-byte updateTime follows the payload depends on the
// WRAPPING op, not on Target itself (task-124 v95 verify pass, live
// GMS_v95.0_U_DEVM.exe port 13341):
//   - teleportrock/serverbound.Use (CWvsContext::SendMapTransferItemUseRequest,
//     v83 0xa0a3bb / v95 0x9e6020) always Encode4(update_time) AFTER
//     RunMapTransferItem succeeds, on every version — genuinely trailing.
//   - cash/serverbound.ItemUseTeleportRock (the RunMapTransferItem call inside
//     CWvsContext::SendConsumeCashItemUseRequest, case 22 @ v95 0x9ee059) has NO
//     such trailing write for MajorVersion()>=87: v95's decompile shows
//     Encode4(update_time) happens ONCE, in the common header prologue
//     (@0x9eb4b7, BEFORE Encode2(nPOS)/Encode4(nItemID) and the case switch),
//     already consumed by the parent ItemUse struct
//     (services/atlas-channel .../character_cash_item_use.go's updateTimeFirst
//     gate). Reserving 4 phantom trailing bytes here for that context would
//     misdecode a genuine 5-byte by-map payload (byName=0 + 4-byte mapId,
//     nothing following) as "no selection" — the caller must say whether a
//     trailing updateTime budget applies via hasTrailingUpdateTime.
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

// Decode reads the target payload. hasTrailingUpdateTime tells Decode whether
// the WRAPPING op still has a 4-byte updateTime to come after this payload —
// true for teleportrock/serverbound.Use (every version) and for
// cash/serverbound.ItemUseTeleportRock on MajorVersion()<87 (v83/v84, trailing
// tail write); false for ItemUseTeleportRock on MajorVersion()>=87 (v87/v95/
// jms — updateTime already consumed by the parent ItemUse header, nothing
// follows the target payload on the wire).
func (t *Target) Decode(_ logrus.FieldLogger, hasTrailingUpdateTime bool) func(r *request.Reader) {
	reserve := 0
	if hasTrailingUpdateTime {
		reserve = trailingUpdateTimeBytes
	}
	return func(r *request.Reader) {
		t.valid = false
		if r.Available() <= reserve {
			return // payload omitted entirely
		}
		t.byName = r.ReadBool()
		if t.byName {
			if r.Available() < 2+reserve {
				return // not even a string length prefix before the reserved budget
			}
			t.targetName = r.ReadAsciiString()
			t.valid = len(t.targetName) > 0
			return
		}
		if r.Available() < 4+reserve {
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
