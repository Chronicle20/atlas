package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestOperationVisitNoErrorNoSomethingRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := OperationVisit{serialNumber: 100, errorCode: 0, something: false}
			output := OperationVisit{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.SerialNumber() != input.SerialNumber() {
				t.Errorf("serialNumber: got %v, want %v", output.SerialNumber(), input.SerialNumber())
			}
			if output.ErrorCode() != input.ErrorCode() {
				t.Errorf("errorCode: got %v, want %v", output.ErrorCode(), input.ErrorCode())
			}
			if output.Something() != input.Something() {
				t.Errorf("something: got %v, want %v", output.Something(), input.Something())
			}
		})
	}
}

func TestOperationVisitWithErrorRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := OperationVisit{serialNumber: 200, errorCode: 3, errorMessage: "birthday failed", something: false}
			output := OperationVisit{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.SerialNumber() != input.SerialNumber() {
				t.Errorf("serialNumber: got %v, want %v", output.SerialNumber(), input.SerialNumber())
			}
			if output.ErrorCode() != input.ErrorCode() {
				t.Errorf("errorCode: got %v, want %v", output.ErrorCode(), input.ErrorCode())
			}
			if output.ErrorMessage() != input.ErrorMessage() {
				t.Errorf("errorMessage: got %v, want %v", output.ErrorMessage(), input.ErrorMessage())
			}
			if output.Something() != input.Something() {
				t.Errorf("something: got %v, want %v", output.Something(), input.Something())
			}
		})
	}
}

func TestOperationVisitWithSomethingRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := OperationVisit{serialNumber: 300, errorCode: 0, something: true, unk1: 5, cashSerialNumber: 999}
			output := OperationVisit{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.SerialNumber() != input.SerialNumber() {
				t.Errorf("serialNumber: got %v, want %v", output.SerialNumber(), input.SerialNumber())
			}
			if output.ErrorCode() != input.ErrorCode() {
				t.Errorf("errorCode: got %v, want %v", output.ErrorCode(), input.ErrorCode())
			}
			if output.Something() != input.Something() {
				t.Errorf("something: got %v, want %v", output.Something(), input.Something())
			}
			if output.Unk1() != input.Unk1() {
				t.Errorf("unk1: got %v, want %v", output.Unk1(), input.Unk1())
			}
			if output.CashSerialNumber() != input.CashSerialNumber() {
				t.Errorf("cashSerialNumber: got %v, want %v", output.CashSerialNumber(), input.CashSerialNumber())
			}
		})
	}
}
