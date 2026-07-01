package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
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

// packet-audit:fname CScriptMan::OnScriptMessage
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

func (m NpcConversation) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.speakerTypeId)
		w.WriteInt(m.speakerTemplateId)
		w.WriteByte(m.msgType)
		// GMS v72 CScriptMan::OnScriptMessage@0x6a0ba9 (GMS_v72.1_U_DEVM.exe, port
		// 13339) reads only Decode1 speakerTypeId, Decode4 speakerTemplateId,
		// Decode1 msgType before dispatching to the per-msgType body — there is NO
		// param byte and NO param&4 secondaryNpcTemplateId at the frame level (the
		// per-handler `param` arg is passed uninitialised, never read from the
		// wire). The param byte + optional secondary int were introduced after the
		// legacy range: v79 OnScriptMessage@0x6c7d3e reads Decode1 param + Decode4
		// secondary when param&4. Legacy GMS (<79) omits both. delta §3.2
		t := tenant.MustFromContext(ctx)
		if !(t.IsRegion("GMS") && !t.MajorAtLeast(79)) {
			w.WriteByte(m.param)
			if m.param&4 != 0 {
				w.WriteInt(m.secondaryNpcTemplateId)
			}
		}
		w.WriteByteArray(m.conversationDetail)
		return w.Bytes()
	}
}

func (m *NpcConversation) Decode(l logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		t := tenant.MustFromContext(ctx)
		legacyNoParam := t.IsRegion("GMS") && !t.MajorAtLeast(79)
		m.speakerTypeId = r.ReadByte()
		m.speakerTemplateId = r.ReadUint32()
		m.msgType = r.ReadByte()
		if !legacyNoParam {
			m.param = r.ReadByte()
			if m.param&4 != 0 {
				m.secondaryNpcTemplateId = r.ReadUint32()
			}
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

func (s *SayImageConversationDetail) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		// GMS v79 CScriptMan::OnSayImage@0x6c8052 reads a SINGLE DecodeStr (one
		// image path) with NO count prefix; the count-prefixed multi-image list
		// was introduced after v79 (v83 @0x961275, v95 @0x6dc310 read Decode1
		// count + count x DecodeStr). For the legacy GMS range emit just the
		// first image string (no count). delta §3.2
		if t, err := tenant.FromContext(ctx)(); err == nil && t.IsRegion("GMS") && !t.MajorAtLeast(83) {
			if len(s.Images) > 0 {
				w.WriteAsciiString(s.Images[0])
			} else {
				w.WriteAsciiString("")
			}
			return w.Bytes()
		}
		w.WriteByte(byte(len(s.Images)))
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

func (s *AskMenuConversationDetail) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteAsciiString(s.Message)
		// GMS v79 merged the avatar-style menu into ASK_MENU: the client
		// (CScriptMan::OnAskMenu @0x6c8863, GMS_v79_1_DEVM.exe port 13340) reads
		// DecodeStr(message) + Decode1(count) + Decode4 x count (avatar look ids,
		// SetUtilDlgEx_AVATAR). v83+ OnAskMenu (v83 @0x746fad, v95 @0x6dce00) read
		// a plain single string with NO count. Atlas uses ASK_MENU only for plain
		// #L#-token text menus, so count is always 0 (no avatar styles). delta §3.2
		if t, err := tenant.FromContext(ctx)(); err == nil && t.IsRegion("GMS") && !t.MajorAtLeast(83) {
			w.WriteByte(0)
		}
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

func (a *AskMemberShopAvatarConversationDetail) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteAsciiString(a.Message)
		// GMS v79 member-shop-avatar candidates are client-inventory-driven: the
		// client (CScriptMan::OnAskMembershopAvatar @0x6c8bc8, GMS_v79_1_DEVM.exe
		// port 13340) reads DecodeStr(message) + Decode1(count) + count x
		// (DecodeBuffer(8)=cash item SN int64 + Decode1 byte). That per-entry
		// format is incompatible with the v83+ int32 style-id list, and Atlas has
		// no server-side SN data to drive it, so count is always 0 for the legacy
		// range (msgType 9). v83+ (CScriptMan::OnAskMembershopAvatar#AskMemberShopAvatar)
		// read Decode4 style ids — unchanged. delta §3.2
		t := tenant.MustFromContext(ctx)
		if t.Region() == "GMS" && t.MajorVersion() < 83 {
			w.WriteByte(0)
		}
		if (t.Region() == "GMS" && t.MajorVersion() >= 83) || t.Region() == "JMS" {
			w.WriteByte(byte(len(a.Candidates)))
			for _, candidate := range a.Candidates {
				w.WriteInt(candidate)
			}
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
		// The leading slideDlgType int is present for GMS v87+ and for
		// JMS185 (CScriptMan::OnAskSlideMenu -> sub_7E2A97@0x7e2a97 reads two
		// leading Decode4s unconditionally). GMS v83..86 omit it (single Decode4);
		// v84..86 == v83 (off-by-one fix). delta §3.2
		if (t.IsRegion("GMS") && t.MajorAtLeast(87)) || t.Region() == "JMS" {
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
