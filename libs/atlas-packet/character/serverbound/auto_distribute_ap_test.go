package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestAutoDistributeApRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := AutoDistributeAp{
				updateTime: 12345,
				nValue:     5,
				distributes: []DistributeEntry{
					{Flag: 64, Value: 3},
					{Flag: 128, Value: 2},
				},
			}
			output := AutoDistributeAp{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.UpdateTime() != input.UpdateTime() {
				t.Errorf("updateTime: got %v, want %v", output.UpdateTime(), input.UpdateTime())
			}
			if output.NValue() != input.NValue() {
				t.Errorf("nValue: got %v, want %v", output.NValue(), input.NValue())
			}
			if len(output.Distributes()) != len(input.Distributes()) {
				t.Fatalf("distributes count: got %v, want %v", len(output.Distributes()), len(input.Distributes()))
			}
			for i, d := range output.Distributes() {
				if d.Flag != input.distributes[i].Flag {
					t.Errorf("distributes[%d].Flag: got %v, want %v", i, d.Flag, input.distributes[i].Flag)
				}
				if d.Value != input.distributes[i].Value {
					t.Errorf("distributes[%d].Value: got %v, want %v", i, d.Value, input.distributes[i].Value)
				}
			}
		})
	}
}
