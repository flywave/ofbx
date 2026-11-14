# ofbx

`ofbx` is an independent, dependency-light Go library for reading and writing
FBX files. FBX files contain geometry, animation, texture and material
information for 3D models.

It supports binary FBX 7.1 – 7.7 and text (ASCII) FBX, in and out, with a
round-trip-oriented design: everything the parser understands can be written
back losslessly, parsed scenes can be edited and re-exported, and complete
files can be built from scratch. Malformed or hostile input fails with
errors, never panics (allocation caps, fuzzed continuously).

## Install

```
go get github.com/flywave/ofbx
```

Requires Go 1.21+. The only dependency is `testify` (tests).

## Reading

```go
scene, err := ofbx.Load(reader) // binary FBX 7.1 – 7.7 and text (ASCII) FBX
for _, mesh := range scene.Meshes {
    // mesh.Geometry.Vertices / Faces / Normals / UVs / Colors / Materials
    // mesh.Materials, scene.AnimationStacks, ...
}
```

While parsing, geometries retain their original source arrays and layer
mapping modes (`Geometry.RawVertices/RawIndices/RawLayers`) so files can be
re-emitted losslessly.

## Writing

Three complementary write paths are provided, each in binary (default
version 7400; 7100/7500/7700 also supported) and text form:

### 1. Verbatim re-encode (max fidelity)

Re-serializes the exact element tree that was parsed:

```go
scene, _ := ofbx.Load(r)
ofbx.Encode(w, scene.RootElement, ofbx.DefaultWriteVersion)      // binary
ofbx.EncodeText(w, scene.RootElement, ofbx.DefaultWriteVersion)  // ASCII (debug friendly)
```

### 2. Export a parsed scene (edit mode)

Load, mutate through the edit helpers, then save. Untouched parts of the file
survive verbatim:

```go
scene, _ := ofbx.Load(r)
ofbx.SetTranslation(scene.Meshes[0], ofbx.Point3{10, 0, 0})
ofbx.SyncMaterial(scene.Meshes[0].Materials[0])  // after editing typed fields
ofbx.Rename(mesh, "NewName")
scene.Meshes[0].Geometry.Vertices[0] = ...      // modify processed data...
ofbx.RebuildGeometryElement(scene.Meshes[0])    // ...then rebuild the element
err = ofbx.SaveScene(scene, w)                  // fresh, clean file
```

Options: `WithVersion(v)` selects the output version (the source file's
version is reused by default), `OnlySupportedObjects()` drops object types
the library does not model.

### 3. Build from scratch

```go
doc := ofbx.NewDocument(ofbx.WithVersion(ofbx.FBXVersion7400))
mesh, _ := doc.NewMesh("Cube", vertices, faces)   // control points + polygons
doc.SetNormals(mesh, cornerNormals)                // ByPolygonVertex/Direct
doc.SetNormalsByVertex(mesh, perControlPoint)      // ByVertice/Direct
doc.SetUVs(mesh, 0, cornerUVs)                     // Direct
doc.SetUVsWithIndex(mesh, 0, uniqueUVs, cornerIdx) // IndexToDirect
doc.SetVertexColors(mesh, cornerColors)
doc.SetMaterialIndices(mesh, "ByPolygon", perFaceIndices)
mat := doc.NewMaterial("Skin")
mat.DiffuseColor = ofbx.Color{R: 1, G: 0, B: 0}
doc.AddMaterial(mesh, mat)
tex := doc.NewTexture("T", "/abs/tex.png", "rel/tex.png")
doc.SetTexture(mat, ofbx.DIFFUSE, tex)

bone, _ := doc.NewNode("Bone", "LimbNode")
doc.SetParent(bone, nil)

stack := doc.NewAnimationStack("Take001", 0, ktime(1*time.Second)) // also writes Takes
layer := doc.NewAnimationLayer("BaseLayer", stack)
curve, _ := doc.NewAnimationCurve("Curve", []time.Duration{0, time.Second}, []float32{0, 10})
doc.NewAnimationCurveNode("T", layer, mesh, "Lcl Translation",
    [3]*ofbx.AnimationCurve{curve, nil, nil})

skin, _ := doc.NewSkin(mesh)
doc.NewCluster(skin, bone, controlPointIndices, weights, transformLinkMatrix)

err := doc.Save(w) // or doc.Bytes(), doc.Prepare() + EncodeText(w, doc.Root(), v)
```

Runnable programs live in [examples/](examples/).

## Status

| Feature | Read | Write |
|---|---|---|
| Geometry (vertices, polygons, normals, UV, colors, materials; multi-layer) | ✅ | ✅ (verbatim/export/from scratch) |
| Transform properties (Lcl Translation/Rotation/Scaling, Geometric) | ✅ | ✅ |
| Materials (Phong) + textures | ✅ | ✅ |
| Skeleton (LimbNode hierarchy) | ✅ | ✅ |
| Skin/Cluster | ✅ | ✅ |
| Animation (Stack/Layer/CurveNode/Curve, Takes) | ✅ | ✅ |
| BlendShapes (morph targets) | ✅ | ✅ |
| Cameras, lights (typed attributes) | ✅ | ✅ |
| Embedded media (Video/Content) + file videos | ✅ | ✅ |
| GlobalSettings | ✅ | ✅ |
| Binary FBX 7.1/7.4 (u32 offsets), 7.5+/7.7 (u64 offsets) | ✅ | ✅ |
| zlib-compressed arrays | ✅ | ✅ |
| Text (ASCII) FBX | ✅ | ✅ |
Robustness: malformed/truncated/hostile input returns errors instead of
panicking or exhausting memory (allocation caps, fuzzed via `FuzzLoadEncode`).

## Performance

Loading runs at roughly 25-32 MB/s on Apple Silicon (dominated by zlib
inflation and garbage collection); encoding at ~240 MB/s. Benchmarks live in
`bench_test.go` (`go test -bench .`).

## History and credits

This project began as a fork of the `ofbx` Go port (oakmound) of
[nem0/OpenFBX](https://github.com/nem0/OpenFBX) and has since been rewritten
and extended independently: a full write pipeline (binary and text encoders,
a document builder, edit mode), ASCII reading, raw layer retention,
animation/skinning authoring, and extensive hardening and fuzz coverage.
The MIT license of the original code is preserved.
