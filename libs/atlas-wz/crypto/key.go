package crypto

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
)

// AES key used by all WZ encryption variants (the key itself is constant).
var wzAESKey = []byte{
	0x13, 0x00, 0x00, 0x00,
	0x08, 0x00, 0x00, 0x00,
	0x06, 0x00, 0x00, 0x00,
	0xB4, 0x00, 0x00, 0x00,
	0x1B, 0x00, 0x00, 0x00,
	0x0F, 0x00, 0x00, 0x00,
	0x33, 0x00, 0x00, 0x00,
	0x52, 0x00, 0x00, 0x00,
}

// WzKey holds the expanded XOR key used for WZ string encryption/decryption.
type WzKey struct {
	key   []byte
	iv    []byte
	block cipher.Block
}

// NewWzKey creates a new WZ key from a 4-byte IV seed.
// The IV is repeated to 16 bytes and used with AES-ECB to generate the XOR table.
func NewWzKey(iv []byte) *WzKey {
	k := &WzKey{}

	// Check if IV is all zeros (no encryption)
	sum := 0
	for _, b := range iv {
		sum += int(b)
	}
	if sum == 0 {
		return k
	}

	// Repeat 4-byte IV to 16 bytes for AES block size
	k.iv = bytes.Repeat(iv, 4)

	block, err := aes.NewCipher(wzAESKey)
	if err != nil {
		return k
	}
	k.block = block

	// Pre-expand to a reasonable initial size
	k.expand(0x10000) // 65536 bytes, same as MapleLib's default

	return k
}

// expand grows the XOR key table to at least the given size.
func (k *WzKey) expand(size int) {
	if k.block == nil || size <= len(k.key) {
		return
	}

	needed := size - len(k.key)
	chunks := needed / 16
	if needed%16 > 0 {
		chunks++
	}

	expandSize := chunks * 16
	buf := make([]byte, expandSize)

	// Use the current IV state for expansion
	iv := make([]byte, 16)
	if len(k.key) == 0 {
		copy(iv, k.iv)
	} else {
		// Continue from where we left off
		copy(iv, k.key[len(k.key)-16:])
	}

	for i := 0; i < expandSize; i += 16 {
		k.block.Encrypt(buf[i:i+16], iv)
		copy(iv, buf[i:i+16])
	}

	k.key = append(k.key, buf...)
}

// Bytes returns the raw XOR key bytes, expanding if needed.
func (k *WzKey) Bytes(size int) []byte {
	k.expand(size)
	return k.key
}

// At returns the key byte at the given index.
func (k *WzKey) At(index int) byte {
	if k.block == nil || len(k.key) == 0 {
		return 0
	}
	k.expand(index + 1)
	return k.key[index]
}

// Len returns the current length of the expanded key.
func (k *WzKey) Len() int {
	return len(k.key)
}

// Transform XOR-transforms data in-place using the expanded key.
func (k *WzKey) Transform(data []byte) {
	if k.block == nil {
		return
	}
	k.expand(len(data))
	for i := range data {
		data[i] ^= k.key[i]
	}
}

// IsEmpty returns true if the key has no encryption (empty IV).
func (k *WzKey) IsEmpty() bool {
	return k.block == nil
}
