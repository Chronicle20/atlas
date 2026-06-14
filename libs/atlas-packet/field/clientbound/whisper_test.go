package clientbound

import (
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestWhisperSendResultRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := WhisperSendResult{mode: 0x0A, targetName: "TargetPlayer", success: true}
			output := WhisperSendResult{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.TargetName() != input.TargetName() {
				t.Errorf("targetName: got %v, want %v", output.TargetName(), input.TargetName())
			}
			if output.Success() != input.Success() {
				t.Errorf("success: got %v, want %v", output.Success(), input.Success())
			}
		})
	}
}

func TestWhisperReceiveRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := WhisperReceive{mode: 0x12, fromName: "SenderPlayer", channelId: 3, gm: false, message: "secret whisper"}
			output := WhisperReceive{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.FromName() != input.FromName() {
				t.Errorf("fromName: got %v, want %v", output.FromName(), input.FromName())
			}
			if output.ChannelId() != input.ChannelId() {
				t.Errorf("channelId: got %v, want %v", output.ChannelId(), input.ChannelId())
			}
			if output.Gm() != input.Gm() {
				t.Errorf("gm: got %v, want %v", output.Gm(), input.Gm())
			}
			if output.Message() != input.Message() {
				t.Errorf("message: got %v, want %v", output.Message(), input.Message())
			}
		})
	}
}

func TestWhisperFindResultCashShopRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := WhisperFindResultCashShop{mode: 0x09, targetName: "ShopPlayer"}
			output := WhisperFindResultCashShop{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.TargetName() != input.TargetName() {
				t.Errorf("targetName: got %v, want %v", output.TargetName(), input.TargetName())
			}
		})
	}
}

func TestWhisperFindResultMapRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := WhisperFindResultMap{mode: 0x09, targetName: "MapPlayer", mapId: 100000000}
			output := WhisperFindResultMap{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.TargetName() != input.TargetName() {
				t.Errorf("targetName: got %v, want %v", output.TargetName(), input.TargetName())
			}
			if output.MapId() != input.MapId() {
				t.Errorf("mapId: got %v, want %v", output.MapId(), input.MapId())
			}
		})
	}
}

func TestWhisperFindResultMapWithXYRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := WhisperFindResultMap{mode: 0x09, targetName: "MapPlayer", mapId: 100000000, includeXY: true, x: 150, y: -200}
			output := WhisperFindResultMap{includeXY: true}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.TargetName() != input.TargetName() {
				t.Errorf("targetName: got %v, want %v", output.TargetName(), input.TargetName())
			}
			if output.MapId() != input.MapId() {
				t.Errorf("mapId: got %v, want %v", output.MapId(), input.MapId())
			}
			if output.X() != input.X() {
				t.Errorf("x: got %v, want %v", output.X(), input.X())
			}
			if output.Y() != input.Y() {
				t.Errorf("y: got %v, want %v", output.Y(), input.Y())
			}
		})
	}
}

func TestWhisperFindResultChannelRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := WhisperFindResultChannel{mode: 0x09, targetName: "ChannelPlayer", channelId: 5}
			output := WhisperFindResultChannel{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.TargetName() != input.TargetName() {
				t.Errorf("targetName: got %v, want %v", output.TargetName(), input.TargetName())
			}
			if output.ChannelId() != input.ChannelId() {
				t.Errorf("channelId: got %v, want %v", output.ChannelId(), input.ChannelId())
			}
		})
	}
}

func TestWhisperFindResultErrorRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := WhisperFindResultError{mode: 0x09, targetName: "MissingPlayer"}
			output := WhisperFindResultError{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.TargetName() != input.TargetName() {
				t.Errorf("targetName: got %v, want %v", output.TargetName(), input.TargetName())
			}
		})
	}
}

func TestWhisperErrorRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := WhisperError{mode: 0x22, targetName: "BlockedPlayer", whispersEnabled: false}
			output := WhisperError{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.TargetName() != input.TargetName() {
				t.Errorf("targetName: got %v, want %v", output.TargetName(), input.TargetName())
			}
			if output.WhispersEnabled() != input.WhispersEnabled() {
				t.Errorf("whispersEnabled: got %v, want %v", output.WhispersEnabled(), input.WhispersEnabled())
			}
		})
	}
}

func TestWhisperWeatherRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := WhisperWeather{mode: 0x92, fromName: "GMPlayer", message: "Weather alert!"}
			output := WhisperWeather{}
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.Mode() != input.Mode() {
				t.Errorf("mode: got %v, want %v", output.Mode(), input.Mode())
			}
			if output.FromName() != input.FromName() {
				t.Errorf("fromName: got %v, want %v", output.FromName(), input.FromName())
			}
			if output.Message() != input.Message() {
				t.Errorf("message: got %v, want %v", output.Message(), input.Message())
			}
		})
	}
}
