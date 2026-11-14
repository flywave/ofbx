package ofbx

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/oakmound/oak/v2/alg/floatgeom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// FuzzLoadEncode feeds arbitrary bytes through Load and, for anything that
// parses, through Encode and SaveScene as well. Parse errors are fine; panics,
// hangs and failed re-loads of our own output are bugs.
func FuzzLoadEncode(f *testing.F) {
	seeds, _ := filepath.Glob("testdata/*.fb?")
	for _, p := range seeds {
		if data, err := os.ReadFile(p); err == nil {
			f.Add(data)
		}
	}
	// hand-made corrupt seeds
	f.Add([]byte("Kaydara FBX Binary  \x00\x1a\x00\xe8\x1c\x00\x00"))
	f.Add([]byte{})
	f.Add([]byte{0})
	f.Add(bytes.Repeat([]byte{0xff}, 64))

	f.Fuzz(func(t *testing.T, data []byte) {
		// oversized mutations make the fuzzing workers GC-thrash (or get
		// OOM-killed) without adding coverage; real files are far smaller
		if len(data) > 256<<10 {
			t.Skip("input too large for fuzzing")
		}
		scene, err := Load(bytes.NewReader(data))
		if err != nil || scene == nil {
			return
		}
		var buf bytes.Buffer
		if err := Encode(&buf, scene.RootElement, 0); err != nil {
			t.Fatalf("Encode failed on a parsed scene: %v", err)
		}
		if _, err := Load(&buf); err != nil {
			t.Fatalf("re-loading Encode output failed: %v", err)
		}
		var buf2 bytes.Buffer
		if err := SaveScene(scene, &buf2); err != nil {
			if scene.RootElement.Child("Objects") != nil {
				t.Fatalf("SaveScene failed on a parsed scene: %v", err)
			}
			return
		}
		if _, err := Load(&buf2); err != nil {
			t.Fatalf("re-loading SaveScene output failed: %v", err)
		}
	})
}

// TestLoadMalformedVertexIndexNoPanic guards the panic fixed by the bounds
// check in parseGeometry.
func TestLoadMalformedVertexIndexNoPanic(t *testing.T) {
	verts, err := NewDoubleArrayProperty([]float64{0, 0, 0, 1, 0, 0, 0, 1, 0}, false)
	require.NoError(t, err)
	idx, err := NewInt32ArrayProperty([]int32{0, 1, 9999}, false)
	require.NoError(t, err)
	el := NewElement("Geometry")
	el.AddProperty(NewInt64Property(1))
	el.AddProperty(NewStringProperty("g\x00\x01Geometry"))
	el.AddProperty(NewStringProperty("Mesh"))
	el.AddChildElement("Vertices", verts)
	el.AddChildElement("PolygonVertexIndex", idx)

	_, err = parseGeometry(&Scene{}, el)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "out of range")
}

// TestLoadTruncatedVariants truncates a real file at many offsets; none of
// the prefixes may panic or hang.
func TestLoadTruncatedVariants(t *testing.T) {
	data, err := os.ReadFile("testdata/cube.fbx")
	require.NoError(t, err)
	for _, n := range []int{0, 1, 21, 26, 27, 30, 100, 1000, 5000, len(data) - 20, len(data) - 1, len(data)} {
		_, _ = Load(bytes.NewReader(data[:n]))
	}
}

// TestLoadMutatedVariants flips bytes across a real file; the parser must
// return an error or a scene, never panic.
func TestLoadMutatedVariants(t *testing.T) {
	data, err := os.ReadFile("testdata/cube.fbx")
	require.NoError(t, err)
	for _, off := range []int{40, 100, 300, 700, 1500, 3000, 6000, 9000, 12000, 14000} {
		mut := append([]byte(nil), data...)
		for i := 0; i < 32; i++ {
			mut[(off+i*7)%len(mut)] ^= 0xff
		}
		_, _ = Load(bytes.NewReader(mut))
	}
}

// TestParseArrayCountLimit guards the decoded-size cap for array properties.
func TestParseArrayCountLimit(t *testing.T) {
	// int32 array claiming 2^29 elements (2 GiB decoded)
	p := &Property{Type: ArrayINT, Count: 1 << 29, Encoding: 0}
	p.value = NewDataView("")
	_, err := parseArrayRawInt(p)
	require.Error(t, err)

	// the same guard applies to float64 arrays
	p = &Property{Type: ArrayDOUBLE, Count: 1 << 28, Encoding: 0}
	p.value = NewDataView("")
	_, err = parseArrayRawFloat64(p)
	require.Error(t, err)
}

