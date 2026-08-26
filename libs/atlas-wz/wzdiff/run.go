package wzdiff

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-wz/wz"
	"github.com/Chronicle20/atlas/libs/atlas-wz/wz/wzxml"
)

// Result is what Run produces: the image-set counts and, for every image
// present on both sides, the deltas Diff found net of the allowlist.
type Result struct {
	// ImagesOurs and ImagesReference are the number of images this
	// archive's tree and the reference dump each enumerate. A gap between
	// them (the "419 vs 421" case in
	// evidence-wz-parse-divergence-reactor.txt) is a divergence in its own
	// right, tracked separately from any property-level Delta.
	ImagesOurs      int
	ImagesReference int
	// Divergent maps an image's HaRepacker file name (e.g.
	// "2006000.img.xml") to the deltas Diff found for it, after any
	// allowlisted deltas have been dropped. An image with no surviving
	// deltas is not a key in this map.
	Divergent map[string][]Delta
	// Allowed counts every Delta that Diff found but that Allowed matched
	// against the allowlist and dropped from Divergent.
	Allowed int
}

// Run walks the image tree rooted at archivePath and compares every image
// present in both that tree and referenceDir (a HaRepacker-style dump,
// one "<image>.img.xml" file per image) against its wzxml-rendered
// counterpart. Images present on only one side are logged by name through
// l at Warn level and counted in Result's Images* fields, never mixed into
// Divergent.
func Run(l logrus.FieldLogger, archivePath, referenceDir string, allow []AllowEntry) (Result, error) {
	f, err := wz.Open(l, archivePath)
	if err != nil {
		return Result{}, fmt.Errorf("wzdiff: open archive %s: %w", archivePath, err)
	}
	defer f.Close()

	ours := map[string]*wz.Image{}
	collectImages(f.Root(), ours)

	refNames, err := referenceImageNames(referenceDir)
	if err != nil {
		return Result{}, fmt.Errorf("wzdiff: read reference dir %s: %w", referenceDir, err)
	}

	result := Result{
		ImagesOurs:      len(ours),
		ImagesReference: len(refNames),
		Divergent:       map[string][]Delta{},
	}

	var onlyOurs, onlyRef []string
	for name := range ours {
		if !refNames[name] {
			onlyOurs = append(onlyOurs, name)
		}
	}
	for name := range refNames {
		if _, ok := ours[name]; !ok {
			onlyRef = append(onlyRef, name)
		}
	}
	sort.Strings(onlyOurs)
	sort.Strings(onlyRef)
	if len(onlyRef) > 0 {
		l.Warnf("wzdiff: images only in HaRepacker dump: %v", onlyRef)
	}
	if len(onlyOurs) > 0 {
		l.Warnf("wzdiff: images only in our parse: %v", onlyOurs)
	}

	var shared []string
	for name := range ours {
		if refNames[name] {
			shared = append(shared, name)
		}
	}
	sort.Strings(shared)

	for _, name := range shared {
		img := ours[name]
		props, err := img.Properties()
		if err != nil {
			return Result{}, fmt.Errorf("wzdiff: parse image %s: %w", name, err)
		}
		oursNodes := FromElements(wzxml.PropertiesToElements(props))

		refPath := filepath.Join(referenceDir, name+".img.xml")
		refNodes, err := LoadImageXML(refPath)
		if err != nil {
			return Result{}, fmt.Errorf("wzdiff: load reference %s: %w", refPath, err)
		}

		deltas := Diff(oursNodes, refNodes)
		var kept []Delta
		for _, d := range deltas {
			if Allowed(allow, name, d) {
				result.Allowed++
				continue
			}
			kept = append(kept, d)
		}
		if len(kept) > 0 {
			result.Divergent[name+".img.xml"] = kept
		}
	}

	return result, nil
}

// collectImages walks dir and its descendants, recording every image
// found by its Name() (no directory-path prefix: both our archive tree
// and a HaRepacker dump identify an image by its bare name, and the
// evidence file's image list does the same).
func collectImages(dir *wz.Directory, out map[string]*wz.Image) {
	for _, img := range dir.Images() {
		out[img.Name()] = img
	}
	for _, sub := range dir.Directories() {
		collectImages(sub, out)
	}
}

// referenceImageNames scans dir for "*.img.xml" files and returns the set
// of image names they name (the ".img.xml" suffix stripped).
func referenceImageNames(dir string) (map[string]bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	out := map[string]bool{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if !strings.HasSuffix(e.Name(), ".img.xml") {
			continue
		}
		out[strings.TrimSuffix(e.Name(), ".img.xml")] = true
	}
	return out, nil
}

// Trace opens archivePath, installs a parse trace hook, parses exactly
// imageName's properties, and writes every TraceEvent observed to w, one
// per line, in decode order. This is the FR-1 tool: it turns "the tree is
// wrong" into "the stream diverged at offset X" for one named image
// without diffing anything.
func Trace(l logrus.FieldLogger, archivePath, imageName string, w io.Writer) error {
	f, err := wz.Open(l, archivePath)
	if err != nil {
		return fmt.Errorf("wzdiff: open archive %s: %w", archivePath, err)
	}
	defer f.Close()

	images := map[string]*wz.Image{}
	collectImages(f.Root(), images)
	img, ok := images[imageName]
	if !ok {
		return fmt.Errorf("wzdiff: image %q not found in %s", imageName, archivePath)
	}

	f.SetTrace(func(ev wz.TraceEvent) {
		fmt.Fprintf(w, "%s kind=%s name=%s type=%d start=%d end=%d %s\n",
			ev.Path, ev.Kind, ev.Name, ev.Type, ev.StartOff, ev.EndOff, ev.Detail)
	})

	if _, err := img.Properties(); err != nil {
		return fmt.Errorf("wzdiff: parse image %s: %w", imageName, err)
	}
	return nil
}

// WriteReport renders result in the two-section, per-image format used by
// evidence-wz-parse-divergence-reactor.txt: a header giving both image
// counts, then one "====...====" block per divergent image with its
// HaRepacker-only and ours-only deltas grouped separately, reference
// before ours (matching Diff's own ordering convention).
func WriteReport(w io.Writer, result Result) {
	fmt.Fprintf(w, "local image count: %d  ours: %d\n", result.ImagesReference, result.ImagesOurs)
	if result.Allowed > 0 {
		fmt.Fprintf(w, "allowlisted deltas dropped: %d\n", result.Allowed)
	}

	names := make([]string, 0, len(result.Divergent))
	for name := range result.Divergent {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		deltas := result.Divergent[name]
		fmt.Fprintln(w, strings.Repeat("=", 70))
		fmt.Fprintln(w, name)

		var refOnly, oursOnly []Delta
		for _, d := range deltas {
			if d.OnlyIn == "reference" {
				refOnly = append(refOnly, d)
			} else {
				oursOnly = append(oursOnly, d)
			}
		}

		fmt.Fprintln(w, "  -- present in HaRepacker dump, ABSENT from our parse:")
		for _, d := range refOnly {
			fmt.Fprintln(w, d.String())
		}
		fmt.Fprintln(w, "  -- present in our parse, ABSENT from HaRepacker dump:")
		for _, d := range oursOnly {
			fmt.Fprintln(w, d.String())
		}
	}
}
