package idasrc

import (
	"context"
	"errors"
)

var ErrMCPUnavailable = errors.New("idasrc: MCP client not configured")

// MCPClient is the small surface MCPSource needs from a JSON-RPC client.
// Real implementation wires in atlas's MCP transport in a follow-up.
type MCPClient interface {
	GetFunctionByName(ctx context.Context, name string) (addr string, ok bool, err error)
	DecompileFunction(ctx context.Context, addr string) (text string, err error)
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
	calls, err := ParseDecompile(text)
	if err != nil {
		return Fields{}, err
	}
	return Fields{Function: fname, Address: addr, Calls: calls}, nil
}

type ErrFunctionNotFound struct{ Name string }

func (e ErrFunctionNotFound) Error() string { return "idasrc: function not found: " + e.Name }

// ParseDecompile is the lexical scanner that pulls CInPacket::DecodeN /
// COutPacket::EncodeN calls out of decompiled C text. Stub for now;
// implementation is part of the Phase-A follow-up that wires MCPSource end-to-end.
func ParseDecompile(_ string) ([]FieldCall, error) {
	return nil, errors.New("idasrc: ParseDecompile not yet implemented")
}
