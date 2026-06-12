package idasrc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// MCPHTTPClient is a JSON-RPC-over-HTTP client for an IDA-MCP server. It is the
// only component that talks to a live IDA-MCP server (Phase 2). It implements
// the MCPClient interface. CI never hits a live server — every test drives this
// type through a fake http.RoundTripper.
//
// Resilience (design §3.1): an empty or error response must fail LOUDLY. A
// silently-empty export would corrupt the audit, so callTool errors on non-200
// HTTP, a JSON-RPC error object, or empty tools/call content.
type MCPHTTPClient struct {
	url       string
	http      *http.Client
	inited    bool
	nextID    int
	sessionID string // Mcp-Session-Id captured from the initialize response.

	// InstancePort, when non-zero, names an ida-pro-mcp instance (multi-IDB
	// setup) to select once via select_instance, lazily after the handshake.
	// 0 means "use the default active instance" (no select_instance call).
	InstancePort     int
	instanceSelected bool
}

// NewMCPHTTPClient builds a client targeting the given MCP endpoint URL. A nil
// *http.Client defaults to a fresh one. No instance is selected (port 0 → the
// server's default active instance).
func NewMCPHTTPClient(url string, hc *http.Client) *MCPHTTPClient {
	return NewMCPHTTPClientWithInstance(url, hc, 0)
}

// NewMCPHTTPClientWithInstance builds a client that, after the handshake,
// selects the ida-pro-mcp instance listening on the given port (multi-IDB:
// v83/v87/v95/jms each run a distinct instance). A port of 0 selects nothing
// and uses the server's default active instance.
func NewMCPHTTPClientWithInstance(url string, hc *http.Client, port int) *MCPHTTPClient {
	if hc == nil {
		hc = &http.Client{}
	}
	return &MCPHTTPClient{url: url, http: hc, InstancePort: port}
}

