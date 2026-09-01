package opcodes

import (
	socket "github.com/Chronicle20/atlas/libs/atlas-socket"
)

// OpCodeSize returns the wire width in bytes of this tenant's opcodes.
// This rule was duplicated verbatim in atlas-channel/main.go and
// atlas-login/main.go; the packet tracer would have been a third copy, so
// it lives here now and both main.go files call OpReadWriterFor.
func OpCodeSize(region string, majorVersion uint16) int {
	if region == "GMS" && majorVersion <= 28 {
		return 1
	}
	return 2
}

// OpReadWriterFor returns the OpReadWriter matching OpCodeSize.
func OpReadWriterFor(region string, majorVersion uint16) socket.OpReadWriter {
	if OpCodeSize(region, majorVersion) == 1 {
		return socket.ByteReadWriter{}
	}
	return socket.ShortReadWriter{}
}
