package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=character/clientbound/AddCharacterError version=gms_v83 ida=0x5fa26c
// packet-audit:verify packet=character/clientbound/AddCharacterError version=gms_v87 ida=0x631b28
// packet-audit:verify packet=character/clientbound/AddCharacterError version=gms_v95 ida=0x5dabcd
// packet-audit:verify packet=character/clientbound/AddCharacterError version=gms_v84 ida=0x60f268
// packet-audit:verify packet=character/clientbound/AddCharacterError version=gms_v79 ida=0x5ceb55
func TestAddCharacterErrorRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := AddCharacterError{code: 3}
			output := AddCharacterError{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Code() != input.Code() {
				t.Errorf("code: got %v, want %v", output.Code(), input.Code())
			}
		})
	}
}

// AddCharacterError v72 byte-fixture.
//
// Client read order — the create-character result error path in
// CLogin::OnCreateNewCharacterResult @0x5B3C65 (op 14; the IDB symbol is rotated
// one step off its body — verified by dispatch case 14 @0x5b2516). On a non-zero
// result code the handler reads only Decode1(code) @0x5b3c80 and shows an error
// modal (no stat/avatar). Matches AddCharacterError.Encode ([byte code]).
//
// packet-audit:verify packet=character/clientbound/AddCharacterError version=gms_v72 ida=0x5b3c65
func TestAddCharacterErrorByteOutputV72(t *testing.T) {
	ctx := pt.CreateContext("GMS", 72, 1)
	got := AddCharacterError{code: 3}.Encode(nil, ctx)(nil)
	want := []byte{0x03} // error code (Decode1) /*0x5b3c80*/
	if len(got) != 1 || got[0] != want[0] {
		t.Errorf("v72 AddCharacterError wire: got %x want %x", got, want)
	}
}
