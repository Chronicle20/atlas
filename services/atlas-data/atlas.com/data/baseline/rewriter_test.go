package baseline

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/google/uuid"
)

// TestRewriterReplacesTenantId covers a binary `uuid` tenant_id column (16-byte
// field) — the form the *_search_index tables dump in. The rewriter must emit
// the target as 16 raw bytes.
func TestRewriterReplacesTenantId(t *testing.T) {
	src := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	dst := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	var in bytes.Buffer
	in.Write(CopyBinarySignature)
	_ = binary.Write(&in, binary.BigEndian, int32(0)) // flags
	_ = binary.Write(&in, binary.BigEndian, int32(0)) // ext area length
	// one row, two fields: id int8, tenant_id uuid16
	_ = binary.Write(&in, binary.BigEndian, int16(2))
	_ = binary.Write(&in, binary.BigEndian, int32(8))
	_ = binary.Write(&in, binary.BigEndian, int64(42))
	_ = binary.Write(&in, binary.BigEndian, int32(16))
	in.Write(src[:])
	_ = binary.Write(&in, binary.BigEndian, int16(-1)) // trailer

	var out bytes.Buffer
	rw := Rewriter{TenantColIndex: 1, Target: dst}
	if err := rw.Stream(&in, &out); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(out.Bytes(), dst[:]) {
		t.Fatalf("target uuid bytes not present in output")
	}
	if bytes.Contains(out.Bytes(), src[:]) {
		t.Fatalf("source uuid bytes still present in output")
	}
}

// TestRewriterReplacesTextTenantId covers a `text` tenant_id column (36-byte
// canonical uuid string) — the form the `documents` table dumps in. The
// rewriter must emit the target as the 36-byte text string, NOT 16 raw bytes;
// emitting raw bytes into a text column is the SQLSTATE 22021
// "invalid byte sequence for encoding UTF8: 0x82" restore failure.
func TestRewriterReplacesTextTenantId(t *testing.T) {
	src := "11111111-1111-1111-1111-111111111111" // text uuid, 36 bytes
	dst := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	var in bytes.Buffer
	in.Write(CopyBinarySignature)
	_ = binary.Write(&in, binary.BigEndian, int32(0)) // flags
	_ = binary.Write(&in, binary.BigEndian, int32(0)) // ext area length
	// one row, two fields: id int8, tenant_id text(36)
	_ = binary.Write(&in, binary.BigEndian, int16(2))
	_ = binary.Write(&in, binary.BigEndian, int32(8))
	_ = binary.Write(&in, binary.BigEndian, int64(42))
	_ = binary.Write(&in, binary.BigEndian, int32(int32(len(src))))
	in.WriteString(src)
	_ = binary.Write(&in, binary.BigEndian, int16(-1)) // trailer

	var out bytes.Buffer
	rw := Rewriter{TenantColIndex: 1, Target: dst}
	if err := rw.Stream(&in, &out); err != nil {
		t.Fatal(err)
	}

	// Parse the rewritten tenant_id field precisely: header(19) + fieldCount(2)
	// + field0 [int32 size=8][8 bytes] = offset 33, then field1 [int32 size][..].
	b := out.Bytes()
	const tenantSizeOff = 11 + 4 + 4 + 2 + 4 + 8
	if len(b) < tenantSizeOff+4 {
		t.Fatalf("output too short: %d bytes", len(b))
	}
	gotSize := int32(binary.BigEndian.Uint32(b[tenantSizeOff : tenantSizeOff+4]))
	want := []byte(dst.String())
	if int(gotSize) != len(want) {
		t.Fatalf("tenant field size = %d, want %d (36-byte text uuid)", gotSize, len(want))
	}
	gotField := b[tenantSizeOff+4 : tenantSizeOff+4+int(gotSize)]
	if !bytes.Equal(gotField, want) {
		t.Fatalf("tenant field = %q, want %q", gotField, want)
	}
	// The 16 raw target bytes must NOT appear — that is the bug shape.
	if bytes.Contains(b, dst[:]) {
		t.Fatalf("target uuid emitted as 16 raw bytes into a text field (the 0x82 restore bug)")
	}
}
