package idasrc

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

type rtFunc func(*http.Request) (*http.Response, error)

func (f rtFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func jsonResp(v any) *http.Response {
	b, _ := json.Marshal(v)
	return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(b)),
		Header: http.Header{"Content-Type": []string{"application/json"}}}
}

// structuredResp builds a NEW-server tools/call response carrying the given
// structuredContent payload (already-parsed JSON) plus a matching text part and
// isError:false — mirroring the live envelope. structured may be any JSON-able
// value (object, array, …).
func structuredResp(structured any) *http.Response {
	text, _ := json.Marshal(structured)
	return jsonResp(map[string]any{
		"jsonrpc": "2.0", "id": 1,
		"result": map[string]any{
			"content":           []map[string]any{{"type": "text", "text": string(text)}},
			"structuredContent": structured,
			"isError":           false,
		},
	})
}

// initResp builds an initialize response carrying the Mcp-Session-Id header the
// real server returns. Subsequent requests must replay that id.
func initResp(sessionID string) *http.Response {
	b, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0", "id": 1,
		"result": map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]any{},
			"serverInfo":      map[string]any{"name": "ida-pro-mcp", "version": "1"},
		},
	})
	return &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewReader(b)),
		Header: http.Header{
			"Content-Type":   []string{"application/json"},
			"Mcp-Session-Id": []string{sessionID},
		}}
}

// notInitErrResp mimics the real server, which does NOT implement
// notifications/initialized and answers it with a JSON-RPC -32601. The client
// must treat that as harmless (it's a notification).
func notInitErrResp() *http.Response {
	return jsonResp(map[string]any{
		"jsonrpc": "2.0", "id": nil,
		"error": map[string]any{"code": -32601, "message": "Method not found: notifications/initialized"},
	})
}

// handshakeOK answers the two handshake methods so a test can focus on the
// tools/call; it returns false for any other method so the caller can supply
// the tool response.
func handshakeOK(method string) (*http.Response, bool) {
	switch method {
	case "initialize":
		return initResp("sid"), true
	case "notifications/initialized":
		return notInitErrResp(), true
	}
	return nil, false
}

func readMethodAndArgs(r *http.Request) (method string, name string, args json.RawMessage) {
	var req struct {
		Method string `json:"method"`
		Params struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		} `json:"params"`
	}
	body, _ := io.ReadAll(r.Body)
	_ = json.Unmarshal(body, &req)
	return req.Method, req.Params.Name, req.Params.Arguments
}

// TestMCPHTTPLookupFuncsFound asserts the NEW lookup_funcs found shape maps to
// (addr, true, nil) and is invoked via tools/call with queries:[name].
func TestMCPHTTPLookupFuncsFound(t *testing.T) {
	var gotTool string
	var gotQueries []string
	rt := rtFunc(func(r *http.Request) (*http.Response, error) {
		method, name, args := readMethodAndArgs(r)
		if resp, ok := handshakeOK(method); ok {
			return resp, nil
		}
		gotTool = name
		var a struct {
			Queries []string `json:"queries"`
		}
		_ = json.Unmarshal(args, &a)
		gotQueries = a.Queries
		return structuredResp(map[string]any{
			"result": []map[string]any{{
				"query": "0xa3f2e8",
				"fn":    map[string]any{"addr": "0xa3f2e8", "name": "?OnFriendResult@CWvsContext@@QAEXAAVCInPacket@@@Z", "size": "0x3d5"},
				"error": nil,
			}},
		}), nil
	})
	c := NewMCPHTTPClient("http://test/mcp", &http.Client{Transport: rt})
	addr, ok, err := c.GetFunctionByName(context.Background(), "?OnFriendResult@CWvsContext@@QAEXAAVCInPacket@@@Z")
	if err != nil || !ok {
		t.Fatalf("GetFunctionByName err=%v ok=%v", err, ok)
	}
	if addr != "0xa3f2e8" {
		t.Errorf("addr = %q, want 0xa3f2e8", addr)
	}
	if gotTool != "lookup_funcs" {
		t.Errorf("tool = %q, want lookup_funcs", gotTool)
	}
	if len(gotQueries) != 1 || gotQueries[0] != "?OnFriendResult@CWvsContext@@QAEXAAVCInPacket@@@Z" {
		t.Errorf("queries = %v, want [<name>]", gotQueries)
	}
}

