package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestTvClearMessageRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewTvClearMessage()
			output := TvClearMessage{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}

// IDA evidence (gms_v83 MapleStory_dump.exe, port 13342):
//
//	CMapleTVMan::OnClearMessage@0x6371ad never touches its CInPacket argument
//	— body is `*this=0; *(this+243)=0; *(this+1)=1;` (local state only).
//	Wire body is EMPTY. Matches TvClearMessage.Encode exactly (already
//	empty — no change needed).
//
// packet-audit:verify packet=tv/clientbound/TvTvClearMessage version=gms_v83 ida=0x6371ad
func TestTvClearMessageByteOutput(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	input := NewTvClearMessage()
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if len(actual) != 0 {
		t.Errorf("payload length: got %d, want 0", len(actual))
	}
}
