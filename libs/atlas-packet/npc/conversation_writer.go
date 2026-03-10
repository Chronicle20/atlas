package npc

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

const NpcConversationWriter = "NPCConversation"

type NpcConversationMessageType string

const (
	NpcConversationMessageTypeSay                 NpcConversationMessageType = "SAY"
	NpcConversationMessageTypeSayImage            NpcConversationMessageType = "SAY_IMAGE"
	NpcConversationMessageTypeAskYesNo            NpcConversationMessageType = "ASK_YES_NO"
	NpcConversationMessageTypeAskText             NpcConversationMessageType = "ASK_TEXT"
	NpcConversationMessageTypeAskNumber           NpcConversationMessageType = "ASK_NUMBER"
	NpcConversationMessageTypeAskMenu             NpcConversationMessageType = "ASK_MENU"
	NpcConversationMessageTypeAskQuiz             NpcConversationMessageType = "ASK_QUIZ"
	NpcConversationMessageTypeAskSpeedQuiz        NpcConversationMessageType = "ASK_SPEED_QUIZ"
	NpcConversationMessageTypeAskAvatar           NpcConversationMessageType = "ASK_AVATAR"
	NpcConversationMessageTypeAskMemberShopAvatar NpcConversationMessageType = "ASK_MEMBER_SHOP_AVATAR"
	NpcConversationMessageTypeAskPet              NpcConversationMessageType = "ASK_PET"
	NpcConversationMessageTypeAskPetAll           NpcConversationMessageType = "ASK_PET_ALL"
	NpcConversationMessageTypeAskYesNoQuest       NpcConversationMessageType = "ASK_YES_NO_QUEST"
	NpcConversationMessageTypeAskBoxText          NpcConversationMessageType = "ASK_BOX_TEXT"
	NpcConversationMessageTypeAskSlideMenu        NpcConversationMessageType = "ASK_SLIDE_MENU"
)

type NpcConversation struct {
	speakerTypeId          byte
	speakerTemplateId      uint32
	secondaryNpcTemplateId uint32
	msgType                byte
	param                  byte
	conversationDetail     []byte
}

func NewNpcConversation(speakerTypeId byte, speakerTemplateId uint32, msgType byte, param byte, secondaryNpcTemplateId uint32, conversationDetail []byte) NpcConversation {
	return NpcConversation{
		speakerTypeId:          speakerTypeId,
		speakerTemplateId:      speakerTemplateId,
		secondaryNpcTemplateId: secondaryNpcTemplateId,
		msgType:                msgType,
		param:                  param,
		conversationDetail:     conversationDetail,
	}
}

func (m NpcConversation) SpeakerTypeId() byte                { return m.speakerTypeId }
func (m NpcConversation) SpeakerTemplateId() uint32          { return m.speakerTemplateId }
func (m NpcConversation) SecondaryNpcTemplateId() uint32     { return m.secondaryNpcTemplateId }
func (m NpcConversation) MsgType() byte                      { return m.msgType }
func (m NpcConversation) Param() byte                        { return m.param }
func (m NpcConversation) ConversationDetail() []byte         { return m.conversationDetail }
func (m NpcConversation) Operation() string                  { return NpcConversationWriter }
func (m NpcConversation) String() string {
	return fmt.Sprintf("speakerTypeId [%d], speakerTemplateId [%d], msgType [%d], param [%d]", m.speakerTypeId, m.speakerTemplateId, m.msgType, m.param)
}

func (m NpcConversation) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.speakerTypeId)
		w.WriteInt(m.speakerTemplateId)
		w.WriteByte(m.msgType)
		w.WriteByte(m.param)
		if m.param&4 != 0 {
			w.WriteInt(m.secondaryNpcTemplateId)
		}
		w.WriteByteArray(m.conversationDetail)
		return w.Bytes()
	}
}

func (m *NpcConversation) Decode(l logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.speakerTypeId = r.ReadByte()
		m.speakerTemplateId = r.ReadUint32()
		m.msgType = r.ReadByte()
		m.param = r.ReadByte()
		if m.param&4 != 0 {
			m.secondaryNpcTemplateId = r.ReadUint32()
		}
		m.conversationDetail = r.ReadBytes(int(r.Available()))
	}
}

// SayConversationDetail encodes a Say conversation message.
type SayConversationDetail struct {
	Message  string
	Next     bool
	Previous bool
}

func (s *SayConversationDetail) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteAsciiString(s.Message)
		w.WriteBool(s.Previous)
		w.WriteBool(s.Next)
		return w.Bytes()
	}
}

// SayImageConversationDetail encodes a Say Image conversation message.
type SayImageConversationDetail struct {
	Images []string
}

func (s *SayImageConversationDetail) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(uint32(len(s.Images)))
		for _, image := range s.Images {
			w.WriteAsciiString(image)
		}
		return w.Bytes()
	}
}

// AskYesNoConversationDetail encodes an Ask Yes/No conversation message.
type AskYesNoConversationDetail struct {
	Message string
}

func (s *AskYesNoConversationDetail) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteAsciiString(s.Message)
		return w.Bytes()
	}
}

