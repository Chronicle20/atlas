package baseline

import (
	"encoding/binary"
	"io"

	"github.com/google/uuid"
)

// CopyBinarySignature is the leading bytes of every COPY binary stream.
var CopyBinarySignature = []byte("PGCOPY\n\xff\r\n\x00")

// Rewriter streams the COPY-binary form of a single table, replacing the
// tenant_id column value (column index given by TenantColIndex) with Target.
type Rewriter struct {
	TenantColIndex int
	Target         uuid.UUID
}

// Stream reads a Postgres COPY binary stream from in, rewriting the tenant_id
// column in each row to rw.Target, and writes the result to out.
func (rw Rewriter) Stream(in io.Reader, out io.Writer) error {
	// 11-byte signature, then 4-byte flags, then 4-byte extension area length and that area.
	if err := copyN(in, out, 11); err != nil {
		return err
	}
	if err := copyN(in, out, 4); err != nil {
		return err
	}
	var extLen uint32
	if err := readU32(in, out, &extLen); err != nil {
		return err
	}
	if err := copyN(in, out, int(extLen)); err != nil {
		return err
	}
	for {
		var fieldCount int16
		if err := binary.Read(in, binary.BigEndian, &fieldCount); err != nil {
			return err
		}
		if err := binary.Write(out, binary.BigEndian, fieldCount); err != nil {
			return err
		}
		if fieldCount == -1 {
			return nil // trailer
		}
		for i := int16(0); i < fieldCount; i++ {
			var size int32
			if err := binary.Read(in, binary.BigEndian, &size); err != nil {
				return err
			}
			if int(i) == rw.TenantColIndex {
				// Discard original, emit target uuid (16 bytes).
				if size > 0 {
					if _, err := io.CopyN(io.Discard, in, int64(size)); err != nil {
						return err
					}
				}
				if err := binary.Write(out, binary.BigEndian, int32(16)); err != nil {
					return err
				}
				if _, err := out.Write(rw.Target[:]); err != nil {
					return err
				}
				continue
			}
			if err := binary.Write(out, binary.BigEndian, size); err != nil {
				return err
			}
			if size > 0 {
				if _, err := io.CopyN(out, in, int64(size)); err != nil {
					return err
				}
			}
		}
	}
}

func copyN(in io.Reader, out io.Writer, n int) error {
	_, err := io.CopyN(out, in, int64(n))
	return err
}

func readU32(in io.Reader, out io.Writer, v *uint32) error {
	if err := binary.Read(in, binary.BigEndian, v); err != nil {
		return err
	}
	return binary.Write(out, binary.BigEndian, *v)
}
