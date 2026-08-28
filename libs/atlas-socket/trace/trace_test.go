package trace

import (
	"bytes"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
)

func TestDump(t *testing.T) {
	cases := []struct {
		name string
		in   []byte
		want string
	}{
		{
			name: "empty",
			in:   []byte{},
			want: "",
		},
		{
			name: "nil",
			in:   nil,
			want: "",
		},
		{
			name: "single byte",
			in:   []byte{0x41},
			want: "0000  41                                                |A|",
		},
		{
			name: "non printable",
			in:   []byte{0x00, 0x1f, 0x7f, 0xff},
			want: "0000  00 1f 7f ff                                       |....|",
		},
		{
			name: "fifteen bytes",
			in:   []byte{0x41, 0x42, 0x43, 0x44, 0x45, 0x46, 0x47, 0x48, 0x49, 0x4a, 0x4b, 0x4c, 0x4d, 0x4e, 0x4f},
			want: "0000  41 42 43 44 45 46 47 48  49 4a 4b 4c 4d 4e 4f     |ABCDEFGHIJKLMNO|",
		},
		{
			name: "sixteen aligned",
			in:   []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f},
			want: "0000  00 01 02 03 04 05 06 07  08 09 0a 0b 0c 0d 0e 0f  |................|",
		},
		{
			name: "two lines with short tail",
			in:   []byte{0x7d, 0x00, 0x01, 0x00, 0x00, 0x00, 0xff, 0xff, 0xff, 0xff, 0x01, 0x05, 0x00, 0x4d, 0x61, 0x70, 0x6c, 0x65, 0x00},
			want: "0000  7d 00 01 00 00 00 ff ff  ff ff 01 05 00 4d 61 70  |}............Map|\n0010  6c 65 00                                          |le.|",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Dump(c.in)
			if got != c.want {
				t.Fatalf("Dump(%v) =\n%q\nwant\n%q", c.in, got, c.want)
			}
		})
	}
}

func TestDump_LargePayloadIsNotTruncated(t *testing.T) {
	bs := bytes.Repeat([]byte{0x41}, 4100)
	got := Dump(bs)

	if n := strings.Count(got, "\n"); n != 256 {
		t.Fatalf("strings.Count(got, \"\\n\") = %d, want 256", n)
	}

	lines := strings.Split(got, "\n")
	last := lines[len(lines)-1]
	if !strings.HasPrefix(last, "1000  41 41 41 41") {
		t.Fatalf("last line = %q, want prefix %q", last, "1000  41 41 41 41")
	}

	if strings.Contains(got, "truncated") {
		t.Fatalf("got contains %q, want no truncation marker", "truncated")
	}
}

func TestFormat(t *testing.T) {
	op := uint16(0x007d)
	sid := uuid.MustParse("3f2a1c88-0000-4000-8000-000000000001")

	twoLineBody := []byte{0x7d, 0x00, 0x01, 0x00, 0x00, 0x00, 0xff, 0xff, 0xff, 0xff, 0x01, 0x05, 0x00, 0x4d, 0x61, 0x70, 0x6c, 0x65, 0x00}
	twoLineDump := "0000  7d 00 01 00 00 00 ff ff  ff ff 01 05 00 4d 61 70  |}............Map|\n0010  6c 65 00                                          |le.|"

	cases := []struct {
		name   string
		header Header
		body   []byte
		want   string
	}{
		{
			name:   "outbound with body",
			header: Header{Outbound, "CHARACTER_DATA", &op, 2, 19, sid},
			body:   twoLineBody,
			want:   "[PKT OUT] writer=CHARACTER_DATA op=0x007d len=19 session=3f2a1c88-0000-4000-8000-000000000001\n" + twoLineDump,
		},
		{
			name:   "inbound byte opcode",
			header: Header{Inbound, "USER_CHAT", &op, 1, 3, sid},
			body:   []byte{0x7d, 0x00, 0x01},
			want:   "[PKT IN ] handler=USER_CHAT op=0x7d len=3 session=3f2a1c88-0000-4000-8000-000000000001" + "\n0000  7d 00 01                                          |}..|",
		},
		{
			name:   "nil opcode",
			header: Header{Outbound, "<hello>", nil, 2, 0, sid},
			body:   []byte{},
			want:   "[PKT OUT] writer=<hello> op=n/a len=0 session=3f2a1c88-0000-4000-8000-000000000001",
		},
		{
			name:   "unresolved handler name",
			header: Header{Inbound, "<none>", &op, 2, 0, sid},
			body:   nil,
			want:   "[PKT IN ] handler=<none> op=0x007d len=0 session=3f2a1c88-0000-4000-8000-000000000001",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Format(c.header, c.body)
			if got != c.want {
				t.Fatalf("Format() =\n%q\nwant\n%q", got, c.want)
			}
		})
	}
}

func TestFormat_IsASingleString(t *testing.T) {
	op := uint16(0x007d)
	sid := uuid.MustParse("3f2a1c88-0000-4000-8000-000000000001")
	body := []byte{0x7d, 0x00, 0x01, 0x00, 0x00, 0x00, 0xff, 0xff, 0xff, 0xff, 0x01, 0x05, 0x00, 0x4d, 0x61, 0x70, 0x6c, 0x65, 0x00}

	got := Format(Header{Outbound, "CHARACTER_DATA", &op, 2, 19, sid}, body)

	if !strings.Contains(got, "\n") {
		t.Fatalf("Format() = %q, want it to contain a newline", got)
	}
	if strings.HasSuffix(got, "\n") {
		t.Fatalf("Format() = %q, want it not to end with a newline", got)
	}
}

type levellessLogger struct {
	logrus.FieldLogger
}

func TestEnabled(t *testing.T) {
	t.Run("flag off short circuits", func(t *testing.T) {
		l := logrus.New()
		l.SetLevel(logrus.DebugLevel)
		if got := Enabled(l, false); got != false {
			t.Fatalf("Enabled() = %v, want false", got)
		}
	})

	t.Run("flag on at info level", func(t *testing.T) {
		l := logrus.New()
		l.SetLevel(logrus.InfoLevel)
		if got := Enabled(l, true); got != false {
			t.Fatalf("Enabled() = %v, want false", got)
		}
	})

	t.Run("flag on at debug level", func(t *testing.T) {
		l := logrus.New()
		l.SetLevel(logrus.DebugLevel)
		if got := Enabled(l, true); got != true {
			t.Fatalf("Enabled() = %v, want true", got)
		}
	})

	t.Run("logger without level probe", func(t *testing.T) {
		null, _ := test.NewNullLogger()
		lvl := levellessLogger{FieldLogger: null}
		if got := Enabled(lvl, true); got != true {
			t.Fatalf("Enabled() = %v, want true", got)
		}
	})
}
