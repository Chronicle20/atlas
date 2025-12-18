package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"os"
)

const (
	encryptHeaderSize = 4
	blockSize         = 1460
)

var key = []byte{0x13, 0x00, 0x00, 0x00, 0x08, 0x00, 0x00, 0x00, 0x06, 0x00, 0x00, 0x00, 0xB4, 0x00, 0x00, 0x00, 0x1B, 0x00, 0x00, 0x00, 0x0F, 0x00, 0x00, 0x00, 0x33, 0x00, 0x00, 0x00, 0x52, 0x00, 0x00, 0x00}

type AESOFB struct {
	key         []byte
	iv          []byte
	ivGenerator IvGenerator
	version     uint16
	cipher      cipher.Block
}

func (a *AESOFB) IV() []byte {
	return a.iv
}

func PacketLength(encryptedHeader []byte) int {
	return int((uint16(encryptedHeader[0]) + uint16(encryptedHeader[1])*0x100) ^
		(uint16(encryptedHeader[2]) + uint16(encryptedHeader[3])*0x100))
}

type EncryptFunc func(input []byte) []byte

type IvGenerator func(input []byte) []byte

// DefaultIvGenerator the default iv generator. This is post 2008 IV generation and default for comment source versions.
func DefaultIvGenerator(input []byte) []byte {
	return multiplyBytes(input, 4, 4)
}

// FillIvZeroGenerator fills the iv with the first byte repeated. This is useful in GMS v12 era.
func FillIvZeroGenerator(input []byte) []byte {
	myIv := make([]byte, 16)
	for x := 0; x < 16; x++ {
		myIv[x] = input[0]
	}
	return myIv
}

func (a *AESOFB) Encrypt(maple bool, aes bool) func(input []byte) []byte {
	return func(input []byte) []byte {
		working := make([]byte, len(input))
		copy(working, input)

		a.generateHeader(working)

		if maple {
			a.mapleCrypt(working[encryptHeaderSize:])
		}

		if aes {
			a.aesCrypt(working[encryptHeaderSize:])
		}

		a.Shuffle()
		return working
	}
}

type DecryptFunc func(input []byte) []byte

func (a *AESOFB) Decrypt(aes bool, maple bool) func(input []byte) []byte {
	return func(input []byte) []byte {
		working := make([]byte, len(input))
		copy(working, input)

		if aes {
			a.aesCrypt(working)
		}

		if maple {
			a.mapleDecrypt(working)
		}

		a.Shuffle()
		return working
	}
}

func (a *AESOFB) aesCrypt(input []byte) {
	var pos, tpos, cbwrite, cb int32 = 0, 0, 0, int32(len(input))
	var first byte = 1

	cb = int32(len(input))

	// I'm not 100% sure what this exactly does but apparently maple
	// decrypts packets in blocks of 1460 bytes to work around
	// packets limitations or something

	for cb > pos {
		tpos = blockSize - int32(first*4)

		if cb > pos+tpos {
			cbwrite = tpos
		} else {
			cbwrite = cb - pos
		}

		myIv := a.ivGenerator(a.iv)
		stream := cipher.NewOFB(a.cipher, myIv[:])
		stream.XORKeyStream(input[pos:pos+cbwrite], input[pos:pos+cbwrite])

		pos += tpos

		if first == 1 {
			first = 0
		}
	}
}

func multiplyBytes(input []byte, count int, mul int) []byte {
	working := make([]byte, count*mul)

	for x := 0; x < count*mul; x++ {
		working[x] = input[x%count]
	}
	return working
}

// Taken from Kagami
func (a *AESOFB) mapleCrypt(input []byte) {
	var j int32
	var b, c byte

	for i := byte(0); i < 3; i++ {
		b = 0

		for j = int32(len(input)); j > 0; j-- {
			c = input[int32(len(input))-j]
			c = rol(c, 3)
			c = byte(int32(c) + j)
			c ^= b
			b = c
			c = ror(b, int(j))
			c ^= 0xFF
			c += 0x48
			input[int32(len(input))-j] = c
		}

		b = 0

		for j = int32(len(input)); j > 0; j-- {
			c = input[j-1]
			c = rol(c, 4)
			c = byte(int32(c) + j)
			c ^= b
			b = c
			c ^= 0x13
			c = ror(c, 3)
			input[j-1] = c
		}
	}
}

