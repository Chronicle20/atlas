package handler

// operator_cancel_unreachable_test.go — machine-checks that the OPERATOR
// pending-change cancel route is unreachable from any socket handler in this
// package, and that no tenant socket-config template binds a clientbound
// cancel-result writer as a handler.
//
// NARROWED PROPERTY. This guard once asserted "no client cancel path exists
// at all" — that the pending-change cancel route was operator-only REST.
// That property is FALSE: commit 4a5d9ff65 landed a real client-initiated
// cancel path, on the strength of two IDA derivations committed at
// docs/tasks/task-227-cash-name-change-world-transfer/cancel-entry-point.md
// and cancel-confirm-semantics.md. The client's name-change/world-transfer
// coupon item-use arm builds a double-confirm CANCELREQUESTS_* dialog chain
// and, only on full confirmation, appends an invariant trailing byte to the
// generic cash item-use packet — see character_cash_item_use.go's
// handleCashCouponCancel, which resolves to the SELF-SCOPED cancel route
// (POST /characters/{id}/pending-changes/cancel, reason "player_cancelled",
// atlas-character pending_change/processor.go:349 CancelForCharacterAndType).
// That is the feature working as derived from the client.
//
// A green guard asserting a falsehood is worse than no guard — it
// manufactures confidence in a property the code does not have. The property
// is narrowed to what is actually true and still worth enforcing: the
// id-based, OPERATOR-only DELETE route (reason "operator_cancelled",
// atlas-character pending_change/resource.go:164 handleCancelPendingChange)
// must stay unreachable from any socket handler, and no template may bind
// the clientbound CashShopCancelNameChangeResult /
// CashShopCancelTransferWorldResult writers as a handler.
//
// The shell equivalent that runs this tree-wide (so the property holds in CI,
// not just this module's `go test`) is tools/operator-cancel-path-guard.sh.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

const (
	operatorCancelReason = "operator_cancelled"
	pendingChangesPath   = "pending-changes"
)

var deleteCallPattern = regexp.MustCompile(`MakeDeleteRequest|http\.MethodDelete|"DELETE"`)

// handlerGoFiles returns every non-test .go file in this package's directory.
func handlerGoFiles(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("reading socket/handler directory: %v", err)
	}
	var files []string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		files = append(files, name)
	}
	if len(files) == 0 {
		t.Fatal("no handler .go files found -- directory scan is broken")
	}
	return files
}

// TestOperatorCancelReasonUnreachableFromHandlers asserts no handler file
// references the operator-only cancel reason string.
func TestOperatorCancelReasonUnreachableFromHandlers(t *testing.T) {
	for _, name := range handlerGoFiles(t) {
		b, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("reading %s: %v", name, err)
		}
		if strings.Contains(string(b), operatorCancelReason) {
			t.Errorf("%s references the operator-only cancel reason %q -- the operator route "+
				"(pending_change/resource.go:164) must never be reachable from a socket handler",
				name, operatorCancelReason)
		}
	}
}

// TestOperatorCancelDeleteRouteUnreachableFromHandlers asserts no handler
// file combines an HTTP DELETE call with the pending-changes resource path.
// The legitimate self-scoped route is a POST to a fixed "/cancel" sub-path
// and never appears with a DELETE call -- that pairing only exists to reach
// the id-based operator route.
func TestOperatorCancelDeleteRouteUnreachableFromHandlers(t *testing.T) {
	for _, name := range handlerGoFiles(t) {
		b, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("reading %s: %v", name, err)
		}
		content := string(b)
		if strings.Contains(content, pendingChangesPath) && deleteCallPattern.MatchString(content) {
			t.Errorf("%s combines an HTTP DELETE call with the %q resource path -- this reaches "+
				"the id-based operator cancel route, which must never be reachable from a socket handler",
				name, pendingChangesPath)
		}
	}
}

// TestSelfScopedPlayerCancelRoutePassesGuard is the positive half: it proves
// the legitimate self-scoped cancel path is NOT banned by the two checks
// above, so a future tightening of this guard cannot silently re-ban the
// real feature landed in commit 4a5d9ff65.
func TestSelfScopedPlayerCancelRoutePassesGuard(t *testing.T) {
	// The real handler that drives the coupon cancel arm must not itself
	// trip either check.
	const realHandlerFile = "character_cash_item_use.go"
	b, err := os.ReadFile(realHandlerFile)
	if err != nil {
		t.Fatalf("reading %s: %v", realHandlerFile, err)
	}
	content := string(b)
	if strings.Contains(content, operatorCancelReason) {
		t.Fatalf("%s unexpectedly references the operator-only cancel reason", realHandlerFile)
	}
	if strings.Contains(content, pendingChangesPath) && deleteCallPattern.MatchString(content) {
		t.Fatalf("%s unexpectedly combines a DELETE call with %q", realHandlerFile, pendingChangesPath)
	}

	// A synthetic self-scoped snippet -- the shape the legitimate route
	// actually takes (POST to a fixed "/cancel" sub-path, reason
	// "player_cancelled") -- must also pass both checks.
	selfScoped := `
package handler

const selfScopedCancelReason = "player_cancelled"

func handleSelfScopedCancel() {
	_ = requests.MakePostRequest[RestModel]("characters/1/pending-changes/cancel")(input)
}
`
	if strings.Contains(selfScoped, operatorCancelReason) {
		t.Fatal("self-scoped snippet unexpectedly matches the operator-only reason")
	}
	if strings.Contains(selfScoped, pendingChangesPath) && deleteCallPattern.MatchString(selfScoped) {
		t.Fatal("self-scoped snippet unexpectedly matches the DELETE+pending-changes pattern " +
			"-- the guard would incorrectly ban the legitimate player-cancel route")
	}
}

// TestOperatorCancelWriterNotBoundAsTemplateHandler asserts no tenant
// socket-config template binds CashShopCancelNameChangeResultWriter or
// CashShopCancelTransferWorldResultWriter (both CLIENTBOUND writers,
// libs/atlas-packet/cash/clientbound/cancel_name_change_result.go and
// cancel_transfer_world_result.go) as a "handler" entry.
func TestOperatorCancelWriterNotBoundAsTemplateHandler(t *testing.T) {
	banned := map[string]bool{
		"CashShopCancelNameChangeResult":    true,
		"CashShopCancelTransferWorldResult": true,
	}

	templateDir := filepath.Join("..", "..", "..", "..", "..", "..",
		"services", "atlas-configurations", "seed-data", "templates")
	matches, err := filepath.Glob(filepath.Join(templateDir, "template_*.json"))
	if err != nil {
		t.Fatalf("globbing templates: %v", err)
	}
	if len(matches) == 0 {
		t.Fatalf("no templates found under %s -- path is wrong", templateDir)
	}

	type entry struct {
		Handler string `json:"handler"`
		OpCode  string `json:"opCode"`
	}
	type socket struct {
		Handlers []entry `json:"handlers"`
	}
	type doc struct {
		Socket socket `json:"socket"`
	}

	for _, path := range matches {
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading %s: %v", path, err)
		}
		var d doc
		if err := json.Unmarshal(b, &d); err != nil {
			t.Fatalf("parsing %s: %v", path, err)
		}
		for _, h := range d.Socket.Handlers {
			if banned[h.Handler] {
				t.Errorf("%s binds clientbound writer %q as a handler @ opCode %s -- it is a "+
					"clientbound writer, not a handler, and this would route a client packet "+
					"into the cancel-result codec",
					filepath.Base(path), h.Handler, h.OpCode)
			}
		}
	}
}
