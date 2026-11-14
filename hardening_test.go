package ofbx

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------- float32 ('f') array coverage: real files store UVs and colors
// as float32 while most tests exercise 'd' ----------

// buildF32Geometry makes a geometry element whose UV and color layers use
// float32 arrays, like Autodesk exports do.
func buildF32Geometry(t *testing.T) *Element {
	t.Helper()
	verts := []float64{0, 0, 0, 1, 0, 0, 1, 1, 0, 0, 1, 0}
	idx := []int32{0, 1, -3, 0, 3, -2} // two triangles

	vertsProp, err := NewDoubleArrayProperty(verts, false)
	require.NoError(t, err)
	idxProp, err := NewInt32ArrayProperty(idx, false)
	require.NoError(t, err)

	el := NewElement("Geometry")
	el.AddProperty(NewInt64Property(1))
	el.AddProperty(NewStringProperty("g\x00\x01Geometry"))
	el.AddProperty(NewStringProperty("Mesh"))
	el.AddChildElement("Vertices", vertsProp)
	el.AddChildElement("PolygonVertexIndex", idxProp)
	el.AddChildElement("GeometryVersion", NewInt32Property(124))

	uvProp, err := NewFloatArrayProperty([]float32{0, 0, 1, 0, 1, 1, 0.5, 0.5}, false)
	require.NoError(t, err)
	uvLayer := el.AddChildElement("LayerElementUV", NewInt64Property(0))
	uvLayer.AddChildElement("Version", NewInt32Property(101))
	uvLayer.AddChildElement("Name", NewStringProperty("UVMap"))
	uvLayer.AddChildElement("MappingInformationType", NewStringProperty("ByPolygonVertex"))
	uvLayer.AddChildElement("ReferenceInformationType", NewStringProperty("Direct"))
	uvLayer.AddChildElement("UV", uvProp)

	colProp, err := NewFloatArrayProperty([]float32{1, 0, 0, 1, 0, 1, 0, 1, 0, 0, 1, 1}, false)
	require.NoError(t, err)
	colLayer := el.AddChildElement("LayerElementColor", NewInt64Property(0))
	colLayer.AddChildElement("Version", NewInt32Property(101))
	colLayer.AddChildElement("Name", NewStringProperty(""))
	colLayer.AddChildElement("MappingInformationType", NewStringProperty("ByPolygonVertex"))
	colLayer.AddChildElement("ReferenceInformationType", NewStringProperty("Direct"))
	colLayer.AddChildElement("Colors", colProp)
	return el
}

// TestFloat32LayersParse checks the 'f' array paths through geometry layers.
func TestFloat32LayersParse(t *testing.T) {
	geom, err := parseGeometry(&Scene{}, buildF32Geometry(t))
	require.NoError(t, err)
	require.Len(t, geom.UVs[0], 4) // 8 floats / 2
	assert.Equal(t, 0.0, geom.UVs[0][0].X())
	assert.Equal(t, 1.0, geom.UVs[0][1].X())
	require.Len(t, geom.Colors, 3) // 12 floats / 4
	assert.Equal(t, 1.0, geom.Colors[0].W())
	assert.Equal(t, 1.0, geom.Colors[1].X())
}

// TestFloat32LayersRoundTrip writes the 'f' geometry via SaveScene and reads
// it back; layers survive with values intact.
func TestFloat32LayersRoundTrip(t *testing.T) {
	doc := NewDocument()
	// splice the hand-built element in by re-using the writer on a fake
	// scene: simplest is to wrap it as the scene's Objects entry
	fake := NewDocument()
	_ = fake
	scene := &Scene{ObjectMap: map[uint64]Obj{}}
	geom := buildF32Geometry(t)
	_ = geom
	_ = scene
	_ = doc
	// covered instead by encoding the element tree directly
	el := buildF32Geometry(t)
	root := NewElement("")
	objs := root.AddChildElement("Objects")
	objs.AddChild(el)

	var buf bytes.Buffer
	require.NoError(t, Encode(&buf, root, 7400))
	reloaded, err := Load(&buf)
	require.NoError(t, err)
	require.Len(t, reloaded.Meshes, 0) // no Model wired, geometry only
	// the geometry still parsed (raw retention in the Objects section)
	objEls := reloaded.RootElement.Child("Objects").ChildrenOf("Geometry")
	require.Len(t, objEls, 1)
	g, err := parseGeometry(reloaded, objEls[0])
	require.NoError(t, err)
	require.Len(t, g.UVs[0], 4)
	assert.Equal(t, 1.0, g.UVs[0][1].X())
	require.Len(t, g.Colors, 3)
}

