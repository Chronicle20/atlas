// Package topicguard implements task-276's standing guard against the topic
// migration regressing: it keeps every topic.Token producer/consumer wiring
// call declared through a typed topic.Token constant (never a bare string
// literal), keeps raw os.Getenv/os.LookupEnv reads of a topic-shaped
// environment variable name out of the tree, and keeps every declared
// topic.Token constant present in the generated
// libs/atlas-kafka/gen/topics.yaml manifest.
//
// Three independent diagnostics:
//
//  1. bare-token-literal — a bare string literal (or a reference to an
//     untyped string constant) reaching a topic.Token-typed function
//     parameter. A defined type alone cannot catch this: Go implicitly
//     converts an untyped string constant to the parameter's named type at
//     the call site, so the type checker alone sees a well-typed call.
//  2. raw-env-topic-read — os.Getenv/os.LookupEnv called with a bare string
//     literal that looks like a topic token name (contains "TOPIC"),
//     instead of `string(SomeTypedConst)`.
//  3. token-not-in-manifest — a topic.Token-typed constant declared in the
//     analyzed package whose value is absent from the checked-in
//     topics.yaml manifest, meaning tools/gen-topics.sh has not been rerun.
//
// Manifest access: topics.yaml lives in libs/atlas-kafka/gen, outside this
// module, and cannot be go:embed'd across the module boundary the way the
// other guards embed their allowlists. The topicguard.manifest flag can
// override the path; unset, run walks up from the analyzed package's
// directory to the repo root (marked by go.work) and appends
// libs/atlas-kafka/gen/topics.yaml, stopping short if the walk passes
// through a "testdata" directory first (see defaultManifestPath). If the
// manifest path cannot be resolved or read — including when analysistest's
// fixtures are being analyzed — diagnostic 3 is skipped silently;
// diagnostics 1 and 2 still run.
package topicguard

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// tokenTypePkgPath and tokenTypeName identify the topic.Token named type
// (libs/atlas-kafka/topic/token.go) exactly the way libs/atlas-kafka/gen's
// own scanner does (gen/scan.go).
const (
	tokenTypePkgPath = "github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	tokenTypeName    = "Token"
)

// manifestRelPath is topics.yaml's path relative to the repo root.
const manifestRelPath = "libs/atlas-kafka/gen/topics.yaml"

// manifestFlagName is the Analyzer's own flag, registered on Analyzer.Flags
// per the brief's "Manifest access" note.
const manifestFlagName = "topicguard.manifest"

var rawEnvTopicPattern = regexp.MustCompile(`^[A-Z0-9_]*TOPIC[A-Z0-9_]*$`)

