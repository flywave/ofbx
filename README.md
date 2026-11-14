# ofbx 

`ofbx` is a library for reading and writing FBX files. FBX files contain geometry, animation, texture and material information for 3D models. 

This is a Go port of [github.com/nem0/OpenFBX](https://github.com/nem0/OpenFBX), extended with a full write pipeline.

## Reading

```go
scene, err := ofbx.Load(reader) // binary FBX 7.1 – 7.7 (text FBX is rejected with ofbx.ErrASCII)
for _, mesh := range scene.Meshes {
    // mesh.Geometry.Vertices / Faces / Normals / UVs / Colors / Materials
    // mesh.Materials, scene.AnimationStacks, ...
}
```

While parsing, geometries retain their original source arrays and layer
mapping modes (`Geometry.RawVertices/RawIndices/RawLayers`) so files can be
re-emitted losslessly.

## Writing

Three complementary write paths are provided, all producing binary FBX
(default version 7400; 7100/7500/7700 also supported):

### 1. Verbatim re-encode (max fidelity)

Re-serializes the exact element tree that was parsed:

```go
scene, _ := ofbx.Load(r)
ofbx.Encode(w, scene.RootElement, ofbx.DefaultWriteVersion)
```

### 2. Export a parsed scene (edit mode)

Load, mutate through the edit helpers, then save. Untouched parts of the file
survive verbatim:

```go
scene, _ := ofbx.Load(r)
ofbx.SetTranslation(scene.Meshes[0], ofbxVec(10, 0, 0))
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
doc.SetUVs(mesh, 0, cornerUVs)
mat := doc.NewMaterial("Skin")
mat.DiffuseColor = ofbx.Color{R: 1, G: 0, B: 0}
doc.AddMaterial(mesh, mat)
tex := doc.NewTexture("T", "/abs/tex.png", "rel/tex.png")
doc.SetTexture(mat, ofbx.DIFFUSE, tex)

bone, _ := doc.NewNode("Bone", "LimbNode")
doc.SetParent(bone, nil)

stack := doc.NewAnimationStack("Take001", 0, ktime(1*time.Second))
layer := doc.NewAnimationLayer("BaseLayer", stack)
curve, _ := doc.NewAnimationCurve("Curve", []time.Duration{0, time.Second}, []float32{0, 10})
doc.NewAnimationCurveNode("T", layer, mesh, "Lcl Translation",
    [3]*ofbx.AnimationCurve{curve, nil, nil})

skin, _ := doc.NewSkin(mesh)
doc.NewCluster(skin, bone, controlPointIndices, weights, transformLinkMatrix)

err := doc.Save(w) // or doc.Bytes()
```

## Status

| Feature | Read | Write |
|---|---|---|
| Geometry (vertices, polygons, normals, UV, colors, materials) | ✅ | ✅ (verbatim/export/from scratch) |
| Transform properties (Lcl Translation/Rotation/Scaling, Geometric) | ✅ | ✅ |
| Materials (Phong) + textures | ✅ | ✅ |
| Skeleton (LimbNode hierarchy) | ✅ | ✅ |
| Skin/Cluster | ✅ | ✅ |
| Animation (Stack/Layer/CurveNode/Curve) | ✅ | ✅ |
| Takes, GlobalSettings | ✅ | ✅ |
| Binary FBX 7.1/7.4 (u32 offsets), 7.5+/7.7 (u64 offsets) | ✅ | ✅ |
| zlib-compressed arrays | ✅ | ✅ |
| Text (ASCII) FBX | ❌ (`ofbx.ErrASCII`) | ❌ |
| BlendShapes, cameras/lights, embedded media | ❌ | ❌ |

The internal API is still evolving.