// ---------- multi UV sets and ByPolygon materials ----------

// TestMultiUVSetsRoundTrip fills all four UV sets and reads them back.
func TestMultiUVSetsRoundTrip(t *testing.T) {
	doc := NewDocument()
	mesh, err := doc.NewMesh("M", cubeVertices, cubeFaces)
	require.NoError(t, err)
	for set := 0; set < MaxUvs; set++ {
		uvs := make([]Point2, 0, cornerCount(cubeFaces))
		for i := 0; i < cornerCount(cubeFaces); i++ {
			uvs = append(uvs, Point2{float64(set), float64(i % 7)})
		}
		require.NoError(t, doc.SetUVs(mesh, set, uvs))
	}
	// out of range set must be rejected
	assert.Error(t, doc.SetUVs(mesh, MaxUvs, nil))

	scene := reloadDoc(t, doc)
	g := scene.Meshes[0].Geometry
	for set := 0; set < MaxUvs; set++ {
		require.NotEmpty(t, g.UVs[set], "uv set %d", set)
		assert.Equal(t, float64(set), g.UVs[set][0].X(), "uv set %d", set)
	}
}

// TestMaterialIndicesByPolygon covers per-face material assignment.
func TestMaterialIndicesByPolygon(t *testing.T) {
	doc := NewDocument()
	mesh, err := doc.NewMesh("M", cubeVertices, cubeFaces)
	require.NoError(t, err)
	m0 := doc.NewMaterial("A")
	m1 := doc.NewMaterial("B")
	doc.AddMaterial(mesh, m0)
	doc.AddMaterial(mesh, m1)

	perFace := make([]int32, len(cubeFaces))
	for i := range perFace {
		perFace[i] = int32(i % 2)
	}
	require.NoError(t, doc.SetMaterialIndices(mesh, "ByPolygon", perFace))
	// wrong count must be rejected
	assert.Error(t, doc.SetMaterialIndices(mesh, "ByPolygon", perFace[:2]))

	scene := reloadDoc(t, doc)
	g := scene.Meshes[0].Geometry
	// ByPolygon/IndexToDirect expands to per-triangle materials: 2 per face
	require.Len(t, g.Materials, 2*len(cubeFaces))
	for i, want := range perFace {
		assert.Equal(t, int(want), g.Materials[2*i], "face %d tri 0", i)
		assert.Equal(t, int(want), g.Materials[2*i+1], "face %d tri 1", i)
	}
}

// ---------- compressed arrays through the text writer ----------

// TestCompressedDocTextRoundTrip forces zlib compression and round-trips
// through EncodeText, which must inflate every array first.
func TestCompressedDocTextRoundTrip(t *testing.T) {
	doc := NewDocument(WithCompressedArrays())
	mesh, err := doc.NewMesh("M", cubeVertices, cubeFaces)
	require.NoError(t, err)
	require.NoError(t, doc.SetNormals(mesh, cubeCornerNormals()))
	require.NoError(t, doc.SetUVs(mesh, 0, cubeCornerUVs()))
	mat := doc.NewMaterial("m")
	mat.DiffuseColor = Color{0.3, 0.6, 0.9}
	doc.AddMaterial(mesh, mat)

	// confirm the arrays really are compressed in the built elements
	normProp := mesh.Geometry.Element().Child("LayerElementNormal").Child("Normals").Properties[0]
	require.Equal(t, uint32(1), normProp.Encoding)

	doc.Prepare()
	var text bytes.Buffer
	require.NoError(t, EncodeText(&text, doc.Root(), 7400))
	assert.Contains(t, text.String(), "a: ")

	scene, err := Load(&text)
	require.NoError(t, err)
	require.Len(t, scene.Meshes, 1)
	g := scene.Meshes[0].Geometry
	assert.Equal(t, cubeVertices, g.Vertices)
	assert.Equal(t, cubeCornerUVs(), g.UVs[0])
	require.Len(t, scene.Meshes[0].Materials, 1)
	assert.Equal(t, Color{0.3, 0.6, 0.9}, scene.Meshes[0].Materials[0].DiffuseColor)
}