// AskTextConversationDetail encodes an Ask Text conversation message.
type AskTextConversationDetail struct {
	Message string
	Def     string
	Min     uint16
	Max     uint16
}

func (a *AskTextConversationDetail) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteAsciiString(a.Message)
		w.WriteAsciiString(a.Def)
		w.WriteShort(a.Min)
		w.WriteShort(a.Max)
		return w.Bytes()
	}
}

// AskNumberConversationDetail encodes an Ask Number conversation message.
type AskNumberConversationDetail struct {
	Message string
	Def     uint32
	Min     uint32
	Max     uint32
}

func (s *AskNumberConversationDetail) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteAsciiString(s.Message)
		w.WriteInt(s.Def)
		w.WriteInt(s.Min)
		w.WriteInt(s.Max)
		return w.Bytes()
	}
}

// AskMenuConversationDetail encodes an Ask Menu conversation message.
type AskMenuConversationDetail struct {
	Message string
}

func (s *AskMenuConversationDetail) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteAsciiString(s.Message)
		return w.Bytes()
	}
}

// AskQuizConversationDetail encodes an Ask Quiz conversation message.
type AskQuizConversationDetail struct {
	Fail          bool
	Title         string
	Problem       string
	Hint          string
	Min           uint32
	Max           uint32
	TimeRemaining uint32
}

func (a *AskQuizConversationDetail) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteBool(a.Fail)
		if !a.Fail {
			w.WriteAsciiString(a.Title)
			w.WriteAsciiString(a.Problem)
			w.WriteAsciiString(a.Hint)
			w.WriteInt(a.Min)
			w.WriteInt(a.Max)
			w.WriteInt(a.TimeRemaining)
		}
		return w.Bytes()
	}
}

// AskSpeedQuizConversationDetail encodes an Ask Speed Quiz conversation message.
type AskSpeedQuizConversationDetail struct {
	Fail          bool
	Type          uint32
	Answer        uint32
	Correct       uint32
	Remain        uint32
	TimeRemaining uint32
}

func (a *AskSpeedQuizConversationDetail) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteBool(a.Fail)
		if !a.Fail {
			w.WriteInt(a.Type)
			w.WriteInt(a.Answer)
			w.WriteInt(a.Correct)
			w.WriteInt(a.Remain)
			w.WriteInt(a.TimeRemaining)
		}
		return w.Bytes()
	}
}

// AskAvatarConversationDetail encodes an Ask Avatar conversation message.
type AskAvatarConversationDetail struct {
	Message string
	Styles  []uint32
}

func (a *AskAvatarConversationDetail) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteAsciiString(a.Message)
		w.WriteByte(byte(len(a.Styles)))
		for _, style := range a.Styles {
			w.WriteInt(style)
		}
		return w.Bytes()
	}
}

// AskMemberShopAvatarConversationDetail encodes an Ask Member Shop Avatar conversation message.
type AskMemberShopAvatarConversationDetail struct {
	Message    string
	Candidates []uint32
}

func (a *AskMemberShopAvatarConversationDetail) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteAsciiString(a.Message)
		w.WriteInt(uint32(len(a.Candidates)))
		for _, candidate := range a.Candidates {
			w.WriteInt(candidate)
		}
		return w.Bytes()
	}
}

// AskPetConversationDetail encodes an Ask Pet conversation message.
type AskPetConversationDetail struct {
	Message string
	CashId  []uint64
}

func (a *AskPetConversationDetail) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteAsciiString(a.Message)
		w.WriteByte(byte(len(a.CashId)))
		for _, cashId := range a.CashId {
			w.WriteLong(cashId)
			w.WriteByte(0) // unused
		}
		return w.Bytes()
	}
}

// AskPetAllConversationDetail encodes an Ask Pet All conversation message.
type AskPetAllConversationDetail struct {
	Message         string
	ExceptionExists bool
	CashIds         []uint64
}

func (a *AskPetAllConversationDetail) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteAsciiString(a.Message)
		w.WriteByte(byte(len(a.CashIds)))
		w.WriteBool(a.ExceptionExists)
		for _, cashId := range a.CashIds {
			w.WriteLong(cashId)
			w.WriteByte(0) // unused
		}
		return w.Bytes()
	}
}

// AskBoxTextConversationDetail encodes an Ask Box Text conversation message.
type AskBoxTextConversationDetail struct {
	Message string
	Def     string
	Col     uint16
	Line    uint16
}

func (a *AskBoxTextConversationDetail) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteAsciiString(a.Message)
		w.WriteAsciiString(a.Def)
		w.WriteShort(a.Col)
		w.WriteShort(a.Line)
		return w.Bytes()
	}
}

// AskSlideMenuConversationDetail encodes an Ask Slide Menu conversation message.
type AskSlideMenuConversationDetail struct {
	Unknown  bool
	MenuType uint32
	Message  string
}

func (a *AskSlideMenuConversationDetail) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		if t.Region() == "GMS" && t.MajorVersion() > 83 {
			if a.Unknown {
				w.WriteInt(1)
			} else {
				w.WriteInt(0)
			}
		}
		w.WriteInt(a.MenuType)
		w.WriteAsciiString(a.Message)
		return w.Bytes()
	}
}
