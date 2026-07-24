package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// ida-notes.md §G4 — v83 CUserLocal::HandleLButtonDblClk @ 0x94fbbf (send path,
// double-click on a miniroom balloon): mode 4 (ENTER/VISIT) carries
// int32 serialNumber, byte hasPassword, [string password], byte 0 (constant
// trailing). No packet-audit:verify marker is added here — a verify marker
// requires a pinned evidence record + audit report, which is a separate
// (heavier) pass this task does not perform; the fname is wired into
// candidatesFromFName below so a future verify pass can pin it.
func TestOperationVisitGameNoPasswordRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := OperationVisitGame{serialNumber: 100, hasPassword: false}
			output := OperationVisitGame{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.SerialNumber() != input.SerialNumber() {
				t.Errorf("serialNumber: got %v, want %v", output.SerialNumber(), input.SerialNumber())
			}
			if output.HasPassword() != input.HasPassword() {
				t.Errorf("hasPassword: got %v, want %v", output.HasPassword(), input.HasPassword())
			}
			if output.Password() != input.Password() {
				t.Errorf("password: got %v, want %v", output.Password(), input.Password())
			}
		})
	}
}

func TestOperationVisitGameWithPasswordRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := OperationVisitGame{serialNumber: 200, hasPassword: true, password: "s3cr3t"}
			output := OperationVisitGame{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.SerialNumber() != input.SerialNumber() {
				t.Errorf("serialNumber: got %v, want %v", output.SerialNumber(), input.SerialNumber())
			}
			if output.HasPassword() != input.HasPassword() {
				t.Errorf("hasPassword: got %v, want %v", output.HasPassword(), input.HasPassword())
			}
			if output.Password() != input.Password() {
				t.Errorf("password: got %v, want %v", output.Password(), input.Password())
			}
		})
	}
}
