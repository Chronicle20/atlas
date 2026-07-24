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

// IDA evidence (gms_v84 GMS_v84.1_U_DEVM.exe, port 13345) —
// CMapleTVMan::OnClearMessage@0x64c7d7 (unnamed sub_64C7D7 on this stripped
// IDB): body is `*this=0; this[243]=0; this[1]=1;` (local state only, no
// Decode* call). Wire body is EMPTY, same as gms_v83/v95.
//
// packet-audit:verify packet=tv/clientbound/TvTvClearMessage version=gms_v84 ida=0x64c7d7
func TestTvClearMessageByteOutputV84(t *testing.T) {
	ctx := pt.CreateContext("GMS", 84, 1)
	input := NewTvClearMessage()
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if len(actual) != 0 {
		t.Errorf("payload length: got %d, want 0", len(actual))
	}
}

// IDA evidence (gms_v87 GMSv87_4GB.exe, port 13343) —
// CMapleTVMan::OnClearMessage@0x67021c: body is
// `*this=0; *(this+275)=0; *(this+1)=1;` — no Decode* call. Wire body is
// EMPTY, same as gms_v83/v84/v95.
//
// packet-audit:verify packet=tv/clientbound/TvTvClearMessage version=gms_v87 ida=0x67021c
func TestTvClearMessageByteOutputV87(t *testing.T) {
	ctx := pt.CreateContext("GMS", 87, 1)
	input := NewTvClearMessage()
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if len(actual) != 0 {
		t.Errorf("payload length: got %d, want 0", len(actual))
	}
}

// IDA evidence (jms_v185 MapleStory_dump_SCY.exe, port 13344) —
// CMapleTVMan::OnClearMessage@0x6ab16d: body is
// `this->m_bShowMessage=0; this->m_nTotalWaitTime=0; this->m_bQueueExists=1;`
// — no Decode* call. Wire body is EMPTY, same as GMS.
//
// packet-audit:verify packet=tv/clientbound/TvTvClearMessage version=jms_v185 ida=0x6ab16d
func TestTvClearMessageByteOutputJms(t *testing.T) {
	ctx := pt.CreateContext("JMS", 185, 1)
	input := NewTvClearMessage()
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if len(actual) != 0 {
		t.Errorf("payload length: got %d, want 0", len(actual))
	}
}

// IDA evidence (gms_v95 GMS_v95.0_U_DEVM.exe, port 13341, PDB-backed) —
// CMapleTVMan::OnClearMessage@0x60f2f0:
//
//	void __thiscall CMapleTVMan::OnClearMessage(this, iPacket) {
//	  this->m_bShowMessage = 0;
//	  this->m_bQueueExists = 1;
//	  this->m_nTotalWaitTime = 0;
//	}
//	The iPacket argument is never touched. Wire body is EMPTY — same as
//	gms_v83.
//
// packet-audit:verify packet=tv/clientbound/TvTvClearMessage version=gms_v95 ida=0x60f2f0
func TestTvClearMessageByteOutputV95(t *testing.T) {
	ctx := pt.CreateContext("GMS", 95, 1)
	input := NewTvClearMessage()
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if len(actual) != 0 {
		t.Errorf("payload length: got %d, want 0", len(actual))
	}
}
