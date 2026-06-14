package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/serverbound/FieldGeneral version=gms_v83 ida=0x52c315
// packet-audit:verify packet=field/serverbound/FieldGeneral version=gms_v87 ida=0x552b67
// packet-audit:verify packet=field/serverbound/FieldGeneral version=gms_v95 ida=0x534000
// packet-audit:verify packet=field/serverbound/FieldGeneral version=jms_v185 ida=0x564a0a
// packet-audit:verify packet=field/serverbound/FieldGeneral version=gms_v84 ida=0x5382d7
func TestGeneralRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := General{updateTime: 100, msg: "hello world", bOnlyBalloon: true}
			output := General{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Msg() != input.Msg() {
				t.Errorf("msg: got %v, want %v", output.Msg(), input.Msg())
			}
			if output.BOnlyBalloon() != input.BOnlyBalloon() {
				t.Errorf("bOnlyBalloon: got %v, want %v", output.BOnlyBalloon(), input.BOnlyBalloon())
			}
		})
	}
}
