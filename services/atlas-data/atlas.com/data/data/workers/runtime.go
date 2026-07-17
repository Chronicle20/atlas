// Package workers wires WZ archives into the per-domain Register/Storage layer
// and emits derived assets (icons, atlases, layers, minimaps) to MinIO.
//
// Workers run inside the ingest pod (MODE=ingest). The Params (Region,
// MajorVersion, MinorVersion, ScopeKey, ScratchDir) come from environment
// variables set by JobCreator. ScopeKey is either "shared" (version-scoped
// canonical id derived via canonical.TenantId(region, major, minor)) or
// "tenants/<uuid>"; workers derive a tenant.Model from this so the
// existing per-tenant document storage continues to work.
package workers

import (
	"atlas-data/canonical"
	"atlas-data/data/wztoxml"
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	stdpng "image/png"
	"os"
	"path/filepath"
	"strings"
	"sync"

	minio "atlas-data/storage/minio"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/Chronicle20/atlas/libs/atlas-wz/wz"
)

// tenantFromParams derives a tenant.Model from the worker Params. For
// scope=shared we use the version-scoped canonical id (canonical.TenantId)
// derived from region/major/minor; for tenants/<uuid> we parse the suffix.
func tenantFromParams(p Params) (tenant.Model, error) {
	var id uuid.UUID
	switch {
	case p.ScopeKey == "shared":
		id = canonical.TenantId(p.Region, p.MajorVersion, p.MinorVersion)
	case strings.HasPrefix(p.ScopeKey, "tenants/"):
		parsed, err := uuid.Parse(strings.TrimPrefix(p.ScopeKey, "tenants/"))
		if err != nil {
			return tenant.Model{}, fmt.Errorf("parse tenant uuid: %w", err)
		}
		id = parsed
	default:
		return tenant.Model{}, fmt.Errorf("invalid scope key: %s", p.ScopeKey)
	}
	return tenant.Create(id, p.Region, p.MajorVersion, p.MinorVersion)
}

// withTenant attaches the worker's tenant to ctx so downstream
// tenant.MustFromContext calls work.
func withTenant(ctx context.Context, p Params) (context.Context, tenant.Model, error) {
	t, err := tenantFromParams(p)
	if err != nil {
		return ctx, tenant.Model{}, err
	}
	return tenant.WithContext(ctx, t), t, nil
}

// WithTenant is the exported entry the dispatcher (data.RunWorkers) calls
// before invoking each Worker.Run, so worker bodies receive an
// already-tenanted ctx. Individual workers MAY still call the unexported
// withTenant defensively — when ctx already carries the tenant, that's a
// no-op except for the local `t` it returns. The discarded-return bug that
// shipped in the Commodity worker (df89b8bee) is impossible to trigger when
// the dispatcher pre-injects, because downstream MustFromContext sees the
// dispatcher's tenant regardless of how the worker handles the return.
func WithTenant(ctx context.Context, p Params) (context.Context, tenant.Model, error) {
	return withTenant(ctx, p)
}

// archiveDir returns the directory where SerializeArchive writes the XML tree
// for a given archive name (e.g. "Item.wz"). Workers compute domain-specific
// subpaths under this root.
func archiveDir(p Params, archive string) string {
	return filepath.Join(p.ScratchDir, "xml", p.Region, fmt.Sprintf("%d.%d", p.MajorVersion, p.MinorVersion), archive)
}

// rootDir returns the parent directory containing per-archive XML trees.
// Compatible with existing `path = <region>/<ver>` callers that expect e.g.
// path/Item.wz/Consume/<id>.img.xml.
func rootDir(p Params) string {
	return filepath.Join(p.ScratchDir, "xml", p.Region, fmt.Sprintf("%d.%d", p.MajorVersion, p.MinorVersion))
}

// serializeArchive writes the WZ file's full directory tree to
// archiveDir(p, file.Name()+".wz"), then returns rootDir(p) so existing
// Register* functions can be called with their accustomed path layout. The
// caller is responsible for cleanup (typically t.TempDir or the pod's
// emptyDir).
func serializeArchive(l logrus.FieldLogger, p Params, file *wz.File) (string, error) {
	root := rootDir(p)
	if err := os.MkdirAll(root, 0o755); err != nil {
		return "", fmt.Errorf("mkdir scratch root: %w", err)
	}
	if err := wztoxml.SerializeToDirectory(l, file, root); err != nil {
		return "", err
	}
	return root, nil
}

// ErrCategoryAbsent reports that the scope stores a monolithic Data.wz whose
// root has no subdirectory for the requested archive (v12 has no
// Quest/Morph/TamingMob). Split-layout scopes (no Data.wz) never produce
// this error — a missing archive there stays a hard failure, matching
// pre-monolithic behavior (task-172 C-3.4).
var ErrCategoryAbsent = errors.New("category absent from monolithic Data.wz")

