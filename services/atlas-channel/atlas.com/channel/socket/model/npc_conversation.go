package model

import (
	"context"

	"github.com/Chronicle20/atlas-socket/packet"
	"github.com/Chronicle20/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

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
	SpeakerTypeId          byte
	SpeakerTemplateId      uint32
	SecondaryNpcTemplateId uint32
	MsgType                NpcConversationMessageType
	Param                  byte
	ConversationDetail     packet.Encoder
}

func NewNpcConversation(npcId uint32, msgType NpcConversationMessageType, speakerByte byte, secondaryNpcId uint32, conversationDetail packet.Encoder) NpcConversation {
	return NpcConversation{
		SpeakerTypeId:          speakerByte,
		SpeakerTemplateId:      npcId,
		SecondaryNpcTemplateId: secondaryNpcId,
		MsgType:                msgType,
		Param:                  speakerByte,
		ConversationDetail:     conversationDetail,
	}
}

func (b *NpcConversation) Encoder(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(b.SpeakerTypeId)
		w.WriteInt(b.SpeakerTemplateId)
		w.WriteByte(getNpcConversationMessageType(l)(options, b.MsgType))
		w.WriteByte(b.Param)
		if b.Param&4 != 0 {
			w.WriteInt(b.SecondaryNpcTemplateId)
		}
		w.WriteByteArray(b.ConversationDetail.Encode(l, ctx)(options))
		return w.Bytes()
	}
}

func getNpcConversationMessageType(l logrus.FieldLogger) func(options map[string]interface{}, key NpcConversationMessageType) byte {
	return func(options map[string]interface{}, key NpcConversationMessageType) byte {
		var genericCodes interface{}
		var ok bool
		if genericCodes, ok = options["messageType"]; !ok {
			l.Errorf("Code [%s] not configured for use. Defaulting to 99 which will likely cause a client crash.", key)
			return 99
		}

		var codes map[string]interface{}
		if codes, ok = genericCodes.(map[string]interface{}); !ok {
			l.Errorf("Code [%s] not configured for use. Defaulting to 99 which will likely cause a client crash.", key)
			return 99
		}

		op, ok := codes[string(key)].(float64)
		if !ok {
			l.Errorf("Code [%s] not configured for use. Defaulting to 99 which will likely cause a client crash.", key)
			return 99
		}
		return byte(op)
	}
}

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
