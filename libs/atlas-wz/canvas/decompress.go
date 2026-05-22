package canvas

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"image"
	"image/color"
	"io"
)

const (
	FormatBGRA4444    = 1
	FormatBGRA8888    = 2
	FormatDXT3Gray    = 3
	FormatARGB1555    = 257
	FormatBGR565      = 513
	FormatBlockRGB565 = 517
	FormatDXT3        = 1026
	FormatDXT5        = 2050
)

// Decompress decompresses WZ canvas data into a Go image.
// The encryptionKey is used for listWz-format encrypted blocks (may be nil for standard zlib).
func Decompress(data []byte, width, height int, format int, encryptionKey []byte) (*image.NRGBA, error) {
	if len(data) == 0 {
		return image.NewNRGBA(image.Rect(0, 0, width, height)), nil
	}

	pixels, err := decompressCanvasData(data, encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("canvas decompression failed (dataLen=%d, format=%d, %dx%d): %w",
			len(data), format, width, height, err)
	}

	return decodePixels(pixels, width, height, format)
}

// decompressCanvasData handles both standard zlib and listWz encrypted block formats.
// Tries multiple strategies in order:
// 1. Standard zlib (if data starts with a zlib header)
// 2. ListWz encrypted blocks with XOR key
// 3. ListWz blocks without XOR (unencrypted blocks)
func decompressCanvasData(data []byte, encryptionKey []byte) ([]byte, error) {
	if len(data) < 2 {
		return nil, fmt.Errorf("data too short: %d bytes", len(data))
	}

	// Strategy 1: Check for standard zlib header
	if isZlibHeader(data[0], data[1]) {
		result, err := decompressZlib(data)
		if err == nil {
			return result, nil
		}
	}

	// Strategy 2: ListWz encrypted block format with XOR key
	if encryptionKey != nil && len(encryptionKey) > 0 {
		result, err := decompressBlocks(data, encryptionKey)
		if err == nil {
			return result, nil
		}
	}

	// Strategy 3: Block format without XOR encryption
	result, err := decompressBlocks(data, nil)
	if err == nil {
		return result, nil
	}

	return nil, fmt.Errorf("all decompression strategies failed (first4bytes=%X)", firstN(data, 4))
}

func isZlibHeader(b0, b1 byte) bool {
	// Zlib streams start with CMF=0x78 (deflate, 32K window) followed by FLG byte.
	return b0 == 0x78 && (b1 == 0x9C || b1 == 0xDA || b1 == 0x01 || b1 == 0x5E)
}

// decompressZlib decompresses a zlib stream, tolerating truncated streams.
// WZ canvas data often uses truncated zlib streams (missing the 4-byte Adler32 checksum),
// which causes Go's zlib.Reader to return io.ErrUnexpectedEOF even though all pixel data
// was successfully decompressed. This function reads in chunks and treats ErrUnexpectedEOF
// as success, matching the behavior of other WZ libraries (MapleLib, wzexplorer).
func decompressZlib(data []byte) ([]byte, error) {
	r, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer r.Close()

	var buf bytes.Buffer
	chunk := make([]byte, 4096)
	for {
		n, readErr := r.Read(chunk)
		if n > 0 {
			buf.Write(chunk[:n])
		}
		if readErr != nil {
			if readErr == io.EOF || readErr == io.ErrUnexpectedEOF {
				break
			}
			// If we already have data, return it despite the error
			if buf.Len() > 0 {
				break
			}
			return nil, readErr
		}
	}
	if buf.Len() == 0 {
		return nil, fmt.Errorf("zlib decompression produced no data")
	}
	return buf.Bytes(), nil
}

