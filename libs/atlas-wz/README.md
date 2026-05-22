# atlas-wz

WZ binary parser, canvas decoder, sprite atlas packer, map layer extractor, icon extractor, and pure type definitions for the manifest + map-layout JSON formats produced by ingest.

## Subpackage import policy

| Subpackage | atlas-renders may import? |
|---|---|
| `wz/`, `wz/property/` | **NO** |
| `crypto/` | **NO** |
| `canvas/` | **NO** |
| `atlas/`, `atlas/pngenc/` | **NO** |
| `mapimage/` | **NO** |
| `icons/` | **NO** |
| `manifest/` | **YES** (pure types) |
| `maplayout/` | **YES** (pure types) |

CI (`go list -deps ./services/atlas-renders/...`) enforces the policy at subpackage granularity.

## Determinism guarantee

`atlas.Pack` produces byte-identical output for identical inputs. This is load-bearing for the canonical baseline reuse. See `atlas/README.md` for the contract.

## Go version pin

Determinism depends on the vendored PNG encoder under `atlas/pngenc/` (frozen Go 1.21 `image/png`). The encoder is self-contained; nonetheless the consuming Dockerfile (atlas-data) pins `golang:1.25.5-alpine3.21` as defense-in-depth.
