package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// v48 CUIGuildBBS::OnRegister @0x608d55 (sub_608D55): COutPacket(109=BBS_OPERATION)+Encode1(0=REGISTER)+Encode1(modify)+[Encode4(threadId) if modify]+Encode1(notice)+EncodeStr(title)+EncodeStr(message)+Encode4(emoticonId). Body == v83.
// packet-audit:verify packet=guild/serverbound/GuildBBSCreateOrEditThread version=gms_v48 ida=0x608d55
// packet-audit:verify packet=guild/serverbound/GuildBBSCreateOrEditThread version=gms_v79 ida=0x786808
// v72 CUIGuildBBS::OnRegister @0x7517ca: COutPacket(153)+Encode1(0)+Encode1(modify)+[if modify:Encode4(threadId)]+Encode1(notice)+EncodeStr(title)+EncodeStr(msg)+Encode4(emoticon), == v79.
// packet-audit:verify packet=guild/serverbound/GuildBBSCreateOrEditThread version=gms_v72 ida=0x7517ca
// packet-audit:verify packet=guild/serverbound/GuildBBSCreateOrEditThread version=gms_v87 ida=0x87a5df
// packet-audit:verify packet=guild/serverbound/GuildBBSCreateOrEditThread version=gms_v95 ida=0x7c4250
// packet-audit:verify packet=guild/serverbound/GuildBBSCreateOrEditThread version=gms_v83 ida=0x8166f6
// v84 OnRegister COutPacket(0x9F)+Encode1(0)+Encode1(modify)+[if modify:Encode4(threadId)]+Encode1(notice)+EncodeStr(title)+EncodeStr(msg)+Encode4(emoticon), IDA-verified.
// packet-audit:verify packet=guild/serverbound/GuildBBSCreateOrEditThread version=gms_v84 ida=0x84198d
// packet-audit:verify packet=guild/serverbound/GuildBBSCreateOrEditThread version=jms_v185 ida=ABSENT
// v61 COutPacket(134)+Encode1(0=REGISTER)+Encode1(modify)+[Encode4(threadId) if modify]+Encode1(notice)+EncodeStr(title)+EncodeStr(message)+Encode4(emoticon), == v72/v83 (CUIGuildBBS::OnRegister @0x6bb129).
// packet-audit:verify packet=guild/serverbound/GuildBBSCreateOrEditThread version=gms_v61 ida=0x6bb129
func TestBBSCreateOrEditThreadRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name+"/create", func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := BBSCreateOrEditThread{modify: false, notice: true, title: "Hello", message: "World", emoticonId: 5}
			output := BBSCreateOrEditThread{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Modify() != input.Modify() {
				t.Errorf("modify: got %v, want %v", output.Modify(), input.Modify())
			}
			if output.Notice() != input.Notice() {
				t.Errorf("notice: got %v, want %v", output.Notice(), input.Notice())
			}
			if output.Title() != input.Title() {
				t.Errorf("title: got %v, want %v", output.Title(), input.Title())
			}
			if output.Message() != input.Message() {
				t.Errorf("message: got %v, want %v", output.Message(), input.Message())
			}
			if output.EmoticonId() != input.EmoticonId() {
				t.Errorf("emoticonId: got %v, want %v", output.EmoticonId(), input.EmoticonId())
			}
		})
		t.Run(v.Name+"/edit", func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := BBSCreateOrEditThread{modify: true, threadId: 42, notice: false, title: "Updated", message: "Content", emoticonId: 3}
			output := BBSCreateOrEditThread{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Modify() != input.Modify() {
				t.Errorf("modify: got %v, want %v", output.Modify(), input.Modify())
			}
			if output.ThreadId() != input.ThreadId() {
				t.Errorf("threadId: got %v, want %v", output.ThreadId(), input.ThreadId())
			}
			if output.Notice() != input.Notice() {
				t.Errorf("notice: got %v, want %v", output.Notice(), input.Notice())
			}
			if output.Title() != input.Title() {
				t.Errorf("title: got %v, want %v", output.Title(), input.Title())
			}
			if output.Message() != input.Message() {
				t.Errorf("message: got %v, want %v", output.Message(), input.Message())
			}
			if output.EmoticonId() != input.EmoticonId() {
				t.Errorf("emoticonId: got %v, want %v", output.EmoticonId(), input.EmoticonId())
			}
		})
	}
}
