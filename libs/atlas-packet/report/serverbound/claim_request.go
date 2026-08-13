package serverbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const ClaimRequestHandle = "ClaimRequest"

// ClaimRequest - CWvsContext::SendClaimRequest. Sent by the CUIClaim report
// window. Body verified on v95, on v72 @0x91f2b4 / v79 @0x9711ff
// (2026-08-04), and on v83 @0xa2719c (named + byte-verified 2026-08-05,
// task-145 Task 23 — COutPacket::COutPacket(&pkt, 0x6A) at 0xa27631, size
// 0x6bf, identical encode order to v95). See packet-findings.md §2.
//
//	byte   bChatClaim   1 = chat/harassment claim, 0 = regular claim
//	string sTargetCharacterName
//	byte   nType        reason type from CUIClaim::GetResult
//	string sContext     free-text description
//	if bChatClaim == 1: string chatLog (client-supplied two-character log)
//
// No version branch: body identical across v72..v95 per findings. The packet
// does not exist below v72 (verified absent on v48/v61). If the v83
// byte-verification ever disagrees, add an inline t.MajorVersion() guard
// here at that point.
// packet-audit:fname CWvsContext::SendClaimRequest
type ClaimRequest struct {
	chatClaim   byte
	targetName  string
	reasonType  byte
	description string
	chatLog     string
}

func NewClaimRequest(chatClaim byte, targetName string, reasonType byte, description string, chatLog string) ClaimRequest {
	return ClaimRequest{chatClaim: chatClaim, targetName: targetName, reasonType: reasonType, description: description, chatLog: chatLog}
}

func (m ClaimRequest) IsChatClaim() bool   { return m.chatClaim == 1 }
func (m ClaimRequest) TargetName() string  { return m.targetName }
func (m ClaimRequest) ReasonType() byte    { return m.reasonType }
func (m ClaimRequest) Description() string { return m.description }
func (m ClaimRequest) ChatLog() string     { return m.chatLog }

func (m ClaimRequest) Operation() string { return ClaimRequestHandle }

func (m ClaimRequest) String() string {
	return fmt.Sprintf("chatClaim [%d], target [%s], type [%d], description [%s]", m.chatClaim, m.targetName, m.reasonType, m.description)
}

func (m ClaimRequest) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.chatClaim)
		w.WriteAsciiString(m.targetName)
		w.WriteByte(m.reasonType)
		w.WriteAsciiString(m.description)
		if m.chatClaim == 1 {
			w.WriteAsciiString(m.chatLog)
		}
		return w.Bytes()
	}
}

func (m *ClaimRequest) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.chatClaim = r.ReadByte()
		m.targetName = r.ReadAsciiString()
		m.reasonType = r.ReadByte()
		m.description = r.ReadAsciiString()
		if m.chatClaim == 1 {
			m.chatLog = r.ReadAsciiString()
		}
	}
}
