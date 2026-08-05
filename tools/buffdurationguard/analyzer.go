// Package buffdurationguard bans seconds→milliseconds scaling in the duration
// fields of the COMMAND_TOPIC_CHARACTER_BUFF command bodies.
//
// The unit contract is owned by
// services/atlas-buffs/atlas.com/buffs/kafka/message/character/kafka.go
// (ApplyCommandBody.Duration — milliseconds). It has been flipped three times
// in prose alone (11e07dfa7, 197324e40, 88d270bf1); this analyzer is the
// mechanical half of task-190 FR-3.2.
//
// The body struct is duplicated under seven different local names, so the
// analyzer fingerprints json tag sets rather than type names.
package buffdurationguard

import (
	"go/ast"
	"go/token"
	"go/types"
	"reflect"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const markerPrefix = "//buffdurationguard:allow"

const diagnostic = "buffdurationguard: duration fields on the character-buff command are MILLISECONDS; " +
	"drop the seconds-to-ms scaling. Contract owner: atlas-buffs kafka/message/character/kafka.go " +
	"(or annotate with //buffdurationguard:allow <justification>)"

// fingerprint identifies one command body by the json tags its struct carries,
// and names the tags whose assigned expressions must not scale.
type fingerprint struct {
	requires []string
	guards   []string
}

var fingerprints = []fingerprint{
	// BD-1 — the buff APPLY body.
	{requires: []string{"sourceId", "duration", "changes"}, guards: []string{"duration"}},
	// BD-2 — the mist create body.
	{requires: []string{"diseaseDuration", "tickIntervalMs"}, guards: []string{"diseaseDuration", "duration"}},
}

// bannedSelectors are the time constants that can only appear in a
// seconds(or coarser)→ms conversion.
var bannedSelectors = map[string]bool{"Second": true, "Minute": true, "Hour": true}

var Analyzer = &analysis.Analyzer{
	Name:     "buffdurationguard",
	Doc:      "bans seconds-to-milliseconds scaling in COMMAND_TOPIC_CHARACTER_BUFF duration fields",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

type lineKey struct {
	file string
	line int
}

func run(pass *analysis.Pass) (interface{}, error) {
	markers := collectMarkers(pass)
	assigns := collectAssignments(pass)
	reported := map[lineKey]bool{}

	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	insp.Preorder([]ast.Node{(*ast.CompositeLit)(nil)}, func(n ast.Node) {
		cl := n.(*ast.CompositeLit)
		if strings.HasSuffix(pass.Fset.Position(cl.Pos()).Filename, "_test.go") {
			return
		}
		st, ok := structOf(pass, cl)
		if !ok {
			return
		}
		tags := tagSet(st)
		for _, fp := range fingerprints {
			if !hasAll(tags, fp.requires) {
				continue
			}
			for _, elt := range cl.Elts {
				kv, ok := elt.(*ast.KeyValueExpr)
				if !ok {
					continue
				}
				key, ok := kv.Key.(*ast.Ident)
				if !ok {
					continue
				}
				tag, ok := jsonTagOfField(st, key.Name)
				if !ok || !contains(fp.guards, tag) {
					continue
				}
				if pos, bad := scalingPos(pass, assigns, kv.Value, 0); bad {
					report(pass, markers, reported, pos)
				}
			}
		}
	})
	return nil, nil
}

func collectMarkers(pass *analysis.Pass) map[lineKey]bool {
	markers := map[lineKey]bool{}
	for _, f := range pass.Files {
		for _, cg := range f.Comments {
			for _, c := range cg.List {
				if !strings.HasPrefix(c.Text, markerPrefix) {
					continue
				}
				p := pass.Fset.Position(c.Pos())
				justification := strings.TrimSpace(strings.TrimPrefix(c.Text, markerPrefix))
				markers[lineKey{p.Filename, p.Line}] = justification != ""
			}
		}
	}
	return markers
}

// collectAssignments maps each local variable to every expression assigned to
// it. Object identity is unique per variable, so no per-function scoping is
// needed. This is what lets the checker follow `durMs` back to its RHS.
func collectAssignments(pass *analysis.Pass) map[types.Object][]ast.Expr {
	out := map[types.Object][]ast.Expr{}
	for _, f := range pass.Files {
		ast.Inspect(f, func(n ast.Node) bool {
			as, ok := n.(*ast.AssignStmt)
			if !ok || len(as.Lhs) != len(as.Rhs) {
				return true
			}
			for i, lhs := range as.Lhs {
				id, ok := lhs.(*ast.Ident)
				if !ok {
					continue
				}
				obj := pass.TypesInfo.ObjectOf(id)
				if _, isVar := obj.(*types.Var); !isVar {
					continue
				}
				out[obj] = append(out[obj], as.Rhs[i])
			}
			return true
		})
	}
	return out
}

func structOf(pass *analysis.Pass, cl *ast.CompositeLit) (*types.Struct, bool) {
	t := pass.TypesInfo.TypeOf(cl)
	if t == nil {
		return nil, false
	}
	st, ok := t.Underlying().(*types.Struct)
	return st, ok
}

func tagSet(st *types.Struct) map[string]bool {
	out := map[string]bool{}
	for i := 0; i < st.NumFields(); i++ {
		if name := jsonName(st.Tag(i)); name != "" {
			out[name] = true
		}
	}
	return out
}

func jsonTagOfField(st *types.Struct, fieldName string) (string, bool) {
	for i := 0; i < st.NumFields(); i++ {
		if st.Field(i).Name() == fieldName {
			name := jsonName(st.Tag(i))
			return name, name != ""
		}
	}
	return "", false
}

func jsonName(tag string) string {
	v := reflect.StructTag(tag).Get("json")
	if v == "" || v == "-" {
		return ""
	}
	return strings.Split(v, ",")[0]
}

func hasAll(set map[string]bool, want []string) bool {
	for _, w := range want {
		if !set[w] {
			return false
		}
	}
	return true
}

func contains(hay []string, needle string) bool {
	for _, h := range hay {
		if h == needle {
			return true
		}
	}
	return false
}

// scalingPos reports the position of a seconds→ms scaling factor inside expr,
// following identifiers one level into the assignments that produced them.
func scalingPos(pass *analysis.Pass, assigns map[types.Object][]ast.Expr, expr ast.Expr, depth int) (token.Pos, bool) {
	var found token.Pos
	var ok bool
	ast.Inspect(expr, func(n ast.Node) bool {
		if ok {
			return false
		}
		switch e := n.(type) {
		case *ast.SelectorExpr:
			if pkg, isIdent := e.X.(*ast.Ident); isIdent && pkg.Name == "time" && bannedSelectors[e.Sel.Name] {
				found, ok = e.Pos(), true
				return false
			}
		case *ast.BasicLit:
			if e.Kind == token.INT && strings.ReplaceAll(e.Value, "_", "") == "1000" {
				found, ok = e.Pos(), true
				return false
			}
		case *ast.Ident:
			if depth >= 1 {
				return true
			}
			obj := pass.TypesInfo.ObjectOf(e)
			if obj == nil {
				return true
			}
			for _, rhs := range assigns[obj] {
				if p, bad := scalingPos(pass, assigns, rhs, depth+1); bad {
					found, ok = p, true
					return false
				}
			}
		}
		return true
	})
	return found, ok
}

func report(pass *analysis.Pass, markers map[lineKey]bool, reported map[lineKey]bool, pos token.Pos) {
	p := pass.Fset.Position(pos)
	k := lineKey{p.Filename, p.Line}
	if reported[k] {
		return
	}
	reported[k] = true

	if justified, found := markerFor(markers, p); found {
		if !justified {
			pass.Reportf(pos, "buffdurationguard: allow marker requires a justification")
		}
		return
	}
	pass.Reportf(pos, "%s", diagnostic)
}

// markerFor accepts a marker trailing on the offending line or on the line
// immediately above it.
func markerFor(markers map[lineKey]bool, pos token.Position) (justified bool, found bool) {
	if justified, found = markers[lineKey{pos.Filename, pos.Line}]; found {
		return justified, found
	}
	justified, found = markers[lineKey{pos.Filename, pos.Line - 1}]
	return justified, found
}
