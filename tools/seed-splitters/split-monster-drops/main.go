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

type rawDrop struct {
	MonsterID       int   `json:"monsterId"`
	ItemID          int   `json:"itemId"`
	MinimumQuantity int   `json:"minimumQuantity"`
	MaximumQuantity int   `json:"maximumQuantity"`
	QuestID         int   `json:"questId"`
	Chance          int64 `json:"chance"`
}

type outDrop struct {
	ItemID          int   `json:"itemId"`
	MinimumQuantity int   `json:"minimumQuantity"`
	MaximumQuantity int   `json:"maximumQuantity"`
	QuestID         int   `json:"questId"`
	Chance          int64 `json:"chance"`
}

type envelope struct {
	Data envelopeData `json:"data"`
}

type envelopeData struct {
	Type       string    `json:"type"`
	ID         string    `json:"id"`
	Attributes attrsBody `json:"attributes"`
}

type attrsBody struct {
	Drops []outDrop `json:"drops"`
}

func main() {
	in := flag.String("input", "", "path to monster_drops.json")
	out := flag.String("output", "", "output directory")
	flag.Parse()
	if *in == "" || *out == "" {
		fmt.Fprintln(os.Stderr, "usage: split-monster-drops --input FILE --output DIR")
		os.Exit(2)
	}
	if err := os.MkdirAll(*out, 0o755); err != nil {
		fail("mkdir: %v", err)
	}
	b, err := os.ReadFile(*in)
	if err != nil {
		fail("read: %v", err)
	}
	var rows []rawDrop
	if err := json.Unmarshal(b, &rows); err != nil {
		fail("parse: %v", err)
	}
	grouped := map[int][]outDrop{}
	for _, r := range rows {
		grouped[r.MonsterID] = append(grouped[r.MonsterID], outDrop{
			ItemID:          r.ItemID,
			MinimumQuantity: r.MinimumQuantity,
			MaximumQuantity: r.MaximumQuantity,
			QuestID:         r.QuestID,
			Chance:          r.Chance,
		})
	}
	ids := make([]int, 0, len(grouped))
	for id := range grouped {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	for _, id := range ids {
		drops := grouped[id]
		sort.SliceStable(drops, func(i, j int) bool { return drops[i].ItemID < drops[j].ItemID })
		env := envelope{Data: envelopeData{
			Type:       "monster-drop",
			ID:         fmt.Sprint(id),
			Attributes: attrsBody{Drops: drops},
		}}
		var buf bytes.Buffer
		enc := json.NewEncoder(&buf)
		enc.SetIndent("", "  ")
		enc.SetEscapeHTML(false)
		if err := enc.Encode(env); err != nil {
			fail("encode %d: %v", id, err)
		}
		if err := os.WriteFile(filepath.Join(*out, fmt.Sprintf("monster-%d.json", id)), buf.Bytes(), 0o644); err != nil {
			fail("write %d: %v", id, err)
		}
	}
}

func fail(format string, a ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", a...)
	os.Exit(1)
}
