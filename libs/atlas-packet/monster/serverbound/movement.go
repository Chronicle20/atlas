package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const MonsterMovementHandle = "MonsterMovementHandle"

// monsterMoveKeyPadTail reports whether the serverbound MOVE_LIFE movement
// blob ends with the keypad history + path bounding rectangle that
// CMovePath::Encode appends after the element loop (see moveKeyPadTail in
// character/serverbound/move.go for the full field derivation:
// count-of-entries byte, ceil(count/2) packed nibble bytes, then left/top/
// right/bottom int16).
//
// Decompile-confirmed for jms v185, NOT live-confirmed: CMob::GenerateMovePath
// @0x6e8892's header — dwMobID, nMobCtrlSN, flag, nAction, ti,
// multiTargetForBall, randTimeForAreaAttack, moveFlags, hackedCode,
// flyCtxTargetX/Y, hackedCodeCRC — matches Atlas field for field up to
// Flush @0x6e9423, confirming the existing header gates independently; only
// the tail (unconditionally written by CMovePath::Encode, see
// character/serverbound/move.go's moveKeyPadTail) was missing. The tail sits
// between the movement blob and the trailing bChasing/hasTarget/bChasing2/
// bChasingHack/tChaseDuration block, which CMob::GenerateMovePath writes
// AFTER calling Flush. See
// docs/tasks/fix-jms185-attack-decode/sibling-movement-ops-findings.md §2.
//
// Gated to JMS because that is the only client this sender was read on. Do
// not extend to GMS without reading each GMS version's CMob::GenerateMovePath
// directly.
func monsterMoveKeyPadTail(t tenant.Model) bool {
	return t.IsRegion("JMS")
}

// packet-audit:fname CMob::GenerateMovePath
type MovementRequest struct {
	uniqueId              uint32
	moveId                int16
	dwFlag                byte
	nActionAndDir         int8
	skillData             uint32
	multiTargetForBall    model.MultiTargetForBall
	randTimeForAreaAttack model.RandTimeForAreaAttack
	moveFlags             byte
	hackedCode            uint32
	flyCtxTargetX         uint32
	flyCtxTargetY         uint32
	hackedCodeCRC         uint32
	movement              model.Movement
	// keyPadStates is the client's per-move keypad history (see
	// monsterMoveKeyPadTail). Nil on versions without the tail.
	keyPadStates []byte
	// moveRect is the bounding rectangle of the whole path, appended after
	// the keypad block.
	moveRectLeft   int16
	moveRectTop    int16
	moveRectRight  int16
	moveRectBottom int16
	bChasing       byte
	hasTarget      byte
	bChasing2      byte
	bChasingHack   byte
	tChaseDuration uint32
}

func (m MovementRequest) UniqueId() uint32   { return m.uniqueId }
func (m MovementRequest) MoveId() int16      { return m.moveId }
func (m MovementRequest) DwFlag() byte       { return m.dwFlag }
func (m MovementRequest) ActionAndDir() int8 { return m.nActionAndDir }
func (m MovementRequest) SkillData() uint32  { return m.skillData }
func (m MovementRequest) SkillId() int16     { return int16(m.skillData & 0xFF) }

func (m MovementRequest) SkillLevel() int16                            { return int16(m.skillData >> 8 & 0xFF) }
func (m MovementRequest) MonsterMoveStartResult() bool                 { return m.dwFlag > 0 }
func (m MovementRequest) MultiTargetForBall() model.MultiTargetForBall { return m.multiTargetForBall }

func (m MovementRequest) RandTimeForAreaAttack() model.RandTimeForAreaAttack {
	return m.randTimeForAreaAttack
}
func (m MovementRequest) MovementData() model.Movement { return m.movement }
func (m MovementRequest) KeyPadStates() []byte         { return m.keyPadStates }

// MoveRect reports the path bounding rectangle the client appends after the
// keypad block: left, top, right, bottom.
func (m MovementRequest) MoveRect() (int16, int16, int16, int16) {
	return m.moveRectLeft, m.moveRectTop, m.moveRectRight, m.moveRectBottom
}

func (m MovementRequest) Operation() string {
	return MonsterMovementHandle
}

func (m MovementRequest) String() string {
	return fmt.Sprintf("uniqueId [%d] moveId [%d] dwFlag [%d] nActionAndDir [%d] skillData [%d] elements [%d]",
		m.uniqueId, m.moveId, m.dwFlag, m.nActionAndDir, m.skillData, len(m.movement.Elements))
}

