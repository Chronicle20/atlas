package mapimage

import (
	"github.com/Chronicle20/atlas/libs/atlas-wz/canvas"
	"github.com/Chronicle20/atlas/libs/atlas-wz/wz"
	"github.com/Chronicle20/atlas/libs/atlas-wz/wz/property"
	"fmt"
	"image"
	"strconv"
	"strings"
)

// sprite is a decoded canvas, its origin anchor, its per-sprite z, and dims.
type sprite struct {
	img *image.NRGBA
	ox  int
	oy  int
	z   int
	w   int
	h   int
}

// decoder caches decoded canvases + mirrored variants for one render call.
type decoder struct {
	f        *wz.File
	cache    map[int64]*sprite
	flipped  map[int64]*image.NRGBA
}

func newDecoder(f *wz.File) *decoder {
	return &decoder{
		f:       f,
		cache:   map[int64]*sprite{},
		flipped: map[int64]*image.NRGBA{},
	}
}

// decode returns the decoded sprite, reading and caching once per canvas offset.
func (d *decoder) decode(cp *property.CanvasProperty) (*sprite, error) {
	if s, ok := d.cache[cp.DataOffset()]; ok {
		return s, nil
	}
	data, err := d.f.ReadCanvasData(cp.DataOffset(), cp.DataSize())
	if err != nil {
		return nil, fmt.Errorf("read canvas: %w", err)
	}
	nrgba, err := canvas.Decompress(data, cp.Width(), cp.Height(), cp.Format(), d.f.CanvasEncryptionKey())
	if err != nil {
		return nil, fmt.Errorf("decompress canvas: %w", err)
	}
	ox, oy := canvasOrigin(cp)
	s := &sprite{
		img: nrgba,
		ox:  ox,
		oy:  oy,
		z:   canvasZ(cp),
		w:   cp.Width(),
		h:   cp.Height(),
	}
	d.cache[cp.DataOffset()] = s
	return s, nil
}

// mirrored returns the X-flipped variant of a cached canvas.
func (d *decoder) mirrored(cp *property.CanvasProperty, src *image.NRGBA) *image.NRGBA {
	if m, ok := d.flipped[cp.DataOffset()]; ok {
		return m
	}
	m := mirrorX(src)
	d.flipped[cp.DataOffset()] = m
	return m
}

// resolveBackSprite returns the decoded Back.wz-equivalent sprite for (bS, no).
func (i *Index) resolveBackSprite(d *decoder, bS string, no int) (*sprite, *property.CanvasProperty, error) {
	img, ok := i.back[strings.ToLower(bS)]
	if !ok {
		return nil, nil, fmt.Errorf("back set %q not found", bS)
	}
	props, err := img.Properties()
	if err != nil {
		return nil, nil, fmt.Errorf("back %q properties: %w", bS, err)
	}
	backSub := findSub(props, "back")
	if backSub == nil {
		return nil, nil, fmt.Errorf("back/%s has no /back", bS)
	}
	cp := findCanvas(backSub.Children(), strconv.Itoa(no))
	if cp == nil {
		// animated — pick frame 0 or first canvas inside sub
		numSub := findSub(backSub.Children(), strconv.Itoa(no))
		if numSub != nil {
			cp = findCanvas(numSub.Children(), "0")
			if cp == nil {
				for _, c := range numSub.Children() {
					if c2, ok := c.(*property.CanvasProperty); ok {
						cp = c2
						break
					}
				}
			}
		}
	}
	if cp == nil {
		return nil, nil, fmt.Errorf("back/%s/back/%d not found", bS, no)
	}
	s, err := d.decode(cp)
	if err != nil {
		return nil, nil, err
	}
	return s, cp, nil
}

// resolveTileSprite returns the decoded Tile sprite for (tS, u, no).
func (i *Index) resolveTileSprite(d *decoder, tS, u, no string) (*sprite, *property.CanvasProperty, error) {
	img, ok := i.tile[strings.ToLower(tS)]
	if !ok {
		return nil, nil, fmt.Errorf("tile set %q not found", tS)
	}
	props, err := img.Properties()
	if err != nil {
		return nil, nil, fmt.Errorf("tile %q properties: %w", tS, err)
	}
	uSub := findSub(props, u)
	if uSub == nil {
		return nil, nil, fmt.Errorf("tile %s/%s not found", tS, u)
	}
	cp := findCanvas(uSub.Children(), no)
	if cp == nil {
		return nil, nil, fmt.Errorf("tile %s/%s/%s not found", tS, u, no)
	}
	s, err := d.decode(cp)
	if err != nil {
		return nil, nil, err
	}
	return s, cp, nil
}

// resolveObjSprite returns the decoded Obj sprite for (oS, l0, l1, l2) frame 0.
func (i *Index) resolveObjSprite(d *decoder, oS, l0, l1, l2 string) (*sprite, *property.CanvasProperty, error) {
	img, ok := i.obj[strings.ToLower(oS)]
	if !ok {
		return nil, nil, fmt.Errorf("obj set %q not found", oS)
	}
	props, err := img.Properties()
	if err != nil {
		return nil, nil, fmt.Errorf("obj %q properties: %w", oS, err)
	}
	l0s := findSub(props, l0)
	if l0s == nil {
		return nil, nil, fmt.Errorf("obj %s/%s not found", oS, l0)
	}
	l1s := findSub(l0s.Children(), l1)
	if l1s == nil {
		return nil, nil, fmt.Errorf("obj %s/%s/%s not found", oS, l0, l1)
	}
	l2s := findSub(l1s.Children(), l2)
	if l2s == nil {
		return nil, nil, fmt.Errorf("obj %s/%s/%s/%s not found", oS, l0, l1, l2)
	}
	cp := findCanvas(l2s.Children(), "0")
	if cp == nil {
		zero := findSub(l2s.Children(), "0")
		if zero != nil {
			cp = findCanvas(zero.Children(), "0")
		}
	}
	if cp == nil {
		return nil, nil, fmt.Errorf("obj %s/%s/%s/%s/0 not a canvas", oS, l0, l1, l2)
	}
	s, err := d.decode(cp)
	if err != nil {
		return nil, nil, err
	}
	return s, cp, nil
}
