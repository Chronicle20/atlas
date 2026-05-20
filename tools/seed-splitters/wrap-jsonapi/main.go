package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func main() {
	var (
		inputDir       = flag.String("input-dir", "", "directory of plain-JSON files")
		outputDir      = flag.String("output-dir", "", "directory to write JSON:API files")
		typ            = flag.String("type", "", "JSON:API data.type")
		idField        = flag.String("id-field", "", "input JSON top-level field name carrying the entity id")
		filenamePrefix = flag.String("filename-prefix", "", "prefix for output filenames (output = <prefix>-<id>.json)")
	)
	flag.Parse()
	if *inputDir == "" || *outputDir == "" || *typ == "" || *idField == "" {
		fmt.Fprintln(os.Stderr, "usage: wrap-jsonapi --input-dir DIR --output-dir DIR --type TYPE --id-field FIELD [--filename-prefix PREFIX]")
		os.Exit(2)
	}
	if err := os.MkdirAll(*outputDir, 0o755); err != nil {
		fail("mkdir output-dir: %v", err)
	}

	entries, err := os.ReadDir(*inputDir)
	if err != nil {
		fail("read input-dir: %v", err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)

	prefix := *filenamePrefix
	for _, name := range names {
		b, err := os.ReadFile(filepath.Join(*inputDir, name))
		if err != nil {
			fail("read %s: %v", name, err)
		}
		var attrs map[string]any
		if err := json.Unmarshal(b, &attrs); err != nil {
			fail("parse %s: %v", name, err)
		}
		idVal, ok := attrs[*idField]
		if !ok {
			fail("%s: missing id field %q", name, *idField)
		}
		id := fmt.Sprint(idVal)
		envelope := map[string]any{
			"data": map[string]any{
				"type":       *typ,
				"id":         id,
				"attributes": attrs,
			},
		}
		var buf bytes.Buffer
		enc := json.NewEncoder(&buf)
		enc.SetIndent("", "  ")
		enc.SetEscapeHTML(false)
		if err := enc.Encode(envelope); err != nil {
			fail("encode %s: %v", name, err)
		}
		outName := id + ".json"
		if prefix != "" {
			outName = prefix + "-" + id + ".json"
		}
		if err := os.WriteFile(filepath.Join(*outputDir, outName), buf.Bytes(), 0o644); err != nil {
			fail("write %s: %v", outName, err)
		}
	}
}

func fail(format string, a ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", a...)
	os.Exit(1)
}
