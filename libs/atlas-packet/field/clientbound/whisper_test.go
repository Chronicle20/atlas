package clientbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/clientbound/FieldWhisperSendResult version=gms_v79 ida=0x51d76d
// packet-audit:verify packet=field/clientbound/FieldWhisperSendResult version=gms_v83 ida=0x53228e
// packet-audit:verify packet=field/clientbound/FieldWhisperSendResult version=gms_v84 ida=0x53e514
// packet-audit:verify packet=field/clientbound/FieldWhisperSendResult version=gms_v87 ida=0x559b1d
// packet-audit:verify packet=field/clientbound/FieldWhisperSendResult version=gms_v95 ida=0x5448a0
// packet-audit:verify packet=field/clientbound/FieldWhisperSendResult version=jms_v185 ida=0x56f4df
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

// packet-audit:verify packet=field/clientbound/FieldWhisperReceive version=gms_v79 ida=0x51d76d
// packet-audit:verify packet=field/clientbound/FieldWhisperReceive version=gms_v83 ida=0x53228e
// packet-audit:verify packet=field/clientbound/FieldWhisperReceive version=gms_v84 ida=0x53e514
// packet-audit:verify packet=field/clientbound/FieldWhisperReceive version=gms_v87 ida=0x559b1d
// packet-audit:verify packet=field/clientbound/FieldWhisperReceive version=gms_v95 ida=0x5448a0
// packet-audit:verify packet=field/clientbound/FieldWhisperReceive version=jms_v185 ida=0x56f4df
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

// packet-audit:verify packet=field/clientbound/FieldWhisperFindResultCashShop version=gms_v79 ida=0x51d76d
// packet-audit:verify packet=field/clientbound/FieldWhisperFindResultCashShop version=gms_v83 ida=0x53228e
// packet-audit:verify packet=field/clientbound/FieldWhisperFindResultCashShop version=gms_v84 ida=0x53e514
// packet-audit:verify packet=field/clientbound/FieldWhisperFindResultCashShop version=gms_v87 ida=0x559b1d
// packet-audit:verify packet=field/clientbound/FieldWhisperFindResultCashShop version=gms_v95 ida=0x5448a0
// packet-audit:verify packet=field/clientbound/FieldWhisperFindResultCashShop version=jms_v185 ida=0x56f4df
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

// packet-audit:verify packet=field/clientbound/FieldWhisperFindResultMap version=gms_v79 ida=0x51d76d
// packet-audit:verify packet=field/clientbound/FieldWhisperFindResultMap version=gms_v83 ida=0x53228e
// packet-audit:verify packet=field/clientbound/FieldWhisperFindResultMap version=gms_v84 ida=0x53e514
// packet-audit:verify packet=field/clientbound/FieldWhisperFindResultMap version=gms_v87 ida=0x559b1d
// packet-audit:verify packet=field/clientbound/FieldWhisperFindResultMap version=gms_v95 ida=0x5448a0
// packet-audit:verify packet=field/clientbound/FieldWhisperFindResultMap version=jms_v185 ida=0x56f4df
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

// packet-audit:verify packet=field/clientbound/FieldWhisperFindResultChannel version=gms_v79 ida=0x51d76d
// packet-audit:verify packet=field/clientbound/FieldWhisperFindResultChannel version=gms_v83 ida=0x53228e
// packet-audit:verify packet=field/clientbound/FieldWhisperFindResultChannel version=gms_v84 ida=0x53e514
// packet-audit:verify packet=field/clientbound/FieldWhisperFindResultChannel version=gms_v87 ida=0x559b1d
// packet-audit:verify packet=field/clientbound/FieldWhisperFindResultChannel version=gms_v95 ida=0x5448a0
// packet-audit:verify packet=field/clientbound/FieldWhisperFindResultChannel version=jms_v185 ida=0x56f4df
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

// packet-audit:verify packet=field/clientbound/FieldWhisperFindResultError version=gms_v79 ida=0x51d76d
// packet-audit:verify packet=field/clientbound/FieldWhisperFindResultError version=gms_v83 ida=0x53228e
// packet-audit:verify packet=field/clientbound/FieldWhisperFindResultError version=gms_v84 ida=0x53e514
// packet-audit:verify packet=field/clientbound/FieldWhisperFindResultError version=gms_v87 ida=0x559b1d
// packet-audit:verify packet=field/clientbound/FieldWhisperFindResultError version=gms_v95 ida=0x5448a0
// packet-audit:verify packet=field/clientbound/FieldWhisperFindResultError version=jms_v185 ida=0x56f4df
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

// packet-audit:verify packet=field/clientbound/FieldWhisperError version=gms_v79 ida=0x51d76d
// packet-audit:verify packet=field/clientbound/FieldWhisperError version=gms_v83 ida=0x53228e
// packet-audit:verify packet=field/clientbound/FieldWhisperError version=gms_v84 ida=0x53e514
// packet-audit:verify packet=field/clientbound/FieldWhisperError version=gms_v87 ida=0x559b1d
// packet-audit:verify packet=field/clientbound/FieldWhisperError version=gms_v95 ida=0x5448a0
// packet-audit:verify packet=field/clientbound/FieldWhisperError version=jms_v185 ida=0x56f4df
// TestWhisperErrorByteOutputV79 pins the gms_v79 WHISPER (op 0x7F) clientbound
// error sub-mode. IDA: CField::OnWhisper @0x51d76d (GMS_v79_1_DEVM.exe), v93==34
// arm reads — Decode1(mode) @0x51d79f, DecodeStr(target) @0x51d87f,
// Decode1(whispersEnabled) @0x51d88a. WriteAsciiString = uint16-LE len + bytes.
func TestWhisperErrorByteOutputV79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)
	input := WhisperError{mode: 0x22, targetName: "BlockedPlayer", whispersEnabled: false}
	expected := []byte{
		0x22,                                                                                     // mode @0x51d79f
		0x0D, 0x00, 0x42, 0x6C, 0x6F, 0x63, 0x6B, 0x65, 0x64, 0x50, 0x6C, 0x61, 0x79, 0x65, 0x72, // target "BlockedPlayer" @0x51d87f
		0x00, // whispersEnabled=false @0x51d88a
	}
	actual := pt.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v79 whisper error golden mismatch: got %v want %v", actual, expected)
	}
}

