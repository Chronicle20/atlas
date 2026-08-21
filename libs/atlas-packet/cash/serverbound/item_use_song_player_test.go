package serverbound

import (
	"bytes"
	"testing"

	pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

func TestItemUseSongPlayerUpdateTimeFirstRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ItemUseSongPlayer{soundLengthMs: 123456, updateTimeFirst: true}
			output := *NewItemUseSongPlayer(true)
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.SoundLengthMs() != 123456 {
				t.Errorf("soundLengthMs: got %v, want %v", output.SoundLengthMs(), 123456)
			}
		})
	}
}

func TestItemUseSongPlayerNoUpdateTimeFirstRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := ItemUseSongPlayer{soundLengthMs: 123456, updateTime: 77777, updateTimeFirst: false}
			output := *NewItemUseSongPlayer(false)
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if output.SoundLengthMs() != 123456 {
				t.Errorf("soundLengthMs: got %v, want %v", output.SoundLengthMs(), 123456)
			}
			if output.UpdateTime() != 77777 {
				t.Errorf("updateTime: got %v, want %v", output.UpdateTime(), 77777)
			}
		})
	}
}

func TestItemUseSongPlayerWireOrder(t *testing.T) {
	// soundLengthMs first, then the trailing updateTime on the versions that
	// trail it (GMS <= v84). Little-endian int32 each.
	// 0x0001E240 == 123456, 0x00012FD1 == 77777.
	tests := []struct {
		name     string
		input    ItemUseSongPlayer
		expected []byte
	}{
		{
			name:     "updateTimeFirst",
			input:    ItemUseSongPlayer{soundLengthMs: 123456, updateTime: 77777, updateTimeFirst: true},
			expected: []byte{0x40, 0xe2, 0x01, 0x00},
		},
		{
			name:     "updateTimeTrails",
			input:    ItemUseSongPlayer{soundLengthMs: 123456, updateTime: 77777, updateTimeFirst: false},
			expected: []byte{0x40, 0xe2, 0x01, 0x00, 0xd1, 0x2f, 0x01, 0x00},
		},
	}

	ctx := pt.CreateContext("GMS", 83, 1)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := pt.Encode(t, ctx, tt.input.Encode, nil)
			if !bytes.Equal(b, tt.expected) {
				t.Errorf("bytes: got %x, want %x", b, tt.expected)
			}
		})
	}
}