// TestReadElementGuards checks the tokenizer's sanity limits.
func TestReadElementGuards(t *testing.T) {
	// property count beyond the cap fails cleanly
	var buf bytes.Buffer
	buf.WriteString("Kaydara FBX Binary  \x00\x1a\x00\xe8\x1c\x00\x00")
	writeUint32(&buf, 100)               // endOffset
	writeUint32(&buf, maxElementProps+1) // propCt
	writeUint32(&buf, 0)                 // propListLen
	buf.WriteByte(1)
	buf.WriteByte('X')
	_, err := tokenize(&buf)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "properties")

	// end offset pointing before the header fails cleanly
	buf.Reset()
	buf.WriteString("Kaydara FBX Binary  \x00\x1a\x00\xe8\x1c\x00\x00")
	writeUint32(&buf, 10) // endOffset before the record header
	writeUint32(&buf, 0)
	writeUint32(&buf, 0)
	buf.WriteByte(1)
	buf.WriteByte('X')
	_, err = tokenize(&buf)
	require.Error(t, err)
}

// ---------- API coverage for previously untested paths ----------

func vec3(x, y, z float64) floatgeom.Point3 {
	return floatgeom.Point3{x, y, z}
}

// reloadDoc saves a Document and parses it back.
func reloadDoc(t *testing.T, d *Document) *Scene {
	t.Helper()
	data, err := d.Bytes()
	require.NoError(t, err)
	scene, err := Load(bytes.NewReader(data))
	require.NoError(t, err)
	return scene
}

// TestSetVertexColors covers the color layer writer.
func TestSetVertexColors(t *testing.T) {
	doc := NewDocument()
	mesh, err := doc.NewMesh("C", cubeVertices, cubeFaces)
	require.NoError(t, err)
	pts := make([]floatgeom.Point4, 0, cornerCount(cubeFaces))
	for i := 0; i < cornerCount(cubeFaces); i++ {
		pts = append(pts, floatgeom.Point4{1, 0.5, 0, 1})
	}
	require.NoError(t, doc.SetVertexColors(mesh, pts))

	scene := reloadDoc(t, doc)
	g := scene.Meshes[0].Geometry
	require.NotEmpty(t, g.Colors)
	for i := range pts {
		assert.Equal(t, pts[i], g.Colors[i], "color %d", i)
	}
}

// TestSetRotationAndScaling covers the remaining transform setters.
func TestSetRotationAndScaling(t *testing.T) {
	doc := NewDocument()
	node, err := doc.NewNode("N", "Null")
	require.NoError(t, err)
	require.NoError(t, SetRotation(node, vec3(10, 20, 30)))
	require.NoError(t, SetScaling(node, vec3(2, 3, 4)))
	require.NoError(t, SetLclDouble(node, "Visibility", 1))

	scene := reloadDoc(t, doc)
	obj := scene.ObjectMap[node.ID()]
	require.NotNil(t, obj)
	assert.Equal(t, vec3(10, 20, 30), getLocalRotation(obj))
	assert.Equal(t, vec3(2, 3, 4), getLocalScaling(obj))
}

// TestElementHelpers exercises the remaining element builders.
func TestElementHelpers(t *testing.T) {
	root := NewElement("")
	a := root.AddChildElement("A", NewStringProperty("x"))
	root.AddChildElement("B", NewInt32Property(2))

	assert.Equal(t, a, root.Child("A"))
	assert.Equal(t, []*Element{a}, root.ChildrenOf("A")) // exact id match only

	c := NewElement("C")
	root.ReplaceChild("A", c)
	assert.Nil(t, root.Child("A"))
	assert.Equal(t, c, root.Child("C"))
	require.Len(t, root.Children, 2)
	root.ReplaceChild("Nope", c) // appends when absent
	require.Len(t, root.Children, 3)

	assert.True(t, root.RemoveChild("C"))
	assert.False(t, root.RemoveChild("Missing"))
}

// TestDocumentOptions covers WithCreator, WithRawSettings and Document.Scene.
func TestDocumentOptions(t *testing.T) {
	doc := NewDocument(WithCreator("unit-test"), WithRawSettings())
	assert.NotNil(t, doc.Scene())
	assert.Equal(t, "unit-test", doc.Creator)

	mesh, err := doc.NewMesh("M", cubeVertices, cubeFaces)
	require.NoError(t, err)
	assert.NotNil(t, doc.Scene().ObjectMap[mesh.ID()])

	data, err := doc.Bytes()
	require.NoError(t, err)
	// raw settings keep the zero UnitScaleFactor (no defaults applied)
	scene, err := Load(bytes.NewReader(data))
	require.NoError(t, err)
	assert.Equal(t, float32(0), scene.Settings.UnitScaleFactor)
}

