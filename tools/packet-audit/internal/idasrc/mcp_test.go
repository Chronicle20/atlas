package idasrc

import (
	"context"
	"errors"
	"testing"
)

func TestMCPSourceWithoutClient(t *testing.T) {
	src := NewMCPSource(nil)
	_, err := src.Resolve(context.Background(), "any")
	if !errors.Is(err, ErrMCPUnavailable) {
		t.Errorf("expected ErrMCPUnavailable, got %v", err)
	}
}

type fakeClient struct {
	addrs     map[string]string       // name -> addr
	decomp    map[string]string       // addr -> text
	decompErr map[string]error        // addr -> error (optional; nil-safe)
	callees   map[string][]Callee     // addr -> callees
	structs   map[string]StructLayout // name -> layout
}

func (f *fakeClient) GetFunctionByName(_ context.Context, n string) (string, bool, error) {
	a, ok := f.addrs[n]
	return a, ok, nil
}
func (f *fakeClient) DecompileFunction(_ context.Context, a string) (string, error) {
	if e := f.decompErr[a]; e != nil {
		return "", e
	}
	return f.decomp[a], nil
}
func (f *fakeClient) GetCallees(_ context.Context, a string) ([]Callee, error) {
	return f.callees[a], nil
}
func (f *fakeClient) StructInfo(_ context.Context, n string) (StructLayout, error) {
	return f.structs[n], nil
}

func TestFakeClientSatisfiesInterface(t *testing.T) {
	var _ MCPClient = (*fakeClient)(nil)
}