// monolith memoizes the scope's Data.wz for the lifetime of one ingest job.
// Same job-scoped reasoning as archiveCache: Params are constant per job and
// the ingest pod exits when the job completes. RunWorkers defers
// CloseMonolith to release the handle and reset the memo.
type monolithState struct {
	once      sync.Once
	file      *wz.File
	localPath string
	found     bool
	err       error
}

var monolith monolithState

func monolithFile(ctx context.Context, l logrus.FieldLogger, mc *minio.Client, p Params) (*wz.File, bool, error) {
	monolith.once.Do(func() {
		key := fmt.Sprintf("%s/regions/%s/versions/%d.%d/Data.wz", p.ScopeKey, p.Region, p.MajorVersion, p.MinorVersion)
		exists, err := mc.Stat(ctx, mc.Cfg().BucketWZ, key)
		if err != nil {
			monolith.err = fmt.Errorf("stat %s: %w", key, err)
			return
		}
		if !exists {
			return
		}
		localPath, err := mc.DownloadToScratch(ctx, mc.Cfg().BucketWZ, key, p.ScratchDir)
		if err != nil {
			monolith.err = fmt.Errorf("download %s: %w", key, err)
			return
		}
		f, err := wz.Open(l, localPath)
		if err != nil {
			_ = os.Remove(localPath)
			monolith.err = fmt.Errorf("open %s: %w", localPath, err)
			return
		}
		l.Infof("monolithic Data.wz detected for scope %s — serving archives as sub-views", p.ScopeKey)
		monolith.file, monolith.localPath, monolith.found = f, localPath, true
	})
	return monolith.file, monolith.found, monolith.err
}

// CloseMonolith closes and removes the memoized Data.wz and resets the memo.
// Deferred by data.RunWorkers; also used by tests between cases. Must only
// run after all workers have finished (sub-views share the handle).
func CloseMonolith() {
	if monolith.file != nil {
		monolith.file.Close()
		_ = os.Remove(monolith.localPath)
	}
	monolith = monolithState{}
}

// monolithSubArchive resolves an archive name against a parsed Data.wz:
// Base.wz maps to the root itself (root-level smap.img/zmap.img play the
// Base.wz role); any other archive maps to the root subdirectory with the
// same stem (Item.wz → Item/).
func monolithSubArchive(mono *wz.File, archive string) (*wz.File, error) {
	stem := strings.TrimSuffix(archive, ".wz")
	if archive == "Base.wz" {
		return wz.NewSubFile(mono, mono.Root(), stem), nil
	}
	for _, d := range mono.Root().Directories() {
		if d.Name() == stem {
			return wz.NewSubFile(mono, d, stem), nil
		}
	}
	return nil, fmt.Errorf("%s: %w", archive, ErrCategoryAbsent)
}

// OpenArchive resolves an archive for the worker scope: the per-archive
// MinIO object when present, else a sub-archive view over the scope's
// monolithic Data.wz (task-172 C-3). The returned cleanup MUST be deferred;
// it is a no-op for monolithic views — the shared Data.wz is closed once by
// CloseMonolith when the job ends.
//
// Concurrency: per-archive opens get a private file as before. Monolithic
// sub-views share one *wz.File across workers; lazy image parsing is
// serialized by the parent's parseMu (see wz.File docs), trading some
// parallelism for correctness.
func OpenArchive(ctx context.Context, l logrus.FieldLogger, mc *minio.Client, p Params, archive string) (*wz.File, func(), error) {
	noop := func() {}
	if mc == nil {
		return nil, noop, fmt.Errorf("minio client unavailable")
	}
	key := fmt.Sprintf("%s/regions/%s/versions/%d.%d/%s", p.ScopeKey, p.Region, p.MajorVersion, p.MinorVersion, archive)
	exists, err := mc.Stat(ctx, mc.Cfg().BucketWZ, key)
	if err != nil {
		return nil, noop, fmt.Errorf("stat %s: %w", key, err)
	}
	if exists {
		localPath, err := mc.DownloadToScratch(ctx, mc.Cfg().BucketWZ, key, p.ScratchDir)
		if err != nil {
			return nil, noop, fmt.Errorf("download %s: %w", key, err)
		}
		f, err := wz.Open(l, localPath)
		if err != nil {
			_ = os.Remove(localPath)
			return nil, noop, fmt.Errorf("open %s: %w", localPath, err)
		}
		return f, func() { f.Close(); _ = os.Remove(localPath) }, nil
	}
	mono, found, err := monolithFile(ctx, l, mc, p)
	if err != nil {
		return nil, noop, err
	}
	if !found {
		return nil, noop, fmt.Errorf("archive %s not found (no per-archive object and no Data.wz in scope)", key)
	}
	sub, err := monolithSubArchive(mono, archive)
	if err != nil {
		return nil, noop, err
	}
	return sub, noop, nil
}

