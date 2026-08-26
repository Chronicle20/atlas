package wztoxml

import (
	"bytes"
	stdxml "encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"

	atlasxml "atlas-data/xml"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-wz/wz"
	"github.com/Chronicle20/atlas/libs/atlas-wz/wz/property"
	"github.com/Chronicle20/atlas/libs/atlas-wz/wz/wzxml"
	"github.com/Chronicle20/atlas/libs/atlas-wz/wztest"
)

// TestRoundTripImage verifies an in-memory wz.Image can be serialized to XML
// and then re-parsed by atlas-data/xml into a Node with the expected shape.
func TestRoundTripImage(t *testing.T) {
	dir := t.TempDir()
	props := []property.Property{
		property.NewSub("info", []property.Property{
			property.NewInt("id", 100000),
			property.NewString("name", "Mushroom"),
		}),
	}
	// We can't easily build a wz.Image directly without exporting more APIs;
	// instead test the inner serializer by writing the XML manually and
	// verifying it parses back into atlas-data/xml.Node.
	root := wzxml.Element{
		XMLName:  stdxml.Name{Local: "imgdir"},
		Name:     "0100000.img",
		Children: wzxml.PropertiesToElements(props),
	}
	path := filepath.Join(dir, "0100000.img.xml")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(stdxml.Header); err != nil {
		t.Fatal(err)
	}
	enc := stdxml.NewEncoder(f)
	enc.Indent("", "  ")
	if err := enc.Encode(root); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	// Parse via atlas-data xml reader.
	n, err := atlasxml.FromPathProvider(path)()
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if n.Name != "0100000.img" {
		t.Errorf("Name=%q", n.Name)
	}
	info, err := n.ChildByName("info")
	if err != nil {
		t.Fatalf("ChildByName: %v", err)
	}
	if info.GetIntegerWithDefault("id", -1) != 100000 {
		t.Errorf("id mismatch")
	}
	if info.GetString("name", "") != "Mushroom" {
		t.Errorf("name mismatch")
	}
}

// wzIntSize and stringBlockSize mirror wztest.Builder.Build's encoding-length
// rules (independent of the encryption key: XOR doesn't change byte length),
// used below to locate a specific image's on-disk bytes to corrupt.
func wzIntSize(v int32) int {
	if v >= -127 && v <= 127 {
		return 1
	}
	return 5
}
func wzStringSize(s string) int    { return 1 + len(s) }
func stringBlockSize(s string) int { return 1 + wzStringSize(s) }

// TestSerializeDirectoryCountsFailures builds a two-image archive where one
// image's content is corrupted so Properties() fails, then asserts
// serializeDirectory logs exactly one per-image failure line and the caller
// logs exactly one per-archive "N of M" summary line - and no per-property
// line, which the Observability NFR forbids.
func TestSerializeDirectoryCountsFailures(t *testing.T) {
	const desc = "Package file test" // wztest.Builder.Build's fixed PKG1 description

	b := wztest.NewBuilder().
		SetVersion(83).
		AddImage(wztest.Img("Good", wztest.Int("price", 50))).
		AddImage(wztest.Img("Bad", wztest.Int("price", 50)))

	data, err := b.Build()
	if err != nil {
		t.Fatalf("build fixture: %v", err)
	}

	contentStart := 4 + 8 + 4 + len(desc) + 1
	imgSize := stringBlockSize("Property") + 2 + // "Property" tag + hasProperty(0,0)
		wzIntSize(1) + // property count
		stringBlockSize("price") + 1 /* Int type byte */ + wzIntSize(50)

	entryGood := 1 /* type */ + wzStringSize("Good.img") + wzIntSize(int32(imgSize)) + wzIntSize(0) + 4
	entryBad := 1 + wzStringSize("Bad.img") + wzIntSize(int32(imgSize)) + wzIntSize(0) + 4
	rootDirSize := wzIntSize(2) + entryGood + entryBad

	// Layout: header + ev(2) + root dir chunk + Good image chunk + Bad image
	// chunk (images are appended, in AddImage order, before the root chunk).
	badOffset := contentStart + 2 + rootDirSize + imgSize
	if data[badOffset] != 0x73 {
		t.Fatalf("computed badOffset=%d does not point at Bad's \"Property\" string tag (got 0x%02X, want 0x73)", badOffset, data[badOffset])
	}
	data[badOffset] = 0xFF // invalid string block tag: Bad.Properties() will fail

	path := filepath.Join(t.TempDir(), "Item.wz")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	var buf bytes.Buffer
	logger := logrus.New()
	logger.SetOutput(&buf)
	logger.SetFormatter(&logrus.TextFormatter{DisableTimestamp: true})

	f, err := wz.Open(logger, path)
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer f.Close()

	outDir := t.TempDir()
	if err := SerializeToDirectory(logger, f, outDir); err != nil {
		t.Fatalf("SerializeToDirectory: %v", err)
	}

	logOutput := buf.String()
	lines := strings.Split(strings.TrimRight(logOutput, "\n"), "\n")

	var perImageLines, summaryLines int
	for _, line := range lines {
		if strings.Contains(line, "unable to serialize image [Bad]") {
			perImageLines++
		}
		if strings.Contains(line, "1 of 2 images failed to serialize") {
			summaryLines++
		}
	}
	if perImageLines != 1 {
		t.Fatalf("per-image failure lines = %d, want 1; log:\n%s", perImageLines, logOutput)
	}
	if summaryLines != 1 {
		t.Fatalf("per-archive summary lines = %d, want 1; log:\n%s", summaryLines, logOutput)
	}
	if strings.Contains(logOutput, "unable to serialize image [Good]") {
		t.Fatalf("Good image logged a failure line; log:\n%s", logOutput)
	}
	// No per-property line: nothing in the log names a property (e.g.
	// "price") independently of an image-level line.
	if strings.Contains(logOutput, "price") {
		t.Fatalf("log contains per-property detail, forbidden by the Observability NFR; log:\n%s", logOutput)
	}
}
