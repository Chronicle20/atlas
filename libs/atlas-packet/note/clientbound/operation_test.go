package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=note/clientbound/NoteRefresh version=gms_v87 ida=0xabccc2
// packet-audit:verify packet=note/clientbound/NoteSendError version=gms_v87 ida=0xabccc2
// packet-audit:verify packet=note/clientbound/NoteSendSuccess version=gms_v87 ida=0xabccc2
// packet-audit:verify packet=note/clientbound/NoteRefresh version=gms_v95 ida=0x9f9da0
// packet-audit:verify packet=note/clientbound/NoteSendError version=gms_v95 ida=0x9f9da0
// packet-audit:verify packet=note/clientbound/NoteSendSuccess version=gms_v95 ida=0x9f9da0
func TestSendSuccessRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := SendSuccess{mode: 4}
			output := SendSuccess{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
		})
	}
}

func TestSendErrorRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := SendError{mode: 5, errorCode: 1}
			output := SendError{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.ErrorCode() != input.ErrorCode() {
				t.Errorf("errorCode: got %v, want %v", output.ErrorCode(), input.ErrorCode())
			}
		})
	}
}

func TestRefreshRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := Refresh{mode: 6}
			output := Refresh{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
		})
	}
}