var Analyzer = &analysis.Analyzer{
	Name:     "topicguard",
	Doc:      "task-276: flags bare topic literals reaching topic.Token parameters, raw os.Getenv/LookupEnv reads of topic-shaped env var names, and topic.Token constants missing from libs/atlas-kafka/gen/topics.yaml",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func init() {
	Analyzer.Flags.String(manifestFlagName, "", "override path to libs/atlas-kafka/gen/topics.yaml (default: walk up from the analyzed package's directory to the repo root)")
}

func run(pass *analysis.Pass) (interface{}, error) {
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	insp.Preorder([]ast.Node{(*ast.CallExpr)(nil)}, func(n ast.Node) {
		call := n.(*ast.CallExpr)
		checkBareTokenLiteral(pass, call)
		checkRawEnvTopicRead(pass, call)
	})

	checkTokensInManifest(pass)

	return nil, nil
}

// --- Diagnostic 1: bare-token-literal ---

// checkBareTokenLiteral flags every call argument reaching a topic.Token
// parameter whose own value is a bare string literal or a reference to an
// untyped string constant — never a properly typed topic.Token constant.
// Excludes _test.go files: the fleet's unit tests legitimately pass
// throwaway placeholder strings ("MY_TOPIC", "T", "test-token", ...) to
// topic.Token parameters as test doubles, the same class of "not a
// production path" call site scopeguard's own call-site rules exclude
// _test.go from (tools/scopeguard/analyzer.go's isTestFile).
func checkBareTokenLiteral(pass *analysis.Pass, call *ast.CallExpr) {
	if isTestFile(pass.Fset.Position(call.Pos())) {
		return
	}
	sig, ok := pass.TypesInfo.TypeOf(call.Fun).(*types.Signature)
	if !ok {
		return
	}
	for i, arg := range call.Args {
		pt := paramTypeAt(sig, i)
		if pt == nil || !isTopicTokenType(pt) {
			continue
		}
		reportIfBareTokenArg(pass, arg)
	}
}

// paramTypeAt returns the type of sig's i'th parameter, expanding a
// variadic final parameter's slice element type for any i at or beyond it.
func paramTypeAt(sig *types.Signature, i int) types.Type {
	params := sig.Params()
	n := params.Len()
	if n == 0 {
		return nil
	}
	if sig.Variadic() && i >= n-1 {
		if sl, ok := params.At(n - 1).Type().(*types.Slice); ok {
			return sl.Elem()
		}
		return params.At(n - 1).Type()
	}
	if i >= n {
		return nil
	}
	return params.At(i).Type()
}

// isTopicTokenType reports whether t is exactly the named type
// …/atlas-kafka/topic.Token.
func isTopicTokenType(t types.Type) bool {
	named, ok := t.(*types.Named)
	if !ok {
		return false
	}
	obj := named.Obj()
	return obj != nil && obj.Pkg() != nil && obj.Pkg().Path() == tokenTypePkgPath && obj.Name() == tokenTypeName
}

// reportIfBareTokenArg reports arg if it is a bare string literal, or a
// reference (identifier or qualified selector) to a constant whose own
// declared type is not topic.Token — an untyped string constant reaching a
// topic.Token parameter only by Go's implicit conversion at the call site —
// and its value looks like a topic token name (design D2b diagnostic 1: the
// same `^[A-Z0-9_]*TOPIC[A-Z0-9_]*$` pattern diagnostic 2 uses). Any other
// argument shape (a conversion, a function call, a variable, a field of
// type topic.Token) is out of scope: it is either already sanctioned or not
// decidable without risking a false positive.
func reportIfBareTokenArg(pass *analysis.Pass, arg ast.Expr) {
	switch e := arg.(type) {
	case *ast.BasicLit:
		if e.Kind != token.STRING {
			return
		}
		val, err := strconv.Unquote(e.Value)
		if err != nil || !rawEnvTopicPattern.MatchString(val) {
			return
		}
		pass.Reportf(arg.Pos(), "bare topic literal %q reaching a topic.Token parameter; declare it as a topic.Token constant", val)
	case *ast.Ident:
		reportIfUntypedConstRef(pass, arg, pass.TypesInfo.Uses[e])
	case *ast.SelectorExpr:
		reportIfUntypedConstRef(pass, arg, pass.TypesInfo.Uses[e.Sel])
	}
}

func reportIfUntypedConstRef(pass *analysis.Pass, arg ast.Expr, obj types.Object) {
	c, ok := obj.(*types.Const)
	if !ok {
		return
	}
	if isTopicTokenType(c.Type()) {
		return
	}
	if c.Val().Kind() != constant.String {
		return
	}
	val := constant.StringVal(c.Val())
	if !rawEnvTopicPattern.MatchString(val) {
		return
	}
	pass.Reportf(arg.Pos(), "bare topic literal %q reaching a topic.Token parameter; declare it as a topic.Token constant", val)
}

// --- Diagnostic 2: raw-env-topic-read ---

// checkRawEnvTopicRead flags os.Getenv/os.LookupEnv called with a bare
// string literal whose name looks like a topic token — the sanctioned form
// is `os.Getenv(string(SomeTypedConst))`, a *ast.CallExpr argument, which
// this never matches. Excludes _test.go files for the same reason
// checkBareTokenLiteral does.
func checkRawEnvTopicRead(pass *analysis.Pass, call *ast.CallExpr) {
	if pass.Pkg.Path() == tokenTypePkgPath {
		// libs/atlas-kafka/topic itself resolves a topic.Token to its
		// environment value via os.LookupEnv(string(token)) — the
		// sanctioned mechanism this diagnostic exists to require
		// everywhere else.
		return
	}
	if isTestFile(pass.Fset.Position(call.Pos())) {
		return
	}

	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}
	fn, ok := pass.TypesInfo.Uses[sel.Sel].(*types.Func)
	if !ok || fn.Pkg() == nil || fn.Pkg().Path() != "os" {
		return
	}
	if fn.Name() != "Getenv" && fn.Name() != "LookupEnv" {
		return
	}
	if len(call.Args) != 1 {
		return
	}
	lit, ok := call.Args[0].(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return
	}
	val, err := strconv.Unquote(lit.Value)
	if err != nil || !rawEnvTopicPattern.MatchString(val) {
		return
	}
	pass.Reportf(call.Pos(), "raw environment read of topic token %q; reference a topic.Token constant instead", val)
}

// --- Diagnostic 3: token-not-in-manifest ---

