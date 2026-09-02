// Command reactor-seed-gen generates tier-1 reactor seed catalogs from the
// tier-1 inventory. It reads the inventory (parse.go), converts each
// section's whitelisted script grammar into a scriptDoc (convert.go),
// derives its description from the source comment (describe.go), then
// marshals and fans it out to all eleven tenant seed directories (emit.go).
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

const (
	defaultInventory = "docs/tasks/task-291-reactor-tier1-conversion/tier1-inventory.md"
	defaultSeedRoot  = "deploy/seed"
)

// builtSeed is one reactor's fully-marshaled seed content, ready to fan out
// or diff against disk.
type builtSeed struct {
	reactorId string
	bytes     []byte
}

func main() {
	inventoryPath := flag.String("inventory", defaultInventory, "path to the tier-1 inventory markdown")
	seedRoot := flag.String("seed-root", defaultSeedRoot, "seed catalog root directory")
	check := flag.Bool("check", false, "regenerate in memory and diff against disk; write nothing")
	flag.Parse()

	if err := run(*inventoryPath, *seedRoot, *check); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// run executes the full pipeline. Any error from any stage aborts the whole
// run with no partial write: results are collected in memory and only
// written out (or diffed, under -check) once every reactor has converted,
// described and marshaled cleanly.
func run(inventoryPath, seedRoot string, check bool) error {
	raw, err := os.ReadFile(inventoryPath)
	if err != nil {
		return fmt.Errorf("reading inventory %s: %w", inventoryPath, err)
	}

	scripts, err := parseInventory(raw)
	if err != nil {
		return fmt.Errorf("parsing inventory: %w", err)
	}

	results := make([]builtSeed, 0, len(scripts))

	for _, s := range scripts {
		doc, err := convertScript(s)
		if err != nil {
			return err
		}

		doc.Description, err = describe(s.Id, s.Comment)
		if err != nil {
			return err
		}

		b, err := marshalEnvelope(doc)
		if err != nil {
			return err
		}

		results = append(results, builtSeed{reactorId: s.Id, bytes: b})
	}

	if check {
		return checkAll(seedRoot, results)
	}

	for _, r := range results {
		if err := fanOut(seedRoot, r.reactorId, r.bytes); err != nil {
			return err
		}
	}

	fmt.Printf("reactor-seed-gen: %d reactors x %d directories = %d files written\n",
		len(results), len(seedDirs), len(results)*len(seedDirs))
	return nil
}

// checkAll regenerates every reactor's seed content in memory and diffs it
// against what is on disk, writing nothing. It exits non-zero listing every
// path that differs or is missing.
func checkAll(seedRoot string, results []builtSeed) error {
	var diffs []string
	for _, r := range results {
		for _, dir := range seedDirs {
			path := filepath.Join(seedRoot, dir, "reactor-actions/reactors", "reactor-"+r.reactorId+".json")
			onDisk, err := os.ReadFile(path)
			if err != nil {
				diffs = append(diffs, fmt.Sprintf("%s: missing (%v)", path, err))
				continue
			}
			if !bytes.Equal(onDisk, r.bytes) {
				diffs = append(diffs, fmt.Sprintf("%s: differs from generated output", path))
			}
		}
	}

	if len(diffs) > 0 {
		for _, d := range diffs {
			fmt.Fprintln(os.Stderr, d)
		}
		return fmt.Errorf("reactor-seed-gen: %d file(s) differ or are missing", len(diffs))
	}

	fmt.Printf("reactor-seed-gen: %d reactors x %d directories = %d files match\n",
		len(results), len(seedDirs), len(results)*len(seedDirs))
	return nil
}
