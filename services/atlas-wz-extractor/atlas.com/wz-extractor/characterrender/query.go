package characterrender

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// RenderQuery is the parsed query component of a render request.
type RenderQuery struct {
	Skin   int
	Hair   int
	Face   int
	Stance string
	Frame  int
	Resize int
	Items  []int
}

// ParseRenderQuery extracts and validates the documented query params. It
// applies defaults: stance=stand1, frame=0, resize=2.
func ParseRenderQuery(q url.Values) (RenderQuery, error) {
	skin, err := requiredInt(q, "skin")
	if err != nil {
		return RenderQuery{}, err
	}
	hair, err := requiredInt(q, "hair")
	if err != nil {
		return RenderQuery{}, err
	}
	face, err := requiredInt(q, "face")
	if err != nil {
		return RenderQuery{}, err
	}
	stance := q.Get("stance")
	if stance == "" {
		stance = "stand1"
	}
	frame := 0
	if v := q.Get("frame"); v != "" {
		f, err := strconv.Atoi(v)
		if err != nil || f < 0 {
			return RenderQuery{}, fmt.Errorf("invalid frame %q", v)
		}
		frame = f
	}
	resize := 2
	if v := q.Get("resize"); v != "" {
		r, err := strconv.Atoi(v)
		if err != nil || r < 1 || r > 4 {
			return RenderQuery{}, fmt.Errorf("invalid resize %q", v)
		}
		resize = r
	}
	items, err := parseItemsCSV(q.Get("items"))
	if err != nil {
		return RenderQuery{}, err
	}
	return RenderQuery{
		Skin: skin, Hair: hair, Face: face,
		Stance: stance, Frame: frame, Resize: resize, Items: items,
	}, nil
}

func requiredInt(q url.Values, name string) (int, error) {
	v := q.Get(name)
	if v == "" {
		return 0, fmt.Errorf("missing %s", name)
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("invalid %s %q", name, v)
	}
	return n, nil
}

func parseItemsCSV(s string) ([]int, error) {
	if s == "" {
		return nil, nil
	}
	out := []int{}
	for _, tok := range strings.Split(s, ",") {
		tok = strings.TrimSpace(tok)
		if tok == "" {
			continue
		}
		n, err := strconv.Atoi(tok)
		if err != nil {
			return nil, fmt.Errorf("invalid items entry %q", tok)
		}
		out = append(out, n)
	}
	return out, nil
}
