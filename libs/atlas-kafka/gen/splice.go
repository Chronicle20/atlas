package main

import (
	"bytes"
	"fmt"
)

// Splice replaces the region between beginMarker and endMarker (both marker
// lines themselves are preserved) with block, leaving every byte outside
// that region -- including the file's existing line endings -- untouched.
//
// It errors if either marker is missing from existing, or if endMarker
// appears before beginMarker.
func Splice(existing []byte, beginMarker, endMarker string, block []byte) ([]byte, error) {
	beginBytes := []byte(beginMarker)
	endBytes := []byte(endMarker)

	beginIdx := bytes.Index(existing, beginBytes)
	if beginIdx == -1 {
		return nil, fmt.Errorf("splice: begin marker %q not found", beginMarker)
	}
	endIdx := bytes.Index(existing, endBytes)
	if endIdx == -1 {
		return nil, fmt.Errorf("splice: end marker %q not found", endMarker)
	}
	if endIdx < beginIdx {
		return nil, fmt.Errorf("splice: end marker %q is out of order relative to begin marker %q", endMarker, beginMarker)
	}

	// The replaced region starts right after the begin marker's line and
	// ends right at the start of the end marker's line, so the markers
	// themselves are preserved and only the content between them changes.
	regionStart := beginIdx + len(beginBytes)
	if nl := bytes.IndexByte(existing[regionStart:], '\n'); nl != -1 {
		regionStart += nl + 1
	}

	var out bytes.Buffer
	out.Write(existing[:regionStart])
	out.Write(block)
	out.Write(existing[endIdx:])
	return out.Bytes(), nil
}