// ---------- text dialect edge cases ----------

// TestTextCRLF checks Windows line endings parse.
func TestTextCRLF(t *testing.T) {
	src := "Root: {\r\n\tA: 1, 2.5\r\n\tB: \"x\"\r\n}\r\n"
	root, err := tokenize(bytes.NewReader([]byte(src)))
	require.NoError(t, err)
	a := root.Children[0].Child("A")
	require.NotNil(t, a)
	assert.Equal(t, int64(1), a.Properties[0].ValueI64())
	assert.Equal(t, 2.5, a.Properties[1].ValueF64())
	assert.NotNil(t, root.Children[0].Child("B"))
}

// TestTextArrayNegativeAndExponents pins harder numeric literals.
func TestTextArrayNegativeAndExponents(t *testing.T) {
	src := `R: {
	N: *5 { a: -1,-2.5,3e2,-4.5e-3,+6 }
	I: *3 { a: -10,-20,-30 }
}`
	root, err := tokenize(bytes.NewReader([]byte(src)))
	require.NoError(t, err)
	fl, err := parseArrayRawFloat64(root.Children[0].Child("N").Properties[0])
	require.NoError(t, err)
	assert.InDeltaSlice(t, []float64{-1, -2.5, 300, -0.0045, 6}, fl, 1e-9)

	ints, err := parseArrayRawInt64(root.Children[0].Child("I").Properties[0])
	require.NoError(t, err)
	assert.Equal(t, []int64{-10, -20, -30}, ints)
}

// ---------- document save semantics ----------

// TestSaveTwiceIdempotent saves the same document twice; outputs parse to
// equal scenes (timestamps aside) and Prepare can run repeatedly.
func TestSaveTwiceIdempotent(t *testing.T) {
	doc := NewDocument()
	mesh, err := doc.NewMesh("M", cubeVertices, cubeFaces)
	require.NoError(t, err)
	_ = doc.NewMaterial("m0")
	doc.AddMaterial(mesh, doc.materials[0])

	data1, err := doc.Bytes()
	require.NoError(t, err)
	doc.Prepare()
	doc.Prepare() // repeated finalization must not duplicate sections
	data2, err := doc.Bytes()
	require.NoError(t, err)

	s1, err := Load(bytes.NewReader(data1))
	require.NoError(t, err)
	s2, err := Load(bytes.NewReader(data2))
	require.NoError(t, err)
	compareScenes(t, s1, s2)

	// structural sanity: exactly one Definitions/Objects/Connections node
	for _, section := range []string{"Definitions", "Objects", "Connections"} {
		assert.Len(t, s2.RootElement.ChildrenOf(section), 1, "section %s", section)
	}
	// a document without animation writes no Takes section at all
	assert.Empty(t, s2.RootElement.ChildrenOf("Takes"))
}

// TestEmptyDocumentSave round-trips a document without objects.
func TestEmptyDocumentSave(t *testing.T) {
	doc := NewDocument()
	data, err := doc.Bytes()
	require.NoError(t, err)
	scene, err := Load(bytes.NewReader(data))
	require.NoError(t, err)
	assert.Empty(t, scene.Meshes)
	assert.NotNil(t, scene.RootNode)
}

// TestDocumentRootAccessor covers Document.Root for custom output.
func TestDocumentRootAccessor(t *testing.T) {
	doc := NewDocument()
	_, err := doc.NewMesh("M", cubeVertices, cubeFaces)
	require.NoError(t, err)
	require.NotNil(t, doc.Root())
	require.NotNil(t, doc.Root().Child("Objects"))
	require.NotNil(t, doc.Scene())
}

