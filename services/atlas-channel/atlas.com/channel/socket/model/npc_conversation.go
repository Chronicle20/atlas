package model

import (
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
	SpeakerTypeId      byte
	SpeakerTemplateId  uint32
	MsgType            NpcConversationMessageType
	Param              byte
	Message            string
	ConversationDetail Encoder
}

func NewNpcConversation(npcId uint32, msgType NpcConversationMessageType, param byte, conversationDetail Encoder) NpcConversation {
	return NpcConversation{
		SpeakerTypeId:      4,
		SpeakerTemplateId:  npcId,
		MsgType:            msgType,
		Param:              param,
		ConversationDetail: conversationDetail,
	}
}

func (b *NpcConversation) Encode(l logrus.FieldLogger, t tenant.Model, ops map[string]interface{}) func(w *response.Writer) {
	return func(w *response.Writer) {
		w.WriteByte(b.SpeakerTypeId)
		w.WriteInt(b.SpeakerTemplateId)
		w.WriteByte(getNpcConversationMessageType(l)(ops, b.MsgType))
		w.WriteByte(b.Param)
		w.WriteAsciiString(b.Message)
		b.ConversationDetail.Encode(l, t, ops)(w)
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

func (s *SayConversationDetail) Encode(_ logrus.FieldLogger, _ tenant.Model, _ map[string]interface{}) func(w *response.Writer) {
	return func(w *response.Writer) {
		w.WriteAsciiString(s.Message)
		w.WriteBool(s.Next)
		w.WriteBool(s.Previous)
	}
}

type SayImageConversationDetail struct {
	Images []string
}

func (s *SayImageConversationDetail) Encode(_ logrus.FieldLogger, _ tenant.Model, _ map[string]interface{}) func(w *response.Writer) {
	return func(w *response.Writer) {
		w.WriteInt(uint32(len(s.Images)))
		for _, image := range s.Images {
			w.WriteAsciiString(image)
		}
	}
}

type AskYesNoConversationDetail struct {
	Message string
}

func (s *AskYesNoConversationDetail) Encode(_ logrus.FieldLogger, _ tenant.Model, _ map[string]interface{}) func(w *response.Writer) {
	return func(w *response.Writer) {
		w.WriteAsciiString(s.Message)
	}
}

type AskTextConversationDetail struct {
	Message string
	Def     string
	Min     uint16
	Max     uint16
}

func (a *AskTextConversationDetail) Encode(_ logrus.FieldLogger, _ tenant.Model, _ map[string]interface{}) func(w *response.Writer) {
	return func(w *response.Writer) {
		w.WriteAsciiString(a.Message)
		w.WriteAsciiString(a.Def)
		w.WriteShort(a.Min)
		w.WriteShort(a.Max)
	}
}

type AskNumberConversationDetail struct {
	Message string
	Def     uint32
	Min     uint32
	Max     uint32
}

func (s *AskNumberConversationDetail) Encode(_ logrus.FieldLogger, _ tenant.Model, _ map[string]interface{}) func(w *response.Writer) {
	return func(w *response.Writer) {
		w.WriteAsciiString(s.Message)
		w.WriteInt(s.Def)
		w.WriteInt(s.Min)
		w.WriteInt(s.Max)
	}
}

type AskMenuConversationDetail struct {
	Message string
}

func (s *AskMenuConversationDetail) Encode(_ logrus.FieldLogger, _ tenant.Model, _ map[string]interface{}) func(w *response.Writer) {
	return func(w *response.Writer) {
		w.WriteAsciiString(s.Message)
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

func (a *AskQuizConversationDetail) Encode(_ logrus.FieldLogger, _ tenant.Model, _ map[string]interface{}) func(w *response.Writer) {
	return func(w *response.Writer) {
		w.WriteBool(a.Fail)
		if !a.Fail {
			w.WriteAsciiString(a.Title)
			w.WriteAsciiString(a.Problem)
			w.WriteAsciiString(a.Hint)
			w.WriteInt(a.Min)
			w.WriteInt(a.Max)
			w.WriteInt(a.TimeRemaining)
		}
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

func (a *AskSpeedQuizConversationDetail) Encode(_ logrus.FieldLogger, _ tenant.Model, _ map[string]interface{}) func(w *response.Writer) {
	return func(w *response.Writer) {
		w.WriteBool(a.Fail)
		if !a.Fail {
			w.WriteInt(a.Type)
			w.WriteInt(a.Answer)
			w.WriteInt(a.Correct)
			w.WriteInt(a.Remain)
			w.WriteInt(a.TimeRemaining)
		}
	}
}

type AskAvatarConversationDetail struct {
	Message string
	Styles  []uint32
}

func (a *AskAvatarConversationDetail) Encode(_ logrus.FieldLogger, _ tenant.Model, _ map[string]interface{}) func(w *response.Writer) {
	return func(w *response.Writer) {
		w.WriteAsciiString(a.Message)
		w.WriteInt(uint32(len(a.Styles)))
		for _, style := range a.Styles {
			w.WriteInt(style)
		}
	}
}

type AskMemberShopAvatarConversationDetail struct {
	Message    string
	Candidates []uint32
}

func (a *AskMemberShopAvatarConversationDetail) Encode(_ logrus.FieldLogger, _ tenant.Model, _ map[string]interface{}) func(w *response.Writer) {
	return func(w *response.Writer) {
		w.WriteAsciiString(a.Message)
		w.WriteInt(uint32(len(a.Candidates)))
		for _, candidate := range a.Candidates {
			w.WriteInt(candidate)
		}
	}
}

type AskPetConversationDetail struct {
	Message string
	CashId  []uint64
}

func (a *AskPetConversationDetail) Encode(_ logrus.FieldLogger, _ tenant.Model, _ map[string]interface{}) func(w *response.Writer) {
	return func(w *response.Writer) {
		w.WriteAsciiString(a.Message)
		w.WriteByte(byte(len(a.CashId)))
		for _, cashId := range a.CashId {
			w.WriteLong(cashId)
			w.WriteByte(0) // unused
		}
	}
}

type AskPetAllConversationDetail struct {
	Message         string
	ExceptionExists bool
	CashIds         []uint64
}

func (a *AskPetAllConversationDetail) Encode(_ logrus.FieldLogger, _ tenant.Model, _ map[string]interface{}) func(w *response.Writer) {
	return func(w *response.Writer) {
		w.WriteAsciiString(a.Message)
		w.WriteByte(byte(len(a.CashIds)))
		w.WriteBool(a.ExceptionExists)
		for _, cashId := range a.CashIds {
			w.WriteLong(cashId)
			w.WriteByte(0) // unused
		}
	}
}

type AskBoxTextConversationDetail struct {
	Message string
	Def     string
	Col     uint16
	Line    uint16
}

func (a *AskBoxTextConversationDetail) Encode(_ logrus.FieldLogger, _ tenant.Model, _ map[string]interface{}) func(w *response.Writer) {
	return func(w *response.Writer) {
		w.WriteAsciiString(a.Message)
		w.WriteAsciiString(a.Def)
		w.WriteShort(a.Col)
		w.WriteShort(a.Line)
	}
}

type AskSlideMenuConversationDetail struct {
	Unknown  bool
	MenuType uint32
	Message  string
}

func (a *AskSlideMenuConversationDetail) Encode(_ logrus.FieldLogger, _ tenant.Model, _ map[string]interface{}) func(w *response.Writer) {
	return func(w *response.Writer) {
		if a.Unknown {
			w.WriteInt(1)
		} else {
			w.WriteInt(0)
		}
		w.WriteInt(a.MenuType)
		w.WriteAsciiString(a.Message)
	}
}
