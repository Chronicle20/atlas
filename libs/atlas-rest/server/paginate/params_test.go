package paginate

import (
	"errors"
	"net/url"
	"testing"
)

func q(kv ...string) url.Values {
	v := url.Values{}
	for i := 0; i+1 < len(kv); i += 2 {
		v.Set(kv[i], kv[i+1])
	}
	return v
}

func TestParseParams(t *testing.T) {
	cases := []struct {
		name       string
		query      url.Values
		wantNumber int
		wantSize   int
		wantErr    bool
	}{
		{"defaults", q(), 1, 50, false},
		{"explicit", q("page[number]", "3", "page[size]", "25"), 3, 25, false},
		{"size at max", q("page[size]", "250"), 1, 250, false},
		{"size over max", q("page[size]", "251"), 0, 0, true},
		{"size zero", q("page[size]", "0"), 0, 0, true},
		{"size negative", q("page[size]", "-5"), 0, 0, true},
		{"size non-integer", q("page[size]", "abc"), 0, 0, true},
		{"number zero", q("page[number]", "0"), 0, 0, true},
		{"number non-integer", q("page[number]", "x"), 0, 0, true},
		{"legacy limit rejected", q("limit", "10"), 0, 0, true},
		{"other params ignored", q("include", "skills", "page[number]", "2"), 2, 50, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p, err := ParseParams(c.query, 50, 250)
			if c.wantErr {
				if !errors.Is(err, ErrInvalidPageParam) {
					t.Fatalf("want ErrInvalidPageParam, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if p.Number != c.wantNumber || p.Size != c.wantSize {
				t.Fatalf("got %+v", p)
			}
		})
	}
}
