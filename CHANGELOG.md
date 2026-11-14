# Changelog

## Unreleased

Complete read/write FBX library, developed independently from the
oakmound/ofbx fork it grew out of.

### Reading
- binary FBX 7.1 – 7.7 (u32 and u64 record offsets) and text (ASCII) FBX,
  including genuine Autodesk ASCII dialect quirks: bare-comma placeholders,
  unescaped backslash paths, base64 embedded media, values continuing on
  the next line
- geometry (multi-layer normals/UVs/colors/tangents/materials with raw
  retention for lossless re-emission), transforms with pivots and
  rotation orders, Phong materials, textures, skeleton hierarchies,
  skin/cluster, animation stacks/layers/curve nodes/curves, takes,
  cameras, lights, blend shapes, embedded and file-linked video media
- hostile input fails with errors, never panics (allocation caps,
  bounds checks, fuzzed continuously by FuzzLoadEncode/FuzzEncodeText)

### Writing
- `Encode`: verbatim binary re-encode of a parsed element tree
- `SaveScene`: clean re-export with edit helpers (SetTranslation/
  SetRotation/SetScaling, SyncMaterial, Rename, SetTextureFilename,
  RebuildGeometryElement, SetCurveKeys, SetLclVec3/SetLclDouble)
- `Document`: from-scratch authoring (meshes with all layer modes,
  materials, textures incl. embedded media, nodes, cameras, lights,
  skins/clusters, blend shapes, full animation graphs with takes)
- `EncodeText`: ASCII output readable by third-party importers
  (validated against the assimp CLI on every test sample)

### Engineering
- dependencies reduced to testify only; go 1.21; built-in vector types
- benchmarks (`go test -bench .`): ~25-32 MB/s load, ~240 MB/s encode
- CI: vet, race tests, fuzz smoke, example builds (GitHub Actions)
- 87% statement coverage; real-world corpus of eleven files from
  Autodesk FBX SDK, Blender and assimp exporters