// decompressBlocks handles WZ listWz canvas block format.
// Each block: [uint32 blockSize][blockSize bytes optionally XOR'd with WzKey]
// After optional decryption, all blocks are concatenated and decompressed as a single zlib stream.
func decompressBlocks(data []byte, encryptionKey []byte) ([]byte, error) {
	var decrypted []byte
	offset := 0

	for offset < len(data) {
		if offset+4 > len(data) {
			break
		}

		blockSize := int(data[offset]) | int(data[offset+1])<<8 | int(data[offset+2])<<16 | int(data[offset+3])<<24
		offset += 4

		if blockSize <= 0 || blockSize > 0x100000 || offset+blockSize > len(data) {
			break
		}

		block := make([]byte, blockSize)
		copy(block, data[offset:offset+blockSize])
		if encryptionKey != nil {
			for i := range block {
				if i < len(encryptionKey) {
					block[i] ^= encryptionKey[i]
				}
			}
		}
		decrypted = append(decrypted, block...)
		offset += blockSize
	}

	if len(decrypted) == 0 {
		return nil, fmt.Errorf("no valid blocks found")
	}

	return decompressZlib(decrypted)
}

func firstN(data []byte, n int) []byte {
	if len(data) < n {
		return data
	}
	return data[:n]
}

func decodePixels(data []byte, width, height int, format int) (*image.NRGBA, error) {
	img := image.NewNRGBA(image.Rect(0, 0, width, height))

	switch format {
	case FormatBGRA4444:
		decodeBGRA4444(data, img)
	case FormatBGRA8888:
		decodeBGRA8888(data, img)
	case FormatDXT3Gray:
		decodeDXT3(data, width, height, img)
	case FormatARGB1555:
		decodeARGB1555(data, img)
	case FormatBGR565:
		decodeBGR565(data, img)
	case FormatBlockRGB565:
		decodeBlockRGB565(data, img)
	case FormatDXT3:
		decodeDXT3(data, width, height, img)
	case FormatDXT5:
		decodeDXT5(data, width, height, img)
	default:
		// Unknown format - try BGRA8888 as fallback
		decodeBGRA8888(data, img)
	}

	return img, nil
}

// decodeBGRA4444 decodes 16-bit BGRA (4 bits per channel).
func decodeBGRA4444(data []byte, img *image.NRGBA) {
	w := img.Bounds().Dx()
	h := img.Bounds().Dy()

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			idx := (y*w + x) * 2
			if idx+1 >= len(data) {
				return
			}
			b1 := data[idx]
			b2 := data[idx+1]

			b := expand4to8(b1 & 0x0F)
			g := expand4to8(b1 >> 4)
			r := expand4to8(b2 & 0x0F)
			a := expand4to8(b2 >> 4)

			img.SetNRGBA(x, y, color.NRGBA{R: r, G: g, B: b, A: a})
		}
	}
}

// decodeBGRA8888 decodes 32-bit BGRA (8 bits per channel).
func decodeBGRA8888(data []byte, img *image.NRGBA) {
	w := img.Bounds().Dx()
	h := img.Bounds().Dy()

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			idx := (y*w + x) * 4
			if idx+3 >= len(data) {
				return
			}
			b := data[idx]
			g := data[idx+1]
			r := data[idx+2]
			a := data[idx+3]

			img.SetNRGBA(x, y, color.NRGBA{R: r, G: g, B: b, A: a})
		}
	}
}

// decodeBGR565 decodes 16-bit BGR (5-6-5 bits).
func decodeBGR565(data []byte, img *image.NRGBA) {
	w := img.Bounds().Dx()
	h := img.Bounds().Dy()

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			idx := (y*w + x) * 2
			if idx+1 >= len(data) {
				return
			}
			pixel := uint16(data[idx]) | uint16(data[idx+1])<<8

			b := byte((pixel & 0x001F) << 3)
			g := byte(((pixel & 0x07E0) >> 5) << 2)
			r := byte(((pixel & 0xF800) >> 11) << 3)

			img.SetNRGBA(x, y, color.NRGBA{R: r, G: g, B: b, A: 255})
		}
	}
}

// decodeARGB1555 decodes 16-bit ARGB (1-5-5-5 bits).
func decodeARGB1555(data []byte, img *image.NRGBA) {
	w := img.Bounds().Dx()
	h := img.Bounds().Dy()

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			idx := (y*w + x) * 2
			if idx+1 >= len(data) {
				return
			}
			pixel := uint16(data[idx]) | uint16(data[idx+1])<<8

			a := byte(0)
			if pixel&0x8000 != 0 {
				a = 255
			}
			r := byte(((pixel >> 10) & 0x1F) * 255 / 31)
			g := byte(((pixel >> 5) & 0x1F) * 255 / 31)
			b := byte((pixel & 0x1F) * 255 / 31)

			img.SetNRGBA(x, y, color.NRGBA{R: r, G: g, B: b, A: a})
		}
	}
}