// TestMCPHTTPLookupFuncsNotFound asserts the {fn:null,error:"Not found"} shape
// maps to ("", false, nil) — a per-function soft miss that must NOT abort an
// export.
func TestMCPHTTPLookupFuncsNotFound(t *testing.T) {
	rt := rtFunc(func(r *http.Request) (*http.Response, error) {
		method, _, _ := readMethodAndArgs(r)
		if resp, ok := handshakeOK(method); ok {
			return resp, nil
		}
		return structuredResp(map[string]any{
			"result": []map[string]any{{
				"query": "Nope::Missing",
				"fn":    nil,
				"error": "Not found",
			}},
		}), nil
	})
	c := NewMCPHTTPClient("http://test/mcp", &http.Client{Transport: rt})
	addr, ok, err := c.GetFunctionByName(context.Background(), "Nope::Missing")
	if err != nil {
		t.Fatalf("unexpected err=%v (Not found is a soft miss)", err)
	}
	if ok || addr != "" {
		t.Errorf("got addr=%q ok=%v, want empty/false", addr, ok)
	}
}

// TestMCPHTTPDecompileSuccess asserts the NEW decompile success shape returns
// the clean code string and sends {addr, include_addresses:false}.
func TestMCPHTTPDecompileSuccess(t *testing.T) {
	const code = "int __thiscall CInPacket::Decode1(CInPacket *this)\n{\n  return CInPacket::Decode1(a2);\n}"
	var gotTool string
	var gotAddr string
	var gotInclude bool
	rt := rtFunc(func(r *http.Request) (*http.Response, error) {
		method, name, args := readMethodAndArgs(r)
		if resp, ok := handshakeOK(method); ok {
			return resp, nil
		}
		gotTool = name
		var a struct {
			Addr         string `json:"addr"`
			IncludeAddrs bool   `json:"include_addresses"`
		}
		_ = json.Unmarshal(args, &a)
		gotAddr = a.Addr
		gotInclude = a.IncludeAddrs
		return structuredResp(map[string]any{
			"addr":  "0xa3f2e8",
			"code":  code,
			"error": nil,
		}), nil
	})
	c := NewMCPHTTPClient("http://test/mcp", &http.Client{Transport: rt})
	text, err := c.DecompileFunction(context.Background(), "0xa3f2e8")
	if err != nil {
		t.Fatalf("DecompileFunction err=%v", err)
	}
	if text != code {
		t.Errorf("text = %q, want %q", text, code)
	}
	if gotTool != "decompile" {
		t.Errorf("tool = %q, want decompile", gotTool)
	}
	if gotAddr != "0xa3f2e8" {
		t.Errorf("addr arg = %q, want 0xa3f2e8", gotAddr)
	}
	if gotInclude {
		t.Errorf("include_addresses = true, want false")
	}
}

// TestMCPHTTPDecompileFailedSoftError asserts a per-item decompile failure
// (code:null, error:"Decompilation failed", isError:false) is returned as an
// error for which IsDecompilationFailed is true — Harvest maps it to an
// Unresolved entry rather than aborting the run.
func TestMCPHTTPDecompileFailedSoftError(t *testing.T) {
	rt := rtFunc(func(r *http.Request) (*http.Response, error) {
		method, _, _ := readMethodAndArgs(r)
		if resp, ok := handshakeOK(method); ok {
			return resp, nil
		}
		return structuredResp(map[string]any{
			"addr":  "0x4e4427",
			"code":  nil,
			"error": "Decompilation failed",
		}), nil
	})
	c := NewMCPHTTPClient("http://test/mcp", &http.Client{Transport: rt})
	_, err := c.DecompileFunction(context.Background(), "0x4e4427")
	if err == nil {
		t.Fatal("expected a decompile-failed error, got nil")
	}
	if !IsDecompilationFailed(err) {
		t.Errorf("IsDecompilationFailed(%v) = false, want true", err)
	}
	if IsFunctionNotFound(err) {
		t.Errorf("IsFunctionNotFound should be false for a decompile-failed error")
	}
}

