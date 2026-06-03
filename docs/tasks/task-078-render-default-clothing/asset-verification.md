# Task 8 — Asset Verification (release-blocking gate)

**Outcome: VERIFIED PRESENT — no id swap required.**

Per plan Task 8 / PRD §9 / design §8, the four default beginner-clothing atlases must
exist in the renders/assets store for the target region/version, otherwise the feature
injects an id that resolves to a `missing atlas` warn-and-skip (FR-7) and renders nothing.

## What was verified

Atlases are read from MinIO bucket `atlas-assets` (`storage/config.go`
`MINIO_BUCKET_ASSETS`, default `atlas-assets`), key scheme
`<scope>/regions/<region>/versions/<version>/atlases/<partClass>/<id>.png` (+`.json`)
(`storage/atlas.go:29-30`). `partClassFor` (`character/composite.go:79`) maps a Top
(classification 104) to `Coat` and a Bottom (106) to `Pants`. Version string is
`fmt.Sprintf("%d.%d", major, minor)` = `83.1`.

| Gender | Slot  | partClass/id   | Present (png+json) |
|--------|-------|----------------|--------------------|
| Male   | Coat  | `Coat/1040036` | ✅ |
| Female | Coat  | `Coat/1041046` | ✅ |
| Male   | Pants | `Pants/1060026`| ✅ |
| Female | Pants | `Pants/1061039`| ✅ |

## Evidence

Live cluster `atlas-main` (env=main), MinIO pod `minio/minio-65954cc9d7-ds6qh`,
`mc find local/atlas-assets --name '<id>.*'` returned, for every one of the four ids,
both a `.png` and a `.json` object under region `GMS`, version `83.1`, in **both** the
`shared` scope and the tenant scope `tenants/ec876921-c363-4cc6-9c51-5bb8d57f9553/`:

```
shared/regions/GMS/versions/83.1/atlases/Coat/1040036.{png,json}
shared/regions/GMS/versions/83.1/atlases/Coat/1041046.{png,json}
shared/regions/GMS/versions/83.1/atlases/Pants/1060026.{png,json}
shared/regions/GMS/versions/83.1/atlases/Pants/1061039.{png,json}
(+ identical keys under tenants/ec876921-.../)
```

Because the atlas ids in `character/gender.go` are the single source of truth and all
four are ingested for the active region/version, no constant swap (Task 8 Step 2 path b)
was required.

## Verification gates (Tasks 6 & 7) — all green

- Go (`services/atlas-renders/atlas.com/renders`): `go test -race ./...` ok,
  `go vet ./...` clean, `go build ./...` clean.
- Repo root: `tools/redis-key-guard.sh` clean (exit 0); `docker buildx bake atlas-renders`
  built `atlas-renders:local` successfully.
- TS (`services/atlas-ui`): changed files lint clean (the 48 repo-wide lint errors are all
  pre-existing in untouched files); `npm run test` 736/736 pass; `npm run build` succeeds.