// checkTokensInManifest flags every topic.Token constant declared in the
// analyzed package whose value is absent from the loaded manifest. If the
// manifest cannot be resolved or read — including when analyzing
// analysistest's testdata fixtures (see defaultManifestPath) — this is a
// silent no-op; diagnostics 1 and 2 above still ran.
//
// Excludes constants declared in _test.go files: libs/atlas-kafka/gen's own
// scanner loads the workspace with Tests: false (design D2a, FR-1.7), so a
// _test.go-declared token never enters topics.yaml in the first place — a
// topic.Token constant declared only for a test fixture would otherwise be
// a permanent, unfixable manifest gap.
func checkTokensInManifest(pass *analysis.Pass) {
	manifest := loadManifestForPass(pass)
	if manifest == nil {
		return
	}
	for _, f := range pass.Files {
		for _, decl := range f.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok || gd.Tok != token.CONST {
				continue
			}
			for _, spec := range gd.Specs {
				vs, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for _, name := range vs.Names {
					if isTestFile(pass.Fset.Position(name.Pos())) {
						continue
					}
					obj, ok := pass.TypesInfo.Defs[name].(*types.Const)
					if !ok || !isTopicTokenType(obj.Type()) || obj.Val().Kind() != constant.String {
						continue
					}
					tok := constant.StringVal(obj.Val())
					if msg := manifestDiagnostic(manifest, tok); msg != "" {
						pass.Reportf(name.Pos(), "%s", msg)
					}
				}
			}
		}
	}
}

// isTestFile reports whether pos falls in a _test.go file — excluded from
// all three diagnostics; see each check function's own doc comment for why.
func isTestFile(pos token.Position) bool {
	return strings.HasSuffix(pos.Filename, "_test.go")
}

// manifestDiagnostic returns the diagnostic message for tok given manifest,
// or "" if tok is present or manifest is nil (unreadable — diagnostic 3 is
// skipped entirely in that case; see checkTokensInManifest).
func manifestDiagnostic(manifest map[string]bool, tok string) string {
	if manifest == nil || manifest[tok] {
		return ""
	}
	return "topic token \"" + tok + "\" is not in " + manifestRelPath + "; run tools/gen-topics.sh"
}

// loadManifestForPass resolves and loads the manifest for the package being
// analyzed, honoring the topicguard.manifest flag override.
func loadManifestForPass(pass *analysis.Pass) map[string]bool {
	path := manifestPathFlag(pass)
	if path == "" {
		path = defaultManifestPath(pass)
	}
	if path == "" {
		return nil
	}
	manifest, err := loadManifest(path)
	if err != nil {
		return nil
	}
	return manifest
}

func manifestPathFlag(pass *analysis.Pass) string {
	f := pass.Analyzer.Flags.Lookup(manifestFlagName)
	if f == nil {
		return ""
	}
	return f.Value.String()
}

// defaultManifestPath walks up from the analyzed package's own directory to
// the repo root (marked by a go.work file, present at the repo root — see
// libs/atlas-kafka/gen/scan.go's own use of it) and appends manifestRelPath.
// Returns "" if no such root is found within the walk, or if the walk
// passes through a directory literally named "testdata" first.
//
// The testdata exclusion matters because analysistest.TestData() (unlike
// analysistest.WriteFiles) does not copy fixtures into an isolated temp
// GOPATH — it loads them in place from this module's own
// testdata/src/atlas-example/... tree, which physically sits inside this
// repo checkout and so WOULD otherwise walk up to this repo's own go.work
// and topics.yaml. Go tooling itself already treats any "testdata"
// directory as out of the build; this reuses that same convention to keep
// diagnostic 3 from firing on fixture-only tokens the real manifest never
// heard of.
func defaultManifestPath(pass *analysis.Pass) string {
	if len(pass.Files) == 0 {
		return ""
	}
	dir := filepath.Dir(pass.Fset.Position(pass.Files[0].Pos()).Filename)
	for {
		if filepath.Base(dir) == "testdata" {
			return ""
		}
		if _, err := os.Stat(filepath.Join(dir, "go.work")); err == nil {
			return filepath.Join(dir, manifestRelPath)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// loadManifest reads path and returns the set of every topic token it
// lists. topics.yaml is machine-generated (libs/atlas-kafka/gen/manifest.go)
// in a fixed, simple shape — a top-level `topics:` list of
// `- token: <value>` entries — so a line-oriented scan avoids adding a yaml
// dependency to this module for a format this analyzer only ever reads, not
// writes. The COMMAND_TOPIC_/EVENT_TOPIC_ prefix-constant entries
// (topics.yaml:5,468) are ordinary entries in that list and are loaded like
// any other — not stray data to filter out.
func loadManifest(path string) (map[string]bool, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	out := map[string]bool{}
	for _, line := range strings.Split(string(raw), "\n") {
		trimmed := strings.TrimSpace(line)
		const prefix = "- token:"
		if !strings.HasPrefix(trimmed, prefix) {
			continue
		}
		tok := strings.TrimSpace(strings.TrimPrefix(trimmed, prefix))
		if tok != "" {
			out[tok] = true
		}
	}
	return out, nil
}