// TestMCPHTTPNewRPCDecompileErrorRecognized asserts the exported test helper
// still produces an error IsDecompilationFailed recognizes (Harvest/validate
// tests depend on this).
func TestMCPHTTPNewRPCDecompileErrorRecognized(t *testing.T) {
	err := NewRPCDecompileError("0x1234")
	if !IsDecompilationFailed(err) {
		t.Errorf("IsDecompilationFailed(NewRPCDecompileError) = false, want true")
	}
}

// TestMCPHTTPGetCalleesArgs asserts callees sends {addrs:[addr], limit:200} and
// parses a structured result best-effort.
func TestMCPHTTPGetCalleesArgs(t *testing.T) {
	var gotTool string
	var gotAddrs []string
	var gotLimit int
	rt := rtFunc(func(r *http.Request) (*http.Response, error) {
		method, name, args := readMethodAndArgs(r)
		if resp, ok := handshakeOK(method); ok {
			return resp, nil
		}
		gotTool = name
		var a struct {
			Addrs []string `json:"addrs"`
			Limit int      `json:"limit"`
		}
		_ = json.Unmarshal(args, &a)
		gotAddrs = a.Addrs
		gotLimit = a.Limit
		return structuredResp(map[string]any{
			"result": []map[string]any{{
				"addr": "0x401000",
				"callees": []map[string]any{
					{"addr": "0x401100", "name": "CLogin::OnBar"},
					{"addr": "0x401200", "name": "CLogin::OnBaz"},
				},
			}},
		}), nil
	})
	c := NewMCPHTTPClient("http://test/mcp", &http.Client{Transport: rt})
	cs, err := c.GetCallees(context.Background(), "0x401000")
	if err != nil {
		t.Fatalf("GetCallees: %v", err)
	}
	if gotTool != "callees" {
		t.Errorf("tool = %q, want callees", gotTool)
	}
	if len(gotAddrs) != 1 || gotAddrs[0] != "0x401000" {
		t.Errorf("addrs = %v, want [0x401000]", gotAddrs)
	}
	if gotLimit != 200 {
		t.Errorf("limit = %d, want 200", gotLimit)
	}
	if len(cs) != 2 || cs[0].Name != "CLogin::OnBar" || cs[0].Addr != "0x401100" {
		t.Errorf("callees = %+v", cs)
	}
}

// TestMCPHTTPSelectInstance asserts SelectInstance issues select_instance with
// {port:N}.
func TestMCPHTTPSelectInstance(t *testing.T) {
	var gotTool string
	var gotPort int
	rt := rtFunc(func(r *http.Request) (*http.Response, error) {
		method, name, args := readMethodAndArgs(r)
		if resp, ok := handshakeOK(method); ok {
			return resp, nil
		}
		gotTool = name
		var a struct {
			Port int `json:"port"`
		}
		_ = json.Unmarshal(args, &a)
		gotPort = a.Port
		return structuredResp(map[string]any{"ok": true}), nil
	})
	c := NewMCPHTTPClient("http://test/mcp", &http.Client{Transport: rt})
	if err := c.SelectInstance(context.Background(), 13338); err != nil {
		t.Fatalf("SelectInstance: %v", err)
	}
	if gotTool != "select_instance" {
		t.Errorf("tool = %q, want select_instance", gotTool)
	}
	if gotPort != 13338 {
		t.Errorf("port = %d, want 13338", gotPort)
	}
}

// TestMCPHTTPInstancePortSelectedAfterHandshake asserts a configured instance
// port triggers a select_instance call once, after the handshake, before the
// first tool call.
func TestMCPHTTPInstancePortSelectedAfterHandshake(t *testing.T) {
	var methods []string
	rt := rtFunc(func(r *http.Request) (*http.Response, error) {
		method, name, _ := readMethodAndArgs(r)
		if name != "" {
			methods = append(methods, name)
		} else {
			methods = append(methods, method)
		}
		if resp, ok := handshakeOK(method); ok {
			return resp, nil
		}
		if name == "select_instance" {
			return structuredResp(map[string]any{"ok": true}), nil
		}
		return structuredResp(map[string]any{
			"result": []map[string]any{{"query": "x", "fn": map[string]any{"addr": "0x1"}, "error": nil}},
		}), nil
	})
	c := NewMCPHTTPClientWithInstance("http://test/mcp", &http.Client{Transport: rt}, 13339)
	if _, _, err := c.GetFunctionByName(context.Background(), "x"); err != nil {
		t.Fatalf("GetFunctionByName: %v", err)
	}
	if _, _, err := c.GetFunctionByName(context.Background(), "y"); err != nil {
		t.Fatalf("GetFunctionByName 2: %v", err)
	}
	want := []string{"initialize", "notifications/initialized", "select_instance", "lookup_funcs", "lookup_funcs"}
	if len(methods) != len(want) {
		t.Fatalf("methods = %v, want %v", methods, want)
	}
	for i := range want {
		if methods[i] != want[i] {
			t.Errorf("methods[%d] = %q, want %q", i, methods[i], want[i])
		}
	}
}

