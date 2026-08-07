package idasrc

import (
	"context"
	"errors"
)

var ErrMCPUnavailable = errors.New("idasrc: MCP client not configured")

// Callee is a single callee entry returned by GetCallees.
type Callee struct {
	Name string
	Addr string
}

// StructField is one field within a StructLayout.
type StructField struct {
	Name   string
	Offset int
	Size   int // bytes
}

// StructLayout describes the memory layout of a named struct.
type StructLayout struct {
	Name   string
	Size   int
	Fields []StructField
}

// MCPClient is the small surface MCPSource needs from a JSON-RPC client.
// Real implementation wires in atlas's MCP transport in a follow-up.
type MCPClient interface {
	GetFunctionByName(ctx context.Context, name string) (addr string, ok bool, err error)
	DecompileFunction(ctx context.Context, addr string) (text string, err error)
	GetCallees(ctx context.Context, addr string) ([]Callee, error)
	StructInfo(ctx context.Context, name string) (StructLayout, error)
}

// BinaryInfoProvider is an OPTIONAL capability a MCPClient implementation may
// additionally support: reporting the underlying IDB's binary module name and
// md5, for the export's top-level "binary"/"md5" provenance fields (the
// evidence-freshness anchors the verification campaign keys on). It is
// intentionally NOT folded into MCPClient — that would force every existing
// unit-test fake across the package to grow a method it has no use for.
// exportRun type-asserts the client against this interface and leaves
// Binary/MD5 empty when the client doesn't implement it (unchanged prior
// behaviour for fakes).
type BinaryInfoProvider interface {
	GetBinaryInfo(ctx context.Context) (name, md5 string, err error)
}

type MCPSource struct {
	client MCPClient
}

func NewMCPSource(c MCPClient) *MCPSource { return &MCPSource{client: c} }

func (s *MCPSource) Resolve(ctx context.Context, fname string) (Fields, error) {
	if s.client == nil {
		return Fields{}, ErrMCPUnavailable
	}
	addr, ok, err := s.client.GetFunctionByName(ctx, fname)
	if err != nil {
		return Fields{}, err
	}
	if !ok {
		return Fields{}, ErrFunctionNotFound{Name: fname}
	}
	text, err := s.client.DecompileFunction(ctx, addr)
	if err != nil {
		return Fields{}, err
	}
	// --verify-export single-function path: not on the export critical path.
	// DirClientbound is an acceptable default here (the verify path re-parses one
	// named function in isolation; its packet direction is not threaded through
	// this entry point).
	calls, err := ParseDecompile(text, DirClientbound)
	if err != nil {
		return Fields{}, err
	}
	dir := "clientbound"
	ef := exportFile{Functions: map[string]exportFn{
		fname: {Address: addr, Direction: dir, Calls: calls},
	}}
	f, err := newExportSourceFromFile(ef).Resolve(ctx, fname)
	if err != nil {
		return Fields{}, err
	}
	return f, nil
}

type ErrFunctionNotFound struct{ Name string }

func (e ErrFunctionNotFound) Error() string { return "idasrc: function not found: " + e.Name }
