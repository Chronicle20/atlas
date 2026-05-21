// Package workers wires WZ archives into the per-domain Register/Storage layer
// and emits derived assets (icons, atlases, layers, minimaps) to MinIO.
//
// Workers run inside the ingest pod (MODE=ingest). The Params (Region,
// MajorVersion, MinorVersion, ScopeKey, ScratchDir) come from environment
// variables set by JobCreator. ScopeKey is either "shared" (canonical sentinel
// tenant) or "tenants/<uuid>"; workers derive a tenant.Model from this so the
// existing per-tenant document storage continues to work.
package workers

import (
	"bytes"
	"context"
	"fmt"
	"image"
	stdpng "image/png"
	"os"
	"path/filepath"
	"strings"

	"atlas-data/canonical"
	"atlas-data/data/wztoxml"
	minio "atlas-data/storage/minio"

	"github.com/Chronicle20/atlas/libs/atlas-wz/wz"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// tenantFromParams derives a tenant.Model from the worker Params. For
// scope=shared we use the canonical sentinel UUID; for tenants/<uuid> we parse
// the suffix.
func tenantFromParams(p Params) (tenant.Model, error) {
	var id uuid.UUID
	switch {
	case p.ScopeKey == "shared":
		parsed, err := uuid.Parse(canonical.TenantUUID)
		if err != nil {
			return tenant.Model{}, fmt.Errorf("parse canonical uuid: %w", err)
		}
		id = parsed
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

// fetchArchive downloads <BucketWZ>/<scope>/regions/<region>/versions/<x.y>/<archive>
// from MinIO into ScratchDir and opens it as a wz.File. Returns the parsed
// file along with a cleanup func the caller MUST invoke (defer) to close the
// file and remove the scratch download. Unlike fetchAndSerializeArchive, this
// helper skips the XML serialization step — workers that only need to read a
// single .img out of the archive (smap.img from Base.wz, gauge data from
// UI.wz) avoid the cost of materializing the whole tree to disk.
func fetchArchive(ctx context.Context, l logrus.FieldLogger, mc *minio.Client, p Params, archive string) (*wz.File, func(), error) {
	if mc == nil {
		return nil, func() {}, fmt.Errorf("minio client unavailable")
	}
	key := fmt.Sprintf("%s/regions/%s/versions/%d.%d/%s", p.ScopeKey, p.Region, p.MajorVersion, p.MinorVersion, archive)
	localPath, err := mc.DownloadToScratch(ctx, mc.Cfg().BucketWZ, key, p.ScratchDir)
	if err != nil {
		return nil, func() {}, fmt.Errorf("download %s: %w", key, err)
	}
	f, err := wz.Open(l, localPath)
	if err != nil {
		_ = os.Remove(localPath)
		return nil, func() {}, fmt.Errorf("open %s: %w", localPath, err)
	}
	cleanup := func() {
		f.Close()
		_ = os.Remove(localPath)
	}
	return f, cleanup, nil
}

// fetchAndSerializeArchive downloads <BucketWZ>/<scope>/regions/<region>/versions/<x.y>/<archive>
// from MinIO into ScratchDir, opens it as a wz.File, and serializes it next to
// the current worker's other archives so domain readers can resolve
// cross-archive references (e.g. String.wz Eqp.img while the worker is on
// Character.wz). Returns the rootDir(p) shared layout root.
func fetchAndSerializeArchive(ctx context.Context, l logrus.FieldLogger, mc *minio.Client, p Params, archive string) (string, error) {
	if mc == nil {
		return "", fmt.Errorf("minio client unavailable")
	}
	key := fmt.Sprintf("%s/regions/%s/versions/%d.%d/%s", p.ScopeKey, p.Region, p.MajorVersion, p.MinorVersion, archive)
	localPath, err := mc.DownloadToScratch(ctx, mc.Cfg().BucketWZ, key, p.ScratchDir)
	if err != nil {
		return "", fmt.Errorf("download %s: %w", key, err)
	}
	defer os.Remove(localPath)
	f, err := wz.Open(l, localPath)
	if err != nil {
		return "", fmt.Errorf("open %s: %w", localPath, err)
	}
	defer f.Close()
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
