package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

// Discrete per-mode body codecs for the CField::OnItemUpgrade dispatcher
// (VICIOUS_HAMMER). The forwarder delegates to CUIItemUpgrade::OnPacket
// (v83 sub_82B2C3 via sub_82B2AD; v95 CUIItemUpgrade::ShowResult 0x7bec20),
// which reads Decode1(mode) and branches: 61 = success (closes the dialog,
// "Increased available upgrade by 1"), 62 = failure (closes with a notice
// keyed by the error code), any other byte = the non-terminal open/arm result
// (arms the gauge: m_nReturnResult = mode, m_nResult = token,
// m_nResultState = 1). Mode values are version-stable across v83/v95 but are
// still config-resolved from the tenant "operations" table by the body funcs
// in field/vicious_hammer_body.go — never hard-coded (DISPATCHER_FAMILY.md).
// The op is absent from the jms registry (jms VERSION-ABSENT).

// ViciousHammerWriter is the registry writer name (Operation()) shared by
// every per-mode VICIOUS_HAMMER body codec in this file.
const ViciousHammerWriter = "ViciousHammer"

// ViciousHammerOpen — the non-terminal open/arm result. Body after the mode
// byte (v83 sub_82B2C3 else-branch: Decode4 + Decode4): token (echoed back by
// the client in ITEM_UPGRADE_UPDATE) and hammerCount (the target's current
// hammersApplied; the client renders "N upgrades are left" as 2 - count).
// packet-audit:fname CField::OnItemUpgrade#Open
type ViciousHammerOpen struct {
	mode        byte
	token       uint32
	hammerCount uint32
}

func NewViciousHammerOpen(mode byte, token uint32, hammerCount uint32) ViciousHammerOpen {
	return ViciousHammerOpen{mode: mode, token: token, hammerCount: hammerCount}
}

func (m ViciousHammerOpen) Mode() byte          { return m.mode }
func (m ViciousHammerOpen) Token() uint32       { return m.token }
func (m ViciousHammerOpen) HammerCount() uint32 { return m.hammerCount }
func (m ViciousHammerOpen) Operation() string   { return ViciousHammerWriter }
func (m ViciousHammerOpen) String() string {
	return fmt.Sprintf("vicious hammer open mode [%d] token [%d] hammerCount [%d]", m.mode, m.token, m.hammerCount)
}

func (m ViciousHammerOpen) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)       // dispatcher mode byte (server-chosen, != 61/62)
		w.WriteInt(m.token)       // Decode4 -> m_nResult (round-trip token)
		w.WriteInt(m.hammerCount) // Decode4 -> current hammersApplied
		return w.Bytes()
	}
}

func (m *ViciousHammerOpen) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.token = r.ReadUint32()
		m.hammerCount = r.ReadUint32()
	}
}

// ViciousHammerSuccess — terminal success (mode 61). Body after the mode byte:
// Decode4(flag); 0 = success, non-0 renders "Unknown error %d". The server
// only ever sends 0.
// packet-audit:fname CField::OnItemUpgrade#Success
type ViciousHammerSuccess struct {
	mode byte
	flag uint32
}

func NewViciousHammerSuccess(mode byte, flag uint32) ViciousHammerSuccess {
	return ViciousHammerSuccess{mode: mode, flag: flag}
}

func (m ViciousHammerSuccess) Mode() byte        { return m.mode }
func (m ViciousHammerSuccess) Flag() uint32      { return m.flag }
func (m ViciousHammerSuccess) Operation() string { return ViciousHammerWriter }
func (m ViciousHammerSuccess) String() string {
	return fmt.Sprintf("vicious hammer success mode [%d] flag [%d]", m.mode, m.flag)
}

func (m ViciousHammerSuccess) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode) // dispatcher mode byte (61)
		w.WriteInt(m.flag)  // Decode4; 0 = success
		return w.Bytes()
	}
}

func (m *ViciousHammerSuccess) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.flag = r.ReadUint32()
	}
}

// ViciousHammerFailure — terminal failure (mode 62). Body after the mode byte:
// Decode4(errorCode); client notices: 1 = "The item is not upgradable",
// 2 = "2 upgrade increases have been used already", 3 = "You can't use
// Vicious Hammer on Horntail Necklace", default = "Unknown error %d".
// packet-audit:fname CField::OnItemUpgrade#Failure
type ViciousHammerFailure struct {
	mode      byte
	errorCode uint32
}

func NewViciousHammerFailure(mode byte, errorCode uint32) ViciousHammerFailure {
	return ViciousHammerFailure{mode: mode, errorCode: errorCode}
}

func (m ViciousHammerFailure) Mode() byte        { return m.mode }
func (m ViciousHammerFailure) ErrorCode() uint32 { return m.errorCode }
func (m ViciousHammerFailure) Operation() string { return ViciousHammerWriter }
func (m ViciousHammerFailure) String() string {
	return fmt.Sprintf("vicious hammer failure mode [%d] errorCode [%d]", m.mode, m.errorCode)
}

func (m ViciousHammerFailure) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)     // dispatcher mode byte (62)
		w.WriteInt(m.errorCode) // Decode4 -> notice selector
		return w.Bytes()
	}
}

func (m *ViciousHammerFailure) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.errorCode = r.ReadUint32()
	}
}