// decodeBlockRGB565 decodes block-compressed RGB565 (format 517).
// Each 16x16 block of pixels shares a single RGB565 color value.
func decodeBlockRGB565(data []byte, img *image.NRGBA) {
	w := img.Bounds().Dx()
	h := img.Bounds().Dy()

	bw := (w + 15) / 16
	bh := (h + 15) / 16

	dataIdx := 0
	for by := 0; by < bh; by++ {
		for bx := 0; bx < bw; bx++ {
			if dataIdx+1 >= len(data) {
				return
			}
			pixel := uint16(data[dataIdx]) | uint16(data[dataIdx+1])<<8
			dataIdx += 2

			b := byte((pixel & 0x001F) * 255 / 31)
			g := byte(((pixel >> 5) & 0x3F) * 255 / 63)
			r := byte(((pixel >> 11) & 0x1F) * 255 / 31)

			for py := 0; py < 16 && by*16+py < h; py++ {
				for px := 0; px < 16 && bx*16+px < w; px++ {
					img.SetNRGBA(bx*16+px, by*16+py, color.NRGBA{R: r, G: g, B: b, A: 255})
				}
			}
		}
	}
}

// decodeDXT3 decodes a DXT3 (BC2) compressed texture.
func decodeDXT3(data []byte, width, height int, img *image.NRGBA) {
	bw := (width + 3) / 4
	bh := (height + 3) / 4

	for by := 0; by < bh; by++ {
		for bx := 0; bx < bw; bx++ {
			blockIdx := (by*bw + bx) * 16
			if blockIdx+16 > len(data) {
				return
			}
			block := data[blockIdx : blockIdx+16]

			// First 8 bytes: explicit alpha (4 bits per pixel, 16 pixels)
			// Next 8 bytes: DXT1 color block
			decodeDXT3Block(block, bx*4, by*4, width, height, img)
		}
	}
}

// decodeDXT5 decodes a DXT5 (BC3) compressed texture.
func decodeDXT5(data []byte, width, height int, img *image.NRGBA) {
	bw := (width + 3) / 4
	bh := (height + 3) / 4

	for by := 0; by < bh; by++ {
		for bx := 0; bx < bw; bx++ {
			blockIdx := (by*bw + bx) * 16
			if blockIdx+16 > len(data) {
				return
			}
			block := data[blockIdx : blockIdx+16]

			// First 8 bytes: interpolated alpha
			// Next 8 bytes: DXT1 color block
			decodeDXT5Block(block, bx*4, by*4, width, height, img)
		}
	}
}

func decodeDXT3Block(block []byte, startX, startY, width, height int, img *image.NRGBA) {
	// Decode alpha (explicit 4-bit per pixel)
	var alphas [16]byte
	for i := 0; i < 16; i++ {
		byteIdx := i / 2
		if i%2 == 0 {
			alphas[i] = expand4to8(block[byteIdx] & 0x0F)
		} else {
			alphas[i] = expand4to8(block[byteIdx] >> 4)
		}
	}

	// Decode colors (DXT1 block at offset 8)
	colors := decodeDXT1Colors(block[8:])

	// Read color indices
	for y := 0; y < 4; y++ {
		if startY+y >= height {
			break
		}
		row := block[12+y]
		for x := 0; x < 4; x++ {
			if startX+x >= width {
				continue
			}
			idx := (row >> (uint(x) * 2)) & 0x03
			c := colors[idx]
			pixIdx := y*4 + x
			img.SetNRGBA(startX+x, startY+y, color.NRGBA{R: c.R, G: c.G, B: c.B, A: alphas[pixIdx]})
		}
	}
}

