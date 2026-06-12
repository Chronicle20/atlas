package idasrc

import "testing"

func TestDemangleQualified(t *testing.T) {
	cases := []struct {
		in     string
		want   string
		wantOK bool
	}{
		{"?Reset@CFriend@CWvsContext@@QAEXAAVCInPacket@@@Z", "CWvsContext::CFriend::Reset", true},
		{"?UpdateFriend@CFriend@CWvsContext@@QAEXAAVCInPacket@@@Z", "CWvsContext::CFriend::UpdateFriend", true},
		{"?Decode1@CInPacket@@QAEEXZ", "CInPacket::Decode1", true},
		{"?Insert@CFriend@CWvsContext@@QAEXAAVCInPacket@@@Z", "CWvsContext::CFriend::Insert", true},
		// A bare top-level name with no scopes demangles to just the name.
		{"?GlobalFn@@YAXXZ", "GlobalFn", true},
		// Not simply demangleable:
		{"sub_A40028", "", false},                       // not a mangled name (no leading '?')
		{"?$ZXString@D@@QAE@XZ", "", false},             // template ('?$')
		{"??4ZXString@@QAEAAV0@ABV0@@Z", "", false},     // operator ('??')
		{"", "", false},                                 // empty
		{"CWvsContext::CFriend::Reset", "", false},      // already demangled
	}
	for _, c := range cases {
		got, ok := demangleQualified(c.in)
		if ok != c.wantOK || got != c.want {
			t.Errorf("demangleQualified(%q) = (%q, %v); want (%q, %v)", c.in, got, ok, c.want, c.wantOK)
		}
	}
}
