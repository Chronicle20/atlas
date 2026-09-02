package drift

import (
	"encoding/json"
	"sort"
	"testing"
)

func canon(t *testing.T, raw string) Doc {
	t.Helper()
	var v any
	if err := json.Unmarshal([]byte(raw), &v); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}
	d, err := Canonicalize(v)
	if err != nil {
		t.Fatalf("Canonicalize: %v", err)
	}
	return d
}

func keys(d Doc) []string {
	ks := make([]string, 0, len(d))
	for k := range d {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestCanonicalizeDropsExcluded(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want []string
	}{
		{
			name: "DropsEveryExcludedKey",
			raw:  `{"environment":"main","region":"GMS","majorVersion":83,"minorVersion":1,"worlds":[{"name":"w"}],"diagnostics":{"tracePackets":true},"usesPin":true}`,
			want: []string{"usesPin"},
		},
		{
			name: "KeepsComparableSections",
			raw:  `{"usesPin":false,"socket":{"handlers":[{"opCode":1}]},"characters":{"templates":[{"gender":0}]},"npcs":[{"npcId":1}],"cashShop":{"commodities":{"hourlyExpirations":[{"templateId":1}]}},"mapleLife":{"looks":[{"gender":0}]}}`,
			want: []string{"cashShop", "characters", "mapleLife", "npcs", "socket", "usesPin"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			d := canon(t, c.raw)
			got := keys(d)
			if !equalStrings(got, c.want) {
				t.Errorf("keys = %v, want %v", got, c.want)
			}
		})
	}
}

func TestCanonicalizePrunesEmptiness(t *testing.T) {
	cases := []struct {
		name string
		a    string
		b    string
	}{
		{
			name: "NullEqualsEmptyArrayAtTopLevel",
			a:    `{"npcs":null,"usesPin":true}`,
			b:    `{"npcs":[],"usesPin":true}`,
		},
		{
			name: "NullEqualsAbsentAtTopLevel",
			a:    `{"npcs":null,"usesPin":true}`,
			b:    `{"usesPin":true}`,
		},
		{
			name: "NestedNullEqualsNestedEmpty",
			a:    `{"cashShop":{"commodities":null},"usesPin":true}`,
			b:    `{"cashShop":{"commodities":[]},"usesPin":true}`,
		},
		{
			name: "EmptyObjectPrunedRecursively",
			a:    `{"cashShop":{"commodities":{}},"usesPin":true}`,
			b:    `{"usesPin":true}`,
		},
		{
			name: "NestedEmptyCollapsesParent",
			a:    `{"cashShop":{"surprise":{"boxTemplateIds":[]}},"usesPin":true}`,
			b:    `{"usesPin":true}`,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			da := canon(t, c.a)
			db := canon(t, c.b)
			ba, err := json.Marshal(da)
			if err != nil {
				t.Fatalf("marshal a: %v", err)
			}
			bb, err := json.Marshal(db)
			if err != nil {
				t.Fatalf("marshal b: %v", err)
			}
			if string(ba) != string(bb) {
				t.Errorf("a = %s, b = %s, want byte-identical", ba, bb)
			}
		})
	}
}

func TestCanonicalizeKeepsFalsyValues(t *testing.T) {
	cases := []struct {
		name string
		a    string
		b    string
	}{
		{
			name: "FalseIsNotPruned",
			a:    `{"usesPin":false}`,
			b:    `{}`,
		},
		{
			name: "ZeroIsNotPruned",
			a:    `{"majorRank":0}`,
			b:    `{}`,
		},
		{
			name: "EmptyStringIsNotPruned",
			a:    `{"note":""}`,
			b:    `{}`,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			da := canon(t, c.a)
			db := canon(t, c.b)
			ba, err := json.Marshal(da)
			if err != nil {
				t.Fatalf("marshal a: %v", err)
			}
			bb, err := json.Marshal(db)
			if err != nil {
				t.Fatalf("marshal b: %v", err)
			}
			if string(ba) == string(bb) {
				t.Errorf("a = %s, b = %s, want different", ba, bb)
			}
		})
	}
}

func TestCanonicalizeContentDiffersFromEmpty(t *testing.T) {
	da := canon(t, `{"npcs":[{"npcId":1}]}`)
	db := canon(t, `{"npcs":[]}`)

	ba, err := json.Marshal(da)
	if err != nil {
		t.Fatalf("marshal a: %v", err)
	}
	bb, err := json.Marshal(db)
	if err != nil {
		t.Fatalf("marshal b: %v", err)
	}
	if string(ba) == string(bb) {
		t.Errorf("a = %s, b = %s, want different", ba, bb)
	}
}

func TestCanonicalizeIsDeterministic(t *testing.T) {
	da := canon(t, `{"socket":{"handlers":[]},"usesPin":true,"npcs":[{"npcId":9}]}`)
	db := canon(t, `{"npcs":[{"npcId":9}],"usesPin":true,"socket":{"handlers":[]}}`)

	ba, err := json.Marshal(da)
	if err != nil {
		t.Fatalf("marshal a: %v", err)
	}
	bb, err := json.Marshal(db)
	if err != nil {
		t.Fatalf("marshal b: %v", err)
	}
	if string(ba) != string(bb) {
		t.Errorf("a = %s, b = %s, want byte-identical", ba, bb)
	}
}
