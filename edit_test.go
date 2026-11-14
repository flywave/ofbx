package ofbx

import (
	"bytes"
	"os"
	"testing"
	"time"

	"github.com/oakmound/oak/v2/alg/floatgeom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func loadScene(t *testing.T, path string) *Scene {
	t.Helper()
	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()
	scene, err := Load(f)
	require.NoError(t, err)
	return scene
}

func saveAndReload(t *testing.T, scene *Scene, opts ...DocOption) *Scene {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, SaveScene(scene, &buf, opts...))
	reloaded, err := Load(&buf)
	require.NoError(t, err)
	return reloaded
}

// TestEditTranslationMutatesModel checks that editing a transform property
// survives a SaveScene round trip.
func TestEditTranslationMutatesModel(t *testing.T) {
	scene := loadScene(t, "testdata/cube.fbx")
	mesh := scene.Meshes[0]

	want := floatgeom.Point3{10, 20, 30}
	require.NoError(t, SetTranslation(mesh, want))
	require.NoError(t, SetScaling(mesh, floatgeom.Point3{2, 2, 2}))

	reloaded := saveAndReload(t, scene)
	require.Len(t, reloaded.Meshes, 1)
	rm := reloaded.Meshes[0]
	assert.Equal(t, want, getLocalTranslation(rm))
	assert.Equal(t, floatgeom.Point3{2, 2, 2}, getLocalScaling(rm))
	// the rest of the scene is untouched
	compareGeometries(t, mesh.Geometry, rm.Geometry)
}

// TestEditSyncLoadedMaterial checks that typed edits to a parsed material
// survive SaveScene after SyncMaterial.
func TestEditSyncLoadedMaterial(t *testing.T) {
	scene := loadScene(t, "testdata/cube.fbx")
	mesh := scene.Meshes[0]
	require.NotEmpty(t, mesh.Materials)

	mat := mesh.Materials[0]
	mat.DiffuseColor = Color{0.25, 0.5, 0.75}
	mat.ShininessExponent = 33
	SyncMaterial(mat)

	reloaded := saveAndReload(t, scene)
	require.Len(t, reloaded.Meshes[0].Materials, 1)
	rm := reloaded.Meshes[0].Materials[0]
	assert.Equal(t, Color{0.25, 0.5, 0.75}, rm.DiffuseColor)
	assert.Equal(t, float64(33), rm.ShininessExponent)
}

// TestEditRebuildPreservesDocLayers rebuilds a Document-created mesh and
// verifies that setter-written layers survive.
func TestEditRebuildPreservesDocLayers(t *testing.T) {
	doc := NewDocument()
	mesh, err := doc.NewMesh("Cube", cubeVertices, cubeFaces)
	require.NoError(t, err)
	require.NoError(t, doc.SetNormals(mesh, cubeCornerNormals()))
	require.NoError(t, doc.SetUVs(mesh, 0, cubeCornerUVs()))

	mesh.Geometry.Vertices[0] = floatgeom.Point3{5, 5, 5}
	require.NoError(t, RebuildGeometryElement(mesh))

	data, err := doc.Bytes()
	require.NoError(t, err)
	scene, err := Load(bytes.NewReader(data))
	require.NoError(t, err)

	g := scene.Meshes[0].Geometry
	assert.Equal(t, floatgeom.Point3{5, 5, 5}, g.Vertices[0])
	assert.Equal(t, normalized(cubeCornerNormals()), g.Normals)
	assert.Equal(t, cubeCornerUVs(), g.UVs[0])
}

// TestEditRebuildGeometry modifies control points, rebuilds the geometry
// element and verifies the modification plus the retained layer modes.
func TestEditRebuildGeometry(t *testing.T) {
	scene := loadScene(t, "testdata/cube.fbx")
	mesh := scene.Meshes[0]
	// make sure the file's normals survive as a raw layer
	require.NotEmpty(t, mesh.Geometry.RawLayers())

	mesh.Geometry.Vertices[0] = floatgeom.Point3{99, 99, 99}
	require.NoError(t, RebuildGeometryElement(mesh))

	reloaded := saveAndReload(t, scene)
	require.Len(t, reloaded.Meshes, 1)
	g := reloaded.Meshes[0].Geometry
	require.NotEmpty(t, g.Vertices)
	assert.Equal(t, floatgeom.Point3{99, 99, 99}, g.Vertices[0])
	assert.Equal(t, mesh.Geometry.Faces, g.Faces)

	// retained layer modes survive the rebuild
	var rawReload, rawOrig []rawLayer
	rawOrig = mesh.Geometry.RawLayers()
	rawReload = g.RawLayers()
	require.Equal(t, len(rawOrig), len(rawReload))
	for i := range rawOrig {
		assert.Equal(t, rawOrig[i].mapping, rawReload[i].mapping, "layer %d mapping", i)
		assert.Equal(t, rawOrig[i].reference, rawReload[i].reference, "layer %d reference", i)
		assert.Equal(t, rawOrig[i].dataF, rawReload[i].dataF, "layer %d data", i)
	}
}

