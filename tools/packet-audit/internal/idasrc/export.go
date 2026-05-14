package idasrc

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
)

type exportFn struct {
	Address   string `json:"address"`
	Direction string `json:"direction"`
	Calls     []struct {
		Op      string `json:"op"`
		Comment string `json:"comment"`
		Guard   string `json:"guard,omitempty"`
	} `json:"calls"`
}

type exportFile struct {
	Binary      string              `json:"binary"`
	MD5         string              `json:"md5"`
	GeneratedAt string              `json:"generated_at"`
	Functions   map[string]exportFn `json:"functions"`
}

type ExportSource struct {
	file exportFile
}

func NewExportSource(path string) (*ExportSource, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var f exportFile
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, err
	}
	return &ExportSource{file: f}, nil
}

// Functions returns all FNames in the export.
func (s *ExportSource) Functions() []string {
	out := make([]string, 0, len(s.file.Functions))
	for k := range s.file.Functions {
		out = append(out, k)
	}
	return out
}

func (s *ExportSource) Resolve(_ context.Context, fname string) (Fields, error) {
	raw, ok := s.file.Functions[fname]
	if !ok {
		return Fields{}, fmt.Errorf("idasrc: function %q not in export", fname)
	}
	dir := DirClientbound
	if raw.Direction == "serverbound" {
		dir = DirServerbound
	}
	out := Fields{Function: fname, Address: raw.Address, Direction: dir}
	for i, c := range raw.Calls {
		op, err := parsePrim(c.Op)
		if err != nil {
			return Fields{}, fmt.Errorf("call %d (%s): %w", i, fname, err)
		}
		out.Calls = append(out.Calls, FieldCall{Op: op, Comment: c.Comment, Guard: c.Guard})
	}
	return out, nil
}

func parsePrim(s string) (Primitive, error) {
	switch s {
	case "Decode1", "Encode1":
		return Decode1, nil
	case "Decode2", "Encode2":
		return Decode2, nil
	case "Decode4", "Encode4":
		return Decode4, nil
	case "Decode8", "Encode8":
		return Decode8, nil
	case "DecodeStr", "EncodeStr":
		return DecodeStr, nil
	case "DecodeBuffer", "EncodeBuffer", "DecodeBuf", "EncodeBuf":
		return DecodeBuf, nil
	}
	return 0, fmt.Errorf("unknown primitive %q", s)
}