// rpcRequest is a JSON-RPC 2.0 request envelope. ID is omitted for
// notifications (which expect no response).
type rpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      *int   `json:"id,omitempty"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

// rpcError is the JSON-RPC 2.0 error object.
type rpcError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

func (e *rpcError) Error() string {
	return fmt.Sprintf("idasrc: MCP rpc error %d: %s", e.Code, e.Message)
}

// Soft-fail classification. The NEW server reports per-function failures NOT as
// transport-level JSON-RPC errors but as a per-item `error` field in the
// tools/call structuredContent payload (with isError:false). These are NOT
// transport/protocol failures: a single missing or undecompilable function must
// not abort a ~1145-function export, so callers (Harvest) branch on them. We
// model them with a dedicated softFailError carrying a kind so the exported
// predicates keep their original semantics while the wire convention changed.
const msgDecompilationFail = "Decompilation failed"

type softFailKind int

const (
	softFailFunctionNotFound softFailKind = iota
	softFailDecompilation
)

// softFailError is a per-item soft failure surfaced by a tool whose call itself
// succeeded (isError:false) but whose result reports a per-function error. It is
// recognized by IsFunctionNotFound / IsDecompilationFailed so Harvest can map it
// to an Unresolved entry instead of aborting the run.
type softFailError struct {
	kind softFailKind
	msg  string
}

func (e *softFailError) Error() string {
	return fmt.Sprintf("idasrc: MCP soft failure: %s", e.msg)
}

// IsFunctionNotFound reports whether err is the soft per-function failure the
// server returns when no function matches a requested name (per-item
// error "Not found").
func IsFunctionNotFound(err error) bool {
	var se *softFailError
	if errors.As(err, &se) {
		return se.kind == softFailFunctionNotFound
	}
	return false
}

// IsDecompilationFailed reports whether err is the soft per-function failure the
// server returns when a function exists but cannot be decompiled (per-item
// error "Decompilation failed ..."). Harvest maps this to an Unresolved entry
// instead of aborting the run.
func IsDecompilationFailed(err error) bool {
	var se *softFailError
	if errors.As(err, &se) {
		return se.kind == softFailDecompilation
	}
	return false
}

// NewRPCDecompileError builds an error recognized by IsDecompilationFailed, for
// the validate command's tests and for any caller that must synthesize a
// decompile soft-fail (the underlying error type is unexported). The name is
// retained for API stability across the server-API re-map.
func NewRPCDecompileError(addr string) error {
	return &softFailError{kind: softFailDecompilation, msg: msgDecompilationFail + " at " + addr}
}

// rpcResponse is a JSON-RPC 2.0 response envelope. Result is parsed by the
// caller; only tools/call interprets the content shape.
type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

// toolResult is the NEW MCP tools/call result shape. The server returns both a
// text content part (a JSON string) and an already-parsed structuredContent
// object on every response, plus an isError flag for tool-level failures.
// Parsing prefers structuredContent (no second unmarshal of the text string).
type toolResult struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	StructuredContent json.RawMessage `json:"structuredContent,omitempty"`
	IsError           bool            `json:"isError"`
}

const mcpProtocolVersion = "2024-11-05"

// post issues one JSON-RPC request. For notifications (id == nil) it is
// fire-and-forget: any 2xx status is accepted (per the MCP Streamable-HTTP
// spec, 202 Accepted is the correct server response), the body is not parsed,
// and nil is returned. A 4xx/5xx is still surfaced as an error. For normal
// requests (id != nil) the response must be exactly 200 OK.
func (c *MCPHTTPClient) post(ctx context.Context, id *int, method string, params any) (*rpcResponse, error) {
	reqBody, err := json.Marshal(rpcRequest{JSONRPC: "2.0", ID: id, Method: method, Params: params})
	if err != nil {
		return nil, fmt.Errorf("idasrc: marshal %s request: %w", method, err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("idasrc: build %s request: %w", method, err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	// The live server issues a session id on initialize and rejects every later
	// request that omits it. Replay it on all post-handshake requests.
	if c.sessionID != "" {
		httpReq.Header.Set("Mcp-Session-Id", c.sessionID)
	}

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("idasrc: %s transport: %w", method, err)
	}
	defer func() { _, _ = io.Copy(io.Discard, resp.Body); _ = resp.Body.Close() }()

	// Capture the session id from the initialize response header; it MUST be
	// replayed on every subsequent request.
	if method == "initialize" {
		if sid := resp.Header.Get("Mcp-Session-Id"); sid != "" {
			c.sessionID = sid
		}
	}

	// Notifications are fire-and-forget: per the MCP Streamable-HTTP spec, any
	// 2xx (200, 202, 204, …) is a valid server acknowledgement — the body is
	// never parsed. Surface genuine transport errors (4xx/5xx) so the caller
	// knows the server actively rejected the notification, but tolerate all 2xx.
	if id == nil {
		if resp.StatusCode/100 == 2 {
			return nil, nil
		}
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("idasrc: %s HTTP %d: %s", method, resp.StatusCode, strings.TrimSpace(string(body)))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("idasrc: read %s response: %w", method, err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("idasrc: %s HTTP %d: %s", method, resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var rpcResp rpcResponse
	if err := json.Unmarshal(body, &rpcResp); err != nil {
		return nil, fmt.Errorf("idasrc: decode %s response: %w", method, err)
	}
	if rpcResp.Error != nil {
		return nil, rpcResp.Error
	}
	return &rpcResp, nil
}

// ensureInit performs the MCP handshake exactly once per client: an initialize
// request followed by the notifications/initialized notification. The
// initialize response is handled LENIENTLY — transport success is enough; we do
// not parse serverInfo/capabilities (real servers return them; the unit fake
// does not). Only tools/call parses content.
func (c *MCPHTTPClient) ensureInit(ctx context.Context) error {
	if c.inited {
		return nil
	}
	id := c.allocID()
	initParams := map[string]any{
		"protocolVersion": mcpProtocolVersion,
		"capabilities":    map[string]any{},
		"clientInfo":      map[string]any{"name": "atlas-packet-audit", "version": "1"},
	}
	if _, err := c.post(ctx, &id, "initialize", initParams); err != nil {
		return err
	}
	// notifications/initialized is a notification: no id, no awaited response.
	if _, err := c.post(ctx, nil, "notifications/initialized", map[string]any{}); err != nil {
		return err
	}
	c.inited = true
	// Multi-IDB: if an instance port is configured, select it exactly once,
	// after the handshake and before any tool call, so subsequent calls target
	// the right IDB. Done here (rather than in a separate guard) so it runs
	// inside the single one-time init path.
	if c.InstancePort != 0 && !c.instanceSelected {
		if err := c.selectInstanceLocked(ctx, c.InstancePort); err != nil {
			return err
		}
		c.instanceSelected = true
	}
	return nil
}

// allocID returns the next monotonically-increasing JSON-RPC request id.
func (c *MCPHTTPClient) allocID() int {
	c.nextID++
	return c.nextID
}

// callStructured issues a tools/call and returns its structuredContent as raw
// JSON. It performs the one-time handshake first, then errors LOUDLY on a
// transport/JSON-RPC error, on isError==true, or on a missing
// structuredContent payload (never silently produces an empty export). Per-item
// soft failures (a function not found, a decompile failure) are NOT visible
// here — they live inside the structuredContent the caller unmarshals.
func (c *MCPHTTPClient) callStructured(ctx context.Context, tool string, args map[string]any) (json.RawMessage, error) {
	if err := c.ensureInit(ctx); err != nil {
		return nil, err
	}
	return c.callStructuredLocked(ctx, tool, args)
}

// callStructuredLocked is callStructured without the handshake — used by the
// post-handshake select_instance call to avoid re-entering ensureInit.
func (c *MCPHTTPClient) callStructuredLocked(ctx context.Context, tool string, args map[string]any) (json.RawMessage, error) {
	id := c.allocID()
	params := map[string]any{"name": tool, "arguments": args}
	resp, err := c.post(ctx, &id, "tools/call", params)
	if err != nil {
		return nil, err
	}
	var tr toolResult
	if err := json.Unmarshal(resp.Result, &tr); err != nil {
		return nil, fmt.Errorf("idasrc: decode tools/call %q result: %w", tool, err)
	}
	if tr.IsError {
		// Tool-level failure: surface the text content (a JSON-RPC error would
		// have been caught earlier). This is loud by design.
		var b strings.Builder
		for _, part := range tr.Content {
			b.WriteString(part.Text)
		}
		return nil, fmt.Errorf("idasrc: tools/call %q failed (isError): %s", tool, strings.TrimSpace(b.String()))
	}
	if len(tr.StructuredContent) == 0 {
		return nil, fmt.Errorf("idasrc: tools/call %q returned no structuredContent", tool)
	}
	return tr.StructuredContent, nil
}

// SelectInstance switches the active ida-pro-mcp instance (multi-IDB) to the one
// listening on the given port, for all subsequent calls. It performs the
// handshake if needed.
func (c *MCPHTTPClient) SelectInstance(ctx context.Context, port int) error {
	if err := c.ensureInit(ctx); err != nil {
		return err
	}
	return c.selectInstanceLocked(ctx, port)
}

func (c *MCPHTTPClient) selectInstanceLocked(ctx context.Context, port int) error {
	if _, err := c.callStructuredLocked(ctx, "select_instance", map[string]any{"port": port}); err != nil {
		return fmt.Errorf("idasrc: select_instance port %d: %w", port, err)
	}
	return nil
}

// lookupFnEntry is one element of the lookup_funcs structuredContent.result
// array. fn is null on a miss; error is "Not found" then.
type lookupFnEntry struct {
	Query string `json:"query"`
	Fn    *struct {
		Addr string `json:"addr"`
		Name string `json:"name"`
		Size string `json:"size"`
	} `json:"fn"`
	Error string `json:"error"`
}

// GetFunctionByName resolves an FName to its address via the NEW batch
// lookup_funcs tool (queries:[name]). The first result entry is inspected: a
// non-null fn yields (fn.addr, true, nil); a null fn / "Not found" error yields
// ("", false, nil) — the resolver turns that into an honest Unresolved, so a
// single missing function never aborts an export. Demangled Class::Method forms
// resolve to "Not found" (acceptable); addresses, sub_XXXX, and MANGLED names
// resolve. Genuine transport/protocol errors stay loud.
func (c *MCPHTTPClient) GetFunctionByName(ctx context.Context, name string) (string, bool, error) {
	raw, err := c.callStructured(ctx, "lookup_funcs", map[string]any{"queries": []string{name}})
	if err != nil {
		return "", false, err
	}
	var payload struct {
		Result []lookupFnEntry `json:"result"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return "", false, fmt.Errorf("idasrc: parse lookup_funcs result: %w", err)
	}
	if len(payload.Result) == 0 {
		return "", false, nil
	}
	e := payload.Result[0]
	if e.Fn == nil {
		// Miss (error typically "Not found"). Soft: ("", false, nil).
		return "", false, nil
	}
	addr := strings.TrimSpace(e.Fn.Addr)
	if addr == "" {
		return "", false, fmt.Errorf("idasrc: lookup_funcs %q: empty addr in result", name)
	}
	return addr, true, nil
}

