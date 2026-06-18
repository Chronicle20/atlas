package clientbound

import (
	"bytes"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/clientbound/FieldOxQuiz version=gms_v83 ida=0x535a57
// packet-audit:verify packet=field/clientbound/FieldOxQuiz version=gms_v84 ida=0x541d5b
// packet-audit:verify packet=field/clientbound/FieldOxQuiz version=gms_v87 ida=0x55d2da
// packet-audit:verify packet=field/clientbound/FieldOxQuiz version=gms_v95 ida=0x537a90
// packet-audit:verify packet=field/clientbound/FieldOxQuiz version=jms_v185 ida=0x572b75
func TestOxQuizGolden(t *testing.T) {
	input := NewOxQuiz(0x01, 0x02, 0x0010)
	ctx := test.CreateContext("GMS", 83, 1)
	expected := []byte{0x01, 0x02, 0x10, 0x00}
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("golden mismatch: got %v want %v", actual, expected)
	}
}

func TestOxQuizRoundTrip(t *testing.T) {
	input := NewOxQuiz(0x01, 0x02, 0x0010)
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			test.RoundTrip(t, ctx, input.Encode, input.Decode, nil)
		})
	}
}
