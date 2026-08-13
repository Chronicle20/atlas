package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const ClaimResultWriter = "ClaimResult"

// ClaimResult (CWvsContext::OnClaimResult) is a mode byte followed by a
// mode-dependent payload where ONLY mode 2 (success) carries data; every
// other verified mode (3, 0x41-0x45, 0x47, 0x48) is the bare mode byte
// rendered as a CUtilDlg::Notice modal. Mode sets identical v72..v95.
// This is the SetSkillResponse-style "mode + conditional payload" shape,
// not a dispatcher family (design.md §3.1).

// ClaimResultSuccess - mode, byte hasRemaining, int32 remaining.
// packet-audit:fname CWvsContext::OnClaimResult#Success
type ClaimResultSuccess struct {
	mode         byte
	hasRemaining bool
	remaining    int32
}

func NewClaimResultSuccess(mode byte, hasRemaining bool, remaining int32) ClaimResultSuccess {
	return ClaimResultSuccess{mode: mode, hasRemaining: hasRemaining, remaining: remaining}
}

func (m ClaimResultSuccess) Mode() byte         { return m.mode }
func (m ClaimResultSuccess) HasRemaining() bool { return m.hasRemaining }
func (m ClaimResultSuccess) Remaining() int32   { return m.remaining }

func (m ClaimResultSuccess) Operation() string { return ClaimResultWriter }

func (m ClaimResultSuccess) String() string {
	return fmt.Sprintf("mode [%d] hasRemaining [%t] remaining [%d]", m.mode, m.hasRemaining, m.remaining)
}

func (m ClaimResultSuccess) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteBool(m.hasRemaining)
		w.WriteInt32(m.remaining)
		return w.Bytes()
	}
}

func (m *ClaimResultSuccess) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.hasRemaining = r.ReadBool()
		m.remaining = r.ReadInt32()
	}
}

// ClaimResultNotice - bare mode byte (modes 3, 0x41-0x45, 0x47, 0x48).
// packet-audit:fname CWvsContext::OnClaimResult#Notice
type ClaimResultNotice struct {
	mode byte
}

func NewClaimResultNotice(mode byte) ClaimResultNotice {
	return ClaimResultNotice{mode: mode}
}

func (m ClaimResultNotice) Mode() byte { return m.mode }

func (m ClaimResultNotice) Operation() string { return ClaimResultWriter }

func (m ClaimResultNotice) String() string {
	return fmt.Sprintf("mode [%d]", m.mode)
}

func (m ClaimResultNotice) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		return w.Bytes()
	}
}

func (m *ClaimResultNotice) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}
