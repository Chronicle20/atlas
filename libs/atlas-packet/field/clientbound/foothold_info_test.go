package clientbound

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-packet/test"
)

// packet-audit:verify packet=field/clientbound/FieldFootholdInfo version=gms_v87 ida=0x560fec
// packet-audit:verify packet=field/clientbound/FieldFootholdInfo version=gms_v95 ida=0x53a810
// packet-audit:verify packet=field/clientbound/FieldFootholdInfo version=jms_v185 ida=0x576a89

// TestFootholdInfoV95Golden exercises the v95/jms count-prefixed id-list form
// with a mode==2 (moving) entry: Decode4(count=1), DecodeStr("fh"), Decode4(mode=2),
// Decode4(idCount=1), Decode4(id=7), then 7 × Decode4 + 2 × Decode1.
func TestFootholdInfoV95Golden(t *testing.T) {
	entry := NewFootholdEntry("fh", 2, []uint32{7}, []uint32{1, 2, 3, 4, 5, 6, 7}, 0x01, 0x00)
	input := NewFootholdInfo([]FootholdEntry{entry})
	ctx := test.CreateContext("GMS", 95, 0)
	expected := []byte{
		0x01, 0x00, 0x00, 0x00, // count = 1
		0x02, 0x00, 0x66, 0x68, // name len 2 "fh"
		0x02, 0x00, 0x00, 0x00, // mode = 2
		0x01, 0x00, 0x00, 0x00, // idCount = 1
		0x07, 0x00, 0x00, 0x00, // id = 7
		0x01, 0x00, 0x00, 0x00, // moveInt 1
		0x02, 0x00, 0x00, 0x00, // moveInt 2
		0x03, 0x00, 0x00, 0x00, // moveInt 3
		0x04, 0x00, 0x00, 0x00, // moveInt 4
		0x05, 0x00, 0x00, 0x00, // moveInt 5
		0x06, 0x00, 0x00, 0x00, // moveInt 6
		0x07, 0x00, 0x00, 0x00, // moveInt 7
		0x01, // reverseVertical
		0x00, // reverseHorizontal
	}
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v95 golden mismatch:\n got  %v\n want %v", actual, expected)
	}
}

// TestFootholdInfoV87Golden exercises the v87 single-entry form with a mode==2
// (moving) entry: DecodeStr("fh"), Decode4(mode=2), then 8 × Decode4 + 2 × Decode1.
func TestFootholdInfoV87Golden(t *testing.T) {
	entry := NewFootholdEntry("fh", 2, nil, []uint32{1, 2, 3, 4, 5, 6, 7, 8}, 0x01, 0x00)
	input := NewFootholdInfo([]FootholdEntry{entry})
	ctx := test.CreateContext("GMS", 87, 0)
	expected := []byte{
		0x02, 0x00, 0x66, 0x68, // name len 2 "fh"
		0x02, 0x00, 0x00, 0x00, // mode = 2
		0x01, 0x00, 0x00, 0x00, // moveInt 1
		0x02, 0x00, 0x00, 0x00, // moveInt 2
		0x03, 0x00, 0x00, 0x00, // moveInt 3
		0x04, 0x00, 0x00, 0x00, // moveInt 4
		0x05, 0x00, 0x00, 0x00, // moveInt 5
		0x06, 0x00, 0x00, 0x00, // moveInt 6
		0x07, 0x00, 0x00, 0x00, // moveInt 7
		0x08, 0x00, 0x00, 0x00, // moveInt 8
		0x01, // reverseVertical
		0x00, // reverseHorizontal
	}
	actual := test.Encode(t, ctx, input.Encode, nil)
	if !bytes.Equal(actual, expected) {
		t.Errorf("v87 golden mismatch:\n got  %v\n want %v", actual, expected)
	}
}

func TestFootholdInfoRoundTrip(t *testing.T) {
	for _, v := range test.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			var input FootholdInfo
			if v.MajorVersion >= 95 {
				// v95/jms count form: 7 move ints + id-list.
				input = NewFootholdInfo([]FootholdEntry{
					NewFootholdEntry("a", 0, []uint32{}, nil, 0x00, 0x00),
					NewFootholdEntry("b", 2, []uint32{10, 20}, []uint32{1, 2, 3, 4, 5, 6, 7}, 0x01, 0x01),
				})
			} else {
				// v87 single-entry form: no count, no id-list, 8 move ints.
				// (GMS < 87 is VERSION-ABSENT for this packet and merely exercises
				// the codec's single-entry fallback; the matrix marks those ⬜ from
				// the registry.)
				input = NewFootholdInfo([]FootholdEntry{
					NewFootholdEntry("fh", 2, nil, []uint32{1, 2, 3, 4, 5, 6, 7, 8}, 0x01, 0x00),
				})
			}
			output := FootholdInfo{}
			test.RoundTrip(t, ctx, input.Encode, output.Decode, nil)
			if !reflect.DeepEqual(normalize(output.Entries()), normalize(input.Entries())) {
				t.Errorf("round-trip mismatch:\n got  %+v\n want %+v", output.Entries(), input.Entries())
			}
		})
	}
}

// normalize coerces nil/empty slices to a canonical empty form so DeepEqual does
// not distinguish nil from a zero-length slice produced by the decoder.
func normalize(entries []FootholdEntry) []FootholdEntry {
	out := make([]FootholdEntry, len(entries))
	for i, e := range entries {
		ids := e.ids
		if len(ids) == 0 {
			ids = nil
		}
		mv := e.moveInts
		if len(mv) == 0 {
			mv = nil
		}
		out[i] = NewFootholdEntry(e.name, e.mode, ids, mv, e.reverseVertical, e.reverseHorizontal)
		if len(ids) == 0 {
			out[i].ids = nil
		}
		if len(mv) == 0 {
			out[i].moveInts = nil
		}
	}
	return out
}
