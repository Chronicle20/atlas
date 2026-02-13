# Domain

## Extraction

### Responsibility

Orchestrates parsing of WZ binary archive files, serializing their contents to XML and extracting entity icon images to PNG.

### Core Models

#### `wz.File`

Represents a parsed WZ archive. Holds the binary reader, parsed directory tree, detected version hash, game version, and encryption key.

Accessors: `Name()`, `Root()`, `Reader()`, `ContentStart()`, `VersionHash()`, `EncryptionKey()`, `CanvasEncryptionKey()`, `ReadCanvasData(offset, size)`.

#### `wz.Directory`

Tree node within a WZ archive. Contains child directories and images. Parsed lazily from binary data. Handles element types 1 (skip), 2 (UOL offset), 3 (subdirectory), and 4 (image).

Accessors: `Name()`, `Directories()`, `Images()`.

#### `wz.Image`

Property container within a directory. Properties are parsed lazily on first access from the binary data at the stored offset. Expects a "Property" tag followed by a property list.

Accessors: `Name()`, `Properties()`.

#### Property Types

All implement the `property.Property` interface (`Name()`, `Type()`, `Children()`):

| Type | Description |
|---|---|
| `NullProperty` | Empty/null marker |
| `ShortProperty` | `int16` value |
| `IntProperty` | `int32` value (WZ compressed) |
| `LongProperty` | `int64` value (WZ compressed) |
| `FloatProperty` | `float32` value |
| `DoubleProperty` | `float64` value |
| `StringProperty` | Encrypted/encoded string |
| `SubProperty` | Nested container (imgdir) |
| `CanvasProperty` | Image data (width, height, format, dataOffset, dataSize, children) |
| `VectorProperty` | 2D coordinates (x, y) |
| `ConvexProperty` | List of child properties |
| `SoundProperty` | Sound stub (no data extraction) |
| `UOLProperty` | Symbolic link to another property path |

#### `crypto.WzKey`

AES-ECB based XOR key for string decryption. Generates a 4-byte IV repeated to 16 bytes, then expands via AES-ECB to produce an XOR table. Lazily expanded to required size.

Three IV seeds: GMS (`0x4D23C72B`), KMS (`0xB97D63E9`), Empty (`0x00000000`).

### Invariants

- WZ files must start with the `PKG1` magic header.
- Version detection brute-forces versions 1--1000 against all three encryption types; extraction fails if no match is found.
- XML output mirrors the WZ directory tree: `{outputDir}/{wzName}.wz/{dirPath}/{imageName}.img.xml`.
- Icon output is organized as: `{outputDir}/{category}/{entityId}/icon.png`.
- Extraction continues on individual file or property errors; partial results are valid.

### Processors

#### `extraction.Processor`

**Interface**: `Extract(l, ctx, xmlOnly, imagesOnly) error`

**Implementation** (`processorImpl`):
- Reads tenant context to determine output path: `{tenantId}/{region}/{majorVersion}.{minorVersion}`.
- Globs `*.wz` files from the input directory.
- For each WZ file:
  - Opens and parses via `wz.Open`.
  - If `imagesOnly` is false: serializes to XML via `xml.SerializeToDirectory`.
  - If `xmlOnly` is false: extracts icons via `image.ExtractIcons`.

#### XML Serialization (`xml.SerializeToDirectory`)

Recursively walks the WZ directory tree and writes one `.img.xml` file per image in HaRepacker-compatible format. Property types map to XML elements:

| Property | XML Element |
|---|---|
| Null | `<null>` |
| Short | `<short>` |
| Int | `<int>` |
| Long | `<long>` |
| Float | `<float>` |
| Double | `<double>` |
| String | `<string>` |
| Sub | `<imgdir>` |
| Canvas | `<canvas>` |
| Vector | `<vector>` |
| Convex | `<extended>` |
| Sound | `<sound>` |
| UOL | `<uol>` |

#### Icon Extraction (`image.ExtractIcons`)

Dispatches by WZ file name:

| WZ File | Category | Canvas Location |
|---|---|---|
| `Npc.wz` | `npc` | `stand/0` -> `move/0` -> first canvas in any sub |
| `Mob.wz` | `mob` | `stand/0` -> `move/0` -> first canvas in any sub |
| `Reactor.wz` | `reactor` | `0/0` |
| `Item.wz` | `item` | `{category}/{itemId}/info/icon` |
| `Skill.wz` | `skill` | `{bookId}/skill/{skillId}/icon` |

Canvas data is read from the WZ file, decompressed (zlib or encrypted block format), decoded from the pixel format, and written as PNG.

Supported pixel formats: BGRA4444, BGRA8888, ARGB1555, BGR565, BlockRGB565, DXT3, DXT5, DXT3Gray.
