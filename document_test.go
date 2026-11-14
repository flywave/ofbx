package ofbx

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// cubeVertices/cubeFaces describe a unit quad cube (8 control points, 6 quads).
var cubeVertices = []Point3{
	{-1, -1, -1}, {1, -1, -1}, {1, 1, -1}, {-1, 1, -1},
	{-1, -1, 1}, {1, -1, 1}, {1, 1, 1}, {-1, 1, 1},
}

var cubeFaces = [][]int{
	{0, 1, 2, 3}, // -Z
	{4, 5, 6, 7}, // +Z (wound consistently enough for a data test)
	{0, 1, 5, 4},
	{2, 3, 7, 6},
	{0, 3, 7, 4},
	{1, 2, 6, 5},
}

// cornerCount returns the total number of polygon vertices.
func cornerCount(faces [][]int) int {
	n := 0
	for _, f := range faces {
		n += len(f)
	}
	return n
}

func cubeCornerNormals() []Point3 {
	faceNormal := func(f []int) Point3 {
		a := cubeVertices[f[0]]
		b := cubeVertices[f[1]]
		c := cubeVertices[f[2]]
		u := a.Sub(b)
		v := c.Sub(b)
		return Point3{
			u.Y()*v.Z() - u.Z()*v.Y(),
			u.Z()*v.X() - u.X()*v.Z(),
			u.X()*v.Y() - u.Y()*v.X(),
		}
	}
	normals := make([]Point3, 0, cornerCount(cubeFaces))
	for _, f := range cubeFaces {
		n := faceNormal(f)
		for range f {
			normals = append(normals, n)
		}
	}
	return normals
}

func cubeCornerUVs() []Point2 {
	uvs := make([]Point2, 0, cornerCount(cubeFaces))
	for _, f := range cubeFaces {
		for i := range f {
			uvs = append(uvs, Point2{float64(i%4) / 3, 0})
		}
	}
	return uvs
}

// normalized returns unit length copies of the input vectors.
func normalized(pts []Point3) []Point3 {
	out := make([]Point3, len(pts))
	for i, p := range pts {
		out[i] = p.Normalize()
	}
	return out
}

// TestDocumentBuildsCube writes a textured, material-assigned cube from
// scratch and re-reads it.
func TestDocumentBuildsCube(t *testing.T) {
	doc := NewDocument()

	mesh, err := doc.NewMesh("Cube", cubeVertices, cubeFaces)
	require.NoError(t, err)
	require.NoError(t, doc.SetNormals(mesh, cubeCornerNormals()))
	require.NoError(t, doc.SetUVs(mesh, 0, cubeCornerUVs()))
	require.NoError(t, doc.SetMaterialIndices(mesh, "AllSame", []int32{0}))

	mat := doc.NewMaterial("CubeMaterial")
	mat.DiffuseColor = Color{0.5, 0.25, 0.75}
	mat.DiffuseFactor = 0.9
	mat.ShininessExponent = 42
	doc.AddMaterial(mesh, mat)

	tex := doc.NewTexture("CubeTex", "/abs/tex.png", "rel/tex.png")
	require.NoError(t, doc.SetTexture(mat, DIFFUSE, tex))

	data, err := doc.Bytes()
	require.NoError(t, err)

	scene, err := Load(bytes.NewReader(data))
	require.NoError(t, err)

	require.Len(t, scene.Meshes, 1)
	m := scene.Meshes[0]
	assert.Contains(t, m.Name(), "Cube")
	require.NotNil(t, m.Geometry)

	g := m.Geometry
	assert.Equal(t, cubeVertices, g.Vertices)
	require.Len(t, g.Faces, len(cubeFaces))
	for i, f := range cubeFaces {
		assert.Equal(t, f, g.Faces[i], "face %d", i)
	}
	// Load normalizes processed normals via MulDirection; the exact written
	// values survive in the raw retention.
	assert.Equal(t, normalized(cubeCornerNormals()), g.Normals)
	assert.Equal(t, cubeCornerUVs(), g.UVs[0])
	var rawNormals []float64
	for _, l := range g.RawLayers() {
		if l.kind == layerNormal {
			rawNormals = l.dataF
		}
	}
	assert.Equal(t, flattenPoint3s(cubeCornerNormals()), rawNormals)

	require.Len(t, m.Materials, 1)
	rm := m.Materials[0]
	assert.Equal(t, Color{0.5, 0.25, 0.75}, rm.DiffuseColor)
	assert.Equal(t, 0.9, rm.DiffuseFactor)
	assert.Equal(t, float64(42), rm.ShininessExponent)
	require.NotNil(t, rm.Textures[DIFFUSE])
	assert.Equal(t, "/abs/tex.png", rm.Textures[DIFFUSE].GetFileName())

	// settings defaults were applied (Y up, Z front, X right, unit scale)
	assert.Equal(t, 1, int(scene.Settings.UpAxis))
	assert.Equal(t, 2, int(scene.Settings.FrontAxis))
	assert.Equal(t, 0, int(scene.Settings.CoordAxis))
	assert.Equal(t, float32(1), scene.Settings.UnitScaleFactor)
}

