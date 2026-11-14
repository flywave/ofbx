package ofbx

import (
	"bytes"
	"testing"
	"time"

	"github.com/oakmound/oak/v2/alg/floatgeom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMultiLayerNormalsRetained writes two normal layers and verifies all of
// them survive a round trip (processed fields from the first, raw retention
// for every layer).
func TestMultiLayerNormalsRetained(t *testing.T) {
	doc := NewDocument()
	mesh, err := doc.NewMesh("M", cubeVertices, cubeFaces)
	require.NoError(t, err)
	first := cubeCornerNormals()
	second := make([]floatgeom.Point3, len(first))
	for i := range second {
		second[i] = first[i].MulConst(-1)
	}
	require.NoError(t, doc.SetNormals(mesh, first))
	require.NoError(t, doc.SetNormals(mesh, second))

	scene := reloadDoc(t, doc)
	g := scene.Meshes[0].Geometry
	// processed normals come from the first layer
	assert.Equal(t, normalized(first), g.Normals)
	// both layers retained
	var rawNormalLayers []rawLayer
	for _, l := range g.RawLayers() {
		if l.kind == layerNormal {
			rawNormalLayers = append(rawNormalLayers, l)
		}
	}
	require.Len(t, rawNormalLayers, 2)
	assert.Equal(t, flattenPoint3s(first), rawNormalLayers[0].dataF)
	assert.Equal(t, flattenPoint3s(second), rawNormalLayers[1].dataF)
}

// TestUnsupportedMaterialMappingLoads checks that a material mapping outside
// the supported set no longer fails the whole load; the layer is retained.
func TestUnsupportedMaterialMappingLoads(t *testing.T) {
	doc := NewDocument()
	mesh, err := doc.NewMesh("M", cubeVertices, cubeFaces)
	require.NoError(t, err)
	// hand-craft a ByPolygonVertex material layer (not in the processed set)
	geomEl := mesh.Geometry.Element()
	layer := geomEl.AddChildElement("LayerElementMaterial", NewInt64Property(0))
	layer.AddChildElement("Version", NewInt32Property(101))
	layer.AddChildElement("Name", NewStringProperty(""))
	layer.AddChildElement("MappingInformationType", NewStringProperty("ByPolygonVertex"))
	layer.AddChildElement("ReferenceInformationType", NewStringProperty("IndexToDirect"))
	mats, err := NewInt32ArrayProperty([]int32{0, 0, 0, 0, 0, 0}, false)
	require.NoError(t, err)
	layer.AddChildElement("Materials", mats)
	addLayerEntryStatic(geomEl, "LayerElementMaterial")

	scene := reloadDoc(t, doc) // must not error
	g := scene.Meshes[0].Geometry
	found := false
	for _, l := range g.RawLayers() {
		if l.kind == layerMaterial {
			found = true
			assert.Equal(t, "ByPolygonVertex", l.mapping)
		}
	}
	assert.True(t, found, "raw material layer retained")
}

// TestSetUVsWithIndex covers the IndexToDirect UV writer.
func TestSetUVsWithIndex(t *testing.T) {
	doc := NewDocument()
	mesh, err := doc.NewMesh("M", cubeVertices, cubeFaces)
	require.NoError(t, err)

	unique := []floatgeom.Point2{{0, 0}, {1, 0}, {1, 1}, {0, 1}}
	idx := make([]int32, 0, cornerCount(cubeFaces))
	for _, f := range cubeFaces {
		for i := range f {
			idx = append(idx, int32(i%4))
		}
	}
	require.NoError(t, doc.SetUVsWithIndex(mesh, 0, unique, idx))

	// out of range index must be rejected
	err = doc.SetUVsWithIndex(mesh, 1, unique, []int32{99})
	assert.Error(t, err)

	scene := reloadDoc(t, doc)
	g := scene.Meshes[0].Geometry
	require.Len(t, g.UVs[0], cornerCount(cubeFaces))
	for i, v := range idx {
		assert.Equal(t, unique[v], g.UVs[0][i], "corner %d", i)
	}
}

// TestSetNormalsByVertex covers the per-control-point normal writer.
func TestSetNormalsByVertex(t *testing.T) {
	doc := NewDocument()
	mesh, err := doc.NewMesh("M", cubeVertices, cubeFaces)
	require.NoError(t, err)

	perVertex := make([]floatgeom.Point3, len(cubeVertices))
	for i := range perVertex {
		perVertex[i] = floatgeom.Point3{0, 1, 0}
	}
	require.NoError(t, doc.SetNormalsByVertex(mesh, perVertex))

	// wrong count must be rejected
	assert.Error(t, doc.SetNormalsByVertex(mesh, perVertex[:3]))

	scene := reloadDoc(t, doc)
	g := scene.Meshes[0].Geometry
	require.Len(t, g.Normals, cornerCount(cubeFaces))
	for _, n := range g.Normals {
		assert.Equal(t, floatgeom.Point3{0, 1, 0}, n)
	}
}

// TestRenameAndTextureFilename covers the object edit helpers.
func TestRenameAndTextureFilename(t *testing.T) {
	doc := NewDocument()
	mesh, err := doc.NewMesh("Cube", cubeVertices, cubeFaces)
	require.NoError(t, err)
	mat := doc.NewMaterial("OldName")
	doc.AddMaterial(mesh, mat)
	tex := doc.NewTexture("OldTex", "/old/tex.png", "old/tex.png")
	require.NoError(t, doc.SetTexture(mat, DIFFUSE, tex))

	require.NoError(t, Rename(mesh, "NewCube"))
	require.NoError(t, Rename(mat, "NewMat"))
	require.NoError(t, Rename(tex, "NewTex"))
	SetTextureFilename(tex, "/new/tex.png", "new/tex.png")

	data, err := doc.Bytes()
	require.NoError(t, err)
	scene, err := Load(bytes.NewReader(data))
	require.NoError(t, err)

	m := scene.Meshes[0]
	assert.Contains(t, m.Name(), "NewCube")
	require.Len(t, m.Materials, 1)
	assert.Contains(t, m.Materials[0].Name(), "NewMat")
	require.NotNil(t, m.Materials[0].Textures[DIFFUSE])
	assert.Equal(t, "/new/tex.png", m.Materials[0].Textures[DIFFUSE].GetFileName())
}

// TestEditOn7700File runs the edit flow against a u64-offset file.
func TestEditOn7700File(t *testing.T) {
	scene := loadScene(t, "testdata/FBXcs2.fbx")
	require.NotEmpty(t, scene.Meshes)
	mesh := scene.Meshes[0]

	require.NoError(t, SetTranslation(mesh, vec3(1, 2, 3)))
	mesh.Geometry.Vertices[0] = vec3(7, 7, 7)
	require.NoError(t, RebuildGeometryElement(mesh))

	reloaded := saveAndReload(t, scene)
	require.Len(t, reloaded.Meshes, len(scene.Meshes))
	rm := reloaded.Meshes[0]
	assert.Equal(t, vec3(1, 2, 3), getLocalTranslation(rm))
	assert.Equal(t, vec3(7, 7, 7), rm.Geometry.Vertices[0])
	compareGeometries(t, mesh.Geometry, rm.Geometry)
}

// TestAnimationTakesWritten verifies the Takes section for created stacks.
func TestAnimationTakesWritten(t *testing.T) {
	doc := NewDocument()
	mesh, err := doc.NewMesh("M", cubeVertices, cubeFaces)
	require.NoError(t, err)
	start, stop := int64(0), stdTimeToFbxTime(2e9)
	stack := doc.NewAnimationStack("Take 001", start, stop)
	layer := doc.NewAnimationLayer("L", stack)
	curve, err := doc.NewAnimationCurve("c", []time.Duration{0, 2e9}, []float32{0, 5})
	require.NoError(t, err)
	_, err = doc.NewAnimationCurveNode("n", layer, mesh, "Lcl Translation", [3]*AnimationCurve{curve, nil, nil})
	require.NoError(t, err)

	scene := reloadDoc(t, doc)
	take := scene.GetTakeInfo("Take 001")
	require.NotNil(t, take, "take info must be written and parsed")
	assert.InDelta(t, 2.0, take.localTimeTo-take.localTimeFrom, 1e-4)

	require.Len(t, scene.AnimationStacks, 1)
	// Definitions gained templates; verify a Model template exists
	defs := scene.RootElement.Child("Definitions")
	require.NotNil(t, defs)
	found := false
	for _, c := range defs.Children {
		if c.ID.String() == "ObjectType" && len(c.Properties) > 0 && c.Properties[0].value.String() == "AnimationStack" {
			require.NotNil(t, c.Child("PropertyTemplate"))
			found = true
		}
	}
	assert.True(t, found, "AnimationStack object type entry with template")
}

// TestClusterTransformMatrix covers the optional Transform matrix.
func TestClusterTransformMatrix(t *testing.T) {
	doc := NewDocument()
	mesh, err := doc.NewMesh("M", cubeVertices, cubeFaces)
	require.NoError(t, err)
	bone, err := doc.NewNode("Bone", "LimbNode")
	require.NoError(t, err)
	skin, err := doc.NewSkin(mesh)
	require.NoError(t, err)

	transform := ScalingMatrix(vec3(2, 2, 2))
	_, err = doc.NewCluster(skin, bone, []int{0, 1}, []float64{1, 1}, makeIdentity(), transform)
	require.NoError(t, err)

	scene := reloadDoc(t, doc)
	c := scene.Meshes[0].Geometry.Skin.Clusters[0]
	assert.Equal(t, transform, c.Transform)
	assert.Equal(t, makeIdentity(), c.TransformLink)
}

// TestFocalLengthCurveNode covers the single-curve special case.
func TestFocalLengthCurveNode(t *testing.T) {
	doc := NewDocument()
	node, err := doc.NewNode("Camera", "Null")
	require.NoError(t, err)
	stack := doc.NewAnimationStack("Take", 0, stdTimeToFbxTime(time.Second))
	layer := doc.NewAnimationLayer("L", stack)
	curve, err := doc.NewAnimationCurve("f", []time.Duration{0, time.Second}, []float32{35, 50})
	require.NoError(t, err)
	_, err = doc.NewAnimationCurveNode("FocalLength", layer, node, "FocalLength", [3]*AnimationCurve{curve, nil, nil})
	require.NoError(t, err)

	scene := reloadDoc(t, doc)
	require.Len(t, scene.AnimationStacks, 1)
	cn := scene.AnimationStacks[0].Layers[0].CurveNodes[0]
	require.NotNil(t, cn.Curves[0].Curve)
	assert.Nil(t, cn.Curves[1].Curve)
	assert.Nil(t, cn.Curves[2].Curve)
	assert.Equal(t, "FocalLength", cn.BoneLinkProp)
	assert.InDelta(t, 42.5, float64(cn.GetNodeLocalTransform(0.5).X()), 1e-3)
}

// TestRealAnimatedSkeletonRoundTrip exercises the real animated, skinned
// sample (assimp's animation_with_skeleton.fbx) through all write paths.
func TestRealAnimatedSkeletonRoundTrip(t *testing.T) {
	scene := loadScene(t, "testdata/anim_skeleton.fbx")
	require.NotEmpty(t, scene.Meshes)
	require.NotEmpty(t, scene.AnimationStacks)
	require.NotEmpty(t, scene.TakeInfos)

	mesh := scene.Meshes[0]
	require.NotNil(t, mesh.Geometry.Skin)
	require.NotEmpty(t, mesh.Geometry.Skin.Clusters)
	require.NotEmpty(t, mesh.Animations())

	// the animation actually animates the mesh or its bones
	stack := scene.AnimationStacks[0]
	require.NotEmpty(t, stack.Layers)
	totalCurves := 0
	for _, l := range stack.Layers {
		for _, cn := range l.CurveNodes {
			for i := 0; i < 3; i++ {
				if cn.Curves[i].Curve != nil {
					totalCurves++
				}
			}
		}
	}
	require.NotZero(t, totalCurves, "sample must contain animation curves")

	// verbatim binary re-encode
	var buf bytes.Buffer
	require.NoError(t, Encode(&buf, scene.RootElement, 7400))
	scene2, err := Load(&buf)
	require.NoError(t, err)
	compareScenes(t, scene, scene2)

	// clean SaveScene export (same version)
	var buf2 bytes.Buffer
	require.NoError(t, SaveScene(scene, &buf2))
	scene3, err := Load(&buf2)
	require.NoError(t, err)
	compareScenes(t, scene, scene3)

	// ASCII export
	var text bytes.Buffer
	require.NoError(t, EncodeText(&text, scene.RootElement, 7400))
	scene4, err := Load(&text)
	require.NoError(t, err)
	compareScenes(t, scene, scene4)
}
