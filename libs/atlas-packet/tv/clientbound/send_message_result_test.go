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

// IDA evidence (gms_v95 GMS_v95.0_U_DEVM.exe, port 13341, PDB-backed) —
// CMapleTVMan::OnSendMessageResult@0x60f5f0:
//
//	v3 = Decode1(iPacket)             // hasError
//	if (!v3) return;                  // success: no further reads
//	v4 = Decode1(iPacket) - 1         // code
//	if (!v4)      code==1 -> GetString(0xF9E=3998), CHATLOG_ADD  (GM_MESSAGE)
//	v5 = v4 - 1
//	if (v5) {
//	  if (v5 != 1) return;
//	  else code==3 -> GetString(0xF9F=3999), CHATLOG_ADD          (QUEUE_TOO_LONG)
//	} else code==2 -> GetString(0xFA0=4000), CHATLOG_ADD          (WRONG_USER)
//	Branch STRUCTURE (and hence code→semantic order: code1=first branch,
//	code2="else"/v5==0 branch, code3=v5==1 branch) is byte-identical to
//	gms_v83's OnSendMessageResult@0x6373a0 — only the StringPool ids differ
//	(3998/3999/4000 vs v83's), confirming the SAME GM_MESSAGE=1/WRONG_USER=2/
//	QUEUE_TOO_LONG=3 code mapping applies to gms_v95. template_gms_95_1.json
//	previously carried the WRONG QUEUE_TOO_LONG=2/WRONG_USER=3 swap (design
//	§1.2's error, propagated uncritically into the seed) — corrected in this
//	commit to GM_MESSAGE=1/WRONG_USER=2/QUEUE_TOO_LONG=3.
//
// packet-audit:verify packet=tv/clientbound/TvTvSendMessageResult version=gms_v95 ida=0x60f5f0
func TestTvSendMessageResultSuccessRoundTripV95(t *testing.T) {
	ctx := pt.CreateContext("GMS", 95, 1)
	input := NewTvSendMessageResultSuccess()
	output := TvSendMessageResult{}
	pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
	if output.HasError() {
		t.Errorf("hasError: got true, want false")
	}
}

func TestTvSendMessageResultErrorRoundTripV95(t *testing.T) {
	ctx := pt.CreateContext("GMS", 95, 1)
	input := NewTvSendMessageResultError(2)
	output := TvSendMessageResult{}
	pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
	if !output.HasError() {
		t.Errorf("hasError: got false, want true")
	}
	if output.Code() != 2 {
		t.Errorf("code: got %v, want 2", output.Code())
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
