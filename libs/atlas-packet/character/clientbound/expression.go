package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const CharacterExpressionWriter = "CharacterExpression"

// CharacterExpression represents the EMOTION packet sent to remote clients.
//
// Wire layout — version-gated (IDA v83 CUserPool::OnUserRemotePacket case 0xC1;
// IDA v87 CUserPool::OnUserRemotePacket case 0xCE; IDA v95 CUser::OnEmotion@0x8e0150):
//
//	Decode4  characterId   — read by CUserPool::OnUserRemotePacket dispatcher
//	Decode4  expression    — emotion/expression ID
//	Decode4  duration      — display duration in ms [GMS>87 || JMS only]
//	Decode1  byItemOption  — item-option emotion flag [GMS>87 || JMS only]
//
// IDA v83: case 0xC1 in CUserPool::OnUserRemotePacket reads only Decode4(expressionId)
// inline — no separate OnEmotion function, no duration, no byItemOption.
// IDA v87: case 0xCE in CUserPool::OnUserRemotePacket@0x9f7492 reads only Decode4(emotionId)
// inline and calls CAvatar::SetEmotion — same as v83, no duration, no byItemOption.
// IDA v95: CUser::OnEmotion@0x8e0150 reads Decode4 + Decode4 + Decode1 (duration+byItemOption added).
type CharacterExpression struct {
	characterId  uint32
	expression   uint32
	duration     uint32
	byItemOption bool
}

func NewCharacterExpression(characterId uint32, expression uint32, duration uint32) CharacterExpression {
	return CharacterExpression{characterId: characterId, expression: expression, duration: duration}
}

func (m CharacterExpression) CharacterId() uint32 { return m.characterId }
func (m CharacterExpression) Expression() uint32  { return m.expression }
func (m CharacterExpression) Duration() uint32    { return m.duration }
func (m CharacterExpression) ByItemOption() bool  { return m.byItemOption }
func (m CharacterExpression) Operation() string   { return CharacterExpressionWriter }
func (m CharacterExpression) String() string {
	return fmt.Sprintf("characterId [%d], expression [%d], duration [%d]", m.characterId, m.expression, m.duration)
}

func (m CharacterExpression) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.characterId)
		w.WriteInt(m.expression)
		// duration and byItemOption added after GMS v87 (first seen in v95).
		// IDA v83 and v87 CUserPool::OnUserRemotePacket read only Decode4(expressionId) inline.
		if (t.Region() == "GMS" && t.MajorVersion() > 87) || t.Region() == "JMS" {
			w.WriteInt(m.duration)
			w.WriteBool(m.byItemOption)
		}
		return w.Bytes()
	}
}

func (m *CharacterExpression) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.characterId = r.ReadUint32()
		m.expression = r.ReadUint32()
		// duration and byItemOption added after GMS v87 (first seen in v95).
		// IDA v83 and v87 CUserPool::OnUserRemotePacket read only Decode4(expressionId) inline.
		if (t.Region() == "GMS" && t.MajorVersion() > 87) || t.Region() == "JMS" {
			m.duration = r.ReadUint32()
			m.byItemOption = r.ReadBool()
		}
	}
}
