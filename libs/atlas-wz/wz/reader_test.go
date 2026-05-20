package wz

import (
	"encoding/binary"
	"math"
	"os"
	"testing"
)

func tempFileWithBytes(t *testing.T, data []byte) *os.File {
	t.Helper()
	f, err := os.CreateTemp("", "wz_reader_test_*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		f.Close()
		os.Remove(f.Name())
	})
	if _, err := f.Write(data); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Seek(0, 0); err != nil {
		t.Fatal(err)
	}
	return f
}

func TestReadByte(t *testing.T) {
	f := tempFileWithBytes(t, []byte{0x42})
	r := NewReader(f)
	b, err := r.ReadByte()
	if err != nil {
		t.Fatalf("ReadByte: %v", err)
	}
	if b != 0x42 {
		t.Errorf("ReadByte = 0x%02X, want 0x42", b)
	}
}

func TestReadBytes(t *testing.T) {
	f := tempFileWithBytes(t, []byte{0x01, 0x02, 0x03, 0x04})
	r := NewReader(f)
	buf, err := r.ReadBytes(3)
	if err != nil {
		t.Fatalf("ReadBytes: %v", err)
	}
	if len(buf) != 3 || buf[0] != 0x01 || buf[1] != 0x02 || buf[2] != 0x03 {
		t.Errorf("ReadBytes = %v, want [1, 2, 3]", buf)
	}
}

func TestReadInt16(t *testing.T) {
	buf := make([]byte, 2)
	binary.LittleEndian.PutUint16(buf, uint16(12345))
	f := tempFileWithBytes(t, buf)
	r := NewReader(f)
	v, err := r.ReadInt16()
	if err != nil {
		t.Fatalf("ReadInt16: %v", err)
	}
	if v != 12345 {
		t.Errorf("ReadInt16 = %d, want 12345", v)
	}
}

func TestReadInt16Negative(t *testing.T) {
	buf := make([]byte, 2)
	binary.LittleEndian.PutUint16(buf, uint16(65436)) // -100 as uint16
	f := tempFileWithBytes(t, buf)
	r := NewReader(f)
	v, err := r.ReadInt16()
	if err != nil {
		t.Fatalf("ReadInt16: %v", err)
	}
	if v != -100 {
		t.Errorf("ReadInt16 = %d, want -100", v)
	}
}

func TestReadInt32(t *testing.T) {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, uint32(123456789))
	f := tempFileWithBytes(t, buf)
	r := NewReader(f)
	v, err := r.ReadInt32()
	if err != nil {
		t.Fatalf("ReadInt32: %v", err)
	}
	if v != 123456789 {
		t.Errorf("ReadInt32 = %d, want 123456789", v)
	}
}

func TestReadUInt32(t *testing.T) {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, 0xDEADBEEF)
	f := tempFileWithBytes(t, buf)
	r := NewReader(f)
	v, err := r.ReadUInt32()
	if err != nil {
		t.Fatalf("ReadUInt32: %v", err)
	}
	if v != 0xDEADBEEF {
		t.Errorf("ReadUInt32 = 0x%X, want 0xDEADBEEF", v)
	}
}

func TestReadInt64(t *testing.T) {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, uint64(9876543210))
	f := tempFileWithBytes(t, buf)
	r := NewReader(f)
	v, err := r.ReadInt64()
	if err != nil {
		t.Fatalf("ReadInt64: %v", err)
	}
	if v != 9876543210 {
		t.Errorf("ReadInt64 = %d, want 9876543210", v)
	}
}

func TestReadFloat32(t *testing.T) {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, math.Float32bits(3.14))
	f := tempFileWithBytes(t, buf)
	r := NewReader(f)
	v, err := r.ReadFloat32()
	if err != nil {
		t.Fatalf("ReadFloat32: %v", err)
	}
	if v != 3.14 {
		t.Errorf("ReadFloat32 = %f, want 3.14", v)
	}
}

func TestReadFloat64(t *testing.T) {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, math.Float64bits(2.718281828))
	f := tempFileWithBytes(t, buf)
	r := NewReader(f)
	v, err := r.ReadFloat64()
	if err != nil {
		t.Fatalf("ReadFloat64: %v", err)
	}
	if v != 2.718281828 {
		t.Errorf("ReadFloat64 = %f, want 2.718281828", v)
	}
}

func TestReadASCIIString(t *testing.T) {
	f := tempFileWithBytes(t, []byte("Hello"))
	r := NewReader(f)
	s, err := r.ReadASCIIString(5)
	if err != nil {
		t.Fatalf("ReadASCIIString: %v", err)
	}
	if s != "Hello" {
		t.Errorf("ReadASCIIString = %q, want %q", s, "Hello")
	}
}