// TestAnimationCreation builds a small animated scene from scratch.
func TestAnimationCreation(t *testing.T) {
	doc := NewDocument()
	mesh, err := doc.NewMesh("AnimCube", cubeVertices, cubeFaces)
	require.NoError(t, err)

	stack := doc.NewAnimationStack("Take001", 0, stdTimeToFbxTime(time.Second))
	layer := doc.NewAnimationLayer("BaseLayer", stack)

	mkCurve := func(v0, v1 float32) *AnimationCurve {
		c, err := doc.NewAnimationCurve("Curve", []time.Duration{0, time.Second}, []float32{v0, v1})
		require.NoError(t, err)
		return c
	}
	curves := [3]*AnimationCurve{
		mkCurve(0, 10),
		mkCurve(0, 20),
		mkCurve(0, 30),
	}
	_, err = doc.NewAnimationCurveNode("T", layer, mesh, "Lcl Translation", curves)
	require.NoError(t, err)

	data, err := doc.Bytes()
	require.NoError(t, err)
	scene, err := Load(bytes.NewReader(data))
	require.NoError(t, err)

	require.Len(t, scene.AnimationStacks, 1)
	st := scene.AnimationStacks[0]
	_ = st
	require.Len(t, st.Layers, 1)
	lay := st.Layers[0]
	require.Len(t, lay.CurveNodes, 1)
	cn := lay.CurveNodes[0]
	require.NotNil(t, cn.Bone)
	assert.Equal(t, mesh.ID(), cn.Bone.ID())
	assert.Equal(t, "Lcl Translation", cn.BoneLinkProp)

	for i := 0; i < 3; i++ {
		require.NotNil(t, cn.Curves[i].Curve, "curve %d", i)
		times := cn.Curves[i].Curve.Times
		require.Len(t, times, 2)
		// times survive at FBX KTime precision; the std-time conversion
		// truncates a float and can lose a microsecond
		assert.InDelta(t, float64(time.Second), float64(times[1]), float64(2*time.Microsecond))
	}
	assert.Equal(t, []float32{0, 10}, cn.Curves[0].Curve.Values)
	assert.Equal(t, []float32{0, 20}, cn.Curves[1].Curve.Values)
	assert.Equal(t, []float32{0, 30}, cn.Curves[2].Curve.Values)

	// sampled transform at t=0.5s: linear interpolation halfway
	pos := cn.GetNodeLocalTransform(0.5)
	assert.InDelta(t, 5.0, pos.X(), 1e-4)
	assert.InDelta(t, 10.0, pos.Y(), 1e-4)
	assert.InDelta(t, 15.0, pos.Z(), 1e-4)
}

// TestSkinCreation builds a skinned cube with one bone.
func TestSkinCreation(t *testing.T) {
	doc := NewDocument()
	mesh, err := doc.NewMesh("SkinnedCube", cubeVertices, cubeFaces)
	require.NoError(t, err)

	bone, err := doc.NewNode("Bone", "LimbNode")
	require.NoError(t, err)
	require.NoError(t, SetTranslation(bone, floatgeom.Point3{0, 0.5, 0}))
	doc.SetParent(bone, nil)

	skin, err := doc.NewSkin(mesh)
	require.NoError(t, err)
	// every control point fully weighted to the bone
	indices := make([]int, len(cubeVertices))
	weights := make([]float64, len(cubeVertices))
	for i := range indices {
		indices[i] = i
		weights[i] = 1
	}
	_, err = doc.NewCluster(skin, bone, indices, weights, makeIdentity())
	require.NoError(t, err)

	data, err := doc.Bytes()
	require.NoError(t, err)
	scene, err := Load(bytes.NewReader(data))
	require.NoError(t, err)

	require.Len(t, scene.Meshes, 1)
	g := scene.Meshes[0].Geometry
	require.NotNil(t, g.Skin)
	require.Len(t, g.Skin.Clusters, 1)
	cluster := g.Skin.Clusters[0]
	require.NotNil(t, cluster.Link)
	assert.Equal(t, bone.ID(), cluster.Link.ID())
	// TransformLink survives (identity written and read back)
	assert.Equal(t, makeIdentity(), cluster.TransformLink)
	// weights survived (indices are remapped to triangle vertices on load,
	// so compare total weight: one weight per triangle-vertex occurrence)
	expected := 0
	for _, f := range cubeFaces {
		expected += 3 * (len(f) - 2) // fan triangulation vertex count
	}
	sum := 0.0
	for _, w := range cluster.Weights {
		sum += w
	}
	assert.InDelta(t, float64(expected), sum, 1e-6)
}

// TestSecondsToFbxTime guards the unit conversion direction.
func TestSecondsToFbxTime(t *testing.T) {
	assert.Equal(t, int64(46186158000), secondsToFbxTime(1))
	assert.Equal(t, int64(46186158000*2), secondsToFbxTime(2))
	assert.InDelta(t, 1.0, fbxTimeToSeconds(secondsToFbxTime(1)), 1e-12)
	// fbxTimetoStdTime divides through a float and truncates; allow 2µs
	assert.InDelta(t, float64(time.Second), float64(fbxTimetoStdTime(secondsToFbxTime(1))), 2000)
}
