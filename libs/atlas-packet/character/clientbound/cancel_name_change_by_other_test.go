package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=character/clientbound/CancelNameChangeByOther version=gms_v83 ida=0xa2a7e6
// packet-audit:verify packet=character/clientbound/CancelNameChangeByOther version=gms_v87 ida=0xac2482
// packet-audit:verify packet=character/clientbound/CancelNameChangeByOther version=gms_v95 ida=0x9f7620
// packet-audit:verify packet=character/clientbound/CancelNameChangeByOther version=gms_v84 ida=0xa75fa9
// packet-audit:verify packet=character/clientbound/CancelNameChangeByOther version=gms_v92 ida=0x9cc170
// packet-audit:verify packet=character/clientbound/CancelNameChangeByOther version=gms_v72 ida=0x922508
// packet-audit:verify packet=character/clientbound/CancelNameChangeByOther version=gms_v79 ida=0x97463d
func TestCancelNameChangeByOtherRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := CancelNameChangeByOther{}
			output := CancelNameChangeByOther{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
		})
	}
}

// TestCancelNameChangeByOtherByteFixture is the hand-encoded byte fixture
// required by the evidence bar — written independently of Encode, not
// derived from it. Every one of the seven applicable receivers
// (CWvsContext::OnCancelNameChangebyOther, re-decompiled this pass on v72
// 0x922508, v79 0x97463d, v83 0xa2a7e6, v84 0xa75fa9, v87 0xac2482, v92
// 0x9cc170, v95 0x9f7620) contains NO `CInPacket::Decode*` call whatsoever —
// only StringPool::GetString + CUtilDlg::Notice + a flag/timestamp write, all
// client-local. The wire body is therefore zero bytes on every applicable
// version; there is no result byte, no sentinel, and no version delta to gate
// on (unlike either CANCEL_*_RESULT sibling, which decode a leading nResult
// byte with per-op success sentinels — see the type doc comment).
func TestCancelNameChangeByOtherByteFixtureEmpty(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			got := CancelNameChangeByOther{}.Encode(nil, ctx)(nil)
			want := []byte{}
			if !bytes.Equal(got, want) {
				t.Errorf("%s CancelNameChangeByOther wire: got %x want empty", v.Name, got)
			}
		})
	}
}

// TestCancelNameChangeByOtherNoVersionDivergence pins that the wire body is
// byte-identical (empty) across every applicable version — derivation.md
// §2.8 shows no field, no sentinel, and no per-version delta of any kind on
// this op, in contrast to both CANCEL_*_RESULT siblings.
func TestCancelNameChangeByOtherNoVersionDivergence(t *testing.T) {
	applicable := []pt.TenantVariant{
		{Name: "GMS v72", Region: "GMS", MajorVersion: 72, MinorVersion: 1},
		{Name: "GMS v79", Region: "GMS", MajorVersion: 79, MinorVersion: 1},
		{Name: "GMS v83", Region: "GMS", MajorVersion: 83, MinorVersion: 1},
		{Name: "GMS v84", Region: "GMS", MajorVersion: 84, MinorVersion: 1},
		{Name: "GMS v87", Region: "GMS", MajorVersion: 87, MinorVersion: 1},
		{Name: "GMS v92", Region: "GMS", MajorVersion: 92, MinorVersion: 1},
		{Name: "GMS v95", Region: "GMS", MajorVersion: 95, MinorVersion: 1},
	}
	var first []byte
	for i, v := range applicable {
		ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
		got := CancelNameChangeByOther{}.Encode(nil, ctx)(nil)
		if i == 0 {
			first = got
			continue
		}
		if !bytes.Equal(got, first) {
			t.Errorf("%s diverges from %s: got %x want %x", v.Name, applicable[0].Name, got, first)
		}
	}
}
