package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

type rawGachapon struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	NpcIDs         []int  `json:"npcIds"`
	CommonWeight   int    `json:"commonWeight"`
	UncommonWeight int    `json:"uncommonWeight"`
	RareWeight     int    `json:"rareWeight"`
}

type rawItem struct {
	GachaponID string `json:"gachaponId"`
	ItemID     int    `json:"itemId"`
	Quantity   int    `json:"quantity"`
	Tier       string `json:"tier"`
}

type rawGlobalItem struct {
	ItemID   int    `json:"itemId"`
	Quantity int    `json:"quantity"`
	Tier     string `json:"tier"`
}

type outItem struct {
	ItemID   int    `json:"itemId"`
	Quantity int    `json:"quantity"`
	Tier     string `json:"tier"`
}

type gachaponAttrs struct {
	Name           string    `json:"name"`
	NpcIDs         []int     `json:"npcIds"`
	CommonWeight   int       `json:"commonWeight"`
	UncommonWeight int       `json:"uncommonWeight"`
	RareWeight     int       `json:"rareWeight"`
	Items          []outItem `json:"items"`
}

type globalAttrs struct {
	Items []outItem `json:"items"`
}

type envelope struct {
	Data envelopeData `json:"data"`
}

type envelopeData struct {
	Type       string `json:"type"`
	ID         string `json:"id"`
	Attributes any    `json:"attributes"`
}

func main() {
	gachaponsPath := flag.String("gachapons", "", "path to gachapons.json")
	itemsPath := flag.String("items", "", "path to gachapon_items.json")
	globalPath := flag.String("global", "", "path to global_gachapon_items.json")
	out := flag.String("output", "", "output directory")
	flag.Parse()
	if *gachaponsPath == "" || *itemsPath == "" || *globalPath == "" || *out == "" {
		fmt.Fprintln(os.Stderr, "usage: split-gachapons --gachapons FILE --items FILE --global FILE --output DIR")
		os.Exit(2)
	}
	if err := os.MkdirAll(*out, 0o755); err != nil {
		fail("mkdir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(*out, "_global"), 0o755); err != nil {
		fail("mkdir _global: %v", err)
	}

	var gachapons []rawGachapon
	mustReadJSON(*gachaponsPath, &gachapons)
	var items []rawItem
	mustReadJSON(*itemsPath, &items)
	var globals []rawGlobalItem
	mustReadJSON(*globalPath, &globals)

	groupedItems := map[string][]outItem{}
	for _, it := range items {
		groupedItems[it.GachaponID] = append(groupedItems[it.GachaponID], outItem{
			ItemID: it.ItemID, Quantity: it.Quantity, Tier: it.Tier,
		})
	}

	sort.SliceStable(gachapons, func(i, j int) bool { return gachapons[i].ID < gachapons[j].ID })
	for _, g := range gachapons {
		drops := groupedItems[g.ID]
		sort.SliceStable(drops, func(i, j int) bool { return drops[i].ItemID < drops[j].ItemID })
		env := envelope{Data: envelopeData{
			Type: "gachapon",
			ID:   g.ID,
			Attributes: gachaponAttrs{
				Name:           g.Name,
				NpcIDs:         g.NpcIDs,
				CommonWeight:   g.CommonWeight,
				UncommonWeight: g.UncommonWeight,
				RareWeight:     g.RareWeight,
				Items:          drops,
			},
		}}
		writeEnvelope(filepath.Join(*out, "gachapon-"+g.ID+".json"), env)
	}

	globalOut := make([]outItem, 0, len(globals))
	for _, gi := range globals {
		globalOut = append(globalOut, outItem{ItemID: gi.ItemID, Quantity: gi.Quantity, Tier: gi.Tier})
	}
	sort.SliceStable(globalOut, func(i, j int) bool { return globalOut[i].ItemID < globalOut[j].ItemID })
	globalEnv := envelope{Data: envelopeData{
		Type:       "gachapon-pool",
		ID:         "_global",
		Attributes: globalAttrs{Items: globalOut},
	}}
	writeEnvelope(filepath.Join(*out, "_global", "items.json"), globalEnv)
}

func mustReadJSON(path string, v any) {
	b, err := os.ReadFile(path)
	if err != nil {
		fail("read %s: %v", path, err)
	}
	if err := json.Unmarshal(b, v); err != nil {
		fail("parse %s: %v", path, err)
	}
}

func writeEnvelope(path string, env envelope) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(env); err != nil {
		fail("encode %s: %v", path, err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		fail("write %s: %v", path, err)
	}
}

func fail(format string, a ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", a...)
	os.Exit(1)
}
