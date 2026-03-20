package serverbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas-packet/test"
)

func TestOperationCreateOmokRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := OperationCreate{roomType: 1, title: "Omok Room", private: true, password: "pass123", nGameSpec: 2}
			output := OperationCreate{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.RoomType() != input.RoomType() {
				t.Errorf("roomType: got %v, want %v", output.RoomType(), input.RoomType())
			}
			if output.Title() != input.Title() {
				t.Errorf("title: got %v, want %v", output.Title(), input.Title())
			}
			if output.Private() != input.Private() {
				t.Errorf("private: got %v, want %v", output.Private(), input.Private())
			}
			if output.Password() != input.Password() {
				t.Errorf("password: got %v, want %v", output.Password(), input.Password())
			}
			if output.NGameSpec() != input.NGameSpec() {
				t.Errorf("nGameSpec: got %v, want %v", output.NGameSpec(), input.NGameSpec())
			}
		})
	}
}

func TestOperationCreateMatchCardRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := OperationCreate{roomType: 2, title: "Match Room", private: false, nGameSpec: 3}
			output := OperationCreate{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.RoomType() != input.RoomType() {
				t.Errorf("roomType: got %v, want %v", output.RoomType(), input.RoomType())
			}
			if output.Title() != input.Title() {
				t.Errorf("title: got %v, want %v", output.Title(), input.Title())
			}
			if output.Private() != input.Private() {
				t.Errorf("private: got %v, want %v", output.Private(), input.Private())
			}
			if output.NGameSpec() != input.NGameSpec() {
				t.Errorf("nGameSpec: got %v, want %v", output.NGameSpec(), input.NGameSpec())
			}
		})
	}
}

func TestOperationCreateTradeRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := OperationCreate{roomType: 3, private: true}
			output := OperationCreate{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.RoomType() != input.RoomType() {
				t.Errorf("roomType: got %v, want %v", output.RoomType(), input.RoomType())
			}
			if output.Private() != input.Private() {
				t.Errorf("private: got %v, want %v", output.Private(), input.Private())
			}
		})
	}
}

func TestOperationCreateShopRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := OperationCreate{roomType: 4, title: "My Shop", private: false, slot: 5, itemId: 1234}
			output := OperationCreate{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.RoomType() != input.RoomType() {
				t.Errorf("roomType: got %v, want %v", output.RoomType(), input.RoomType())
			}
			if output.Title() != input.Title() {
				t.Errorf("title: got %v, want %v", output.Title(), input.Title())
			}
			if output.Private() != input.Private() {
				t.Errorf("private: got %v, want %v", output.Private(), input.Private())
			}
			if output.Slot() != input.Slot() {
				t.Errorf("slot: got %v, want %v", output.Slot(), input.Slot())
			}
			if output.ItemId() != input.ItemId() {
				t.Errorf("itemId: got %v, want %v", output.ItemId(), input.ItemId())
			}
		})
	}
}

func TestOperationCreateCashTradeRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := OperationCreate{roomType: 6, private: false}
			output := OperationCreate{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.RoomType() != input.RoomType() {
				t.Errorf("roomType: got %v, want %v", output.RoomType(), input.RoomType())
			}
			if output.Private() != input.Private() {
				t.Errorf("private: got %v, want %v", output.Private(), input.Private())
			}
		})
	}
}
