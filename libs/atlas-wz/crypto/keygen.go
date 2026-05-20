package crypto

import "fmt"

// Region IV seeds for WZ string encryption.
var (
	IvGMS   = []byte{0x4D, 0x23, 0xC7, 0x2B} // GMS (Global MapleStory)
	IvKMS   = []byte{0xB9, 0x7D, 0x63, 0xE9} // KMS/EMS (Korea/Europe)
	IvEmpty = []byte{0x00, 0x00, 0x00, 0x00} // No encryption (newer versions)
)

// EncryptionType identifies the WZ file encryption variant.
type EncryptionType int

const (
	EncryptionNone EncryptionType = iota
	EncryptionGMS
	EncryptionKMS
)

// GetIVForEncryption returns the IV bytes for the given encryption type.
func GetIVForEncryption(enc EncryptionType) []byte {
	switch enc {
	case EncryptionGMS:
		return IvGMS
	case EncryptionKMS:
		return IvKMS
	default:
		return IvEmpty
	}
}

// GetKeyForRegion returns the appropriate WZ key for the given encryption type.
func GetKeyForRegion(enc EncryptionType) *WzKey {
	return NewWzKey(GetIVForEncryption(enc))
}

// AllEncryptionTypes returns all encryption types to try during auto-detection.
func AllEncryptionTypes() []EncryptionType {
	return []EncryptionType{EncryptionGMS, EncryptionKMS, EncryptionNone}
}

// CalculateVersionHash computes the version hash from a game version number.
// Used to verify and decrypt WZ file offsets.
func CalculateVersionHash(version int) (encryptedVersion uint16, hash uint32) {
	s := []byte(fmt.Sprintf("%d", version))
	var h int
	for _, b := range s {
		h = (h << 5) + int(b) + 1
	}
	hash = uint32(h)

	ev := uint16(0xFF)
	for i := 0; i < 4; i++ {
		ev ^= uint16((h >> (i * 8)) & 0xFF)
	}
	encryptedVersion = ev
	return
}