// archiveSerialization memoizes one cross-archive fetch+serialize per archive
// name. Multiple workers (Mob/Npc/Skill/Map/Character) need to read the SAME
// String.wz, Etc.wz, or UI.wz, and each calling fetchAndSerializeArchive
// independently races on:
//
//  1. The shared download path $scratch/<archive> — os.Create truncates and
//     defer os.Remove leaves a window where peers see a missing file
//     (observed: Skill worker logged "String.wz unavailable" on PR-544).
//  2. The XML output dir $scratch/xml/<region>/<ver>/<archive>/*.img.xml —
//     wztoxml.SerializeToDirectory calls os.Create on every file; concurrent
//     re-serializations briefly truncate, and a reader that happens to grab
//     the file during that window parses an empty <imgdir/> with zero
//     ChildNodes. Silent: no error returned, the consuming InitString just
//     adds nothing to its registry, and downstream document Name fields stay
//     blank. PR-544 evidence: all 1568 MONSTER + 1620 NPC docs had name=”.
//
// The map is keyed by archive name only because Params (ScopeKey/Region/
// Version) are constant for the lifetime of one ingest job. sync.Once
// provides the happens-before relationship for archiveResult.root/err.
type archiveResult struct {
	once sync.Once
	root string
	err  error
}

var archiveCache sync.Map // archive name -> *archiveResult

// fetchAndSerializeArchive downloads <BucketWZ>/<scope>/regions/<region>/versions/<x.y>/<archive>
// from MinIO into ScratchDir, opens it as a wz.File, and serializes it next to
// the current worker's other archives so domain readers can resolve
// cross-archive references (e.g. String.wz Eqp.img while the worker is on
// Character.wz). Returns the rootDir(p) shared layout root.
//
// Per-archive memoized: only the first caller for a given archive name does
// the work; later callers wait via sync.Once and read the cached result.
func fetchAndSerializeArchive(ctx context.Context, l logrus.FieldLogger, mc *minio.Client, p Params, archive string) (string, error) {
	if mc == nil {
		return "", fmt.Errorf("minio client unavailable")
	}
	v, _ := archiveCache.LoadOrStore(archive, &archiveResult{})
	r := v.(*archiveResult)
	r.once.Do(func() {
		r.root, r.err = fetchAndSerializeArchiveOnce(ctx, l, mc, p, archive)
	})
	return r.root, r.err
}

// fetchAndSerializeArchiveOnce is the actual fetch+serialize. Always called
// at most once per (archive, ingest-process) by fetchAndSerializeArchive's
// sync.Once gate.
func fetchAndSerializeArchiveOnce(ctx context.Context, l logrus.FieldLogger, mc *minio.Client, p Params, archive string) (string, error) {
	f, cleanup, err := OpenArchive(ctx, l, mc, p, archive)
	if err != nil {
		return "", err
	}
	defer cleanup()
	return serializeArchive(l, p, f)
}

// minioAssetPrefix returns the MinIO key prefix shared by all derived assets
// under the worker's scope/region/version. Concrete keys are formed by
// appending the per-domain suffix (e.g. "item/1000000/icon.png").
func minioAssetPrefix(p Params) string {
	return fmt.Sprintf("%s/regions/%s/versions/%d.%d", p.ScopeKey, p.Region, p.MajorVersion, p.MinorVersion)
}

// putPNG encodes the image with the std library PNG encoder and PUTs it to
// MinIO. Used for icons / minimap / map layers. Deterministic-encoder
// (libs/atlas-wz/atlas/pngenc) is reserved for the canonical-baseline atlas
// path (Character.wz).
func putPNG(ctx context.Context, mc *minio.Client, key string, img image.Image) error {
	var buf bytes.Buffer
	if err := stdpng.Encode(&buf, img); err != nil {
		return fmt.Errorf("encode png: %w", err)
	}
	return mc.Put(ctx, mc.Cfg().BucketAssets, key, bytes.NewReader(buf.Bytes()), int64(buf.Len()), "image/png")
}

// putJSON PUTs raw JSON bytes to MinIO with content-type application/json.
func putJSON(ctx context.Context, mc *minio.Client, key string, data []byte) error {
	return mc.Put(ctx, mc.Cfg().BucketAssets, key, bytes.NewReader(data), int64(len(data)), "application/json")
}

// putBytes PUTs raw bytes with the supplied content type.
func putBytes(ctx context.Context, mc *minio.Client, key string, data []byte, contentType string) error {
	return mc.Put(ctx, mc.Cfg().BucketAssets, key, bytes.NewReader(data), int64(len(data)), contentType)
}
