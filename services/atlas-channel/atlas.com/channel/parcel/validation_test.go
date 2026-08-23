package parcel

import "testing"

func baselineSendInput() SendInput {
	return SendInput{
		MesoAmount:  1_000,
		Quantity:    1,
		Quick:       false,
		Message:     "",
		SenderLevel: 30,
		SenderMeso:  1_000_000,
	}
}

func repeatChar(c byte, n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = c
	}
	return string(b)
}

func TestValidateSend(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		in := baselineSendInput()
		if got := ValidateSend(in); got != RejectNone {
			t.Errorf("ValidateSend() = %v, want %v", got, RejectNone)
		}
	})

	t.Run("nothing attached", func(t *testing.T) {
		in := baselineSendInput()
		in.MesoAmount = 0
		in.Quantity = 0
		if got := ValidateSend(in); got != RejectIncorrectRequest {
			t.Errorf("ValidateSend() = %v, want %v", got, RejectIncorrectRequest)
		}
	})

	t.Run("item only", func(t *testing.T) {
		in := baselineSendInput()
		in.MesoAmount = 0
		in.Quantity = 1
		if got := ValidateSend(in); got != RejectNone {
			t.Errorf("ValidateSend() = %v, want %v", got, RejectNone)
		}
	})

	t.Run("meso only", func(t *testing.T) {
		in := baselineSendInput()
		in.MesoAmount = 1_000
		in.Quantity = 0
		if got := ValidateSend(in); got != RejectNone {
			t.Errorf("ValidateSend() = %v, want %v", got, RejectNone)
		}
	})

	t.Run("over the parcel cap", func(t *testing.T) {
		in := baselineSendInput()
		in.MesoAmount = 100_000_001
		if got := ValidateSend(in); got != RejectIncorrectRequest {
			t.Errorf("ValidateSend() = %v, want %v", got, RejectIncorrectRequest)
		}
	})

	t.Run("at the parcel cap", func(t *testing.T) {
		in := baselineSendInput()
		in.MesoAmount = 100_000_000
		in.SenderMeso = 4_294_967_295
		if got := ValidateSend(in); got != RejectNone {
			t.Errorf("ValidateSend() = %v, want %v", got, RejectNone)
		}
	})

	t.Run("overflow", func(t *testing.T) {
		in := baselineSendInput()
		in.MesoAmount = 4_294_000_000
		if got := ValidateSend(in); got != RejectIncorrectRequest {
			t.Errorf("ValidateSend() = %v, want %v", got, RejectIncorrectRequest)
		}
	})

	t.Run("low level over limit", func(t *testing.T) {
		in := baselineSendInput()
		in.SenderLevel = 15
		in.MesoAmount = 1_000_001
		in.SenderMeso = 4_294_967_295
		if got := ValidateSend(in); got != RejectMesoLimit {
			t.Errorf("ValidateSend() = %v, want %v", got, RejectMesoLimit)
		}
	})

	t.Run("low level at limit", func(t *testing.T) {
		in := baselineSendInput()
		in.SenderLevel = 15
		in.MesoAmount = 1_000_000
		in.SenderMeso = 4_294_967_295
		if got := ValidateSend(in); got != RejectNone {
			t.Errorf("ValidateSend() = %v, want %v", got, RejectNone)
		}
	})

	t.Run("level 16 over limit", func(t *testing.T) {
		in := baselineSendInput()
		in.SenderLevel = 16
		in.MesoAmount = 1_000_001
		in.SenderMeso = 4_294_967_295
		if got := ValidateSend(in); got != RejectNone {
			t.Errorf("ValidateSend() = %v, want %v", got, RejectNone)
		}
	})

	t.Run("cannot afford", func(t *testing.T) {
		in := baselineSendInput()
		in.MesoAmount = 1_000_000
		in.SenderMeso = 1_000_000
		if got := ValidateSend(in); got != RejectNotEnoughMesos {
			t.Errorf("ValidateSend() = %v, want %v", got, RejectNotEnoughMesos)
		}
	})

	t.Run("can afford exactly", func(t *testing.T) {
		in := baselineSendInput()
		in.MesoAmount = 1_000_000
		in.SenderMeso = 1_023_000
		if got := ValidateSend(in); got != RejectNone {
			t.Errorf("ValidateSend() = %v, want %v", got, RejectNone)
		}
	})

	t.Run("message too long", func(t *testing.T) {
		in := baselineSendInput()
		in.Quick = true
		in.Message = repeatChar('a', 101)
		if got := ValidateSend(in); got != RejectIncorrectRequest {
			t.Errorf("ValidateSend() = %v, want %v", got, RejectIncorrectRequest)
		}
	})

	t.Run("message at the limit", func(t *testing.T) {
		in := baselineSendInput()
		in.Quick = true
		in.Message = repeatChar('a', 100)
		if got := ValidateSend(in); got != RejectNone {
			t.Errorf("ValidateSend() = %v, want %v", got, RejectNone)
		}
	})

	t.Run("message on a non-quick send", func(t *testing.T) {
		in := baselineSendInput()
		in.Quick = false
		in.Message = repeatChar('a', 101)
		if got := ValidateSend(in); got != RejectNone {
			t.Errorf("ValidateSend() = %v, want %v", got, RejectNone)
		}
	})
}