func (m MovementRequest) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.uniqueId)
		w.WriteInt16(m.moveId)
		w.WriteByte(m.dwFlag)
		w.WriteInt8(m.nActionAndDir)
		w.WriteInt(m.skillData)

		if (t.IsRegion("GMS") && t.MajorAtLeast(84)) || t.Region() == "JMS" { // multiTarget/randTime are v84+ (CONFIRMED: v83 CMob::GenerateMovePath @0x66b6fc has neither; v84 sub_6818C3 inserts both between skillData and moveFlags)
			w.WriteByteArray(m.multiTargetForBall.Encode(l, ctx)(options))
			w.WriteByteArray(m.randTimeForAreaAttack.Encode(l, ctx)(options))
		}

		w.WriteByte(m.moveFlags)
		if (t.IsRegion("GMS") && t.MajorAtLeast(61)) || t.Region() == "JMS" { // hackedCode added between v48 and v61: v48 sub_550383 @0x5508f2 goes moveFlags->CMovePath::Flush with NO hackedCode Encode4; v61 CMob::GenerateMovePath @0x5cada5 inserts it. Legacy (<61) omits.
			w.WriteInt(m.hackedCode)
		}
		if (t.IsRegion("GMS") && t.MajorAtLeast(79)) || t.Region() == "JMS" { // flyCtxTargetX/Y added at v79; legacy GMS (v72 sub_61AA54 @0x61af58 goes hackedCode->Flush with no flyCtx) omits them
			w.WriteInt(m.flyCtxTargetX)
			w.WriteInt(m.flyCtxTargetY)
		}
		if (t.IsRegion("GMS") && t.MajorAtLeast(87)) || t.Region() == "JMS" { // v87+ fields; v84..86 == v83 (off-by-one fix). delta §3.2
			w.WriteInt(m.hackedCodeCRC)
		}

		w.WriteByteArray(m.movement.Encode(l, ctx)(options))

		// Keypad history + path bounding rect — see monsterMoveKeyPadTail.
		// Sits between the movement blob and the bChasing/hasTarget/
		// bChasing2/bChasingHack/tChaseDuration block: CMob::GenerateMovePath
		// calls CMovePath::Flush @0x6e9423 (which writes this tail) BEFORE
		// writing those five fields.
		if monsterMoveKeyPadTail(t) {
			w.WriteByte(byte(len(m.keyPadStates)))
			for i := 0; i < len(m.keyPadStates); i += 2 {
				b := m.keyPadStates[i] & 0x0F
				if i != len(m.keyPadStates)-1 {
					b |= m.keyPadStates[i+1] << 4
				}
				w.WriteByte(b)
			}
			w.WriteShort(uint16(m.moveRectLeft))
			w.WriteShort(uint16(m.moveRectTop))
			w.WriteShort(uint16(m.moveRectRight))
			w.WriteShort(uint16(m.moveRectBottom))
		}

		if (t.IsRegion("GMS") && t.MajorAtLeast(87)) || t.Region() == "JMS" { // v87+ fields; v84..86 == v83 (off-by-one fix). delta §3.2
			w.WriteByte(m.bChasing)
			w.WriteByte(m.hasTarget)
			w.WriteByte(m.bChasing2)
			w.WriteByte(m.bChasingHack)
			w.WriteInt(m.tChaseDuration)
		}
		return w.Bytes()
	}
}

func (m *MovementRequest) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.uniqueId = r.ReadUint32()
		m.moveId = r.ReadInt16()
		m.dwFlag = r.ReadByte()
		m.nActionAndDir = r.ReadInt8()
		m.skillData = r.ReadUint32()

		if (t.IsRegion("GMS") && t.MajorAtLeast(84)) || t.Region() == "JMS" { // multiTarget/randTime are v84+ (mirror of Encode; v83 CMob::GenerateMovePath has neither, v84 sub_6818C3 has both)
			m.multiTargetForBall.Decode(l, ctx)(r, options)
			m.randTimeForAreaAttack.Decode(l, ctx)(r, options)
		}

		m.moveFlags = r.ReadByte()
		if (t.IsRegion("GMS") && t.MajorAtLeast(61)) || t.Region() == "JMS" { // mirror of Encode: hackedCode added between v48 and v61
			m.hackedCode = r.ReadUint32()
		}
		if (t.IsRegion("GMS") && t.MajorAtLeast(79)) || t.Region() == "JMS" { // mirror of Encode: flyCtxTargetX/Y added at v79
			m.flyCtxTargetX = r.ReadUint32()
			m.flyCtxTargetY = r.ReadUint32()
		}
		if (t.IsRegion("GMS") && t.MajorAtLeast(87)) || t.Region() == "JMS" { // v87+ fields; v84..86 == v83 (off-by-one fix). delta §3.2
			m.hackedCodeCRC = r.ReadUint32()
		}

		m.movement.Decode(l, ctx)(r, options)

		// Keypad history + path bounding rect, serverbound only — see
		// monsterMoveKeyPadTail. Sits between the movement blob and the
		// bChasing block; mirror of Encode.
		if monsterMoveKeyPadTail(t) {
			count := int(r.ReadByte())
			states := make([]byte, 0, count)
			var packed byte
			for i := 0; i < count; i++ {
				if i%2 == 0 {
					packed = r.ReadByte()
				} else {
					packed >>= 4
				}
				states = append(states, packed&0x0F)
			}
			m.keyPadStates = states
			m.moveRectLeft = r.ReadInt16()
			m.moveRectTop = r.ReadInt16()
			m.moveRectRight = r.ReadInt16()
			m.moveRectBottom = r.ReadInt16()
		}

		if (t.IsRegion("GMS") && t.MajorAtLeast(87)) || t.Region() == "JMS" { // v87+ fields; v84..86 == v83 (off-by-one fix). delta §3.2
			m.bChasing = r.ReadByte()
			m.hasTarget = r.ReadByte()
			m.bChasing2 = r.ReadByte()
			m.bChasingHack = r.ReadByte()
			m.tChaseDuration = r.ReadUint32()
		}
	}
}