// TestMCPHTTPSessionIDReplayed asserts the Mcp-Session-Id from the initialize
// response header is replayed on the tools/call request.
func TestMCPHTTPSessionIDReplayed(t *testing.T) {
	const sid = "11112222-3333-4444-5555-666677778888"
	var toolCallSession string
	rt := rtFunc(func(r *http.Request) (*http.Response, error) {
		method, name, _ := readMethodAndArgs(r)
		switch method {
		case "initialize":
			return initResp(sid), nil
		case "notifications/initialized":
			return notInitErrResp(), nil
		default: // tools/call
			if name == "lookup_funcs" {
				toolCallSession = r.Header.Get("Mcp-Session-Id")
			}
			return structuredResp(map[string]any{
				"result": []map[string]any{{"query": "x", "fn": map[string]any{"addr": "0x1"}, "error": nil}},
			}), nil
		}
	})
	c := NewMCPHTTPClient("http://test/mcp", &http.Client{Transport: rt})
	if _, _, err := c.GetFunctionByName(context.Background(), "A::B"); err != nil {
		t.Fatalf("GetFunctionByName: %v", err)
	}
	if toolCallSession != sid {
		t.Errorf("tools/call Mcp-Session-Id = %q, want %q", toolCallSession, sid)
	}
}

// TestMCPHTTPNotificationsInitializedTolerated asserts a JSON-RPC error response
// to notifications/initialized does NOT abort the handshake; a subsequent
// tools/call still succeeds.
func TestMCPHTTPNotificationsInitializedTolerated(t *testing.T) {
	rt := rtFunc(func(r *http.Request) (*http.Response, error) {
		method, _, _ := readMethodAndArgs(r)
		switch method {
		case "initialize":
			return initResp("sid"), nil
		case "notifications/initialized":
			return notInitErrResp(), nil
		default:
			return structuredResp(map[string]any{
				"result": []map[string]any{{"query": "x", "fn": map[string]any{"addr": "0x1"}, "error": nil}},
			}), nil
		}
	})
	c := NewMCPHTTPClient("http://test/mcp", &http.Client{Transport: rt})
	addr, ok, err := c.GetFunctionByName(context.Background(), "A::B")
	if err != nil || !ok {
		t.Fatalf("err=%v ok=%v (handshake should tolerate -32601)", err, ok)
	}
	if addr != "0x1" {
		t.Errorf("addr = %q, want 0x1", addr)
	}
}

// TestMCPHTTPInitOnce verifies the handshake (initialize +
// notifications/initialized) happens exactly once across two tool calls.
func TestMCPHTTPInitOnce(t *testing.T) {
	var methods []string
	rt := rtFunc(func(r *http.Request) (*http.Response, error) {
		method, _, _ := readMethodAndArgs(r)
		methods = append(methods, method)
		if resp, ok := handshakeOK(method); ok {
			return resp, nil
		}
		return structuredResp(map[string]any{
			"result": []map[string]any{{"query": "x", "fn": map[string]any{"addr": "0x1"}, "error": nil}},
		}), nil
	})
	c := NewMCPHTTPClient("http://test/mcp", &http.Client{Transport: rt})
	if _, _, err := c.GetFunctionByName(context.Background(), "A::B"); err != nil {
		t.Fatalf("call 1: %v", err)
	}
	if _, _, err := c.GetFunctionByName(context.Background(), "C::D"); err != nil {
		t.Fatalf("call 2: %v", err)
	}
	want := []string{"initialize", "notifications/initialized", "tools/call", "tools/call"}
	if len(methods) != len(want) {
		t.Fatalf("methods = %v, want %v", methods, want)
	}
	for i := range want {
		if methods[i] != want[i] {
			t.Errorf("methods[%d] = %q, want %q", i, methods[i], want[i])
		}
	}
}

