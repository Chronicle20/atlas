package wz

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
)

// Reader provides WZ-specific binary reading operations.
type Reader struct {
	f     *os.File
	key   []byte
	order binary.ByteOrder
}

// NewReader creates a Reader from an open file.
func NewReader(f *os.File) *Reader {
	return &Reader{
		f:     f,
		order: binary.LittleEndian,
	}
}

// SetKey sets the encryption key for string decryption.
func (r *Reader) SetKey(key []byte) {
	r.key = key
}

// Key returns the current encryption key.
func (r *Reader) Key() []byte {
	return r.key
}

// Pos returns the current file position.
func (r *Reader) Pos() (int64, error) {
	return r.f.Seek(0, io.SeekCurrent)
}

// Seek sets the file position.
func (r *Reader) Seek(offset int64, whence int) (int64, error) {
	return r.f.Seek(offset, whence)
}

// Skip advances the reader by n bytes.
func (r *Reader) Skip(n int64) error {
	_, err := r.f.Seek(n, io.SeekCurrent)
	return err
}

// ReadByte reads a single byte.
func (r *Reader) ReadByte() (byte, error) {
	var b [1]byte
	_, err := io.ReadFull(r.f, b[:])
	return b[0], err
}

// ReadBytes reads n bytes.
func (r *Reader) ReadBytes(n int) ([]byte, error) {
	buf := make([]byte, n)
	_, err := io.ReadFull(r.f, buf)
	return buf, err
}

// ReadInt16 reads a little-endian int16.
func (r *Reader) ReadInt16() (int16, error) {
	var buf [2]byte
	if _, err := io.ReadFull(r.f, buf[:]); err != nil {
		return 0, err
	}
	return int16(r.order.Uint16(buf[:])), nil
}

// ReadUInt16 reads a little-endian uint16.
func (r *Reader) ReadUInt16() (uint16, error) {
	var buf [2]byte
	if _, err := io.ReadFull(r.f, buf[:]); err != nil {
		return 0, err
	}
	return r.order.Uint16(buf[:]), nil
}

// ReadInt32 reads a little-endian int32.
func (r *Reader) ReadInt32() (int32, error) {
	var buf [4]byte
	if _, err := io.ReadFull(r.f, buf[:]); err != nil {
		return 0, err
	}
	return int32(r.order.Uint32(buf[:])), nil
}

// ReadUInt32 reads a little-endian uint32.
func (r *Reader) ReadUInt32() (uint32, error) {
	var buf [4]byte
	if _, err := io.ReadFull(r.f, buf[:]); err != nil {
		return 0, err
	}
	return r.order.Uint32(buf[:]), nil
}

// ReadInt64 reads a little-endian int64.
func (r *Reader) ReadInt64() (int64, error) {
	var buf [8]byte
	if _, err := io.ReadFull(r.f, buf[:]); err != nil {
		return 0, err
	}
	return int64(r.order.Uint64(buf[:])), nil
}

// ReadFloat32 reads a little-endian float32.
func (r *Reader) ReadFloat32() (float32, error) {
	var buf [4]byte
	if _, err := io.ReadFull(r.f, buf[:]); err != nil {
		return 0, err
	}
	bits := r.order.Uint32(buf[:])
	return math.Float32frombits(bits), nil
}

// ReadFloat64 reads a little-endian float64.
func (r *Reader) ReadFloat64() (float64, error) {
	var buf [8]byte
	if _, err := io.ReadFull(r.f, buf[:]); err != nil {
		return 0, err
	}
	bits := r.order.Uint64(buf[:])
	return math.Float64frombits(bits), nil
}

// ReadASCIIString reads a fixed-length ASCII string.
func (r *Reader) ReadASCIIString(length int) (string, error) {
	buf, err := r.ReadBytes(length)
	if err != nil {
		return "", err
	}
	return string(buf), nil
}

// ReadASCIIZString reads a null-terminated ASCII string.
func (r *Reader) ReadASCIIZString() (string, error) {
	var result []byte
	for {
		b, err := r.ReadByte()
		if err != nil {
			return string(result), err
		}
		if b == 0 {
			break
		}
		result = append(result, b)
	}
	return string(result), nil
}

// ReadWzInt reads a WZ compressed integer.
// If the first byte is -128 (0x80), reads a full int32. Otherwise, the byte IS the value.
func (r *Reader) ReadWzInt() (int32, error) {
	b, err := r.ReadByte()
	if err != nil {
		return 0, err
	}
	if int8(b) == -128 {
		return r.ReadInt32()
	}
	return int32(int8(b)), nil
}

// ReadWzLong reads a WZ compressed long.
// If the first byte is -128 (0x80), reads a full int64. Otherwise, the byte IS the value.
func (r *Reader) ReadWzLong() (int64, error) {
	b, err := r.ReadByte()
	if err != nil {
		return 0, err
	}
	if int8(b) == -128 {
		return r.ReadInt64()
	}
	return int64(int8(b)), nil
}