func (a *AESOFB) mapleDecrypt(input []byte) {
	var j int32
	var d, b, c byte

	for i := byte(0); i < 3; i++ {
		d = 0
		b = 0

		for j = int32(len(input)); j > 0; j-- {
			c = input[j-1]
			c = rol(c, 3)
			c ^= 0x13
			d = c
			c ^= b
			c = byte(int32(c) - j)
			c = ror(c, 4)
			b = d
			input[j-1] = c
		}

		d = 0
		b = 0

		for j = int32(len(input)); j > 0; j-- {
			c = input[int32(len(input))-j]
			c -= 0x48
			c ^= 0xFF
			c = rol(c, int(j))
			d = c
			c ^= b
			c = byte(int32(c) - j)
			c = ror(c, 3)
			b = d
			input[int32(len(input))-j] = c
		}
	}
}

func (a *AESOFB) generateHeader(input []byte) {
	bodyLength := int32(len(input[encryptHeaderSize:]))
	iiv := int32(a.iv[3] & 255)
	iiv |= int32(a.iv[2]) << 8 & '\uff00'
	iiv ^= int32(a.version)

	mLen := bodyLength<<8&'\uff00' | bodyLength>>8
	xoredIv := iiv ^ mLen
	input[0] = byte(iiv >> 8 & 255)
	input[1] = byte(iiv & 255)
	input[2] = byte(xoredIv >> 8 & 255)
	input[3] = byte(xoredIv & 255)
}

// Taken from Kagami
func ror(val byte, num int) byte {
	for i := 0; i < num; i++ {
		var lowbit int

		if val&1 > 0 {
			lowbit = 1
		} else {
			lowbit = 0
		}

		val >>= 1
		val |= byte(lowbit << 7)
	}

	return val
}

// Taken from Kagami
func rol(val byte, num int) byte {
	var highBit int

	for i := 0; i < num; i++ {
		if val&0x80 > 0 {
			highBit = 1
		} else {
			highBit = 0
		}

		val <<= 1
		val |= byte(highBit)
	}

	return val
}

func (a *AESOFB) Shuffle() {
	newIV := []byte{0xF2, 0x53, 0x50, 0xC6}

	for i := 0; i < 4; i++ {
		input := a.iv[i]
		shiftVal := ivShiftKey[input]

		newIV[0] += ivShiftKey[newIV[1]] - input
		newIV[1] -= newIV[2] ^ shiftVal
		newIV[2] ^= ivShiftKey[newIV[3]] + input
		newIV[3] -= newIV[0] - shiftVal

		val := uint32(newIV[3])<<24 | uint32(newIV[2])<<16 | uint32(newIV[1])<<8 | uint32(newIV[0])
		shift := val>>0x1D | val<<0x03

		newIV[0] = byte(shift & uint32(0xFF))
		newIV[1] = byte(shift >> 8 & uint32(0xFF))
		newIV[2] = byte(shift >> 16 & uint32(0xFF))
		newIV[3] = byte(shift >> 24 & uint32(0xFF))
	}

	copy(a.iv[:], newIV[:])
}