// TestRebuildProcessedLayerFallback drives appendProcessedLayers: a geometry
// without raw retention and without element layers falls back to the
// processed per-corner data.
func TestRebuildProcessedLayerFallback(t *testing.T) {
	scene := loadScene(t, "testdata/cube.fbx")
	mesh := scene.Meshes[0]
	g := mesh.Geometry

	// drop raw retention and element layers, then rebuild from processed data
	g.raw = rawGeometry{}
	g.Element().Children = nil
	normals := make([]floatgeom.Point3, 0)
	for _, f := range g.Faces {
		for range f {
			normals = append(normals, floatgeom.Point3{0, 1, 0})
		}
	}
	g.Normals = normals
	require.NoError(t, RebuildGeometryElement(mesh))

	reloaded := saveAndReload(t, scene)
	rg := reloaded.Meshes[0].Geometry
	require.NotEmpty(t, rg.Normals)
	for _, n := range rg.Normals {
		assert.Equal(t, floatgeom.Point3{0, 1, 0}, n)
	}
}

// TestGetFramerateFromTimeMode covers the full enum switch.
func TestGetFramerateFromTimeMode(t *testing.T) {
	expected := map[FrameRate]float32{
		FrameRateDefault: 1, FrameRate120: 120, FrameRate100: 100,
		FrameRate60: 60, FrameRate50: 50, FrameRate48: 48, FrameRate30: 30,
		FrameRate30Drop: 30, FrameRateNTSCDropFrame: 29.9700262,
		FrameRateNTSCFullFrame: 29.9700262, FrameRatePAL: 25,
		FrameRateCinema: 24, FrameRate1000: 1000, FrameRateCinemaND: 23.976,
		FrameRateCustom: 17.5,
	}
	for mode, want := range expected {
		assert.InDelta(t, want, GetFramerateFromTimeMode(mode, 17.5), 1e-5, "mode %d", mode)
	}
}

// TestMeshAnimations covers Animations() incl. the no-skin early return.
func TestMeshAnimations(t *testing.T) {
	// no geometry at all: must not panic, returns empty
	empty := &Mesh{}
	empty.Object.scene = &Scene{}
	assert.Empty(t, empty.Animations())

	// animated mesh resolves its stack
	doc := NewDocument()
	mesh, err := doc.NewMesh("M", cubeVertices, cubeFaces)
	require.NoError(t, err)
	stack := doc.NewAnimationStack("Take", 0, stdTimeToFbxTime(time.Second))
	layer := doc.NewAnimationLayer("L", stack)
	curve, err := doc.NewAnimationCurve("c", []time.Duration{0, time.Second}, []float32{0, 1})
	require.NoError(t, err)
	_, err = doc.NewAnimationCurveNode("n", layer, mesh, "Lcl Translation", [3]*AnimationCurve{curve, nil, nil})
	require.NoError(t, err)

	scene := reloadDoc(t, doc)
	require.NotEmpty(t, scene.Meshes[0].Animations())
}

// TestSaveSceneIdempotent saves twice; the second export must converge.
func TestSaveSceneIdempotent(t *testing.T) {
	scene := loadScene(t, "testdata/cube.fbx")
	first := saveAndReload(t, scene)
	second := saveAndReload(t, first)
	compareScenes(t, first, second)
}

// TestDefinitionsCountConsistency asserts the Definitions Count equals the
// ObjectType counts plus GlobalSettings.
func TestDefinitionsCountConsistency(t *testing.T) {
	doc := NewDocument()
	_, err := doc.NewMesh("M", cubeVertices, cubeFaces)
	require.NoError(t, err)
	_ = doc.NewMaterial("mat")

	data, err := doc.Bytes()
	require.NoError(t, err)
	scene, err := Load(bytes.NewReader(data))
	require.NoError(t, err)

	defs := scene.RootElement.Child("Definitions")
	require.NotNil(t, defs)
	var total int32
	for _, c := range defs.Children {
		if c.ID.String() != "ObjectType" {
			continue
		}
		cnt := c.Child("Count")
		require.NotNil(t, cnt)
		total += cnt.Properties[0].value.toInt32()
	}
	countEl := defs.Child("Count")
	require.NotNil(t, countEl)
	// the ObjectType entries include GlobalSettings, so their sum already
	// matches the total
	assert.Equal(t, int(total), int(countEl.Properties[0].value.toInt32()))
}

// TestBinaryLittleEndianSanity pins the header layout assumptions the guards
// rely on.
func TestBinaryLittleEndianSanity(t *testing.T) {
	var buf bytes.Buffer
	writeUint32(&buf, 0x12345678)
	assert.Equal(t, []byte{0x78, 0x56, 0x34, 0x12}, buf.Bytes())
}