func decodeDXT5Block(block []byte, startX, startY, width, height int, img *image.NRGBA) {
	// Decode interpolated alpha
	alpha0 := block[0]
	alpha1 := block[1]

	var alphaTable [8]byte
	alphaTable[0] = alpha0
	alphaTable[1] = alpha1
	if alpha0 > alpha1 {
		alphaTable[2] = byte((6*uint16(alpha0) + 1*uint16(alpha1) + 3) / 7)
		alphaTable[3] = byte((5*uint16(alpha0) + 2*uint16(alpha1) + 3) / 7)
		alphaTable[4] = byte((4*uint16(alpha0) + 3*uint16(alpha1) + 3) / 7)
		alphaTable[5] = byte((3*uint16(alpha0) + 4*uint16(alpha1) + 3) / 7)
		alphaTable[6] = byte((2*uint16(alpha0) + 5*uint16(alpha1) + 3) / 7)
		alphaTable[7] = byte((1*uint16(alpha0) + 6*uint16(alpha1) + 3) / 7)
	} else {
		alphaTable[2] = byte((4*uint16(alpha0) + 1*uint16(alpha1) + 2) / 5)
		alphaTable[3] = byte((3*uint16(alpha0) + 2*uint16(alpha1) + 2) / 5)
		alphaTable[4] = byte((2*uint16(alpha0) + 3*uint16(alpha1) + 2) / 5)
		alphaTable[5] = byte((1*uint16(alpha0) + 4*uint16(alpha1) + 2) / 5)
		alphaTable[6] = 0
		alphaTable[7] = 255
	}

	// Read 48 bits of alpha indices (3 bits per pixel, 16 pixels)
	alphaBits := uint64(block[2]) | uint64(block[3])<<8 | uint64(block[4])<<16 |
		uint64(block[5])<<24 | uint64(block[6])<<32 | uint64(block[7])<<40

	var alphas [16]byte
	for i := 0; i < 16; i++ {
		alphaIdx := (alphaBits >> (uint(i) * 3)) & 0x07
		alphas[i] = alphaTable[alphaIdx]
	}

	// Decode colors (DXT1 block at offset 8)
	colors := decodeDXT1Colors(block[8:])

	// Read color indices
	for y := 0; y < 4; y++ {
		if startY+y >= height {
			break
		}
		row := block[12+y]
		for x := 0; x < 4; x++ {
			if startX+x >= width {
				continue
			}
			idx := (row >> (uint(x) * 2)) & 0x03
			c := colors[idx]
			pixIdx := y*4 + x
			img.SetNRGBA(startX+x, startY+y, color.NRGBA{R: c.R, G: c.G, B: c.B, A: alphas[pixIdx]})
		}
	}
}

// decodeDXT1Colors decodes the 4-color palette from a DXT1 color block (4 bytes of colors + 4 bytes of indices).
func decodeDXT1Colors(block []byte) [4]color.NRGBA {
	c0 := uint16(block[0]) | uint16(block[1])<<8
	c1 := uint16(block[2]) | uint16(block[3])<<8

	var colors [4]color.NRGBA
	colors[0] = rgb565ToNRGBA(c0)
	colors[1] = rgb565ToNRGBA(c1)

	if c0 > c1 {
		colors[2] = lerpColor(colors[0], colors[1], 1, 2)
		colors[3] = lerpColor(colors[0], colors[1], 2, 2)
	} else {
		colors[2] = lerpColor(colors[0], colors[1], 1, 1)
		colors[3] = color.NRGBA{R: 0, G: 0, B: 0, A: 0}
	}

	return colors
}

func rgb565ToNRGBA(c uint16) color.NRGBA {
	r := byte(((c >> 11) & 0x1F) * 255 / 31)
	g := byte(((c >> 5) & 0x3F) * 255 / 63)
	b := byte((c & 0x1F) * 255 / 31)
	return color.NRGBA{R: r, G: g, B: b, A: 255}
}

func lerpColor(c0, c1 color.NRGBA, num, denom int) color.NRGBA {
	total := denom + 1
	return color.NRGBA{
		R: byte((int(c0.R)*(total-num) + int(c1.R)*num) / total),
		G: byte((int(c0.G)*(total-num) + int(c1.G)*num) / total),
		B: byte((int(c0.B)*(total-num) + int(c1.B)*num) / total),
		A: 255,
	}
}

func expand4to8(v byte) byte {
	return v | (v << 4)
}
