package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// IDA evidence (gms_v83 MapleStory_dump.exe, port 13342) —
// CMapleTVMan::OnSendMessageResult@0x6373a0:
//
//	if (Decode1(a2)) {          // hasError
//	  v2 = Decode1(a2) - 1;     // code
//	  if (v2==0)      code==1 -> "non-GM character tried to send GM message"  (GM_MESSAGE)
//	  else if (v2==1) code==2 -> "you've entered the wrong user name"          (WRONG_USER)
//	  else if (v2==2) code==3 -> "waiting line is longer than an hour"         (QUEUE_TOO_LONG)
//	}
//	Confirms struct shape (hasError bool + optional code byte) already
//	correct. Fixes the seed table: gms_v83 template had WRONG_USER/
//	QUEUE_TOO_LONG swapped (2<->3) — corrected in this commit.
//
// packet-audit:verify packet=tv/clientbound/TvTvSendMessageResult version=gms_v83 ida=0x6373a0
func TestTvSendMessageResultSuccessRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewTvSendMessageResultSuccess()
			output := TvSendMessageResult{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.HasError() {
				t.Errorf("hasError: got true, want false")
			}
		})
	}
}

func TestTvSendMessageResultSuccessByteOutput(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	input := NewTvSendMessageResultSuccess()
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if len(actual) != 1 {
		t.Fatalf("payload length: got %d, want 1", len(actual))
	}
	if actual[0] != 0 {
		t.Errorf("payload byte: got %d, want 0", actual[0])
	}
}

func TestTvSendMessageResultErrorRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := NewTvSendMessageResultError(2)
			output := TvSendMessageResult{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if !output.HasError() {
				t.Errorf("hasError: got false, want true")
			}
			if output.Code() != 2 {
				t.Errorf("code: got %v, want 2", output.Code())
			}
		})
	}
}