// TestMCPHTTPErrorOnIsError asserts a tools/call response with isError:true is
// surfaced loudly (never swallowed into an empty export).
func TestMCPHTTPErrorOnIsError(t *testing.T) {
	rt := rtFunc(func(r *http.Request) (*http.Response, error) {
		method, _, _ := readMethodAndArgs(r)
		if resp, ok := handshakeOK(method); ok {
			return resp, nil
		}
		return jsonResp(map[string]any{
			"jsonrpc": "2.0", "id": 1,
			"result": map[string]any{
				"content": []map[string]any{{"type": "text", "text": "boom"}},
				"isError": true,
			},
		}), nil
	})
	c := NewMCPHTTPClient("http://test/mcp", &http.Client{Transport: rt})
	_, err := c.DecompileFunction(context.Background(), "0x1")
	if err == nil {
		t.Fatal("expected error on isError:true, got nil")
	}
}

// TestMCPHTTPErrorOnRPCError asserts a JSON-RPC error object on a tools/call is
// surfaced loudly (transport/protocol failure, not a per-item soft fail).
func TestMCPHTTPErrorOnRPCError(t *testing.T) {
	rt := rtFunc(func(r *http.Request) (*http.Response, error) {
		method, _, _ := readMethodAndArgs(r)
		if resp, ok := handshakeOK(method); ok {
			return resp, nil
		}
		return jsonResp(map[string]any{
			"jsonrpc": "2.0", "id": 1,
			"error": map[string]any{"code": -32602, "message": "Invalid params"},
		}), nil
	})
	c := NewMCPHTTPClient("http://test/mcp", &http.Client{Transport: rt})
	_, err := c.DecompileFunction(context.Background(), "0x1")
	if err == nil {
		t.Fatal("expected error on JSON-RPC error object, got nil")
	}
}

// TestMCPHTTPErrorOnNon200 asserts a non-200 HTTP status errors loudly.
func TestMCPHTTPErrorOnNon200(t *testing.T) {
	rt := rtFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 500,
			Body:       io.NopCloser(bytes.NewReader([]byte("internal error"))),
			Header:     http.Header{},
		}, nil
	})
	c := NewMCPHTTPClient("http://test/mcp", &http.Client{Transport: rt})
	_, _, err := c.GetFunctionByName(context.Background(), "A::B")
	if err == nil {
		t.Fatal("expected error on non-200 status, got nil")
	}
}

// TestMCPHTTPNotification202Tolerated asserts that a 202 Accepted (no body)
// response to notifications/initialized does NOT abort the handshake and a
// subsequent tools/call still succeeds. Per the MCP Streamable-HTTP spec,
// 202 Accepted is the correct server response to a notification.
func TestMCPHTTPNotification202Tolerated(t *testing.T) {
	rt := rtFunc(func(r *http.Request) (*http.Response, error) {
		method, _, _ := readMethodAndArgs(r)
		switch method {
		case "initialize":
			resp := jsonResp(map[string]any{"jsonrpc": "2.0", "id": 1, "result": map[string]any{}})
			resp.Header.Set("Mcp-Session-Id", "sess-1")
			return resp, nil
		case "notifications/initialized":
			return &http.Response{StatusCode: 202, Body: io.NopCloser(bytes.NewReader(nil)),
				Header: http.Header{}}, nil
		default: // tools/call
			return structuredResp(map[string]any{
				"result": []map[string]any{{"query": "x", "fn": map[string]any{"addr": "0xabc"}, "error": nil}},
			}), nil
		}
	})
	c := NewMCPHTTPClient("http://test/mcp", &http.Client{Transport: rt})
	addr, ok, err := c.GetFunctionByName(context.Background(), "Foo::Bar")
	if err != nil || !ok || addr != "0xabc" {
		t.Fatalf("GetFunctionByName err=%v ok=%v addr=%q (202 on notification must be tolerated)", err, ok, addr)
	}
}

// Compile-time assertion: the HTTP client still satisfies MCPClient.
var _ MCPClient = (*MCPHTTPClient)(nil)
