package crypto

import "encoding/binary"

// DecryptASCIIString decrypts a WZ ASCII string using the XOR key.
func DecryptASCIIString(data []byte, key *WzKey) string {
	if key == nil || key.IsEmpty() {
		return string(data)
	}
	result := make([]byte, len(data))
	copy(result, data)
	keyBytes := key.Bytes(len(data))
	for i := range result {
		result[i] ^= keyBytes[i]
	}
	return string(result)
}

// DecryptUnicodeString decrypts a WZ Unicode string using the XOR key.
func DecryptUnicodeString(data []byte, key *WzKey) string {
	if key == nil || key.IsEmpty() {
		return decodeUTF16LE(data)
	}
	result := make([]byte, len(data))
	copy(result, data)
	keyBytes := key.Bytes(len(data))
	for i := range result {
		result[i] ^= keyBytes[i]
	}
	return decodeUTF16LE(result)
}

func decodeUTF16LE(data []byte) string {
	runes := make([]rune, len(data)/2)
	for i := 0; i < len(data)/2; i++ {
		runes[i] = rune(binary.LittleEndian.Uint16(data[i*2:]))
	}
	return string(runes)
}