func TestReadASCIIZString(t *testing.T) {
	f := tempFileWithBytes(t, []byte("test\x00extra"))
	r := NewReader(f)
	s, err := r.ReadASCIIZString()
	if err != nil {
		t.Fatalf("ReadASCIIZString: %v", err)
	}
	if s != "test" {
		t.Errorf("ReadASCIIZString = %q, want %q", s, "test")
	}
}

func TestReadWzIntSmall(t *testing.T) {
	// Value fits in a byte (not -128)
	f := tempFileWithBytes(t, []byte{42})
	r := NewReader(f)
	v, err := r.ReadWzInt()
	if err != nil {
		t.Fatalf("ReadWzInt: %v", err)
	}
	if v != 42 {
		t.Errorf("ReadWzInt = %d, want 42", v)
	}
}

func TestReadWzIntNegativeSmall(t *testing.T) {
	// Negative value in single byte: -5 = 0xFB
	f := tempFileWithBytes(t, []byte{0xFB})
	r := NewReader(f)
	v, err := r.ReadWzInt()
	if err != nil {
		t.Fatalf("ReadWzInt: %v", err)
	}
	if v != -5 {
		t.Errorf("ReadWzInt = %d, want -5", v)
	}
}

func TestReadWzIntLong(t *testing.T) {
	// Tag 0x80 (-128) followed by int32
	data := []byte{0x80, 0x00, 0x00, 0x00, 0x00}
	binary.LittleEndian.PutUint32(data[1:], uint32(300000))
	f := tempFileWithBytes(t, data)
	r := NewReader(f)
	v, err := r.ReadWzInt()
	if err != nil {
		t.Fatalf("ReadWzInt: %v", err)
	}
	if v != 300000 {
		t.Errorf("ReadWzInt = %d, want 300000", v)
	}
}

func TestReadWzLongSmall(t *testing.T) {
	f := tempFileWithBytes(t, []byte{10})
	r := NewReader(f)
	v, err := r.ReadWzLong()
	if err != nil {
		t.Fatalf("ReadWzLong: %v", err)
	}
	if v != 10 {
		t.Errorf("ReadWzLong = %d, want 10", v)
	}
}

func TestReadWzLongLong(t *testing.T) {
	data := make([]byte, 9)
	data[0] = 0x80
	binary.LittleEndian.PutUint64(data[1:], uint64(999999999999))
	f := tempFileWithBytes(t, data)
	r := NewReader(f)
	v, err := r.ReadWzLong()
	if err != nil {
		t.Fatalf("ReadWzLong: %v", err)
	}
	if v != 999999999999 {
		t.Errorf("ReadWzLong = %d, want 999999999999", v)
	}
}

func TestPosAndSeek(t *testing.T) {
	f := tempFileWithBytes(t, []byte{0x01, 0x02, 0x03, 0x04})
	r := NewReader(f)

	pos, err := r.Pos()
	if err != nil {
		t.Fatalf("Pos: %v", err)
	}
	if pos != 0 {
		t.Errorf("initial Pos = %d, want 0", pos)
	}

	_, _ = r.ReadByte()
	_, _ = r.ReadByte()

	pos, err = r.Pos()
	if err != nil {
		t.Fatalf("Pos: %v", err)
	}
	if pos != 2 {
		t.Errorf("Pos after 2 reads = %d, want 2", pos)
	}

	if _, err := r.Seek(0, 0); err != nil {
		t.Fatalf("Seek: %v", err)
	}
	pos, _ = r.Pos()
	if pos != 0 {
		t.Errorf("Pos after Seek(0) = %d, want 0", pos)
	}
}

func TestSkip(t *testing.T) {
	f := tempFileWithBytes(t, []byte{0x01, 0x02, 0x03, 0x04})
	r := NewReader(f)

	if err := r.Skip(2); err != nil {
		t.Fatalf("Skip: %v", err)
	}

	b, err := r.ReadByte()
	if err != nil {
		t.Fatalf("ReadByte after Skip: %v", err)
	}
	if b != 0x03 {
		t.Errorf("ReadByte after Skip(2) = 0x%02X, want 0x03", b)
	}
}

func TestRotateLeft32(t *testing.T) {
	// rotateLeft32(1, 1) should be 2
	if got := rotateLeft32(1, 1); got != 2 {
		t.Errorf("rotateLeft32(1, 1) = %d, want 2", got)
	}
	// rotateLeft32(0x80000000, 1) should be 1
	if got := rotateLeft32(0x80000000, 1); got != 1 {
		t.Errorf("rotateLeft32(0x80000000, 1) = %d, want 1", got)
	}
	// rotateLeft32 with 0 shift should be identity
	if got := rotateLeft32(0xDEADBEEF, 0); got != 0xDEADBEEF {
		t.Errorf("rotateLeft32(0xDEADBEEF, 0) = 0x%X, want 0xDEADBEEF", got)
	}
}
