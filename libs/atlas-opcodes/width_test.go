package opcodes

import (
	"testing"

	"github.com/stretchr/testify/require"

	socket "github.com/Chronicle20/atlas/libs/atlas-socket"
)

func opCodeSizeTable() []struct {
	name          string
	region        string
	majorVersion  uint16
	wantSize      int
	wantReadWrite socket.OpReadWriter
} {
	return []struct {
		name          string
		region        string
		majorVersion  uint16
		wantSize      int
		wantReadWrite socket.OpReadWriter
	}{
		{name: "gms v28 is byte", region: "GMS", majorVersion: 28, wantSize: 1, wantReadWrite: socket.ByteReadWriter{}},
		{name: "gms v27 is byte", region: "GMS", majorVersion: 27, wantSize: 1, wantReadWrite: socket.ByteReadWriter{}},
		{name: "gms v29 is short", region: "GMS", majorVersion: 29, wantSize: 2, wantReadWrite: socket.ShortReadWriter{}},
		{name: "gms v83 is short", region: "GMS", majorVersion: 83, wantSize: 2, wantReadWrite: socket.ShortReadWriter{}},
		{name: "jms v185 is short", region: "JMS", majorVersion: 185, wantSize: 2, wantReadWrite: socket.ShortReadWriter{}},
		{name: "jms v28 is short", region: "JMS", majorVersion: 28, wantSize: 2, wantReadWrite: socket.ShortReadWriter{}},
	}
}

func TestOpCodeSize(t *testing.T) {
	for _, tt := range opCodeSizeTable() {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.wantSize, OpCodeSize(tt.region, tt.majorVersion))
		})
	}
}

func TestOpReadWriterFor(t *testing.T) {
	for _, tt := range opCodeSizeTable() {
		t.Run(tt.name, func(t *testing.T) {
			require.IsType(t, tt.wantReadWrite, OpReadWriterFor(tt.region, tt.majorVersion))
		})
	}
}
