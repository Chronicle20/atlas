package wzdiff

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-wz/wz"
)

// Violation records one type-9 sub-object whose decode did not end where
// its own declared size said it would — the drift that
// image.go's recovery reseek would otherwise silently heal (task-262).
type Violation struct {
	Image       string
	Path        string
	Name        string
	DeclaredEnd int64
	ActualEnd   int64
}

// ImageError records one image that failed to parse at all: a single bad
// image must not hide the state of every other image in the archive, so
// SelfCheck records these and keeps walking rather than aborting.
type ImageError struct {
	Image string
	Err   error
}

// SelfCheckResult is the outcome of a whole-archive size-accounting walk:
// how many images and type-9 sub-objects were examined, and every
// disagreement found between a sub-object's declared size and where its
// decode actually ended, or between an image and a successful parse.
type SelfCheckResult struct {
	Images      int
	SubObjects  int
	Violations  []Violation
	ParseErrors []ImageError
}

// SelfCheck opens archivePath and walks every image in it, installing a
// parse trace hook that records a Violation for every type-9 sub-object
// whose DeclaredEnd (from its own declared size) disagrees with its
// ActualEnd (where the decode actually stopped, captured before the
// recovery reseek at wz/image.go:368-370 heals the drift).
//
// This needs no external reference dump: the archive's own declared sizes
// are the oracle. An image that fails to parse is recorded in
// ParseErrors and does not stop the walk — the returned error is reserved
// for failures that do (the archive itself failing to open).
func SelfCheck(l logrus.FieldLogger, archivePath string) (SelfCheckResult, error) {
	f, err := wz.Open(l, archivePath)
	if err != nil {
		return SelfCheckResult{}, fmt.Errorf("wzdiff: open archive %s: %w", archivePath, err)
	}
	defer f.Close()

	images := map[string]*wz.Image{}
	collectImages(f.Root(), images)

	result := SelfCheckResult{Images: len(images)}

	var currentImage string
	f.SetTrace(func(ev wz.TraceEvent) {
		if ev.Kind != "sub" {
			return
		}
		result.SubObjects++
		if ev.DeclaredEnd != ev.ActualEnd {
			result.Violations = append(result.Violations, Violation{
				Image:       currentImage,
				Path:        ev.Path,
				Name:        ev.Name,
				DeclaredEnd: ev.DeclaredEnd,
				ActualEnd:   ev.ActualEnd,
			})
		}
	})

	names := make([]string, 0, len(images))
	for name := range images {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		currentImage = name
		if _, err := images[name].Properties(); err != nil {
			result.ParseErrors = append(result.ParseErrors, ImageError{Image: name, Err: err})
		}
	}

	return result, nil
}

// WriteSelfCheckReport renders r following WriteReport's convention: a
// totals line first, then one line per violation naming its image, path,
// and the two offsets that disagree, then one line per image that failed
// to parse at all.
func WriteSelfCheckReport(w io.Writer, r SelfCheckResult) {
	_, _ = fmt.Fprintf(w, "images: %d  sub-objects: %d  violations: %d  parse errors: %d\n",
		r.Images, r.SubObjects, len(r.Violations), len(r.ParseErrors))

	if len(r.Violations) > 0 {
		_, _ = fmt.Fprintln(w, strings.Repeat("=", 70))
		_, _ = fmt.Fprintln(w, "size-accounting violations:")
		for _, v := range r.Violations {
			_, _ = fmt.Fprintf(w, "  %s %s (%s): declaredEnd=%d actualEnd=%d\n",
				v.Image, v.Path, v.Name, v.DeclaredEnd, v.ActualEnd)
		}
	}

	if len(r.ParseErrors) > 0 {
		_, _ = fmt.Fprintln(w, strings.Repeat("=", 70))
		_, _ = fmt.Fprintln(w, "images that failed to parse:")
		for _, pe := range r.ParseErrors {
			_, _ = fmt.Fprintf(w, "  %s: %v\n", pe.Image, pe.Err)
		}
	}
}
