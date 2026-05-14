package serverbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const CharacterExpressionHandle = "CharacterExpressionHandle"

// ExpressionRequest - CWvsContext::SendEmotionChange
//
// Wire layout — version-gated (IDA v83@0xa24470, v87@0xabbfbb, v95@0x9f9320;
// IDA JMS v185 CWvsContext::SendEmotionChange@0xb0b8be):
//
//	Encode4  emote          — emotion/expression ID
//	Encode4  duration       — display duration in ms [GMS>87 only]
//	Encode1  byItemOption   — 1 = triggered by item option [GMS>87 only]
//
// IDA v83 CWvsContext::SendEmotionChange@0xa24470: encodes only Encode4(emotionId).
// IDA v87 CWvsContext::SendEmotionChange@0xabbfbb: encodes only Encode4(emotionId) — same as v83.
// IDA v95 CWvsContext::SendEmotionChange@0x9f9320: encodes Encode4 + Encode4 + Encode1 (duration+byItemOption added).
// IDA JMS v185 CWvsContext::SendEmotionChange@0xb0b8be: encodes only Encode4(charId) — fundamentally
// different wire: JMS sends the sender's characterId rather than emotionId+duration+byItemOption.
// The JMS serverbound opcode 0x2B carries only a charId; atlas-server reads emotionId from this
// slot. JMS ExpressionRequest decodes emote field only (the charId is reinterpreted as emotionId
// server-side). No duration or byItemOption for JMS.
type ExpressionRequest struct {
	emote        uint32
	duration     int32
	byItemOption bool
}

func (m ExpressionRequest) Emote() uint32       { return m.emote }
func (m ExpressionRequest) Duration() int32     { return m.duration }
func (m ExpressionRequest) ByItemOption() bool  { return m.byItemOption }

func (m ExpressionRequest) Operation() string {
	return CharacterExpressionHandle
}

func (m ExpressionRequest) String() string {
	return fmt.Sprintf("emote [%d], duration [%d], byItemOption [%v]", m.emote, m.duration, m.byItemOption)
}

func (m ExpressionRequest) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.emote)
		// duration and byItemOption added after GMS v87 (first seen in v95).
		// IDA v83 and v87 CWvsContext::SendEmotionChange encode only Encode4(emotionId).
		// IDA JMS v185 SendEmotionChange@0xb0b8be encodes only Encode4(charId) — no duration/byItemOption.
		if t.Region() == "GMS" && t.MajorVersion() > 87 {
			w.WriteInt32(m.duration)
			w.WriteBool(m.byItemOption)
		}
		return w.Bytes()
	}
}

func (m *ExpressionRequest) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.emote = r.ReadUint32()
		// duration and byItemOption added after GMS v87 (first seen in v95).
		// IDA v83 and v87 CWvsContext::SendEmotionChange encode only Encode4(emotionId).
		// IDA JMS v185 SendEmotionChange@0xb0b8be encodes only Encode4(charId) — no duration/byItemOption.
		if t.Region() == "GMS" && t.MajorVersion() > 87 {
			m.duration = r.ReadInt32()
			m.byItemOption = r.ReadBool()
		}
	}
}
