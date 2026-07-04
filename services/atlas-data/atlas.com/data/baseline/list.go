package baseline

import (
	"context"
	"encoding/hex"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	minio "atlas-data/storage/minio"

	"github.com/sirupsen/logrus"
)

// baselinePrefix is the canonical-bucket prefix every published baseline
// lives under (see DumpKey).
const baselinePrefix = "baseline/regions/"

// objectStore is the narrow slice of *minio.Client the Lister consumes.
// Tests inject a map-backed fake; *minio.Client satisfies it.
type objectStore interface {
	List(ctx context.Context, bucket, prefix string) ([]minio.ObjectInfo, error)
	Get(ctx context.Context, bucket, key string) (io.ReadCloser, error)
}

// Lister derives the published-baseline collection from canonical-bucket
// objects. Same construction shape as Publisher/Restorer.
type Lister struct {
	MC     objectStore
	Bucket string
	L      logrus.FieldLogger
}

// List enumerates baseline/regions/, keeps keys that parse as dump objects,
// reads each sha sidecar (degrading to "" on any failure), and returns the
// collection sorted by (region, major, minor) ascending so the response is
// deterministic. One bad key never fails the listing.
func (li Lister) List(ctx context.Context) ([]ListItemModel, error) {
	objs, err := li.MC.List(ctx, li.Bucket, baselinePrefix)
	if err != nil {
		return nil, err
	}
	items := make([]ListItemModel, 0)
	for _, o := range objs {
		if !strings.HasSuffix(o.Key, "/documents.dump") {
			continue
		}
		region, major, minor, ok := parseDumpKey(o.Key)
		if !ok {
			li.L.Warnf("baseline list: skipping unparseable key %s", o.Key)
			continue
		}
		items = append(items, ListItemModel{
			Id:           PublishOutputId(region, major, minor),
			Region:       region,
			MajorVersion: major,
			MinorVersion: minor,
			Sha256:       li.readSha(ctx, region, major, minor),
			PublishedAt:  o.LastModified.UTC().Format(time.RFC3339),
			SizeBytes:    o.Size,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Region != items[j].Region {
			return items[i].Region < items[j].Region
		}
		if items[i].MajorVersion != items[j].MajorVersion {
			return items[i].MajorVersion < items[j].MajorVersion
		}
		return items[i].MinorVersion < items[j].MinorVersion
	})
	return items, nil
}

// readSha reads the .sha256 sidecar and expects exactly 64 hex characters.
// Any failure (missing object, read error, malformed content) logs a WARN
// and returns "" so a partially-published baseline is visible rather than
// hidden.
func (li Lister) readSha(ctx context.Context, region string, major, minor int) string {
	rc, err := li.MC.Get(ctx, li.Bucket, ShaKey(region, major, minor))
	if err != nil {
		li.L.Warnf("baseline list: sha sidecar unavailable for %s/%d.%d: %v", region, major, minor, err)
		return ""
	}
	defer rc.Close()
	b, err := io.ReadAll(rc)
	if err != nil {
		li.L.Warnf("baseline list: sha sidecar read failed for %s/%d.%d: %v", region, major, minor, err)
		return ""
	}
	sum := strings.TrimSpace(string(b))
	if raw, decErr := hex.DecodeString(sum); decErr != nil || len(raw) != 32 {
		li.L.Warnf("baseline list: sha sidecar malformed for %s/%d.%d", region, major, minor)
		return ""
	}
	return sum
}

// parseDumpKey extracts (region, major, minor) from a canonical dump key of
// the exact shape DumpKey produces:
// baseline/regions/<region>/versions/<major>.<minor>/documents.dump.
// Keys that do not parse are the caller's cue to skip-and-warn, never fail.
func parseDumpKey(key string) (region string, major, minor int, ok bool) {
	parts := strings.Split(key, "/")
	if len(parts) != 6 || parts[0] != "baseline" || parts[1] != "regions" ||
		parts[3] != "versions" || parts[5] != "documents.dump" {
		return "", 0, 0, false
	}
	region = parts[2]
	if region == "" {
		return "", 0, 0, false
	}
	ver := parts[4]
	dot := strings.LastIndex(ver, ".")
	if dot <= 0 || dot == len(ver)-1 {
		return "", 0, 0, false
	}
	major, err := strconv.Atoi(ver[:dot])
	if err != nil || major < 0 {
		return "", 0, 0, false
	}
	minor, err = strconv.Atoi(ver[dot+1:])
	if err != nil || minor < 0 {
		return "", 0, 0, false
	}
	return region, major, minor, true
}
