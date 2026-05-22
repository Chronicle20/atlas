# atlas

Sprite atlas packer. Pack() takes a slice of named, anchored sprite images and emits one sheet PNG + one manifest JSON.

## Determinism contract

- Inputs are pre-sorted by (width desc, height desc, name asc) before packing.
- Free-rectangle list updates use slice operations only — no map iteration.
- Bin sizes grow 256, 512, 1024, 2048, 4096.
- The PNG encoder is the vendored `pngenc/` (frozen Go 1.21 image/png).
- The manifest encoder is `manifest.Marshal` (key-sorted recursive).

A "pack twice, byte-compare" test runs on every PR.
