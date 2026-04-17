package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestOperationCashTradeOpenInitiateRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := OperationCashTradeOpen{nProc: 0, roomType: 6, targetCharacterId: 12345}
			output := OperationCashTradeOpen{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.NProc() != input.NProc() {
				t.Errorf("nProc: got %v, want %v", output.NProc(), input.NProc())
			}
			if output.RoomType() != input.RoomType() {
				t.Errorf("roomType: got %v, want %v", output.RoomType(), input.RoomType())
			}
			if output.TargetCharacterId() != input.TargetCharacterId() {
				t.Errorf("targetCharacterId: got %v, want %v", output.TargetCharacterId(), input.TargetCharacterId())
			}
		})
	}
}

func TestOperationCashTradeOpenCashTradeRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := OperationCashTradeOpen{nProc: 4, roomType: 6, spw: 111, dwSN: 222, unk2: 1}
			output := OperationCashTradeOpen{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.NProc() != input.NProc() {
				t.Errorf("nProc: got %v, want %v", output.NProc(), input.NProc())
			}
			if output.RoomType() != input.RoomType() {
				t.Errorf("roomType: got %v, want %v", output.RoomType(), input.RoomType())
			}
			if output.Spw() != input.Spw() {
				t.Errorf("spw: got %v, want %v", output.Spw(), input.Spw())
			}
			if output.DwSN() != input.DwSN() {
				t.Errorf("dwSN: got %v, want %v", output.DwSN(), input.DwSN())
			}
			if output.Unk2() != input.Unk2() {
				t.Errorf("unk2: got %v, want %v", output.Unk2(), input.Unk2())
			}
		})
	}
}

func TestOperationCashTradeOpenMerchantRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := OperationCashTradeOpen{nProc: 4, roomType: 5, spw: 333, shopId: 444, unk2: 2, position: 55, serialNumber: 666}
			output := OperationCashTradeOpen{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.NProc() != input.NProc() {
				t.Errorf("nProc: got %v, want %v", output.NProc(), input.NProc())
			}
			if output.RoomType() != input.RoomType() {
				t.Errorf("roomType: got %v, want %v", output.RoomType(), input.RoomType())
			}
			if output.Spw() != input.Spw() {
				t.Errorf("spw: got %v, want %v", output.Spw(), input.Spw())
			}
			if output.ShopId() != input.ShopId() {
				t.Errorf("shopId: got %v, want %v", output.ShopId(), input.ShopId())
			}
			if output.Unk2() != input.Unk2() {
				t.Errorf("unk2: got %v, want %v", output.Unk2(), input.Unk2())
			}
			if output.Position() != input.Position() {
				t.Errorf("position: got %v, want %v", output.Position(), input.Position())
			}
			if output.SerialNumber() != input.SerialNumber() {
				t.Errorf("serialNumber: got %v, want %v", output.SerialNumber(), input.SerialNumber())
			}
		})
	}
}

func TestOperationCashTradeOpenBirthdayRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := OperationCashTradeOpen{nProc: 11, roomType: 4, birthday: 19900101}
			output := OperationCashTradeOpen{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.NProc() != input.NProc() {
				t.Errorf("nProc: got %v, want %v", output.NProc(), input.NProc())
			}
			if output.RoomType() != input.RoomType() {
				t.Errorf("roomType: got %v, want %v", output.RoomType(), input.RoomType())
			}
			if output.Birthday() != input.Birthday() {
				t.Errorf("birthday: got %v, want %v", output.Birthday(), input.Birthday())
			}
		})
	}
}