// TestSaveSceneOnlySupportedAnimated drops nothing on a fully modeled file.
func TestSaveSceneOnlySupportedAnimated(t *testing.T) {
	scene := loadScene(t, "testdata/anim_skeleton.fbx")
	var buf bytes.Buffer
	require.NoError(t, SaveScene(scene, &buf, OnlySupportedObjects()))
	reloaded, err := Load(&buf)
	require.NoError(t, err)
	// skin, animation and takes all survive the filter
	compareScenes(t, scene, reloaded)
}

// TestEditLargeJyj runs the edit flow on the largest sample (750KB, 51
// meshes): move the first mesh, tint a material, rebuild geometry, export.
func TestEditLargeJyj(t *testing.T) {
	scene := loadScene(t, "testdata/jyj.FBX")
	require.Greater(t, len(scene.Meshes), 10)
	mesh := scene.Meshes[0]
	require.NotNil(t, mesh.Geometry)

	require.NoError(t, SetTranslation(mesh, Point3{5, 5, 5}))
	require.NoError(t, RebuildGeometryElement(mesh))
	if len(mesh.Materials) > 0 {
		mesh.Materials[0].DiffuseColor = Color{0.5, 0.5, 0.5}
		SyncMaterial(mesh.Materials[0])
	}

	reloaded := saveAndReload(t, scene)
	rm := reloaded.Meshes[0]
	assert.Equal(t, Point3{5, 5, 5}, getLocalTranslation(rm))
	if len(rm.Materials) > 0 {
		assert.Equal(t, Color{0.5, 0.5, 0.5}, rm.Materials[0].DiffuseColor)
	}
}

// ---------- error paths ----------

// TestClusterCorruptIndexFailsCleanly guards the cluster bounds check: a
// corrupt cluster index must fail the load with an error, not panic.
func TestClusterCorruptIndexFailsCleanly(t *testing.T) {
	// assemble a minimal scene: geometry + skin + cluster with bad indexes
	doc := NewDocument()
	mesh, err := doc.NewMesh("M", cubeVertices, cubeFaces)
	require.NoError(t, err)
	bone, err := doc.NewNode("B", "LimbNode")
	require.NoError(t, err)
	skin, err := doc.NewSkin(mesh)
	require.NoError(t, err)

	// bypass validation: write an out-of-range index array directly
	idxProp, err := NewInt32ArrayProperty([]int32{0, 99999}, false)
	require.NoError(t, err)
	wProp, err := NewDoubleArrayProperty([]float64{1, 1}, false)
	require.NoError(t, err)
	tl := makeIdentity()
	tlArr := tl.ToArray()
	tlProp, err := NewDoubleArrayProperty(tlArr[:], false)
	require.NoError(t, err)

	el := NewElement("Deformer")
	el.AddProperty(NewInt64Property(777))
	el.AddProperty(NewStringProperty("Cluster\x00\x01Cluster"))
	el.AddProperty(NewStringProperty("Cluster"))
	el.AddChildElement("Indexes", idxProp)
	el.AddChildElement("Weights", wProp)
	el.AddChildElement("TransformLink", tlProp)
	cluster, err := parseCluster(doc.Scene(), el)
	require.NoError(t, err)
	doc.Scene().ObjectMap[777] = cluster
	cluster.SetID(777)
	doc.objs.AddChild(el)
	skin.Clusters = append(skin.Clusters, cluster)
	cluster.Skin = skin
	doc.connect(cluster, skin, "")
	doc.connect(bone, cluster, "")

	data, err := doc.Bytes()
	require.NoError(t, err)
	_, err = Load(bytes.NewReader(data))
	require.Error(t, err, "corrupt cluster index must fail the load")
}

// TestByteArrayErrors covers the byte array decode error paths.
func TestByteArrayErrors(t *testing.T) {
	// wrong type
	p, err := NewInt32ArrayProperty([]int32{1}, false)
	require.NoError(t, err)
	_, err = parseArrayRawBytes(p)
	require.Error(t, err)

	// uncompressed payload shorter than the count claims
	p2 := &Property{Type: ArrayBYTE, Count: 8, Encoding: 0}
	p2.value = NewDataViewBytes([]byte{1, 2, 3})
	_, err = parseArrayRawBytes(p2)
	require.Error(t, err)
}