// ReadWzOffset reads a WZ-encrypted offset using the version hash.
func (r *Reader) ReadWzOffset(fileStart uint32, versionHash uint32) (uint32, error) {
	pos, err := r.Pos()
	if err != nil {
		return 0, err
	}

	offset := uint32(pos) - fileStart
	offset = ^offset
	offset *= versionHash
	offset -= 0x581C3F6D
	offset = rotateLeft32(offset, byte(offset&0x1F))

	encryptedOffset, err := r.ReadUInt32()
	if err != nil {
		return 0, err
	}

	offset ^= encryptedOffset
	offset += fileStart * 2

	return offset, nil
}

// ReadWzString reads a WZ-encoded string (length-prefixed, possibly encrypted).
func (r *Reader) ReadWzString() (string, error) {
	tag, err := r.ReadByte()
	if err != nil {
		return "", err
	}

	switch int8(tag) {
	case 0:
		return "", nil
	case 127:
		// Unicode string (int32 length)
		return r.readWzUnicodeString()
	case -128:
		// ASCII string (negative length = long form)
		return r.readWzASCIIStringLong()
	default:
		if int8(tag) > 0 {
			// Unicode string with inline length
			return r.readWzUnicodeStringInline(int(int8(tag)))
		}
		// ASCII string with inline length
		return r.readWzASCIIStringInline(int(-int8(tag)))
	}
}

func (r *Reader) readWzUnicodeString() (string, error) {
	length, err := r.ReadInt32()
	if err != nil {
		return "", err
	}
	if length <= 0 {
		return "", nil
	}
	return r.readWzUnicodeStringInline(int(length))
}

func (r *Reader) readWzUnicodeStringInline(length int) (string, error) {
	buf, err := r.ReadBytes(length * 2)
	if err != nil {
		return "", err
	}

	// Decrypt unicode string
	mask := uint16(0xAAAA)
	chars := make([]uint16, length)
	for i := 0; i < length; i++ {
		c := r.order.Uint16(buf[i*2:])
		c ^= mask
		if r.key != nil && i*2+1 < len(r.key) {
			c ^= r.order.Uint16(r.key[i*2:])
		}
		chars[i] = c
		mask++
	}

	// Convert uint16 slice to string
	runes := make([]rune, length)
	for i, c := range chars {
		runes[i] = rune(c)
	}
	return string(runes), nil
}

func (r *Reader) readWzASCIIStringLong() (string, error) {
	length, err := r.ReadInt32()
	if err != nil {
		return "", err
	}
	if length <= 0 {
		return "", nil
	}
	return r.readWzASCIIStringInline(int(length))
}

func (r *Reader) readWzASCIIStringInline(length int) (string, error) {
	buf, err := r.ReadBytes(length)
	if err != nil {
		return "", err
	}

	// Decrypt ASCII string
	mask := byte(0xAA)
	for i := 0; i < length; i++ {
		buf[i] ^= mask
		if r.key != nil && i < len(r.key) {
			buf[i] ^= r.key[i]
		}
		mask++
	}
	return string(buf), nil
}

// ReadWzStringBlock reads a WZ string with possible offset reference.
// The tag byte determines how the string is stored.
func (r *Reader) ReadWzStringBlock(fileStart int64) (string, error) {
	tag, err := r.ReadByte()
	if err != nil {
		return "", err
	}

	switch tag {
	case 0x00, 0x73:
		// Inline string
		return r.ReadWzString()
	case 0x01, 0x1B:
		// String at offset
		offset, err := r.ReadInt32()
		if err != nil {
			return "", err
		}
		// Save position, seek to offset, read string, restore position
		pos, err := r.Pos()
		if err != nil {
			return "", err
		}
		if _, err := r.Seek(fileStart+int64(offset), io.SeekStart); err != nil {
			return "", err
		}
		s, err := r.ReadWzString()
		if err != nil {
			return "", err
		}
		if _, err := r.Seek(pos, io.SeekStart); err != nil {
			return "", err
		}
		return s, nil
	default:
		return "", fmt.Errorf("unknown string block tag: 0x%02X", tag)
	}
}

// Peek saves the current position, executes fn, then restores the position.
func (r *Reader) Peek(fn func() error) error {
	pos, err := r.Pos()
	if err != nil {
		return err
	}
	if err := fn(); err != nil {
		return err
	}
	_, err = r.Seek(pos, io.SeekStart)
	return err
}

// rotateLeft32 performs a bitwise left rotation on a uint32.
func rotateLeft32(value uint32, count byte) uint32 {
	n := uint(count) % 32
	return (value << n) | (value >> (32 - n))
}