var ivShiftKey = [...]byte{
	0xEC, 0x3F, 0x77, 0xA4, 0x45, 0xD0, 0x71, 0xBF, 0xB7, 0x98, 0x20, 0xFC, 0x4B, 0xE9, 0xB3, 0xE1,
	0x5C, 0x22, 0xF7, 0x0C, 0x44, 0x1B, 0x81, 0xBD, 0x63, 0x8D, 0xD4, 0xC3, 0xF2, 0x10, 0x19, 0xE0,
	0xFB, 0xA1, 0x6E, 0x66, 0xEA, 0xAE, 0xD6, 0xCE, 0x06, 0x18, 0x4E, 0xEB, 0x78, 0x95, 0xDB, 0xBA,
	0xB6, 0x42, 0x7A, 0x2A, 0x83, 0x0B, 0x54, 0x67, 0x6D, 0xE8, 0x65, 0xE7, 0x2F, 0x07, 0xF3, 0xAA,
	0x27, 0x7B, 0x85, 0xB0, 0x26, 0xFD, 0x8B, 0xA9, 0xFA, 0xBE, 0xA8, 0xD7, 0xCB, 0xCC, 0x92, 0xDA,
	0xF9, 0x93, 0x60, 0x2D, 0xDD, 0xD2, 0xA2, 0x9B, 0x39, 0x5F, 0x82, 0x21, 0x4C, 0x69, 0xF8, 0x31,
	0x87, 0xEE, 0x8E, 0xAD, 0x8C, 0x6A, 0xBC, 0xB5, 0x6B, 0x59, 0x13, 0xF1, 0x04, 0x00, 0xF6, 0x5A,
	0x35, 0x79, 0x48, 0x8F, 0x15, 0xCD, 0x97, 0x57, 0x12, 0x3E, 0x37, 0xFF, 0x9D, 0x4F, 0x51, 0xF5,
	0xA3, 0x70, 0xBB, 0x14, 0x75, 0xC2, 0xB8, 0x72, 0xC0, 0xED, 0x7D, 0x68, 0xC9, 0x2E, 0x0D, 0x62,
	0x46, 0x17, 0x11, 0x4D, 0x6C, 0xC4, 0x7E, 0x53, 0xC1, 0x25, 0xC7, 0x9A, 0x1C, 0x88, 0x58, 0x2C,
	0x89, 0xDC, 0x02, 0x64, 0x40, 0x01, 0x5D, 0x38, 0xA5, 0xE2, 0xAF, 0x55, 0xD5, 0xEF, 0x1A, 0x7C,
	0xA7, 0x5B, 0xA6, 0x6F, 0x86, 0x9F, 0x73, 0xE6, 0x0A, 0xDE, 0x2B, 0x99, 0x4A, 0x47, 0x9C, 0xDF,
	0x09, 0x76, 0x9E, 0x30, 0x0E, 0xE4, 0xB2, 0x94, 0xA0, 0x3B, 0x34, 0x1D, 0x28, 0x0F, 0x36, 0xE3,
	0x23, 0xB4, 0x03, 0xD8, 0x90, 0xC8, 0x3C, 0xFE, 0x5E, 0x32, 0x24, 0x50, 0x1F, 0x3A, 0x43, 0x8A,
	0x96, 0x41, 0x74, 0xAC, 0x52, 0x33, 0xF0, 0xD9, 0x29, 0x80, 0xB1, 0x16, 0xD3, 0xAB, 0x91, 0xB9,
	0x84, 0x7F, 0x61, 0x1E, 0xCF, 0xC5, 0xD1, 0x56, 0x3D, 0xCA, 0xF4, 0x05, 0xC6, 0xE5, 0x08, 0x49}

type Configurator func(a *AESOFB)

func SetKey(nk []byte) Configurator {
	return func(a *AESOFB) {
		a.key = nk
	}
}

func SetIvGenerator(generator IvGenerator) Configurator {
	return func(a *AESOFB) {
		a.ivGenerator = generator
	}
}

//goland:noinspection GoUnusedExportedFunction
func NewAESOFB(iv []byte, version uint16, configurators ...Configurator) *AESOFB {
	a := &AESOFB{
		key:         key,
		iv:          iv,
		ivGenerator: DefaultIvGenerator,
		version:     version>>8&255 | version<<8&uint16(uint32('\uff00')),
	}

	for _, configure := range configurators {
		configure(a)
	}

	var err error
	a.cipher, err = aes.NewCipher(a.key)
	if err != nil {
		os.Exit(0)
	}

	return a
}