// wstrV79 builds a WriteAsciiString wire field: uint16-LE length + ASCII bytes.
func wstrV79(s string) []byte {
	out := []byte{byte(len(s)), byte(len(s) >> 8)}
	return append(out, []byte(s)...)
}

// TestWhisperVariantsByteOutputV79 pins the remaining gms_v79 WHISPER (op 0x7F)
// clientbound sub-modes so the op-cell lifts off worst-of-siblings. IDA:
// CField::OnWhisper @0x51d76d (GMS_v79_1_DEVM.exe) switch on Decode1(mode) @0x51d79f:
//   case 10  SendResult  : DecodeStr(target)@0x51db21 + Decode1(success)@0x51db3a
//   case 18  Receive     : DecodeStr(from)@0x51d965-region + Decode1(ch)@0x51d951 +
//                          Decode1(gm)@0x51d95c + DecodeStr(msg)@0x51d965
//   case 9/72 FindResult : DecodeStr(target)@0x51dc37 + Decode1(findMode)@0x51dc52 +
//                          Decode4(value)@0x51dc5a  [Map adds Decode4 x,y @0x51ddbb/bd]
//   case 146 Weather     : DecodeStr(from)@0x51d7e5 + Decode1(flag)@0x51d7f3 +
//                          DecodeStr(msg)@0x51d7fe
func TestWhisperVariantsByteOutputV79(t *testing.T) {
	ctx := pt.CreateContext("GMS", 79, 1)

	send := NewWhisperSendResult(0x0A, "TargetPlayer", true)
	if got := pt.Encode(t, ctx, send.Encode, nil); !bytes.Equal(got, append(append([]byte{0x0A}, wstrV79("TargetPlayer")...), 0x01)) {
		t.Errorf("v79 sendResult: got %v", got)
	}

	recv := NewWhisperReceive(0x12, "SenderPlayer", 3, false, "secret whisper")
	var rw []byte
	rw = append(rw, 0x12)
	rw = append(rw, wstrV79("SenderPlayer")...)
	rw = append(rw, 0x03, 0x00)
	rw = append(rw, wstrV79("secret whisper")...)
	if got := pt.Encode(t, ctx, recv.Encode, nil); !bytes.Equal(got, rw) {
		t.Errorf("v79 receive: got %v want %v", got, rw)
	}

	cash := NewWhisperFindResultCashShop(0x09, "ShopPlayer")
	if got := pt.Encode(t, ctx, cash.Encode, nil); !bytes.Equal(got, append(append([]byte{0x09}, wstrV79("ShopPlayer")...), 0x02, 0xFF, 0xFF, 0xFF, 0xFF)) {
		t.Errorf("v79 cashShop: got %v", got)
	}

	mp := NewWhisperFindResultMap(0x09, "MapPlayer", 100000000)
	if got := pt.Encode(t, ctx, mp.Encode, nil); !bytes.Equal(got, append(append([]byte{0x09}, wstrV79("MapPlayer")...), 0x01, 0x00, 0xE1, 0xF5, 0x05)) {
		t.Errorf("v79 map: got %v", got)
	}

	ch := NewWhisperFindResultChannel(0x09, "ChannelPlayer", 5)
	if got := pt.Encode(t, ctx, ch.Encode, nil); !bytes.Equal(got, append(append([]byte{0x09}, wstrV79("ChannelPlayer")...), 0x03, 0x05, 0x00, 0x00, 0x00)) {
		t.Errorf("v79 channel: got %v", got)
	}

	fe := NewWhisperFindResultError(0x09, "MissingPlayer")
	if got := pt.Encode(t, ctx, fe.Encode, nil); !bytes.Equal(got, append(append([]byte{0x09}, wstrV79("MissingPlayer")...), 0x00, 0x00, 0x00, 0x00, 0x00)) {
		t.Errorf("v79 findError: got %v", got)
	}

	wx := NewWhisperWeather(0x92, "GMPlayer", "Weather alert!")
	var ww []byte
	ww = append(ww, 0x92)
	ww = append(ww, wstrV79("GMPlayer")...)
	ww = append(ww, 0x01)
	ww = append(ww, wstrV79("Weather alert!")...)
	if got := pt.Encode(t, ctx, wx.Encode, nil); !bytes.Equal(got, ww) {
		t.Errorf("v79 weather: got %v want %v", got, ww)
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

// packet-audit:verify packet=field/clientbound/FieldWhisperWeather version=gms_v79 ida=0x51d76d
// packet-audit:verify packet=field/clientbound/FieldWhisperWeather version=gms_v83 ida=0x53228e
// packet-audit:verify packet=field/clientbound/FieldWhisperWeather version=gms_v84 ida=0x53e514
// packet-audit:verify packet=field/clientbound/FieldWhisperWeather version=gms_v87 ida=0x559b1d
// packet-audit:verify packet=field/clientbound/FieldWhisperWeather version=gms_v95 ida=0x5448a0
// packet-audit:verify packet=field/clientbound/FieldWhisperWeather version=jms_v185 ida=0x56f4df
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
