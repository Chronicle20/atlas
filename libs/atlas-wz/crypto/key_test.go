package crypto

import (
	"bytes"
	"testing"
)

func TestNewWzKeyEmpty(t *testing.T) {
	k := NewWzKey([]byte{0x00, 0x00, 0x00, 0x00})
	if !k.IsEmpty() {
		t.Error("NewWzKey with zero IV should be empty")
	}
	if k.Len() != 0 {
		t.Errorf("Len() = %d, want 0", k.Len())
	}
	if k.At(0) != 0 {
		t.Errorf("At(0) = %d, want 0", k.At(0))
	}
}

func TestNewWzKeyGMS(t *testing.T) {
	iv := GetIVForEncryption(EncryptionGMS)
	k := NewWzKey(iv)
	if k.IsEmpty() {
		t.Error("NewWzKey with GMS IV should not be empty")
	}
	if k.Len() == 0 {
		t.Error("Len() should be > 0 after initialization")
	}
}

func TestNewWzKeyKMS(t *testing.T) {
	iv := GetIVForEncryption(EncryptionKMS)
	k := NewWzKey(iv)
	if k.IsEmpty() {
		t.Error("NewWzKey with KMS IV should not be empty")
	}
}

func TestWzKeyExpansion(t *testing.T) {
	iv := GetIVForEncryption(EncryptionGMS)
	k := NewWzKey(iv)

	b := k.Bytes(100)
	if len(b) < 100 {
		t.Errorf("Bytes(100) returned %d bytes, want >= 100", len(b))
	}

	// Deterministic: second call returns same data
	b2 := k.Bytes(100)
	if !bytes.Equal(b[:100], b2[:100]) {
		t.Error("Bytes() is not deterministic")
	}
}

func TestWzKeyAt(t *testing.T) {
	iv := GetIVForEncryption(EncryptionGMS)
	k := NewWzKey(iv)

	b := k.Bytes(1)
	if k.At(0) != b[0] {
		t.Errorf("At(0) = 0x%02X, want 0x%02X", k.At(0), b[0])
	}
}

func TestWzKeyTransformRoundTrip(t *testing.T) {
	iv := GetIVForEncryption(EncryptionGMS)
	k := NewWzKey(iv)

	original := []byte("Hello, WZ encryption!")
	data := make([]byte, len(original))
	copy(data, original)

	// Encrypt
	k.Transform(data)
	if bytes.Equal(data, original) {
		t.Error("Transform should change data")
	}

	// Decrypt (XOR is self-inverse)
	k.Transform(data)
	if !bytes.Equal(data, original) {
		t.Errorf("double Transform = %q, want %q", data, original)
	}
}

func TestWzKeyTransformEmpty(t *testing.T) {
	k := NewWzKey([]byte{0x00, 0x00, 0x00, 0x00})
	data := []byte("unchanged")
	original := make([]byte, len(data))
	copy(original, data)

	k.Transform(data)
	if !bytes.Equal(data, original) {
		t.Errorf("Transform with empty key changed data: %q", data)
	}
}

func TestDecryptASCIIStringNoKey(t *testing.T) {
	data := []byte("Hello")
	result := DecryptASCIIString(data, nil)
	if result != "Hello" {
		t.Errorf("DecryptASCIIString(nil key) = %q, want %q", result, "Hello")
	}
}

func TestDecryptASCIIStringEmptyKey(t *testing.T) {
	emptyKey := NewWzKey([]byte{0x00, 0x00, 0x00, 0x00})
	data := []byte("Hello")
	result := DecryptASCIIString(data, emptyKey)
	if result != "Hello" {
		t.Errorf("DecryptASCIIString(empty key) = %q, want %q", result, "Hello")
	}
}

func TestDecryptASCIIStringWithKey(t *testing.T) {
	iv := GetIVForEncryption(EncryptionGMS)
	k := NewWzKey(iv)

	original := []byte("MapleStory")
	encrypted := make([]byte, len(original))
	copy(encrypted, original)
	keyBytes := k.Bytes(len(encrypted))
	for i := range encrypted {
		encrypted[i] ^= keyBytes[i]
	}

	result := DecryptASCIIString(encrypted, k)
	if result != "MapleStory" {
		t.Errorf("DecryptASCIIString = %q, want %q", result, "MapleStory")
	}
}

func TestDecryptUnicodeStringNoKey(t *testing.T) {
	// "Hi" in UTF-16LE: H=0x0048, i=0x0069
	data := []byte{0x48, 0x00, 0x69, 0x00}
	result := DecryptUnicodeString(data, nil)
	if result != "Hi" {
		t.Errorf("DecryptUnicodeString(nil key) = %q, want %q", result, "Hi")
	}
}

func TestDecodeUTF16LE(t *testing.T) {
	tests := []struct {
		data []byte
		want string
	}{
		{[]byte{0x48, 0x00, 0x65, 0x00, 0x6C, 0x00, 0x6C, 0x00, 0x6F, 0x00}, "Hello"},
		{[]byte{}, ""},
		{[]byte{0x41, 0x00}, "A"},
	}
	for _, tt := range tests {
		got := decodeUTF16LE(tt.data)
		if got != tt.want {
			t.Errorf("decodeUTF16LE(%v) = %q, want %q", tt.data, got, tt.want)
		}
	}
}