// TestDocumentBuildsNodeHierarchy checks nodes and parent links.
func TestDocumentBuildsNodeHierarchy(t *testing.T) {
	doc := NewDocument()
	root, err := doc.NewNode("RootNode", "Root")
	require.NoError(t, err)
	bone, err := doc.NewNode("Bone1", "LimbNode")
	require.NoError(t, err)
	bone2, err := doc.NewNode("Bone2", "LimbNode")
	require.NoError(t, err)
	doc.SetParent(root, nil)
	doc.SetParent(bone, root)
	doc.SetParent(bone2, bone)

	data, err := doc.Bytes()
	require.NoError(t, err)
	scene, err := Load(bytes.NewReader(data))
	require.NoError(t, err)

	boneObj := scene.ObjectMap[bone.ID()]
	require.NotNil(t, boneObj)
	parent := getParent(boneObj)
	require.NotNil(t, parent)
	assert.Equal(t, root.ID(), parent.ID())

	// LimbNode carried a Skeleton node attribute
	require.NotNil(t, boneObj.NodeAttribute())
	na, ok := boneObj.NodeAttribute().(*NodeAttribute)
	require.True(t, ok)
	assert.Equal(t, "Skeleton", na.Attribute.String())

	bone2Obj := scene.ObjectMap[bone2.ID()]
	require.NotNil(t, bone2Obj)
	parent2 := getParent(bone2Obj)
	require.NotNil(t, parent2)
	assert.Equal(t, bone.ID(), parent2.ID())
}

// TestDocumentVersionOptions writes the same document at several versions.
func TestDocumentVersionOptions(t *testing.T) {
	for _, version := range []uint32{7100, 7400, 7500, 7700} {
		doc := NewDocument(WithVersion(version), WithCompressedArrays())
		_, err := doc.NewMesh("M", cubeVertices, cubeFaces)
		require.NoError(t, err)
		data, err := doc.Bytes()
		require.NoError(t, err, "version %d", version)

		scene, err := Load(bytes.NewReader(data))
		require.NoError(t, err, "version %d", version)
		require.Len(t, scene.Meshes, 1, "version %d", version)
		assert.Equal(t, cubeVertices, scene.Meshes[0].Geometry.Vertices, "version %d", version)
	}
}

// TestDocumentRejectsBadGeometry guards the mesh validation paths.
func TestDocumentRejectsBadGeometry(t *testing.T) {
	doc := NewDocument()

	_, err := doc.NewMesh("Empty", nil, cubeFaces)
	assert.Error(t, err)

	_, err = doc.NewMesh("BadIndex", cubeVertices, [][]int{{0, 1, 99}})
	assert.Error(t, err)

	mesh, err := doc.NewMesh("OK", cubeVertices, cubeFaces)
	require.NoError(t, err)
	err = doc.SetNormals(mesh, cubeCornerNormals()[:3])
	assert.Error(t, err)
	err = doc.SetUVs(mesh, MaxUvs, cubeCornerUVs())
	assert.Error(t, err)
	err = doc.SetMaterialIndices(mesh, "Bogus", []int32{0})
	assert.Error(t, err)
}

// TestSaveSceneRoundTripCube exports a parsed scene through SaveScene.
func TestSaveSceneRoundTripCube(t *testing.T) {
	f, err := os.Open("testdata/cube.fbx")
	require.NoError(t, err)
	scene1, err := Load(f)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	var buf bytes.Buffer
	require.NoError(t, SaveScene(scene1, &buf))

	scene2, err := Load(&buf)
	require.NoError(t, err)
	compareScenes(t, scene1, scene2)
}

// TestSaveSceneRoundTripSkinnedAnim exports the rigged, animated character.
func TestSaveSceneRoundTripSkinnedAnim(t *testing.T) {
	f, err := os.Open("testdata/jyj.FBX")
	require.NoError(t, err)
	scene1, err := Load(f)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	var buf bytes.Buffer
	require.NoError(t, SaveScene(scene1, &buf))

	scene2, err := Load(&buf)
	require.NoError(t, err)
	compareScenes(t, scene1, scene2)
}

// TestSaveSceneVersionOverride re-exports a 7700 file as 7400.
func TestSaveSceneVersionOverride(t *testing.T) {
	f, err := os.Open("testdata/FBXcs.fbx")
	require.NoError(t, err)
	scene1, err := Load(f)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	var buf bytes.Buffer
	require.NoError(t, SaveScene(scene1, &buf, WithVersion(7400)))

	data := buf.Bytes()
	assert.Equal(t, uint32(7400), sourceVersionFromBytes(t, data))

	scene2, err := Load(bytes.NewReader(data))
	require.NoError(t, err)
	compareScenes(t, scene1, scene2)
}

// TestSaveSceneOnlySupported drops unmodeled object types.
func TestSaveSceneOnlySupported(t *testing.T) {
	f, err := os.Open("testdata/FBXcs.fbx")
	require.NoError(t, err)
	scene1, err := Load(f)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	var all, filtered bytes.Buffer
	require.NoError(t, SaveScene(scene1, &all))
	require.NoError(t, SaveScene(scene1, &filtered, OnlySupportedObjects()))
	assert.Less(t, filtered.Len(), all.Len(), "filtered export should be smaller")

	// geometry and animation must survive the filter
	scene2, err := Load(&filtered)
	require.NoError(t, err)
	require.NotEmpty(t, scene2.Meshes)
	require.NotEmpty(t, scene2.AnimationStacks)
}

func sourceVersionFromBytes(t *testing.T, data []byte) uint32 {
	t.Helper()
	scene, err := Load(bytes.NewReader(data))
	require.NoError(t, err)
	return sourceVersion(scene)
}
