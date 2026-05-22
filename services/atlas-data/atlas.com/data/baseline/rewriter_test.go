package baseline

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/google/uuid"
)

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