// DecompileFunction returns the Hex-Rays pseudocode for addr via the NEW
// decompile tool ({addr, include_addresses:false}). On a per-item failure
// (code==null, error set, e.g. "Decompilation failed") it returns an error for
// which IsDecompilationFailed reports true, so Harvest can map it to an
// Unresolved entry; it is not swallowed. The code is returned as-is (it is now
// clean Hex-Rays with no "/* line: N */" prefix; the later parser handles it).
func (c *MCPHTTPClient) DecompileFunction(ctx context.Context, addr string) (string, error) {
	raw, err := c.callStructured(ctx, "decompile", map[string]any{"addr": addr, "include_addresses": false})
	if err != nil {
		return "", err
	}
	var payload struct {
		Addr  string  `json:"addr"`
		Code  *string `json:"code"`
		Error string  `json:"error"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return "", fmt.Errorf("idasrc: parse decompile result: %w", err)
	}
	if payload.Code != nil {
		return *payload.Code, nil
	}
	// code==null: a per-item soft failure. Classify it so IsDecompilationFailed
	// recognizes it.
	if strings.Contains(payload.Error, msgDecompilationFail) {
		return "", &softFailError{kind: softFailDecompilation, msg: payload.Error + " at " + addr}
	}
	if payload.Error != "" {
		return "", fmt.Errorf("idasrc: decompile %s: %s", addr, payload.Error)
	}
	return "", fmt.Errorf("idasrc: decompile %s: null code, no error", addr)
}

// GetCallees returns the direct callees of addr via the NEW callees tool
// ({addrs:[addr], limit:200}). It is off the critical path; the parse is
// best-effort (callee names/addrs only).
func (c *MCPHTTPClient) GetCallees(ctx context.Context, addr string) ([]Callee, error) {
	raw, err := c.callStructured(ctx, "callees", map[string]any{"addrs": []string{addr}, "limit": 200})
	if err != nil {
		return nil, err
	}
	return parseCallees(raw)
}

// StructInfo is a stub on the NEW server: it is NOT used by Harvest or validate
// (the critical path), so it returns an empty StructLayout without calling a
// tool. Wiring the new struct tool can follow if a caller ever needs it.
func (c *MCPHTTPClient) StructInfo(ctx context.Context, name string) (StructLayout, error) {
	_ = ctx
	_ = name
	return StructLayout{}, nil
}

// --- Tool-payload parsers (NEW server structuredContent shapes). ---

// parseCallees parses the NEW callees tool structuredContent (best-effort, off
// the critical path). The shape is {result:[{addr, callees:[{addr,name}]}]};
// since GetCallees passes a single addr, the first result entry's callees are
// flattened. A bare array of callee objects is also accepted defensively.
func parseCallees(raw json.RawMessage) ([]Callee, error) {
	raw = json.RawMessage(strings.TrimSpace(string(raw)))
	if len(raw) == 0 {
		return nil, nil
	}
	type jsCallee struct {
		Name    string `json:"name"`
		Address string `json:"address"`
		Addr    string `json:"addr"`
	}
	collect := func(rawCallees []jsCallee) []Callee {
		out := make([]Callee, 0, len(rawCallees))
		for _, r := range rawCallees {
			a := r.Address
			if a == "" {
				a = r.Addr
			}
			out = append(out, Callee{Name: r.Name, Addr: a})
		}
		return out
	}
	// Primary shape: {result:[{addr, callees:[...]}]}.
	var wrapped struct {
		Result []struct {
			Addr    string     `json:"addr"`
			Callees []jsCallee `json:"callees"`
		} `json:"result"`
	}
	if err := json.Unmarshal(raw, &wrapped); err == nil && len(wrapped.Result) > 0 {
		var all []jsCallee
		for _, r := range wrapped.Result {
			all = append(all, r.Callees...)
		}
		return collect(all), nil
	}
	// Defensive: a bare array of callee objects.
	var bare []jsCallee
	if err := json.Unmarshal(raw, &bare); err == nil {
		return collect(bare), nil
	}
	return nil, nil
}