// TestPropFloatIntTypeMatrix covers the P value converters across types.
func TestPropFloatIntTypeMatrix(t *testing.T) {
	cases := []struct {
		p *Property
		f float64
		i int
	}{
		{NewBoolProperty(true), 1, 1},
		{NewBoolProperty(false), 0, 0},
		{NewInt16Property(-300), -300, -300},
		{NewInt32Property(7), 7, 7},
		{NewInt64Property(-9), -9, -9},
		{NewFloat32Property(2.5), 2.5, 2},
		{NewFloat64Property(-3.75), -3.75, -3},
	}
	for _, c := range cases {
		assert.Equal(t, c.f, propFloat(c.p))
		assert.Equal(t, c.i, propInt(c.p))
	}
	// nil safety
	assert.Equal(t, 0.0, propFloat(nil))
	assert.Equal(t, 0, propInt(nil))
}

// TestNewBoolAndFloat32Ctors covers the remaining constructors.
func TestNewBoolAndFloat32Ctors(t *testing.T) {
	bp, err := NewBoolArrayProperty([]bool{true, false, true}, false)
	require.NoError(t, err)
	vals, err := bp.getValuesF32()
	require.NoError(t, err)
	_ = vals
	raw := bp.rawBytes()
	assert.Equal(t, []byte{1, 0, 1}, raw)

	f := NewFloat32Property(1.25)
	assert.Equal(t, float64(1.25), float64(f.value.toFloat()))
}

// TestRebuildProcessedUVFallback drives the UV branch of the processed-layer
// fallback (normals were covered by TestRebuildProcessedLayerFallback).
func TestRebuildProcessedUVFallback(t *testing.T) {
	scene := loadScene(t, "testdata/cube.fbx")
	mesh := scene.Meshes[0]
	g := mesh.Geometry

	g.raw = rawGeometry{}
	g.Element().Children = nil
	g.Normals = nil
	cornerUV := make([]Point2, 0)
	for _, f := range g.Faces {
		for i := range f {
			cornerUV = append(cornerUV, Point2{float64(i % 2), 0.25})
		}
	}
	g.UVs[0] = cornerUV
	require.NoError(t, RebuildGeometryElement(mesh))

	reloaded := saveAndReload(t, scene)
	rg := reloaded.Meshes[0].Geometry
	require.NotEmpty(t, rg.UVs[0])
	assert.Equal(t, cornerUV, rg.UVs[0])
}

// TestGlobalMatrixAndTextureAccessors covers GetGlobalMatrix and the
// texture relative filename accessor.
func TestGlobalMatrixAndTextureAccessors(t *testing.T) {
	doc := NewDocument()
	root, err := doc.NewNode("Root", "Root")
	require.NoError(t, err)
	child, err := doc.NewNode("Child", "Null")
	require.NoError(t, err)
	doc.SetParent(root, nil)
	doc.SetParent(child, root)
	require.NoError(t, SetTranslation(child, Point3{1, 0, 0}))
	require.NoError(t, SetTranslation(root, Point3{0, 2, 0}))

	scene := reloadDoc(t, doc)
	rc := scene.ObjectMap[child.ID()]
	m := GetGlobalMatrix(rc)
	// parent translation + own translation compose
	assert.Equal(t, 1.0, m.ToArray()[12])
	assert.Equal(t, 2.0, m.ToArray()[13])

	tex := doc.NewTexture("T", "/a.png", "rel/a.png")
	assert.Equal(t, "/a.png", tex.GetFileName())
	assert.NotNil(t, tex.GetRelativeFileName())
	assert.Equal(t, "rel/a.png", tex.GetRelativeFileName().String())

	// ArrayF64 accessor
	arr, err := NewDoubleArrayProperty([]float64{1.5, -2.5}, false)
	require.NoError(t, err)
	vals, err := arr.ArrayF64()
	require.NoError(t, err)
	assert.Equal(t, []float64{1.5, -2.5}, vals)
}
